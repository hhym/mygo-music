package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The sidebar's appearance button lists the themes, and one chosen becomes the
// app's, kept.
func TestAppearancePicker(t *testing.T) {
	a := newTestApp()
	a.settings.Appearance = "sky"
	tt := ui.NewTester(a.view, 1100, 760)
	tt.Frame()
	if err := tt.Click("Appearance"); err != nil {
		t.Fatal("the appearance button isn't there")
	}
	tt.Frame()
	if err := tt.Click("◑ Midnight Ember"); err != nil {
		t.Fatal("the themes aren't listed")
	}
	if a.settings.Appearance != "dusk" {
		t.Errorf("appearance %q, want dusk", a.settings.Appearance)
	}
	if a.themeOpen {
		t.Error("the picker stays open")
	}
}
