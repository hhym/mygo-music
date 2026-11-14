package qianqian

import (
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Settings 是桌面版的可配置项，持久化在 <数据目录>/settings.json。
type Settings struct {
	Theme       string   `json:"theme"`       // dark | light | amber | teal
	Volume      float64  `json:"volume"`      // 0 - 1
	AutoPlay    bool     `json:"autoPlay"`    // 选中即播
	ShowLyric   bool     `json:"showLyric"`   // 播放条显示歌词行
	DefaultTab  string   `json:"defaultTab"`  // 启动落到哪个页
	Sources     []string `json:"sources"`     // 参与聚合搜索的源
	SearchLimit int      `json:"searchLimit"` // 每源取多少条
	LastTrackID string   `json:"lastTrackId"`
	LocalDirs   []string `json:"localDirs"`   // 本地乐库授权的音乐目录

	// LookMigrated 记录「默认外观已迁到 2.7.14 那套（桌面橘黄）」，只跑一次，
	// 之后用户自己选的主题不再被覆盖。
	LookMigrated bool `json:"lookMigrated"`

	// 只读，回填给前端展示用
	ProxyPort int    `json:"proxyPort"`
	DataDir   string `json:"dataDir"`
	Version   string `json:"version"`
}

const appVersion = "2.7.19"

func defaultSettings() Settings {
	return Settings{
		Theme:       "amber", // 2.7.14 起桌面默认橘黄
		Volume:      0.8,
		AutoPlay:    true,
		ShowLyric:   true,
		DefaultTab:  "search",
		Sources:     []string{"ne", "qq"},
		SearchLimit: 20,
	}
}

func settingsPath() string { return filepath.Join(dataDir(), "settings.json") }

func (a *App) loadSettings() Settings {
	s := defaultSettings()
	b, err := os.ReadFile(settingsPath())
	if err == nil {
		json.Unmarshal(b, &s)
	}
	if s.Volume <= 0 || s.Volume > 1 {
		s.Volume = 0.8
	}
	if s.SearchLimit <= 0 || s.SearchLimit > 200 {
		s.SearchLimit = 20
	}
	if len(s.Sources) == 0 {
		s.Sources = []string{"ne", "qq"}
	}
	// 老用户的 settings.json 里存的是旧默认主题，只改 defaultSettings 对他们永远不生效，
	// 所以这里补一次迁移：标记没写过就按新默认写回一次，之后不再动。
	if !s.LookMigrated {
		s.Theme = "amber"
		s.LookMigrated = true
		if b2, e2 := json.MarshalIndent(s, "", "  "); e2 == nil {
			_ = os.WriteFile(settingsPath(), b2, 0o644)
		}
	}
	return s
}

func (a *App) saveSettings(s Settings) error {
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(settingsPath(), b, 0o644)
}

// fillRuntime 把端口、数据目录这些运行时信息补进去，前端只读展示。
func (a *App) fillRuntime(s Settings) Settings {
	s.DataDir = dataDir()
	s.Version = appVersion
	if base := a.proxyBase(); base != "" {
		if u, err := url.Parse(base); err == nil {
			if p := u.Port(); p != "" {
				if n, err := atoiSafe(p); err == nil {
					s.ProxyPort = n
				}
			}
		}
	}
	return s
}

func atoiSafe(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// ---------- 绑定给前端的方法 ----------

func (a *App) GetSettings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.fillRuntime(a.loadSettings())
}

func (a *App) SaveSettings(s Settings) (Settings, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.saveSettings(s); err != nil {
		return a.fillRuntime(a.loadSettings()), err.Error()
	}
	return a.fillRuntime(a.loadSettings()), ""
}

func (a *App) ResetSettings() (Settings, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.saveSettings(defaultSettings()); err != nil {
		return a.fillRuntime(defaultSettings()), err.Error()
	}
	return a.fillRuntime(a.loadSettings()), ""
}

// OpenDataDir 在资源管理器里打开数据目录，方便用户自己看配置和插件。
func (a *App) OpenDataDir() (string, string) {
	dir := dataDir()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return dir, err.Error()
	}
	return dir, ""
}

// ClearLibrary 清空收藏。
func (a *App) ClearLibrary() ([]Track, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lib = []Track{}
	a.saveLibrary()
	return a.lib, ""
}
