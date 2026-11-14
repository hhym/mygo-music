# 千千静听界面版 MyGo Music

这个 fork 把 [egoist/mygo-music](https://github.com/egoist/mygo-music) 的原生 UI 换成
千千静听（qianqian-jingting）的 WebView 前端：皮肤界面、迷你浮卡、正在播放页、
三档主题（橘黄 / 冷白浅蓝 / 紫夜深色）都是千千静听那一套，Go 侧继续用 MyGo 的
插件宿主能力（插件是前端在 WebView 里跑的 MyGo 模块）。

## 怎么编译

需要 Go 1.27（go.mod 里写的就是 1.27.3）。

```bash
# Windows（需要 WebView2，Win11 自带）
go build -o qqjt.exe ./cmd/qqjt
```

跑起来：`./qqjt`，界面就是千千静听的桌面版。

原来的原生 UI 入口没有删，`go build .` 出来的还是 mygo-music 原版界面，
两个入口并存，互不影响。

## 目录

| 路径 | 是什么 |
| --- | --- |
| `cmd/qqjt/main.go` | 新入口：一个 MyGo 窗口 + `LoadHTML` 加载打包页面 |
| `qianqian/page.html` | 千千静听前端打完包的整页（JS/CSS 全内联），外加一层 `window.go` 兼容 shim |
| `qianqian/app.go` | 从千千静听搬来的宿主能力（搜索、收藏、插件、播客、电台、设置） |
| `qianqian/plugin*.go` | MyGo 插件的安装、扫描、源码读取、存储、设置 |
| `qianqian/httpreq.go`, `http.go`, `proxy.go` | 绕 CORS 的 HTTP 请求、播放地址代理 |
| `qianqian/local_music.go` | 本地音乐目录扫描与授权 |
| `qianqian/window.go` | 迷你浮卡（348×124 置顶小窗）与拖动 |

## 前端怎么调 Go

千千静听的前端是按 Wails 写的，调 `window.go.main.App.X(...)`。
`page.html` 顶部注入了一段 shim，把这层换成 MyGo 的 IPC：

```js
window.mygo.call('App.X', ...args)
```

所以前端源码一行没改。

## 与千千静听原版的差别

- **窗口裁剪**：原版用 Win32 `SetWindowRgn` 按卡片轮廓裁窗口；这里改用 MyGo 的
  `Frameless` 窗口 + 页面 CSS 的圆角与阴影，效果一样但更简单，且三平台一致。
- **迷你浮卡尺寸**：仍是 348×124，`SetMiniMode` 里切换尺寸与置顶。
- **拖动**：仍走前端 `WinMoveBy(dx, dy)` → 窗口移动。

## 绑定层为什么多一个 API

MyGo 的绑定只接受这两种方法签名：单返回值，或者 `(值, error)`。
千千静听原来的方法全是 `(值, 错误串)`，第二个是 `string`，直接 Bind 会在注册阶段
报 `the second result must be an error`，整个服务绑不上、窗口里什么都点不动。

所以 `qianqian/bind.go` 里多了一层 `API`：42 个方法逐个转签名，把错误串包成
`errors.New`，参数与值原样转发。原 `app.go` 一行没动，仍然是千千静听那套逻辑。

`cmd/qqjt/main.go` 里用 `mygo.BindAs("App", qianqian.NewAPI(a))`，服务名固定是
`App`，所以前端 `window.go.main.App.X(...)` 照旧。

## 还没做的

- 皮肤页面里的六款皮肤缩略图（截图 assets）需要重新打包进 `page.html`。
- 原生 UI 入口（`main.go`）里还有若干 mygo-music 独有页面（评论、电台榜单等），
  没有对应的千千静听页面。
- 迷你浮卡现在是「同一窗口缩到 348×124 + 置顶」，窗口仍带系统标题栏。
  要像截图那样干净地浮在桌面上，需要真机调无边框/透明窗口，或者像上游
  `ui_notch.go` 那样单开一个 `Frameless` 小窗，这一项留着等你真机反馈。
