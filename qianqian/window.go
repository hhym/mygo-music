package qianqian

// 迷你浮卡：窗口缩成 348x124 并置顶。MyGo 的窗口是透明无边框的，
// 圆角与阴影交给页面里的 CSS（.mini-card / .mini-player），不再像
// Win32 那版那样用 SetWindowRgn 裁轮廓。

const (
	miniW, miniH = 348, 124
	deskW, deskH = 1190, 780
)

func (a *App) miniOn() string {
	if a.win == nil {
		return "窗口还没准备好"
	}
	a.win.SetAlwaysOnTop(true)
	a.win.SetSize(miniW, miniH)
	return ""
}

func (a *App) miniOff() string {
	if a.win == nil {
		return "窗口还没准备好"
	}
	a.win.SetSize(deskW, deskH)
	a.win.SetAlwaysOnTop(false)
	return ""
}

// Wails 那版叫 WinMoveBy；这里做同样的换算，给页面拖动迷你卡用。
func (a *App) WinMoveBy(dx, dy int) string {
	if a.win == nil {
		return "窗口还没准备好"
	}
	x, y := a.win.Position()
	a.win.SetPosition(x+dx, y+dy)
	return ""
}
