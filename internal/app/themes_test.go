package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The themes offered are the system's own light and dark, and the app's four
// palettes, each with the base light or dark it draws on.
func TestThemeChoices(t *testing.T) {
	want := map[string]bool{
		"system": false, "light": false, "dark": true,
		"green": false, "amber": false, "sky": false, "gold": true,
		"neon": true, "sakura": false, "navy": true, "sepia": false, "brass": true, "wine": false,
	}
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
	if themeChoiceOf("teal").id != "sky" {
		t.Errorf("deep teal, which sky took the place of, isn't sky")
	}
	if themeChoiceOf("nonsense").id != "system" {
		t.Errorf("a theme no longer offered isn't the system's")
	}
}

// Each palette is its own, and the light and dark ones are the themes' own
// colors.
func TestPalettes(t *testing.T) {
	palettes := map[string]palette{
		"green":  greenPalette(),
		"amber":  amberPalette(),
		"sky":    skyPalette(),
		"gold":   goldPalette(),
		"neon":   neonPalette(),
		"sakura": sakuraPalette(),
		"navy":   navyPalette(),
		"sepia":  sepiaPalette(),
		"brass":  brassPalette(),
		"wine":   winePalette(),
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
	for _, id := range []string{"green", "amber", "sky", "sakura", "sepia", "wine"} {
		if palettes[id].dark {
			t.Errorf("%s should be light", id)
		}
	}
	for _, id := range []string{"gold", "neon", "navy", "brass"} {
		if !palettes[id].dark {
			t.Errorf("%s should be dark", id)
		}
	}
}
