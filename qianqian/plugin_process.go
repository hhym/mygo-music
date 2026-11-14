package qianqian

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

/* 插件调用本机程序的受控执行。

   MyGo 插件的 host.spawn(bin, args) 落到这里。两个约束：

     1. 只允许 ffmpeg（用来把 HE-AAC 这类应用自身解不了的音频转成可播放的格式），
        别的程序一律拒绝，不让插件变成任意命令执行的后门；
     2. 还必须有一个「已启用」的插件在自己的 plugin.json 里 permissions.process
        声明过它，没声明过的插件调不动。

   和插件那边的契约对齐：插件拿到的是带 wait() / errorOutput() 的对象，
   所以后端一次性跑完再返回，前端包装成那两个方法。 */

type ProcessResult struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// 硬编码白名单：不管插件声明什么，能跑的只有这些。
var processAllowList = map[string]bool{
	"ffmpeg":     true,
	"ffmpeg.exe": true,
}

// 已启用插件在 plugin.json 里声明过的可执行程序（basename 归一化）。
func declaredProcesses() map[string]bool {
	out := map[string]bool{}
	dir := pluginsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	state := loadPluginState()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if v, ok := state[id]; ok && !v {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, id, "plugin.json"))
		if err != nil {
			continue
		}
		var m struct {
			Permissions struct {
				Process []string `json:"process"`
			} `json:"permissions"`
		}
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		for _, p := range m.Permissions.Process {
			if p == "" {
				continue
			}
			out[strings.ToLower(filepath.Base(p))] = true
		}
	}
	return out
}

func tailString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func (a *App) RunProcess(bin string, args []string, timeoutSec int) (ProcessResult, string) {
	if strings.TrimSpace(bin) == "" {
		return ProcessResult{}, "没有指定要运行的程序"
	}
	base := strings.ToLower(filepath.Base(bin))
	if !processAllowList[base] {
		return ProcessResult{}, "桌面版只允许插件运行 ffmpeg，其它程序一律拒绝：" + filepath.Base(bin)
	}
	if !declaredProcesses()[base] {
		return ProcessResult{}, "没有启用的插件声明过 " + filepath.Base(bin) + " 权限"
	}
	if timeoutSec <= 0 || timeoutSec > 600 {
		timeoutSec = 180
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return ProcessResult{}, "启动失败：" + err.Error()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		code := cmd.ProcessState.ExitCode()
		res := ProcessResult{Code: code, Stdout: tailString(stdout.String(), 8192), Stderr: tailString(stderr.String(), 8192)}
		if err != nil && code == 0 {
			return res, err.Error()
		}
		return res, ""
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		<-done
		msg := fmt.Sprintf("执行超时（%s 秒），已终止", strconv.Itoa(timeoutSec))
		return ProcessResult{Code: -1, Stderr: msg}, msg
	}
}

/* ---------- 插件自己的设置项 ----------
   plugin.json 的 settings[] 是插件声明的配置（例如蜻蜓的 ffmpegPath / transcode）。
   用户的取值存在 <dataDir>/plugin-settings.json，按插件 id 分组。 */

func pluginSettingsPath() string { return filepath.Join(dataDir(), "plugin-settings.json") }

func loadAllPluginSettings() map[string]map[string]string {
	m := map[string]map[string]string{}
	if b, err := os.ReadFile(pluginSettingsPath()); err == nil {
		json.Unmarshal(b, &m)
	}
	return m
}

func (a *App) PluginSettings(id string) (map[string]string, string) {
	m := loadAllPluginSettings()
	if v, ok := m[id]; ok {
		return v, ""
	}
	return map[string]string{}, ""
}

func (a *App) SavePluginSetting(id, key, value string) (map[string]string, string) {
	if id == "" || key == "" {
		return map[string]string{}, "插件 id 和设置项不能为空"
	}
	m := loadAllPluginSettings()
	if m[id] == nil {
		m[id] = map[string]string{}
	}
	m[id][key] = value
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(pluginSettingsPath(), b, 0o600); err != nil {
		return m[id], err.Error()
	}
	return m[id], ""
}
