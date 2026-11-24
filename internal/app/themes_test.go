package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The themes offered are the system's own light and dark, and the app's four
// palettes, each with the base light or dark it draws on.
func TestThemeChoices(t *testing.T) {
	want := map[string]bool{"system": false, "light": false, "dark": true, "green": false, "amber": false, "teal": true, "gold": true}
	for _, c := range themeChoices {
		dark, ok := want[c.id]
		if !ok {
			t.Errorf("theme %q isn't one of them", c.id)
			continue
		}
		if got := themeDark(c, false); got != dark {
			t.Errorf("%s is dark %v, want %v", c.id, got, dark)
		}
	}
	if themeChoiceOf("nonsense").id != "system" {
		t.Errorf("a theme no longer offered isn't the system's")
	}
}

// Each palette is its own, and the light and dark ones are the themes' own
// colors.
func TestPalettes(t *testing.T) {
	palettes := map[string]palette{
		"green": greenPalette(),
		"amber": amberPalette(),
		"teal":  tealPalette(),
		"gold":  goldPalette(),
	}
	bg := map[ui.Color]string{}
	for id, p := range palettes {
		if other, ok := bg[p.bg]; ok {
			t.Errorf("%s and %s share a background", id, other)
		}
		bg[p.bg] = id
		if p.accent == ui.Transparent {
			t.Errorf("%s has no accent", id)
		}
	}
	if greenPalette().dark || amberPalette().dark {
		t.Error("the reading themes should be light")
	}
	if !tealPalette().dark || !goldPalette().dark {
		t.Error("deep teal and black gold should be dark")
	}
}
