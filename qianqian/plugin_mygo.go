package qianqian

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

/* MyGo Music 插件支持。

   MyGo 的插件是一个目录：plugin.json（清单）+ index.js（ESM 导出）+ locales 等附属文件。
   index.js 里用 `export async function search({query, kind, offset}) {…}` 这种写法，
   网络请求直接用全局 fetch，持久化用全局 host.storage —— 所以真正执行仍然在前端，
   后端只负责：把目录原样落盘、按需把文件读给前端、给 host.storage 提供落盘能力。 */

// PluginManifest 是 plugin.json 的内容，只取桌面版用得上的字段。
type PluginManifest struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	MinAppVersion string            `json:"minAppVersion"`
	Main          string            `json:"main"`
	Author        string            `json:"author"`
	Description   string            `json:"description"`
	Homepage      string            `json:"homepage"`
	Playback      string            `json:"playback"`
	Icon          string            `json:"icon"`
	Permissions   map[string][]string `json:"permissions"`
	Navigation    []map[string]any  `json:"navigation"`
	Settings      []map[string]any  `json:"settings"`
}

func pluginDir(id string) string { return filepath.Join(pluginsDir(), id) }

// mygoManifest 读取目录型插件的清单；不是目录型插件就返回 nil。
func mygoManifest(id string) *PluginManifest {
	b, err := os.ReadFile(filepath.Join(pluginDir(id), "plugin.json"))
	if err != nil {
		return nil
	}
	var m PluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	if m.Main == "" {
		m.Main = "index.js"
	}
	return &m
}

// pluginEntryFile 返回插件主文件路径（目录型返回 index.js 一类，单文件型返回 <id>.js）。
func pluginEntryFile(id string) (string, bool) {
	if m := mygoManifest(id); m != nil {
		p := filepath.Join(pluginDir(id), filepath.FromSlash(m.Main))
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		p = filepath.Join(pluginDir(id), "index.js")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		return "", false
	}
	p := filepath.Join(pluginsDir(), id+".js")
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// pluginKind 报给前端：mygo（目录型）/ musicfree（单文件）。
func pluginKind(id string) string {
	if mygoManifest(id) != nil {
		return "mygo"
	}
	return "musicfree"
}

// PluginDir 把插件的安装目录告诉前端：插件的 host.dir 就是它，
// 插件自带的程序（如 bin/win32-amd64/ffmpeg.exe）靠这个路径拼出来。
func (a *App) PluginDir(id string) string {
	if mygoManifest(id) == nil {
		return pluginsDir()
	}
	return pluginDir(id)
}

// PluginFile 读插件目录里的任意文件，供前端解析 index.js 里的相对 import。
func (a *App) PluginFile(id, rel string) (string, string) {
	rel = strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "./")
	if rel == "" || strings.Contains(rel, "..") {
		return "", "路径不合法"
	}
	base := pluginDir(id)
	if mygoManifest(id) == nil {
		base = pluginsDir()
		rel = id + ".js"
	}
	p := filepath.Join(base, filepath.FromSlash(rel))
	clean := filepath.Clean(p)
	if !strings.HasPrefix(clean, filepath.Clean(base)+string(os.PathSeparator)) && clean != filepath.Clean(p) {
		return "", "路径不合法"
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", "读不到文件：" + err.Error()
    }
	return string(b), ""
}

// ---------- host.storage 落盘 ----------

func pluginDataDir() string {
	dir := filepath.Join(dataDir(), "plugin-data")
	os.MkdirAll(dir, 0o755)
	return dir
}

func pluginDataPath(id string) string {
	return filepath.Join(pluginDataDir(), sanitizeID(id)+".json")
}

// PluginStorage 读回插件自己的持久化数据（host.storage 的底层）。
func (a *App) PluginStorage(id string) (string, string) {
	b, err := os.ReadFile(pluginDataPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return "{}", ""
		}
		return "{}", err.Error()
	}
	return string(b), ""
}

// SavePluginStorage 覆盖写入插件数据。
func (a *App) SavePluginStorage(id, data string) string {
	if !json.Valid([]byte(data)) {
		return "数据不是合法 JSON"
	}
	if err := os.WriteFile(pluginDataPath(id), []byte(data), 0o600); err != nil {
		return err.Error()
	}
	return ""
}

// ---------- 安装：目录 / zip ----------

// installPluginDir 把一个 MyGo 插件目录整体复制进插件目录。
func (a *App) installPluginDir(src string) (string, error) {
	b, err := os.ReadFile(filepath.Join(src, "plugin.json"))
	if err != nil {
		return "", fmt.Errorf("这个文件夹里没有 plugin.json")
	}
	var m PluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return "", fmt.Errorf("plugin.json 解析失败：%w", err)
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		if m.Name != "" {
			id = sanitizeID(m.Name)
		}
	}
	if id == "" {
		return "", fmt.Errorf("plugin.json 里没有 id")
	}
	dst := pluginDir(id)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	os.MkdirAll(dst, 0o755)
	err = filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// 源码目录里的 node_modules / .git 之类不搬
		base := filepath.Base(rel)
		if info.IsDir() && (base == "node_modules" || base == ".git") {
			return filepath.SkipDir
		}
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		of, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer of.Close()
		_, err = io.Copy(of, io.LimitReader(in, 32<<20))
		return err
	})
	if err != nil {
		return "", err
	}
	state := loadPluginState()
	state[id] = true
	savePluginState(state)
	return id, nil
}

// installPluginZip 解出压缩包里的 MyGo 插件目录（含 plugin.json 的那一层）再安装。
func (a *App) installPluginZip(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("压缩包解析失败：%w", err)
	}
	// 找到 plugin.json 所在目录前缀
	prefix := ""
	for _, f := range zr.File {
		if strings.EqualFold(filepath.Base(f.Name), "plugin.json") {
			prefix = strings.TrimSuffix(f.Name, "plugin.json")
			break
		}
	}
	if prefix == "" {
		return "", fmt.Errorf("压缩包里没有 plugin.json，不是 MyGo 插件包")
	}
	tmp, err := os.MkdirTemp("", "hulu-plugin-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" {
			continue
		}
		out := filepath.Join(tmp, filepath.FromSlash(rel))
		if f.FileInfo().IsDir() {
			os.MkdirAll(out, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		of, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, err = io.Copy(of, io.LimitReader(rc, 32<<20))
		of.Close()
		rc.Close()
		if err != nil {
			return "", err
		}
	}
	return a.installPluginDir(tmp)
}

// scanMyGoPlugins 列出所有目录型插件。
func scanMyGoPlugins(state map[string]bool) []Plugin {
	dir := pluginsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		 return []Plugin{}
	}
	out := make([]Plugin, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		m := mygoManifest(id)
		if m == nil {
			continue
		}
		var size int64
		filepath.Walk(pluginDir(id), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				size += info.Size()
			}
			return nil
		})
		name := m.Name
		if name == "" {
			name = id
		}
		ver := m.Version
		if ver == "" {
		}
		desc := m.Description
		if desc == "" {
			desc = "MyGo 插件"
		}
		fi, _ := e.Info()
		installed := ""
		if fi != nil {
			installed = fi.ModTime().Format("2006-01-02 15:04")
		}
		out = append(out, Plugin{
			ID:          id,
			Name:        name,
			Version:     ver,
			Author:      m.Author,
			Homepage:    m.Homepage,
			Description: desc,
			Enabled:     state[id],
			Compatible:  true,
			Size:        size,
			InstalledAt: installed,
			File:        m.Main,
			Kind:        "mygo",
			Navigation:  navigationJSON(m),
			Actions:     actionList(m, id),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func navigationJSON(m *PluginManifest) string {
	if len(m.Navigation) == 0 {
		return ""
	}
	b, _ := json.Marshal(m.Navigation)
	return string(b)
}

// actionList 把插件支持的入口列出来，界面上按这个决定给不给搜/播放/登录。
//
// plugin.json 只声明了 playback 和 navigation，像 signIn / account 这类
// 是插件直接导出的函数，manifest 里根本不提。所以这里读一遍入口文件，
// 看看实际导出名，否则「扫码登录」按钮永远不出现。
func actionList(m *PluginManifest, id string) []string {
	out := []string{}
	if m.Playback != "" {
		out = append(out, "stream")
	}
	if len(m.Navigation) > 0 {
		out = append(out, "nav")
	}

	entry, ok := pluginEntryFile(id)
	if !ok {
		return out
	}
	src, err := os.ReadFile(entry)
	if err != nil {
		return out
	}
	body := string(src)
	for _, a := range detectActions(body) {
		if !actionHas(out, a) {
			out = append(out, a)
		}
	}
	return out
}

// detectActions 从插件源码里找它导出了哪些能力。
//
// 匹配几种常见写法：export function signIn / export const signIn = /
// signIn: async (啊对象字面量) / "signIn" 字符串键。插件名是死的，够用。
func detectActions(body string) []string {
	probes := []struct{ name, action string }{
		{"signIn", "signIn"},
		{"account", "account"},
		{"search", "search"},
		{"stream", "stream"},
		{"lyrics", "lyrics"},
		{"home", "home"},
		{"stations", "stations"},
		{"browse", "browse"},
	}
	out := []string{}
	for _, p := range probes {
		if strings.Contains(body, p.name+"(") ||
			strings.Contains(body, p.name+":") ||
			strings.Contains(body, p.name+" =") ||
			strings.Contains(body, "\""+p.name+"\"") ||
			strings.Contains(body, "'"+p.name+"'") {
			out = append(out, p.action)
		}
	}
	return out
}

func actionHas(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
