package app

import (
	"sync"

	"mygo-music/qianqian"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 葫芦音乐的皮肤界面和迷你卡片：装在原生界面旁边的两个可选窗口。
//
// 它们是加装，不是替换——原生桌面界面、它的乐库、播放器和设置一概不碰。
// 葫芦窗口跑的是 qianqian 包里那份内置 WebView 页面（千千静听那套皮肤），
// 用的是它自己的数据目录（~/.hulu-music-desktop），两边的数据互不影响。
var hulu = struct {
	once sync.Once
	app  *qianqian.App
	skin *mygo.Window
	mini *mygo.Window
}{}

// huluHost 第一次用到时才把葫芦的后端起起来：开一个本地 http 服务提供页面，
// 并把它的 Go 方法绑成前端能调的 App.xxx。不用的时候什么都不占。
func huluHost() *qianqian.App {
	hulu.once.Do(func() {
		hulu.app = qianqian.NewApp()
		hulu.app.Start()
		mygo.BindAs("App", qianqian.NewAPI(hulu.app))
	})
	return hulu.app
}

// huluURL 葫芦页面地址，windowed=1 告诉页面「我是附带窗口，× 只关我自己」。
func huluURL(extra string) string {
	host := huluHost()
	base := host.PageURL()
	if base == "" {
		return ""
	}
	q := "?windowed=1"
	if extra != "" {
		q += "&" + extra
	}
	return base + q
}

// openHuluSkin 打开皮肤界面。已经开着就把它提到最前面，不重复开。
func (a *app) openHuluSkin() {
	url := huluURL("")
	if url == "" {
		return
	}
	if hulu.skin != nil {
		hulu.skin.Show()
		hulu.skin.Focus()
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:        "葫芦音乐 · 皮肤",
		Width:        1000,
		Height:       720,
		MinWidth:     720,
		MinHeight:    500,
		Frameless:    true,
		Transparent:  true,
		CornerRadius: 12,
		URL:          url,
	})
	// 皮肤窗口里的最小化/关闭/切迷你，都作用在它自己身上。
	huluHost().AttachWindow(win)
	hulu.skin = win
	win.OnClosed(func() { hulu.skin = nil })
}

// openHuluMini 打开迷你卡片：348×124 的无边框浮卡，常在最前。
func (a *app) openHuluMini() {
	url := huluURL("mini=1")
	if url == "" {
		return
	}
	if hulu.mini != nil {
		hulu.mini.Show()
		hulu.mini.Focus()
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:        "葫芦音乐 · 迷你",
		Width:        qianqian.MiniWidth(),
		Height:       qianqian.MiniHeight(),
		MinWidth:     0,
		MinHeight:    0,
		Frameless:    true,
		Transparent:  true,
		CornerRadius: 12,
		AlwaysOnTop:  true,
		URL:          url,
	})
	win.OnClosed(func() { hulu.mini = nil })
}

// closeHulu 退出前把葫芦窗口收掉，别留下关不掉的进程。
func closeHulu() {
	if hulu.skin != nil {
		hulu.skin.Close()
	}
	if hulu.mini != nil {
		hulu.mini.Close()
	}
}

// huluRow 是侧栏底部那一行入口：原生界面的其余部分一个像素都没动，整个
// 「加装」在界面上就只有这一行和菜单里的两项。
func (a *app) huluRow(c *ui.Context) {
	p := a.pal
	b := ui.ButtonBase(c.Key("hulu-skin")).Height(34).PaddingX(10).Gap(10).Radius(8).
		AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	if b.Hovered() {
		b.Background(p.hover)
	}
	b.Label("葫芦音乐皮肤")
	b.Children(func() {
		ui.Icon(c, iconShirt).Size(17, 17).TextColor(p.muted)
		ui.Text(c, "葫芦音乐皮肤").FontSize(13).FontWeight(500).TextColor(p.muted).
			SingleLine().Shrink(1).MinWidth(0)
	})
	if b.Clicked() {
		a.openHuluSkin()
	}
}
