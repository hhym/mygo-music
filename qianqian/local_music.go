package qianqian

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/egoist/mygo"
)

/* 本地乐库。

   用户挑一个放音乐的文件夹，这里递归扫音频文件做成 Track 列表。
   播放走本地代理（file:// 通道），但代理原先只认数据目录里的文件，
   所以要把用户点过「授权」的目录登记成白名单，否则扫出来的歌点了会 403。 */

var audioExts = map[string]bool{
	".mp3": true, ".flac": true, ".m4a": true, ".aac": true,
	".wav": true, ".ogg": true, ".opus": true, ".wma": true, ".ape": true,
}

var localMu sync.RWMutex
var localRoots []string

func setLocalRoots(dirs []string) {
	localMu.Lock()
	defer localMu.Unlock()
	clean := make([]string, 0, len(dirs))
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if abs, err := filepath.Abs(d); err == nil {
			clean = append(clean, abs)
		}
	}
	localRoots = clean
}

func localRootsSnapshot() []string {
	localMu.RLock()
	defer localMu.RUnlock()
	out := make([]string, len(localRoots))
	copy(out, localRoots)
	return out
}

// within 判断 abs 是否在 root 目录之下（按目录边界比，避免 /a/bc 命中 /a/b）。
func within(abs, root string) bool {
	abs = strings.ToLower(filepath.Clean(abs))
	root = strings.ToLower(filepath.Clean(root))
	if abs == root {
		return true
	}
	return strings.HasPrefix(abs, root+string(os.PathSeparator))
}

// pathAllowed 媒体代理的放行规则：数据目录内，或用户授权过的音乐目录内。
func pathAllowed(abs string) bool {
	if within(abs, dataDir()) {
		return true
	}
	for _, r := range localRootsSnapshot() {
		if within(abs, r) {
			return true
		}
	}
	return false
}

// PickLocalDir 弹目录选择框。返回空字符串表示用户取消。
func (a *App) PickLocalDir() (string, string) {
	if a.ctx == nil {
		return "", "窗口还没就绪"
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:    a.win,
		Title:     "选择放音乐的文件夹",
		Directory: true,
	})
	if err != nil {
		return "", err.Error()
	}
	if len(paths) == 0 {
		return "", ""
	}
	dir := strings.TrimSpace(paths[0])
	if dir == "" {
		return "", ""
	}
	a.addLocalRoot(dir)
	return dir, ""
}

// LocalDirs 已授权的音乐目录。
func (a *App) LocalDirs() []string {
	return localRootsSnapshot()
}

// ForgetLocalDir 移除一个目录（只是不再扫描，不清用户的歌）。
func (a *App) ForgetLocalDir(dir string) []string {
	abs, _ := filepath.Abs(strings.TrimSpace(dir))
	keep := make([]string, 0, len(localRoots))
	for _, d := range localRootsSnapshot() {
		if !strings.EqualFold(filepath.Clean(d), filepath.Clean(abs)) {
			keep = append(keep, d)
		}
	}
	a.persistLocalRoots(keep)
	return localRootsSnapshot()
}

func (a *App) addLocalRoot(dir string) {
	abs, err := filepath.Abs(strings.TrimSpace(dir))
	if err != nil {
		return
	}
	keep := make([]string, 0, len(localRoots)+1)
	keep = append(keep, abs)
	for _, d := range localRootsSnapshot() {
		if !strings.EqualFold(filepath.Clean(d), filepath.Clean(abs)) {
			keep = append(keep, d)
		}
	}
	a.persistLocalRoots(keep)
}

// persistLocalRoots 写回 settings.json 并同步白名单。
func (a *App) persistLocalRoots(dirs []string) {
	setLocalRoots(dirs)
	s := a.loadSettings()
	s.LocalDirs = localRootsSnapshot()
	_ = a.saveSettings(s)
}

// ScanLocalMusic 递归扫一个目录，返回曲目列表。同名 .lrc 会被 Lyric 读到。
// LocalScan 扫描结果。带统计是为了能回答「为什么扫不到歌」：
// 是目录没读到、全是图片，还是真的空。
type LocalScan struct {
	Tracks   []Track `json:"tracks"`
	Root     string  `json:"root"`
	Scanned  int     `json:"scanned"`  // 遍历过的文件总数
	Audio    int     `json:"audio"`    // 命中音频扩展名的数量
	Dirs     int     `json:"dirs"`     // 遍历过的目录数
	Skipped  int     `json:"skipped"`  // 读不进去的目录/文件
	Err      string  `json:"err"`
}

// ScanLocalMusic 递归扫一个目录，返回曲目列表 + 扫描统计。同名 .lrc 会被 Lyric 读到。
func (a *App) ScanLocalMusic(dir string) LocalScan {
	res := LocalScan{Tracks: []Track{}}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		res.Err = "先选一个音乐文件夹"
		return res
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Root = abs
	fi, err := os.Stat(abs)
	if err != nil {
		res.Err = "这个目录打不开：" + err.Error()
		return res
	}
	if !fi.IsDir() {
		res.Err = "选中的不是文件夹"
		return res
	}
	// 扫过就记住，方便下次直接放行
	a.addLocalRoot(abs)

	out := make([]Track, 0, 512)
	walkErr := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 根目录都读不了，必须让用户看到原因，不能静默返回空列表
			if p == abs {
				return err
			}
			res.Skipped++
			return nil
		}
		if d.IsDir() {
			res.Dirs++
			name := d.Name()
			if p != abs && (name == "node_modules" || name == "__MACOSX" || strings.HasPrefix(name, "$RECYCLE.BIN")) {
				return filepath.SkipDir
			}
			return nil
		}
		res.Scanned++
		if len(out) >= 20000 {
			return filepath.SkipAll
		}
		if !audioExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		res.Audio++
		out = append(out, localTrack(p))
		return nil
	})
	if walkErr != nil {
		res.Err = "扫描中断：" + walkErr.Error()
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	res.Tracks = out
	return res
}

// localTrack 从文件名推标题：优先按「艺人 - 歌名」拆，拆不出就当整段是歌名。
func localTrack(p string) Track {
	base := filepath.Base(p)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	artist, title := "", strings.TrimSpace(stem)
	// 兼容半角与全角连接符
	for _, sep := range []string{" - ", " – ", " — "} {
		if i := strings.Index(stem, sep); i > 0 {
			artist = strings.TrimSpace(stem[:i])
			title = strings.TrimSpace(stem[i+len(sep):])
			break
		}
	}
	return Track{
		ID:     "loc:" + base64.RawURLEncoding.EncodeToString([]byte(p)),
		Name:   title,
		Artist: artist,
		Album:  filepath.Base(filepath.Dir(p)),
		Source: "local",
	}
}

// localTrackPath 从 loc: id 还原文件路径。
func localTrackPath(id string) (string, error) {
	p, err := decodeTarget(strings.TrimPrefix(id, "loc:"))
	if err != nil || strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("本地曲目 id 损坏")
	}
	return p, nil
}

// localLyric 找同名 .lrc（同一目录、同基名）。
func localLyric(p string) []LyricLine {
	stem := strings.TrimSuffix(p, filepath.Ext(p))
	for _, ext := range []string{".lrc", ".LRC", ".Lrc"} {
		if b, err := os.ReadFile(stem + ext); err == nil {
			if lines := parseLRC(string(b)); len(lines) > 0 {
				return lines
			}
		}
	}
	return []LyricLine{}
}
