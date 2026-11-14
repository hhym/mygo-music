package qianqian

/* 网络电台：直接沿用千千静听插件里的那批公开直播源。
   这些是「真·网络收音机」，流是无限的，播的是当前正在放的内容。 */

type Station struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Logo   string `json:"logo"`
	URL    string `json:"url"`
	Region string `json:"region"`
}

const somafmImg = "https://somafm.com/img3/"

var radioStations = []Station{
	{ID: "radio:qt_pop", Name: "华语流行台", Desc: "蜻蜓FM · 华语流行直播", Region: "中国", URL: "https://lhttp.qtfm.cn/live/4915/64k.mp3", Logo: somafmImg + "poptron-400.jpg"},
	{ID: "radio:qt_city", Name: "城市音乐台", Desc: "蜻蜓FM · 城市音乐直播", Region: "中国", URL: "https://lhttp.qtfm.cn/live/4930/64k.mp3", Logo: somafmImg + "indiepop-400.jpg"},
	{ID: "radio:qt_story", Name: "华语音乐台", Desc: "蜻蜓FM · 华语音乐直播", Region: "中国", URL: "https://lhttp.qtfm.cn/live/4911/64k.mp3", Logo: somafmImg + "deephouse-400.jpg"},
	{ID: "radio:qt_more", Name: "华语广播台", Desc: "蜻蜓FM · 华语广播直播", Region: "中国", URL: "https://lhttp.qtfm.cn/live/4916/64k.mp3", Logo: somafmImg + "poptron-400.jpg"},
	{ID: "radio:fip", Name: "FIP 法国音乐台", Desc: "Radio France · 现代流行", Region: "法国", URL: "https://icecast.radiofrance.fr/fip-midfi.mp3", Logo: somafmImg + "suburbsofgoa-400.jpg"},
	{ID: "radio:soma", Name: "SomaFM 音乐台", Desc: "SomaFM · 英文驰放", Region: "美国", URL: "https://ice1.somafm.com/groovesalad-128-mp3", Logo: somafmImg + "groovesalad-400.jpg"},
	{ID: "radio:soma2", Name: "SomaFM 环境台", Desc: "SomaFM · 氛围电子", Region: "美国", URL: "https://ice1.somafm.com/dronezone-128-mp3", Logo: somafmImg + "dronezone-400.jpg"},
	{ID: "radio:nts", Name: "NTS Radio", Desc: "NTS · 英伦地下/世界音乐", Region: "英国", URL: "https://stream-relay-geo.ntslive.net/stream", Logo: somafmImg + "beatblender-400.jpg"},
	{ID: "radio:rp", Name: "Radio Paradise", Desc: "Mix · 精选综合", Region: "美国", URL: "https://stream.radioparadise.com/mp3-192", Logo: somafmImg + "fluid-400.jpg"},
	{ID: "radio:classic", Name: "Classic FM 古典", Desc: "UK · 古典音乐", Region: "英国", URL: "https://media-ice.musicradio.com/ClassicFMMP3", Logo: somafmImg + "lush-400.jpg"},
	{ID: "radio:antenne", Name: "Antenne 巴伐利亚", Desc: "DE · 流行", Region: "德国", URL: "https://s5-webradio.antenne.de/antenne/stream/mp3", Logo: somafmImg + "covers-400.jpg"},
}

func stationByID(id string) *Station {
	for i := range radioStations {
		if radioStations[i].ID == id {
			return &radioStations[i]
		}
	}
	return nil
}

// Stations 返回电台列表
func (a *App) Stations() []Station {
	return radioStations
}
