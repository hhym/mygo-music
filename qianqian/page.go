package qianqian

import _ "embed"

// Page 是打完包的整个界面：index.html 里已经内联了全部 JS 与 CSS，
// 一个 <embed> 文件就是全部前端。
//
//go:embed page.html
var page string

func Page() string { return page }
