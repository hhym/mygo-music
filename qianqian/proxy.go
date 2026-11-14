package qianqian

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

/* 音频代理。
   前端 <audio> 直接拉三方地址会被 Referer/防盗链拦（网易云、QQ 都要特定头），
   而且跨域 m3u8 也麻烦。所以本地起一个 HTTP 服务，把目标 URL 编码进 query，
   由 Go 带上正确的头去取，再原样转发给播放器。 */

var proxyOnce sync.Once
var proxyBase string

func (a *App) ensureProxy() string {
	proxyOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return
		}
		port := ln.Addr().(*net.TCPAddr).Port
		mux := http.NewServeMux()
		mux.HandleFunc("/a", handleAudioProxy)
		srv := &http.Server{Handler: mux}
		go srv.Serve(ln)
		proxyBase = fmt.Sprintf("http://127.0.0.1:%d/a", port)
	})
	return proxyBase
}

// proxyBase 返回已启动的代理基址（未启动则为空）。
func (a *App) proxyBase() string {
	a.ensureProxy()
	return proxyBase
}

func encodeTarget(u string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(u))
}

func decodeTarget(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// 音频接口给前端 WebAudio（均衡器）用，必须允许跨源读取。Wails 页面在
// wails.localhost，音频在 127.0.0.1 的另一端口，属于跨源；不带这些头，
// createMediaElementSource 拿到的会是静音。
func allowAudioCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges, Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func handleAudioProxy(w http.ResponseWriter, r *http.Request) {
	if allowAudioCORS(w, r) {
		return
	}
	enc := r.URL.Query().Get("u")
	target, err := decodeTarget(enc)
	if err != nil || target == "" {
		http.Error(w, "bad url", 400)
		return
	}
	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, "bad url", 400)
		return
	}
	// 插件把音频转码到本地后会返回 file:// 地址（蜻蜓的 HE-AAC 就是这么处理的），
	// 这种不走 HTTP，直接从磁盘读，并且只认数据目录下的文件。
	if u.Scheme == "file" {
		serveLocalAudio(w, r, u)
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		http.Error(w, "bad url", 400)
		return
	}

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	req.Header.Set("User-Agent", uaDesktop)
	switch {
	case strings.Contains(target, "music.163.com"), strings.Contains(target, "music.126.net"):
		req.Header.Set("Referer", "https://music.163.com/")
		req.Header.Set("Cookie", "NMTID=1; os=pc; appver=8.9.70")
	case strings.Contains(target, "qqmusic"), strings.Contains(target, "gtimg"):
		req.Header.Set("Referer", "https://y.qq.com/")
	}
	// 插件给的 URL 往往需要自己的 Referer，用 h=<base64url(referer)> 传进来覆盖默认值
	if encH := r.URL.Query().Get("h"); encH != "" {
		if h, err := decodeTarget(encH); err == nil && h != "" {
			req.Header.Set("Referer", h)
		}
	}

	// Range 透传，播放器拖动进度条要靠它
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	// 点播用超时 client；直播流是无限长连接，必须用无整体超时的 client，
	// 否则 30 秒一到就被掐断（听起来像「播一会儿就停」）
	isLive := r.URL.Query().Get("r") == "1"
	var client *http.Client
	if isLive {
		client = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 15 * time.Second}}
	} else {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream error", 502)
		return
	}
	defer res.Body.Close()

	// 网易云/QQ 对无版权曲会 302 到一个 HTML 页，这里直接当失败处理
	ct := res.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		http.Error(w, "版权受限，无可用音频", http.StatusNotFound)
		return
	}

	liveMP3 := r.URL.Query().Get("r") == "1" && strings.Contains(ct, "mpeg")

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Content-Disposition"} {
		if liveMP3 && (h == "Content-Length" || h == "Content-Range") {
			continue // 直播是无限流，带长度会让浏览器一直等
		}
		if v := res.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if res.Header.Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	w.WriteHeader(res.StatusCode)

	// 电台直播流：剥掉 junk 头与 ICY 元数据，输出干净 MP3
	if liveMP3 {
		copyMP3Clean(w, res.Body, w.(flusher))
		return
	}
	io.Copy(w, res.Body)
}

// ---------------- 歌词 ----------------

type neLyricResp struct {
	Lrc struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
	Code int `json:"code"`
}

func lyricNetease(id string) ([]LyricLine, error) {
	u := "https://music.163.com/api/song/lyric?id=" + url.QueryEscape(id) + "&lv=-1&kv=-1&tv=-1"
	txt, err := getText(u, map[string]string{
		"Referer": "https://music.163.com/",
		"Cookie":  "NMTID=1; os=pc",
	})
	if err != nil {
		return nil, err
	}
	var j neLyricResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return nil, fmt.Errorf("歌词返回不是 JSON")
	}
	return parseLRC(j.Lrc.Lyric), nil
}

type qqLyricResp struct {
	Code int `json:"code"`
	Lyric string `json:"lyric"`
}

func lyricQQ(songMid string) ([]LyricLine, error) {
	u := "https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?" + url.Values{
		"songmid":     {songMid},
		"format":      {"json"},
		"inCharset":   {"utf8"},
		"outCharset":  {"utf-8"},
		"notice":      {"0"},
		"platform":    {"yqq.json"},
		"needNewCode": {"0"},
	}.Encode()
	txt, err := getText(u, map[string]string{
		"Referer": "https://y.qq.com/",
	})
	if err != nil {
		return nil, err
	}
	var j qqLyricResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return nil, fmt.Errorf("QQ 歌词返回不是 JSON")
	}
	// QQ 的 lyric 字段是 base64 存的，先解码
	if dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(j.Lyric)); err == nil {
		return parseLRC(string(dec)), nil
	}
	if dec, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(j.Lyric)); err == nil {
		return parseLRC(string(dec)), nil
	}
	return parseLRC(j.Lyric), nil
}

func parseLRC(s string) []LyricLine {
	lines := make([]LyricLine, 0, 128)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		rb := strings.Index(line, "]")
		if rb < 0 {
			continue
		}
		stamp := line[1:rb]
		text := strings.TrimSpace(line[rb+1:])
		parts := strings.Split(stamp, ":")
		if len(parts) < 2 {
			continue
		}
		var m, sec float64
		fmt.Sscanf(parts[0], "%f", &m)
		fmt.Sscanf(parts[1], "%f", &sec)
		ms := int(m*60*1000 + sec*1000)
		if text == "" {
			continue
		}
		lines = append(lines, LyricLine{Time: ms, Text: text})
	}
	return lines
}

/* 本地音频文件服务。

   只允许读「数据目录」内的文件（插件转码产物存在 <dataDir>/plugin-data/<id>/），
   别的路径一律拒绝，避免被构造成任意文件读取。支持 Range，播放器可以拖动进度。 */

func serveLocalAudio(w http.ResponseWriter, r *http.Request, u *url.URL) {
	if allowAudioCORS(w, r) {
		return
	}
	native := u.Path
	if native == "" {
		native = u.Opaque
	}
	// file:///C:/x 在 Windows 上是 /C:/x，盘符前的斜杠要去掉；Unix 的 /home/x 保持原样
	if len(native) >= 3 && native[0] == '/' && native[2] == ':' {
		native = native[1:]
	}
	native = strings.ReplaceAll(native, "/", string(os.PathSeparator))

	abs, err := filepath.Abs(native)
	if err != nil {
		http.Error(w, "bad path", 400)
		return
	}
	// 放行范围：数据目录（插件转码产物）+ 用户授权过的本地音乐目录
	if !pathAllowed(abs) {
		http.Error(w, "forbidden", 403)
		return
	}
	fi, err := os.Stat(abs)
	if err != nil || fi.IsDir() {
		http.Error(w, "not found", 404)
		return
	}

	ext := strings.ToLower(filepath.Ext(abs))
	ct := "audio/mpeg"
	switch ext {
	case ".m4a", ".mp4", ".aac":
		ct = "audio/mp4"
	case ".ogg", ".opus":
		ct = "audio/ogg"
	case ".flac":
		ct = "audio/flac"
	case ".wav":
		ct = "audio/wav"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Accept-Ranges", "bytes")

	f, err := os.Open(abs)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	defer f.Close()

	// Range 交给标准库的 ServeContent 处理
	http.ServeContent(w, r, filepath.Base(abs), fi.ModTime(), f)
}

// fileURL 把本地路径转成 file:// 地址。Windows 的 C:\x 要写成 file:///C:/x，
// serveLocalAudio 那边会把盘符前的斜杠去掉。
func fileURL(abs string) string {
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}
