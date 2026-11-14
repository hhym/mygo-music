package qianqian

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

/* 插件管理。

   桌面版本体不内置任何音源，插件由用户自己导入，运行时在前端 WebView 里执行
   （见 frontend/src/mygoRuntime.ts）。规范只有一种：MyGo Music 的插件目录

       plugins/<id>/plugin.json        清单：id / name / version / main / navigation ...
       plugins/<id>/index.js           入口，ESM 导出 search / stream / lyrics / home ...
       plugins/<id>/locales/*.json     可选

   后端只做文件层面的增删启停，以及把插件文件按需读给前端。
   旧的 MusicFree 单文件插件（module.exports）已不再支持。 */

type Plugin struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Homepage    string   `json:"homepage"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Compatible  bool     `json:"compatible"`
	Size        int64    `json:"size"`
	InstalledAt string   `json:"installedAt"`
	File        string   `json:"file"`
	Kind        string   `json:"kind"`
	Navigation  string   `json:"navigation"`
	Actions     []string `json:"actions"`
}

func pluginsDir() string {
	dir := filepath.Join(dataDir(), "plugins")
	os.MkdirAll(dir, 0o755)
	return dir
}

func pluginsStatePath() string { return filepath.Join(dataDir(), "plugins.json") }

func loadPluginState() map[string]bool {
	m := map[string]bool{}
	if b, err := os.ReadFile(pluginsStatePath()); err == nil {
		json.Unmarshal(b, &m)
	}
	return m
}

func savePluginState(m map[string]bool) {
	b, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(pluginsStatePath(), b, 0o644)
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r > 127:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *App) scanPlugins() []Plugin {
	out := scanMyGoPlugins(loadPluginState())
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ---------- 绑定给前端的方法 ----------

func (a *App) Plugins() ([]Plugin, string) {
	return a.scanPlugins(), ""
}

// InstallPluginText MyGo 插件有目录结构，粘贴一段源码装不了。
func (a *App) InstallPluginText(code string) PickResult {
	if strings.TrimSpace(code) == "" {
		return PickResult{Plugins: a.scanPlugins(), Err: "内容是空的"}
	}
	return PickResult{Plugins: a.scanPlugins(), Err: "MyGo 插件是「目录 + plugin.json」的形式，不能只粘贴源码：请用「选文件夹」挑插件目录，或装打包好的 .zip"}
}

// InstallPluginFile 从本地导入：选文件夹，或选含 plugin.json 的 .zip。
//
// 返回结构体而不是 (列表, 错误) 两个值：Wails 会把多返回值打包成数组，
// 前端一旦解包方式不对，就会出现「界面提示装好了、列表里其实是空的」。
func (a *App) InstallPluginFile(path string) PickResult {
	path = strings.Trim(strings.TrimSpace(path), "\"")
	if path == "" {
		return PickResult{Plugins: a.scanPlugins(), Err: "没有选文件"}
	}
	fi, err := os.Stat(path)
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "路径不存在：" + err.Error()}
	}
	if fi.IsDir() {
		id, err := a.installPluginDir(path)
		if err != nil {
			return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
		}
		return PickResult{Plugins: a.scanPlugins(), Err: "", InstalledID: id}
	}
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		b, err := os.ReadFile(path)
		if err != nil {
			return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
		}
		id, err := a.installPluginZip(b)
		if err != nil {
			return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
		}
		return PickResult{Plugins: a.scanPlugins(), Err: "", InstalledID: id}
	}
	return PickResult{Plugins: a.scanPlugins(), Err: "只支持 MyGo 插件文件夹，或里面带 plugin.json 的 .zip"}
}

// InstallPluginURL 只接受打包好的插件 zip（里面要有 plugin.json）。
func (a *App) InstallPluginURL(rawURL string) PickResult {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "地址不合法"}
	}
	req.Header.Set("User-Agent", uaDesktop)
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "下载失败：" + err.Error()}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return PickResult{Plugins: a.scanPlugins(), Err: fmt.Sprintf("下载失败：HTTP %d", res.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 60<<20))
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "读取失败：" + err.Error()}
	}
	if !bytes.HasPrefix(body, []byte("PK\x03\x04")) {
		return PickResult{Plugins: a.scanPlugins(), Err: "这不是插件 zip（MyGo 插件要打包成含 plugin.json 的压缩包）"}
	}
	id, err := a.installPluginZip(body)
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
	}
	return PickResult{Plugins: a.scanPlugins(), Err: "", InstalledID: id}
}

func (a *App) SetPluginEnabled(id string, on bool) ([]Plugin, string) {
	state := loadPluginState()
	state[id] = on
	savePluginState(state)
	return a.scanPlugins(), ""
}

func (a *App) RemovePlugin(id string) ([]Plugin, string) {
	if err := os.RemoveAll(pluginDir(id)); err != nil {
		return a.scanPlugins(), err.Error()
	}
	state := loadPluginState()
	delete(state, id)
	savePluginState(state)
	return a.scanPlugins(), ""
}

// PluginSource 把插件入口文件交给前端执行。
func (a *App) PluginSource(id string) (string, string) {
	p, ok := pluginEntryFile(id)
	if !ok {
		return "", "读不到插件：" + id
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", "读不到插件：" + err.Error()
	}
	return string(b), ""
}

func (a *App) PluginExists(id string) bool {
	_, ok := pluginEntryFile(id)
	return ok
}
