package app

import (
	"github.com/egoist/mygo/ui"
)

// themeChoice is a theme the settings offer: the colors of the windows, and
// the name it goes by, marked with its own sign.
type themeChoice struct {
	id, label string
}

// themeChoices are the appearances, the system's own light and dark first, then
// the palettes of the app. IDs are saved in the settings, so they stay.
var themeChoices = []themeChoice{
	{"system", N("Follow System")},
	{"light", N("✻ Light")},
	{"dark", N("◐ Dark")},
	{"green", N("✿ Eye Care")},
	{"amber", N("☆ Amber")},
	{"teal", N("❋ Deep Teal")},
	{"gold", N("☾ Black Gold")},
}

// themeChoiceOf is the choice of id, the system's own for one no longer offered,
// as settings saved before may hold.
func themeChoiceOf(id string) themeChoice {
	for _, c := range themeChoices {
		if c.id == id {
			return c
		}
	}
	return themeChoices[0]
}

// themeDark reports whether the appearance is dark, the system's own when it
// follows the system.
func themeDark(choice themeChoice, systemDark bool) bool {
	switch choice.id {
	case "light", "green", "amber":
		return false
	case "dark", "teal", "gold":
		return true
	}
	return systemDark
}

// applyTheme makes c light, dark or one of the app's palettes, and returns the
// palette the views draw with. The system's light or dark theme is the base of
// each: the palette colors what the app draws, and the theme what MyGo's own
// widgets draw, so the two agree.
func applyTheme(c *ui.Context, appearance string) palette {
	choice := themeChoiceOf(appearance)
	dark := themeDark(choice, c.Theme().Dark)
	p := paletteOf(choice.id, dark)
	t := *c.Theme()
	t.Dark = dark
	t.Background = p.bg
	t.Surface, t.SurfaceHover, t.SurfacePressed = p.surface, p.surfaceHover, p.surfacePressed
	t.Border = p.border
	t.Text, t.TextMuted = p.text, p.muted
	t.Scrollbar = ui.RGBA(0, 0, 0, 0.32)
	t.Selection = p.accent.Alpha(0.3)
	if dark {
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.35)
		t.Selection = p.accent.Alpha(0.4)
	}
	useAccent(&t, p)
	c.SetTheme(&t)
	return p
}

// paletteOf is the palette of the appearance, by the base light or dark one.
func paletteOf(id string, dark bool) palette {
	switch id {
	case "green":
		return greenPalette()
	case "amber":
		return amberPalette()
	case "teal":
		return tealPalette()
	case "gold":
		return goldPalette()
	}
	if dark {
		return darkPalette()
	}
	return lightPalette()
}

// greenPalette is 护眼绿: the pale green of a reading page, with a moss accent.
func greenPalette() palette {
	return palette{
		bg:             ui.Hex("#edf2e6"),
		sidebar:        ui.Hex("#e2e9d8"),
		text:           ui.Hex("#22301d"),
		muted:          ui.RGBA(45, 65, 38, 0.66),
		faint:          ui.RGBA(45, 65, 38, 0.36),
		hairline:       ui.RGBA(40, 60, 30, 0.12),
		hover:          ui.RGBA(40, 70, 25, 0.06),
		press:          ui.RGBA(40, 70, 25, 0.11),
		accent:         ui.Hex("#4a7c4e"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(35, 45, 30, 0.16),
		surface:        ui.Hex("#e4eadb"),
		surfaceHover:   ui.Hex("#dbe3d0"),
		surfacePressed: ui.Hex("#d0dac4"),
		border:         ui.Hex("#cbd5be"),
	}
}

// amberPalette is 橘黄: warm paper, with a burned orange accent.
func amberPalette() palette {
	return palette{
		bg:             ui.Hex("#fdf6ea"),
		sidebar:        ui.Hex("#f6ecda"),
		text:           ui.Hex("#35281a"),
		muted:          ui.RGBA(70, 50, 30, 0.66),
		faint:          ui.RGBA(70, 50, 30, 0.36),
		hairline:       ui.RGBA(90, 60, 25, 0.12),
		hover:          ui.RGBA(120, 80, 20, 0.06),
		press:          ui.RGBA(120, 80, 20, 0.11),
		accent:         ui.Hex("#c06a18"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(60, 40, 20, 0.16),
		surface:        ui.Hex("#f8efe0"),
		surfaceHover:   ui.Hex("#f2e4cc"),
		surfacePressed: ui.Hex("#ebd9bc"),
		border:         ui.Hex("#e3cfae"),
	}
}

// tealPalette is 深青: a dark of deep water, with a bright teal accent.
func tealPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#0c191c"),
		sidebar:        ui.Hex("#102428"),
		text:           ui.Hex("#e4f1ef"),
		muted:          ui.RGBA(200, 235, 230, 0.58),
		faint:          ui.RGBA(200, 235, 230, 0.32),
		hairline:       ui.RGBA(120, 220, 210, 0.14),
		hover:          ui.RGBA(120, 220, 210, 0.07),
		press:          ui.RGBA(120, 220, 210, 0.12),
		accent:         ui.Hex("#2fb8a6"),
		onAccent:       ui.Hex("#04211e"),
		cardShade:      ui.RGBA(0, 0, 0, 0.5),
		surface:        ui.Hex("#123033"),
		surfaceHover:   ui.Hex("#183c40"),
		surfacePressed: ui.Hex("#1e4649"),
		border:         ui.Hex("#2a5451"),
	}
}

// goldPalette is 黑金: near black, with an antique gold accent.
func goldPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#0c0b09"),
		sidebar:        ui.Hex("#121008"),
		text:           ui.Hex("#f1e7d2"),
		muted:          ui.RGBA(235, 220, 190, 0.58),
		faint:          ui.RGBA(235, 220, 190, 0.32),
		hairline:       ui.RGBA(212, 175, 55, 0.16),
		hover:          ui.RGBA(212, 175, 55, 0.08),
		press:          ui.RGBA(212, 175, 55, 0.14),
		accent:         ui.Hex("#d4af37"),
		onAccent:       ui.Hex("#1a1508"),
		cardShade:      ui.RGBA(0, 0, 0, 0.55),
		surface:        ui.Hex("#17140c"),
		surfaceHover:   ui.Hex("#211c12"),
		surfacePressed: ui.Hex("#2a2417"),
		border:         ui.Hex("#3a3120"),
	}
}
