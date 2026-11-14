package qianqian

import (
	"context"

	"github.com/egoist/mygo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Quit 供迷你卡右上角的关闭按钮用：直接退出进程，不是把窗口还原回桌面。
// 关窗前会走 OnBeforeClose，日志里能看到「窗口被要求关闭」这一行。
func (a *App) Quit() {
	mygo.App.Quit()
}

// App 是 Wails 绑定到前端的入口。所有首字母大写的方法前端都能直接调。
type App struct {
	win    *mygo.Window
	ctx    context.Context
	mu     sync.Mutex
	lib    []Track
	libPth string
}

func NewApp() *App { return &App{} }

// AttachWindow 把 MyGo 的窗口交给 App：迷你模式要改它的尺寸与置顶。
func (a *App) AttachWindow(w *mygo.Window) { a.win = w }

// appLog 往数据目录下追加一行日志（带毫秒时间戳）。出错就静默返回，绝不影响主流程。
func appLog(name string, msg string) {
	f, err := os.OpenFile(filepath.Join(dataDir(), name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, time.Now().Format("2006-01-02 15:04:05.000"), msg)
}

// OpenExternal 用系统默认浏览器打开外部网页（设置页里的平台主页入口）。
func (a *App) OpenExternal(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "只允许打开 http/https 地址"
	}
	if err := mygo.Shell.OpenExternal(url); err != nil {
		return err.Error()
	}
	return ""
}

// SetMiniMode 切换「迷你播放器」窗口形态：缩成一条小窗并置顶，方便挂在桌面角落看歌词。
// 具体 Win32 细节在 win_floating.go（miniOn / miniOff）里，顺序是「先缩尺寸、再按尺寸裁轮廓」。
func (a *App) SetMiniMode(on bool) string {
	if a.ctx == nil {
		return "窗口还没准备好"
	}
	if on {
		return a.miniOn()
	}
	return a.miniOff()
}

// Start 在窗口建好后调一次，把目录、收藏、代理准备好。
func (a *App) Start() { a.startup(context.Background()) }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureProxy()
	a.libPth = filepath.Join(dataDir(), "library.json")
	a.loadLibrary()
	// 本地乐库：把上次授权过的音乐目录装进代理白名单
	setLocalRoots(a.loadSettings().LocalDirs)
}

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".hulu-music-desktop")
	os.MkdirAll(dir, 0o755)
	return dir
}

func (a *App) loadLibrary() {
	b, err := os.ReadFile(a.libPth)
	if err != nil {
		return
	}
	json.Unmarshal(b, &a.lib)
}

func (a *App) saveLibrary() {
	b, _ := json.MarshalIndent(a.lib, "", "  ")
	os.WriteFile(a.libPth, b, 0o644)
}

// ---------- 搜索 ----------

// Search 聚合搜索，sources 可传 ["ne","qq"]，为空则两个都搜。
func (a *App) Search(query string, sources []string) SearchResult {
	t0 := time.Now()
	if strings.TrimSpace(query) == "" {
		return SearchResult{Tracks: []Track{}}
	}
	// 音源已全部改为插件提供（前端在 WebView 里执行插件），
	// 内置的 ne/qq 只保留给 StreamURL/Lyric 用来播旧收藏，不再参与搜索。
	if len(sources) == 0 {
		return SearchResult{Tracks: []Track{}, Error: "音源已改为插件：请装一个 MyGo 插件后在「插件」页搜索", Source: "plugin"}
	}

	type chunk struct {
		src    string
		tracks []Track
		err    error
	}
	ch := make(chan chunk, len(sources))
	for _, s := range sources {
		go func(src string) {
			switch src {
			case "ne":
				t, err := searchNetease(query, 20)
				ch <- chunk{"ne", t, err}
			case "qq":
				t, err := searchQQ(query, 20)
				ch <- chunk{"qq", t, err}
			default:
				ch <- chunk{src, nil, fmt.Errorf("未知源 %s", src)}
			}
		}(s)
	}

	merged := make([]Track, 0, 40)
	var errs []string
	for i := 0; i < len(sources); i++ {
		c := <-ch
		if c.err != nil {
			errs = append(errs, c.src+": "+c.err.Error())
			continue
		}
		merged = append(merged, c.tracks...)
	}
	return SearchResult{
		Tracks: merged,
		Error:  strings.Join(errs, " | "),
		MS:     int(time.Since(t0).Milliseconds()),
	}
}

// ---------- 榜单 ----------

func (a *App) Charts() []Chart {
	out := make([]Chart, 0, len(qqCharts))
	for _, c := range qqCharts {
		cc := c
		cc.Cover = "https://y.gtimg.cn/music/photo_new/T003R300x300M000" + strings.TrimPrefix(c.ID, "qqc:") + ".jpg"
		out = append(out, cc)
	}
	return out
}

func (a *App) ChartTracks(id string) ([]Track, string) {
	topid := strings.TrimPrefix(id, "qqc:")
	t, err := qqChartTracks(topid)
	if err != nil {
		return []Track{}, err.Error()
	}
	return t, ""
}

// ---------- 取流 ----------

// StreamURL 返回一个本地代理地址，前端直接丢给 <audio>。
func (a *App) StreamURL(id string) (string, string) {
	base := a.ensureProxy()
	if base == "" {
		return "", "本地代理没起来"
	}
	var target string
	var err error
	live := false

	switch {
	case strings.HasPrefix(id, "qq:"):
		target, err = qqStreamURL(strings.TrimPrefix(id, "qq:"))
	case strings.HasPrefix(id, "ne:"):
		neID := strings.TrimPrefix(id, "ne:")
		target = "https://music.163.com/song/media/outer/url?id=" + neID + ".mp3"
	case strings.HasPrefix(id, "radio:"):
		st := stationByID(id)
		if st == nil {
			return "", "没有这个电台"
		}
		target = st.URL
		live = true
	case strings.HasPrefix(id, "loc:"):
		p, e := localTrackPath(id)
		if e != nil {
			return "", e.Error()
		}
		if !pathAllowed(p) {
			return "", "这个文件不在已授权的音乐目录里，重选一次文件夹"
		}
		target = fileURL(p)
	case strings.HasPrefix(id, "pode:"):
		target, err = decodeTarget(strings.TrimPrefix(id, "pode:"))
	default:
		return "", "不认识的曲目 id"
	}
	if err != nil {
		return "", err.Error()
	}
	suffix := ""
	if live {
		suffix = "&r=1" // 直播流要过 MP3 帧净化器，剥掉 junk 头和 ICY 元数据
	}
	return base + "?u=" + encodeTarget(target) + suffix, ""
}

// ---------- 歌词 ----------

func (a *App) Lyric(id string) ([]LyricLine, string) {
	var lines []LyricLine
	var err error
	switch {
	case strings.HasPrefix(id, "qq:"):
		lines, err = lyricQQ(strings.TrimPrefix(id, "qq:"))
	case strings.HasPrefix(id, "ne:"):
		lines, err = lyricNetease(strings.TrimPrefix(id, "ne:"))
	case strings.HasPrefix(id, "loc:"):
		if p, e := localTrackPath(id); e == nil {
			lines = localLyric(p)
		}
	default:
		return []LyricLine{}, "不认识的曲目 id"
	}
	if err != nil {
		return []LyricLine{}, err.Error()
	}
	return lines, ""
}

// ---------- 收藏 ----------

func (a *App) Library() []Track {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Track, len(a.lib))
	copy(out, a.lib)
	return out
}

func (a *App) ToggleLibrary(t Track) []Track {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, x := range a.lib {
		if x.ID == t.ID {
			a.lib = append(a.lib[:i], a.lib[i+1:]...)
			a.saveLibrary()
			return a.lib
		}
	}
	a.lib = append(a.lib, t)
	a.saveLibrary()
	return a.lib
}

func (a *App) InLibrary(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, x := range a.lib {
		if x.ID == id {
			return true
		}
	}
	return false
}
