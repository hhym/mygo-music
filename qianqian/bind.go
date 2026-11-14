package qianqian

import "errors"

// API 是给前端用的绑定层。千千静听原来的方法返回 (值, 错误串)，
// MyGo 的绑定只收 (值, error) 或单值，所以这里统一包一层：
// 错误串变成 error，前端 catch 到的仍是同一句中文提示。
type API struct{ *App }

func NewAPI(a *App) API { return API{a} }

func errOf(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

// Quit 迷你卡右上角 ×：直接退出进程，不是把窗口还原。
func (x API) Quit() { x.App.Quit() }

func (x API) OpenExternal(url string) string {
	return x.App.OpenExternal(url)
}

func (x API) SetMiniMode(on bool) string {
	return x.App.SetMiniMode(on)
}

func (x API) Search(query string, sources []string) SearchResult {
	return x.App.Search(query, sources)
}

func (x API) Charts() []Chart {
	return x.App.Charts()
}

func (x API) ChartTracks(id string) ([]Track, error) {
	v, e := x.App.ChartTracks(id)
	return v, errOf(e)
}

func (x API) StreamURL(id string) (string, error) {
	v, e := x.App.StreamURL(id)
	return v, errOf(e)
}

func (x API) Lyric(id string) ([]LyricLine, error) {
	v, e := x.App.Lyric(id)
	return v, errOf(e)
}

func (x API) Library() []Track {
	return x.App.Library()
}

func (x API) ToggleLibrary(t Track) []Track {
	return x.App.ToggleLibrary(t)
}

func (x API) InLibrary(id string) bool {
	return x.App.InLibrary(id)
}

func (x API) HttpRequest(rawURL, method string, headers map[string]string, body string, manualRedirect bool) HttpResult {
	return x.App.HttpRequest(rawURL, method, headers, body, manualRedirect)
}

func (x API) PluginPlayURL(rawURL, referer string) (string, error) {
	v, e := x.App.PluginPlayURL(rawURL, referer)
	return v, errOf(e)
}

func (x API) PickLocalDir() (string, error) {
	v, e := x.App.PickLocalDir()
	return v, errOf(e)
}

func (x API) LocalDirs() []string {
	return x.App.LocalDirs()
}

func (x API) ForgetLocalDir(dir string) []string {
	return x.App.ForgetLocalDir(dir)
}

func (x API) ScanLocalMusic(dir string) LocalScan {
	return x.App.ScanLocalMusic(dir)
}

func (x API) Plugins() ([]Plugin, error) {
	v, e := x.App.Plugins()
	return v, errOf(e)
}

func (x API) InstallPluginText(code string) PickResult {
	return x.App.InstallPluginText(code)
}

func (x API) InstallPluginFile(path string) PickResult {
	return x.App.InstallPluginFile(path)
}

func (x API) InstallPluginURL(rawURL string) PickResult {
	return x.App.InstallPluginURL(rawURL)
}

func (x API) SetPluginEnabled(id string, on bool) ([]Plugin, error) {
	v, e := x.App.SetPluginEnabled(id, on)
	return v, errOf(e)
}

func (x API) RemovePlugin(id string) ([]Plugin, error) {
	v, e := x.App.RemovePlugin(id)
	return v, errOf(e)
}

func (x API) PluginSource(id string) (string, error) {
	v, e := x.App.PluginSource(id)
	return v, errOf(e)
}

func (x API) PluginExists(id string) bool {
	return x.App.PluginExists(id)
}

func (x API) PluginDir(id string) string {
	return x.App.PluginDir(id)
}

func (x API) PluginFile(id, rel string) (string, error) {
	v, e := x.App.PluginFile(id, rel)
	return v, errOf(e)
}

func (x API) PluginStorage(id string) (string, error) {
	v, e := x.App.PluginStorage(id)
	return v, errOf(e)
}

func (x API) SavePluginStorage(id, data string) string {
	return x.App.SavePluginStorage(id, data)
}

func (x API) PickPluginZip() PickResult {
	return x.App.PickPluginZip()
}

func (x API) PickPluginDir() PickResult {
	return x.App.PickPluginDir()
}

func (x API) RunProcess(bin string, args []string, timeoutSec int) (ProcessResult, error) {
	v, e := x.App.RunProcess(bin, args, timeoutSec)
	return v, errOf(e)
}

func (x API) PluginSettings(id string) (map[string]string, error) {
	v, e := x.App.PluginSettings(id)
	return v, errOf(e)
}

func (x API) SavePluginSetting(id, key, value string) (map[string]string, error) {
	v, e := x.App.SavePluginSetting(id, key, value)
	return v, errOf(e)
}

func (x API) PodcastSearch(term string) ([]Show, error) {
	v, e := x.App.PodcastSearch(term)
	return v, errOf(e)
}

func (x API) PodcastEpisodes(feedURL string) ([]Track, error) {
	v, e := x.App.PodcastEpisodes(feedURL)
	return v, errOf(e)
}

func (x API) Stations() []Station {
	return x.App.Stations()
}

func (x API) GetSettings() Settings {
	return x.App.GetSettings()
}

func (x API) SaveSettings(s Settings) (Settings, error) {
	v, e := x.App.SaveSettings(s)
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

func (x API) WinMoveBy(dx, dy int) string {
	return x.App.WinMoveBy(dx, dy)
}
