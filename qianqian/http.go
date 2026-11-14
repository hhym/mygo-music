package qianqian

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	uaDesktop = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	uaMobile  = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
)

var httpClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return nil
	},
}

// getText 发一个 GET，返回 body 文本
func getText(rawURL string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", uaDesktop)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// postJSON 发一个 JSON POST
func postJSON(rawURL string, body interface{}, headers map[string]string) (string, error) {
	var r io.Reader
	switch v := body.(type) {
	case string:
		r = strings.NewReader(v)
	default:
		b, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		r = strings.NewReader(string(b))
	}
	req, err := http.NewRequest("POST", rawURL, r)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", uaDesktop)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func msToDuration(v float64) int {
	if v <= 0 {
		return 0
	}
	if v < 3000 {
		return int(v * 1000)
	}
	return int(v)
}

func qs(v url.Values) string { return v.Encode() }

// ---------------- 网易云 ----------------

type neSearchResp struct {
	Result struct {
		Songs []struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			Duration int    `json:"duration"`
			Artists  []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Name      string `json:"name"`
				PicURL    string `json:"picUrl"`
				ArtistIDs []int64
			} `json:"album"`
	} `json:"songs"`
	} `json:"result"`
	Code int `json:"code"`
}

func searchNetease(query string, limit int) ([]Track, error) {
	u := "https://music.163.com/api/search/get/web?" + qs(url.Values{
		"s": {query}, "type": {"1"}, "offset": {"0"}, "limit": {fmt.Sprint(limit)},
	})
	txt, err := getText(u, map[string]string{
		"Referer": "https://music.163.com/",
		"Cookie":  "NMTID=1; os=pc; appver=8.9.70",
	})
	if err != nil {
		return nil, err
	}
	var j neSearchResp
	if err := json.Unmarshal([]byte(txt), &j); err != nil {
		return nil, fmt.Errorf("网易云返回不是 JSON")
	}
	out := make([]Track, 0, len(j.Result.Songs))
	for _, s := range j.Result.Songs {
		names := make([]string, 0, len(s.Artists))
		for _, a := range s.Artists {
			names = append(names, a.Name)
		}
		out = append(out, Track{
			ID:       fmt.Sprintf("ne:%d", s.ID),
			Name:     s.Name,
			Artist:   strings.Join(names, "/"),
			Album:    s.Album.Name,
			Cover:    s.Album.PicURL,
			Duration: msToDuration(float64(s.Duration)),
			Source:   "ne",
		})
	}
	return out, nil
}
