package qianqian

// 统一的数据模型：前端只认这一套
type Track struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"` // 毫秒
	Source   string `json:"source"`   // ne | qq
}

type LyricLine struct {
	Time int    `json:"time"` // 毫秒
	Text string `json:"text"`
}

type Chart struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Cover  string `json:"cover"`
	Source string `json:"source"`
}

type SearchResult struct {
	Tracks []Track `json:"tracks"`
	Source string  `json:"source"`
	Error  string  `json:"error"`
	MS     int     `json:"ms"`
}
