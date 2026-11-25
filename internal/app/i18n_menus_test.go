package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A text field's own menu, which MyGo writes rather than the app, says its
// items in the listener's language: Chinese for a Chinese app.
func TestTextFieldMenuInChinese(t *testing.T) {
	var text string
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		ui.TextInput(c, &text).Label("Name").Fill()
	}, 400, 300)
	tt.SetLocale("zh-Hans")
	tt.Frame()
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"剪切", "复制", "粘贴", "删除", "全选"} {
		if !tt.HasText(want) {
			t.Errorf("the menu has no %q: %q", want, tt.Texts())
		}
	}
}
