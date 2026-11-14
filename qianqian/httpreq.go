package qianqian

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

/* 插件运行在 WebView 里，直接 fetch 三方接口会被 CORS 拦掉。
   所有插件网络请求都绕到后端来发，顺带把 UA / Referer 这些头补上。 */

type HttpResult struct {
	Status  int               `json:"status"`
	BodyB64 string            `json:"bodyB64"`
	Body    string            `json:"body"`
	Err     string            `json:"err"`
	Headers map[string]string `json:"headers"`
}

// 响应头要回传给插件：不少平台（酷我、B站）靠 Set-Cookie 或自定义头下发凭证，
// 插件拿不到头就没法继续拿数据。
func collectHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vs := range h {
		out[strings.ToLower(k)] = strings.Join(vs, "\n")
	}
	return out
}

func (a *App) HttpRequest(rawURL, method string, headers map[string]string, body string, manualRedirect bool) HttpResult {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return HttpResult{Err: "只支持 http/https 地址"}
	}
	if method == "" {
		method = "GET"
	}
	var rdr io.Reader
	if body != "" && method != "GET" && method != "HEAD" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(strings.ToUpper(method), rawURL, rdr)
	if err != nil {
		return HttpResult{Err: "请求构造失败：" + err.Error()}
	}
	req.Header.Set("User-Agent", uaDesktop)
	for k, v := range headers {
		if k == "" || v == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	// 插件显式要求 redirect: 'manual' 时不要自动跟随：
	// QQ 登录靠 check_sig 返回的 302 和 Location 拿授权票据，跟随后就看不到 302 了
	if manualRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return HttpResult{Err: "请求失败：" + err.Error()}
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	if err != nil {
		return HttpResult{Status: res.StatusCode, Err: "读取响应失败：" + err.Error()}
	}
	// 文本类响应给原文；图片/音频这类二进制响应额外给 base64，
	// 否则插件里 `(await fetch(...)).arrayBuffer()` 拿到的字节是错的
	ct := res.Header.Get("Content-Type")
	resp := HttpResult{Status: res.StatusCode, Body: string(b), Headers: collectHeaders(res.Header)}
	// 判断「文本还是二进制」很容易翻车：QQ 的二维码接口连 Content-Type 都不给，
	// 只看头会把它当文本，字节就全坏了。所以小响应一律额外带上 base64 原文，
	// 插件要 arrayBuffer 时直接用它，不必猜。
	// 大响应（>1MB，基本是音频/图片包体）才只按类型给一种。
	if len(b) <= 1<<20 {
		resp.BodyB64 = base64.StdEncoding.EncodeToString(b)
	} else if !isTextualType(ct) || !utf8.Valid(b) {
		resp.BodyB64 = base64.StdEncoding.EncodeToString(b)
	}
	return resp
}

// isTextualType 判断响应是不是能安全当作文本交给前端。
func isTextualType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return true
	}
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "urlencoded")
}

// PluginPlayURL 把插件拿到的音频地址包装成本地代理地址。
// 插件管的站点往往要自己的 Referer，所以一并带过去。
func (a *App) PluginPlayURL(rawURL, referer string) (string, string) {
	base := a.ensureProxy()
	if base == "" {
		return "", "本地代理没起来"
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "插件返回的地址不合法"
	}
	// 插件转码后给的是 file:// 本地文件，也走代理（代理内部只读数据目录）
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "file" {
		return "", "插件返回的地址不合法"
	}
	out := base + "?u=" + encodeTarget(rawURL)
	if referer != "" {
		out += "&h=" + encodeTarget(referer)
	}
	return out, ""
}
