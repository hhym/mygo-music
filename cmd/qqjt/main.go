package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"mygo-music/qianqian"
)

// bootLog 写一行启动日志：双击 exe 没有控制台，出问题只能靠这个文件。
func bootLog(format string, args ...any) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".hulu-music-desktop")
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "qqjt-boot.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func main() {
	a := qianqian.NewApp()
	mygo.BindAs("App", qianqian.NewAPI(a))
	mygo.App.WhenReady(func() {
		// 先把本地服务起起来，拿到真实 http origin，再把地址交给窗口创建。
		// 早先的做法是先建空窗口、1.2 秒后再 LoadURL，Windows 上会多出一个
		// 空白的第二个窗口（WebView2 没挂到我们建的窗口上），桌面上看到就是
		// 一块黑。窗口创建时直接给 URL 走 mygo 的标准路径，只有一个窗口。
		a.Start()
		url := a.PageURL()
		opts := mygo.WindowOptions{
			Title:     "千千静听 · MyGo",
			Width:     1190,
			Height:    780,
			MinWidth:  900,
			MinHeight: 560,
			// 无边框 + 透明：系统标题栏不要了，改由页面顶部的 mygo-dragbar 拖动、
			// 自绘最小化/关闭。透明是为了迷你模式——窗口缩到 348×124 时页面背景
			// 变透明，只剩那张带圆角的迷你卡露在外面，看起来就是贴在桌面上的浮卡。
			Frameless:    true,
			Transparent:  true,
			CornerRadius: 12,
			Page:         mygo.PageOptions{DevTools: mygo.DevToolsEnabled},
		}
		if url != "" {
			opts.URL = url
		}
		win := mygo.NewWindow(opts)
		a.AttachWindow(win)
		bootLog("窗口已建，page url: %q", url)

		// 页面 <title> 是「千千静听 · Wails 桌面版」，会盖掉窗口标题、和
		// Wails 版播放器撞名，这里拦下来保持自己的名字。
		if p := win.Page(); p != nil {
			p.OnPageTitleUpdated(func(e *mygo.TitleEvent) { e.PreventDefault() })
		}

		// 回读一次真实状态，方便从日志判断到底加载到了什么。
		go func() {
			time.Sleep(2500 * time.Millisecond)
			page := win.Page()
			if page == nil {
				bootLog("page handle 是 nil，无法回读")
				return
			}
			v, err := page.Eval("document.location.href + '|' + document.body.innerHTML.length")
			bootLog("加载后状态: %v err=%v", v, err)
			bad := err != nil
			if s, ok := v.(string); ok {
				if strings.Contains(s, "about:blank") || strings.HasSuffix(s, "|0") {
					bad = true
				}
			}
			if bad {
				bootLog("本地服务页面没起来，回退内联页面")
				page.LoadHTML(qianqian.Page(), "")
				time.Sleep(1500 * time.Millisecond)
				v2, _ := page.Eval("document.location.href + '|' + document.body.innerHTML.length + '|' + (typeof window.go)")
				bootLog("回退后状态: %v", v2)
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		bootLog("Run 退出: %v", err)
		log.Fatal(err)
	}
}
