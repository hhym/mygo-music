package main

import (
	"log"

	"github.com/egoist/mygo"

	"mygo-music/qianqian"
)

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
		if p := win.Page(); p != nil {
			p.LoadHTML(qianqian.Page(), "")
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
