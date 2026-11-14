package qianqian

import (
	"encoding/json"
	"errors"
)

// API 是给前端用的绑定层。
//
// 两件事必须在这里做：
//  1. 千千静听的方法返回 (值, 错误串)，MyGo 只认 (值, error)，错误串要包成 error；
//  2. 前端经 Proxy 转发，参数偶尔会多裹一层数组，这里统一用 any 收再宽松还原，
//     避免 invalid argument: cannot unmarshal ... 之类的小事故。
type API struct{ *App }

func NewAPI(a *App) API { return API{a} }

// unwrap 剥掉多余的包裹：单元素数组取第一个，单键 map 取值。
func unwrap(v any) any {
	switch t := v.(type) {
	case []any:
		if len(t) == 1 {
			return unwrap(t[0])
		}
	case map[string]any:
		if len(t) == 1 {
			for _, x := range t {
				return unwrap(x)
			}
		}
	}
	return v
}

// as 用 JSON 中转把任意值还原成目标类型，失败给零值。
func as[T any](v any) T {
	var out T
	b, err := json.Marshal(unwrap(v))
	if err != nil {
		return out
	}
	_ = json.Unmarshal(b, &out)
	return out
}

func asStr(v any) string { return as[string](v) }

func errOf(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func (x API) OpenExternal(url any) string {
	return x.App.OpenExternal(asStr(url))
}

func (x API) SetMiniMode(on any) string {
	return x.App.SetMiniMode(as[bool](on))
}

func (x API) Search(query any, sources any) SearchResult {
	return x.App.Search(asStr(query), as[[]string](sources))
}

func (x API) Charts() []Chart {
	return x.App.Charts()
}

func (x API) ChartTracks(id any) ([]Track, error) {
	v, e := x.App.ChartTracks(asStr(id))
	return v, errOf(e)
}

func (x API) StreamURL(id any) (string, error) {
	v, e := x.App.StreamURL(asStr(id))
	return v, errOf(e)
}

func (x API) Lyric(id any) ([]LyricLine, error) {
	v, e := x.App.Lyric(asStr(id))
	return v, errOf(e)
}

func (x API) Library() []Track {
	return x.App.Library()
}

func (x API) ToggleLibrary(t any) []Track {
	return x.App.ToggleLibrary(as[Track](t))
}

func (x API) InLibrary(id any) bool {
	return x.App.InLibrary(asStr(id))
}

func (x API) HttpRequest(rawURL any, method any, headers any, body any, manualRedirect any) HttpResult {
	return x.App.HttpRequest(asStr(rawURL), asStr(method), as[map[string]string](headers), asStr(body), as[bool](manualRedirect))
}

func (x API) PluginPlayURL(rawURL any, referer any) (string, error) {
	v, e := x.App.PluginPlayURL(asStr(rawURL), asStr(referer))
	return v, errOf(e)
}

func (x API) PickLocalDir() (string, error) {
	v, e := x.App.PickLocalDir()
	return v, errOf(e)
}

func (x API) LocalDirs() []string {
	return x.App.LocalDirs()
}

func (x API) ForgetLocalDir(dir any) []string {
	return x.App.ForgetLocalDir(asStr(dir))
}

func (x API) ScanLocalMusic(dir any) LocalScan {
	return x.App.ScanLocalMusic(asStr(dir))
}

func (x API) Plugins() ([]Plugin, error) {
	v, e := x.App.Plugins()
	return v, errOf(e)
}

func (x API) InstallPluginText(code any) PickResult {
	return x.App.InstallPluginText(asStr(code))
}

func (x API) InstallPluginFile(path any) PickResult {
	return x.App.InstallPluginFile(asStr(path))
}

func (x API) InstallPluginURL(rawURL any) PickResult {
	return x.App.InstallPluginURL(asStr(rawURL))
}

func (x API) SetPluginEnabled(id any, on any) ([]Plugin, error) {
	v, e := x.App.SetPluginEnabled(asStr(id), as[bool](on))
	return v, errOf(e)
}

func (x API) RemovePlugin(id any) ([]Plugin, error) {
	v, e := x.App.RemovePlugin(asStr(id))
	return v, errOf(e)
}

func (x API) PluginSource(id any) (string, error) {
	v, e := x.App.PluginSource(asStr(id))
	return v, errOf(e)
}

func (x API) PluginExists(id any) bool {
	return x.App.PluginExists(asStr(id))
}

func (x API) PluginDir(id any) string {
	return x.App.PluginDir(asStr(id))
}

func (x API) PluginFile(id any, rel any) (string, error) {
	v, e := x.App.PluginFile(asStr(id), asStr(rel))
	return v, errOf(e)
}

func (x API) PluginStorage(id any) (string, error) {
	v, e := x.App.PluginStorage(asStr(id))
	return v, errOf(e)
}

func (x API) SavePluginStorage(id any, data any) string {
	return x.App.SavePluginStorage(asStr(id), asStr(data))
}

func (x API) PickPluginZip() PickResult {
	return x.App.PickPluginZip()
}

func (x API) PickPluginDir() PickResult {
	return x.App.PickPluginDir()
}

func (x API) RunProcess(bin any, args any, timeoutSec any) (ProcessResult, error) {
	v, e := x.App.RunProcess(asStr(bin), as[[]string](args), as[int](timeoutSec))
	return v, errOf(e)
}

func (x API) PluginSettings(id any) (map[string]string, error) {
	v, e := x.App.PluginSettings(asStr(id))
	return v, errOf(e)
}

func (x API) SavePluginSetting(id any, key any, value any) (map[string]string, error) {
	v, e := x.App.SavePluginSetting(asStr(id), asStr(key), asStr(value))
	return v, errOf(e)
}

func (x API) PodcastSearch(term any) ([]Show, error) {
	v, e := x.App.PodcastSearch(asStr(term))
	return v, errOf(e)
}

func (x API) PodcastEpisodes(feedURL any) ([]Track, error) {
	v, e := x.App.PodcastEpisodes(asStr(feedURL))
	return v, errOf(e)
}

func (x API) Stations() []Station {
	return x.App.Stations()
}

func (x API) GetSettings() Settings {
	return x.App.GetSettings()
}

func (x API) SaveSettings(s any) (Settings, error) {
	v, e := x.App.SaveSettings(as[Settings](s))
	return v, errOf(e)
}

func (x API) ResetSettings() (Settings, error) {
	v, e := x.App.ResetSettings()
	return v, errOf(e)
}

func (x API) OpenDataDir() (string, error) {
	v, e := x.App.OpenDataDir()
	return v, errOf(e)
}

func (x API) ClearLibrary() ([]Track, error) {
	v, e := x.App.ClearLibrary()
	return v, errOf(e)
}

func (x API) WinMoveBy(dx any, dy any) string {
	return x.App.WinMoveBy(as[int](dx), as[int](dy))
}

// Quit 迷你卡右上角 ×：直接退出进程。
func (x API) WinMinimize() { x.App.WinMinimize() }

func (x API) WinClose() { x.App.WinClose() }

func (x API) Quit() { x.App.Quit() }
