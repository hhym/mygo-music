package qianqian

import (
	"github.com/egoist/mygo"
)

/* 本地插件选择。

   以前让用户手输路径，不好用：Windows 的路径既长又要转义反斜杠。
   这里直接弹系统文件对话框挑 zip，或者挑文件夹。

   返回值用结构体而不是 (list, err) 两个值——取消要能和「装失败」区分开，
   双返回值被前端合并后没法表达三种状态。 */

type PickResult struct {
	Plugins     []Plugin `json:"plugins"`
	Err         string   `json:"err"`
	Canceled    bool     `json:"canceled"`
	InstalledID string   `json:"installedId"`
}

// PickPluginZip 弹文件对话框选插件压缩包（里面要有 plugin.json）。
func (a *App) PickPluginZip() PickResult {
	if a.ctx == nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "窗口还没就绪"}
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  a.win,
		Title:   "选择 MyGo 插件压缩包",
		Filters: []mygo.FileFilter{{Name: "MyGo 插件 (*.zip)", Extensions: []string{"zip"}}},
	})
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
	}
	if len(paths) == 0 {
		return PickResult{Plugins: a.scanPlugins(), Canceled: true}
	}
	path := paths[0]
	if path == "" {
		return PickResult{Plugins: a.scanPlugins(), Canceled: true}
	}
	return a.InstallPluginFile(path)
}

// PickPluginDir 弹目录选择，直接装一个插件文件夹。
func (a *App) PickPluginDir() PickResult {
	if a.ctx == nil {
		return PickResult{Plugins: a.scanPlugins(), Err: "窗口还没就绪"}
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:    a.win,
		Title:     "选择 MyGo 插件文件夹（里面要有 plugin.json）",
		Directory: true,
	})
	if err != nil {
		return PickResult{Plugins: a.scanPlugins(), Err: err.Error()}
	}
	if len(paths) == 0 {
		return PickResult{Plugins: a.scanPlugins(), Canceled: true}
	}
	dir := paths[0]
	if dir == "" {
		return PickResult{Plugins: a.scanPlugins(), Canceled: true}
	}
	return a.InstallPluginFile(dir)
}
