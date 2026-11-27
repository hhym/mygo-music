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

// The card's power makes its plugin the source in use, the page staying; the
// word asks before removing it.
func TestPluginCardPowerAndRemove(t *testing.T) {
	a := newTestApp()
	b := addFake(a, newFakeSource("b"))
	tt := pluginsView(a, 600)
	tt.Frame()
	if err := tt.Click("Use b"); err != nil {
		t.Fatalf("no power on the card: %v", err)
	}
	tt.Frame()
	if a.library != b {
		t.Error("b isn't the source in use")
	}
	if a.route.page != pagePlugins {
		t.Errorf("the page became %v", a.route.page)
	}

	asked := ""
	was := confirmRemovePlugin
	t.Cleanup(func() { confirmRemovePlugin = was })
	confirmRemovePlugin = func(_ *app, name string, yes func()) { asked = name; yes() }
	if err := tt.Click("Remove b"); err != nil {
		t.Fatalf("no remove on the card: %v", err)
	}
	tt.Frame()
	if asked != "b" {
		t.Errorf("asked to remove %q", asked)
	}
}
