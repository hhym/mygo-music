package qianqian

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	qqSearchAPI = "https://c.y.qq.com/soso/fcgi-bin/search_for_qq_cp"
	qqMusicuAPI = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	qqTopList   = "https://c.y.qq.com/v8/fcg-bin/fcg_v8_toplist_cp.fcg"
)

type qqSearchResp struct {
	Code int `json:"code"`
	Data struct {
		Song struct {
			List []struct {
				SongMid   string `json:"songmid"`
				SongName  string `json:"songname"`
				MediaMid  string `json:"media_mid"`
				Interval  int    `json:"interval"`
				AlbumMid  string `json:"albummid"`
				AlbumName string `json:"albumname"`
				Singer    []struct {
					Name string `json:"name"`
					Mid  string `json:"mid"`
				} `json:"singer"`
			} `json:"list"`
		} `json:"song"`
	} `json:"data"`
}

func qqCover(albumMid string) string {
	if albumMid == "" {
		return ""
	}
	return "https://y.gtimg.cn/music/photo_new/T002R300x300M000" + albumMid + ".jpg"
}

func searchQQ(query string, limit int) ([]Track, error) {
	u := qqSearchAPI + "?" + url.Values{
		"remoteplace": {"txt.mqq.all"},
		"w":           {query},
		"format":      {"json"},
		"n":           {fmt.Sprint(limit)},
		"p":           {"1"},
		"platform":    {"yqq.json"},
		"needNewCode": {"0"},
	}.Encode()

	txt, err := getText(u, map[string]string{
		"Referer":    "https://y.qq.com/",
		"User-Agent": uaMobile,
	})
	if err != nil {
		return nil, err
	}
	var j qqSearchResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return nil, fmt.Errorf("QQ 返回不是 JSON")
	}
	if j.Code != 0 {
		return nil, fmt.Errorf("QQ 接口错误 %d", j.Code)
	}
	out := make([]Track, 0, len(j.Data.Song.List))
	for _, s := range j.Data.Song.List {
		if s.SongMid == "" {
			continue
		}
		names := make([]string, 0, len(s.Singer))
		for _, a := range s.Singer {
			names = append(names, a.Name)
		}
		media := s.MediaMid
		if media == "" {
			media = s.SongMid
		}
		qqMediaCache[s.SongMid] = media
		out = append(out, Track{
			ID:       "qq:" + s.SongMid,
			Name:     s.SongName,
			Artist:   strings.Join(names, "/"),
			Album:    s.AlbumName,
			Cover:    qqCover(s.AlbumMid),
			Duration: s.Interval * 1000,
			Source:   "qq",
		})
	}
	return out, nil
}

// qqMediaCache 记 songmid → media_mid 的对应，取流时 filename 要用 media_mid
var qqMediaCache = map[string]string{}

type qqVkeyResp struct {
	Code  int `json:"code"`
	Req0  struct {
		Code int `json:"code"`
		Data struct {
			Midurlinfo []struct {
				Purl     string `json:"purl"`
				Filename string `json:"filename"`
			} `json:"midurlinfo"`
			Sip []string `json:"sip"`
		} `json:"data"`
	} `json:"req_0"`
}

func qqStreamURL(songMid string) (string, error) {
	media := qqMediaCache[songMid]
	if media == "" {
		media = songMid
	}
	body := map[string]interface{}{
		"comm": map[string]interface{}{"ct": 24, "cv": 0, "uin": "0", "format": "json"},
		"req_0": map[string]interface{}{
			"module": "vkey.GetVkeyServer",
			"method": "CgiGetVkey",
			"param": map[string]interface{}{
				"guid":     "10000",
				"songmid":  []string{songMid},
				"songtype": []int{0},
				"uin":      "0",
				"loginflag": 1,
				"platform": "24",
				"filename": []string{"M500" + media + ".mp3"},
			},
		},
	}
	txt, err := postJSON(qqMusicuAPI, body, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return "", err
	}
	var j qqVkeyResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return "", fmt.Errorf("QQ 取流返回不是 JSON")
	}
	if j.Req0.Code != 0 || len(j.Req0.Data.Midurlinfo) == 0 {
		return "", fmt.Errorf("QQ 取流失败 code=%d", j.Req0.Code)
	}
	purl := j.Req0.Data.Midurlinfo[0].Purl
	if purl == "" {
		return "", fmt.Errorf("这首歌 QQ 侧没有可播地址")
	}
	sip := "http://ws.stream.qqmusic.qq.com/"
	if len(j.Req0.Data.Sip) > 0 {
		sip = j.Req0.Data.Sip[0]
	}
	if strings.HasPrefix(purl, "http") {
		return purl, nil
	}
	return sip + purl, nil
}

// ---------------- QQ 榜单 ----------------

// QQ 榜单 topid：26 热歌 / 27 新歌 / 62 飙升 / 4 流行指数 / 6 内地 / 29 欧美 ...
var qqCharts = []Chart{
	{ID: "qqc:26", Name: "热歌榜", Source: "qq"},
	{ID: "qqc:62", Name: "飙升榜", Source: "qq"},
	{ID: "qqc:27", Name: "新歌榜", Source: "qq"},
	{ID: "qqc:4", Name: "流行指数榜", Source: "qq"},
	{ID: "qqc:6", Name: "内地榜", Source: "qq"},
	{ID: "qqc:29", Name: "欧美榜", Source: "qq"},
	{ID: "qqc:23", Name: "华语金曲榜", Source: "qq"},
}

type qqTopResp struct {
	Code int `json:"code"`
	Topinfo struct {
		TopID   string `json:"topID"`
		Title   string `json:"ListName"`
		PicURL  string `json:"pic_v12"`
	} `json:"topinfo"`
	Songlist []struct {
		Data struct {
			SongMid   string `json:"songmid"`
			SongName  string `json:"songname"`
			MediaMid  string `json:"media_mid"`
			Interval  int    `json:"interval"`
			AlbumMid  string `json:"albummid"`
			AlbumName string `json:"albumname"`
			Singer    []struct {
				Name string `json:"name"`
			} `json:"singer"`
		} `json:"data"`
	} `json:"songlist"`
}

func qqChartTracks(topid string) ([]Track, error) {
	u := qqTopList + "?" + url.Values{
		"topid":           {topid},
		"format":          {"json"},
		"inCharset":       {"utf8"},
		"outCharset":      {"utf-8"},
		"notice":          {"0"},
		"platform":        {"yqq.json"},
		"needNewCode":     {"0"},
		"tpl":             {"3"},
		"page":            {"detail"},
		"type":            {"top"},
		"song_begin":      {"0"},
		"song_num":        {"40"},
	}.Encode()
	txt, err := getText(u, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return nil, err
	}
	var j qqTopResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return nil, fmt.Errorf("QQ 榜单返回不是 JSON")
	}
	out := make([]Track, 0, len(j.Songlist))
	for _, x := range j.Songlist {
		s := x.Data
		if s.SongMid == "" {
			continue
		}
		names := make([]string, 0, len(s.Singer))
		for _, a := range s.Singer {
			names = append(names, a.Name)
		}
		media := s.MediaMid
		if media == "" {
			media = s.SongMid
		}
		qqMediaCache[s.SongMid] = media
		out = append(out, Track{
			ID:       "qq:" + s.SongMid,
			Name:     s.SongName,
			Artist:   strings.Join(names, "/"),
			Album:    s.AlbumName,
			Cover:    qqCover(s.AlbumMid),
			Duration: s.Interval * 1000,
			Source:   "qq",
		})
	}
	return out, nil
}
