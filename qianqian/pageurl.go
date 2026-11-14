package qianqian

// PageURL 是界面自己的地址：本地代理服务上的根路径。
// 用真实 http origin 而不是 about:blank，localStorage 才可写
// （千千静听的前端把皮肤、主题、布局都放在 localStorage 里）。
func (a *App) PageURL() string {
	a.ensureProxy()
	if proxyRoot == "" {
		return ""
	}
	return proxyRoot + "/"
}
