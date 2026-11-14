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
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "千千静听",
			Width:     1190,
			Height:    780,
			MinWidth:  900,
			MinHeight: 560,
		})
		a.AttachWindow(win)
		a.Start()
		bootLog("窗口已建，page url: %q", a.PageURL())

		// 导航到本地服务（真实 origin，localStorage 可用）。窗口刚建好时
		// webview 可能还没接管，所以稍等一拍；再等两秒回读，仍没加载出来
		// 就退回内联页面（那种情况下 localStorage 由页面里的内存版顶着）。
		go func() {
			time.Sleep(1200 * time.Millisecond)
			page := win.Page()
			if page == nil {
				bootLog("page handle 是 nil，无法加载")
				return
			}
			url := a.PageURL()
			if url != "" {
				if err := page.LoadURL(url); err != nil {
					bootLog("LoadURL 失败: %v", err)
				}
			}
			time.Sleep(2500 * time.Millisecond)
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
