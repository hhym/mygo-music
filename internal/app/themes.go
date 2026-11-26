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
	{"sky", N("☁ Sky")},
	{"gold", N("☾ Black Gold")},
	{"neon", N("⚡ Neon Purple")},
	{"sakura", N("❀ Sakura")},
	{"navy", N("⚓ Navy")},
	{"sepia", N("✤ Retro Brown")},
	{"brass", N("❦ Dark Green Brass")},
	{"wine", N("❧ Wine Cream")},
}

// themeChoiceOf is the choice of id, the system's own for one no longer offered,
// as settings saved before may hold.
func themeChoiceOf(id string) themeChoice {
	if id == "teal" { // 深青, which 天青 took the place of
		id = "sky"
	}
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
	case "light", "green", "amber", "sky", "sakura", "sepia", "wine":
		return false
	case "dark", "gold", "neon", "navy", "brass":
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
	case "sky":
		return skyPalette()
	case "gold":
		return goldPalette()
	case "neon":
		return neonPalette()
	case "sakura":
		return sakuraPalette()
	case "navy":
		return navyPalette()
	case "sepia":
		return sepiaPalette()
	case "brass":
		return brassPalette()
	case "wine":
		return winePalette()
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

// skyPalette is 天青: the pale cyan of a clear morning sky, with a deep azure
// accent.
func skyPalette() palette {
	return palette{
		bg:             ui.Hex("#eef7f7"),
		sidebar:        ui.Hex("#e2f0f1"),
		text:           ui.Hex("#163339"),
		muted:          ui.RGBA(22, 51, 57, 0.62),
		faint:          ui.RGBA(22, 51, 57, 0.34),
		hairline:       ui.RGBA(20, 80, 90, 0.12),
		hover:          ui.RGBA(20, 90, 100, 0.06),
		press:          ui.RGBA(20, 90, 100, 0.11),
		accent:         ui.Hex("#2a9cb8"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(20, 50, 60, 0.14),
		surface:        ui.Hex("#e7f2f3"),
		surfaceHover:   ui.Hex("#dcecee"),
		surfacePressed: ui.Hex("#d0e4e7"),
		border:         ui.Hex("#c6dee2"),
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

// neonPalette is 紫霓虹: a night of near black violet, lit by neon purple.
func neonPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#0b0710"),
		sidebar:        ui.Hex("#130d1b"),
		text:           ui.Hex("#f0e8ff"),
		muted:          ui.RGBA(230, 215, 255, 0.58),
		faint:          ui.RGBA(230, 215, 255, 0.30),
		hairline:       ui.RGBA(180, 120, 255, 0.16),
		hover:          ui.RGBA(180, 120, 255, 0.08),
		press:          ui.RGBA(180, 120, 255, 0.14),
		accent:         ui.Hex("#b06bff"),
		onAccent:       ui.Hex("#14061f"),
		cardShade:      ui.RGBA(0, 0, 0, 0.55),
		surface:        ui.Hex("#180f24"),
		surfaceHover:   ui.Hex("#221430"),
		surfacePressed: ui.Hex("#2b1a3c"),
		border:         ui.Hex("#38224a"),
	}
}

// sakuraPalette is 樱花粉: the pink of blossoms on a spring page.
func sakuraPalette() palette {
	return palette{
		bg:             ui.Hex("#fdf2f5"),
		sidebar:        ui.Hex("#fae7ee"),
		text:           ui.Hex("#3a2229"),
		muted:          ui.RGBA(90, 50, 62, 0.62),
		faint:          ui.RGBA(90, 50, 62, 0.34),
		hairline:       ui.RGBA(150, 60, 90, 0.12),
		hover:          ui.RGBA(200, 100, 130, 0.06),
		press:          ui.RGBA(200, 100, 130, 0.11),
		accent:         ui.Hex("#d9678c"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(120, 60, 80, 0.14),
		surface:        ui.Hex("#fbe9ef"),
		surfaceHover:   ui.Hex("#f6dde6"),
		surfacePressed: ui.Hex("#f0d0dc"),
		border:         ui.Hex("#e8c4d1"),
	}
}

// navyPalette is 海军蓝: a dark the color of a dress uniform, with a blue accent.
func navyPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#08111f"),
		sidebar:        ui.Hex("#0c1a2c"),
		text:           ui.Hex("#e6effa"),
		muted:          ui.RGBA(200, 220, 245, 0.60),
		faint:          ui.RGBA(200, 220, 245, 0.32),
		hairline:       ui.RGBA(110, 160, 230, 0.16),
		hover:          ui.RGBA(110, 160, 230, 0.08),
		press:          ui.RGBA(110, 160, 230, 0.14),
		accent:         ui.Hex("#4c86d9"),
		onAccent:       ui.Hex("#061020"),
		cardShade:      ui.RGBA(0, 0, 0, 0.50),
		surface:        ui.Hex("#102138"),
		surfaceHover:   ui.Hex("#162c47"),
		surfacePressed: ui.Hex("#1c3655"),
		border:         ui.Hex("#284465"),
	}
}

// sepiaPalette is 暖褐复古: the brown of old paper and worn leather.
func sepiaPalette() palette {
	return palette{
		bg:             ui.Hex("#f2e7d5"),
		sidebar:        ui.Hex("#ead9c0"),
		text:           ui.Hex("#3b2c1c"),
		muted:          ui.RGBA(80, 60, 40, 0.64),
		faint:          ui.RGBA(80, 60, 40, 0.36),
		hairline:       ui.RGBA(110, 80, 45, 0.14),
		hover:          ui.RGBA(140, 100, 55, 0.06),
		press:          ui.RGBA(140, 100, 55, 0.11),
		accent:         ui.Hex("#8f5c2e"),
		onAccent:       ui.Hex("#fff8ec"),
		cardShade:      ui.RGBA(90, 60, 30, 0.16),
		surface:        ui.Hex("#ecdfc9"),
		surfaceHover:   ui.Hex("#e3d2b8"),
		surfacePressed: ui.Hex("#d9c4a5"),
		border:         ui.Hex("#cdb491"),
	}
}

// brassPalette is 墨绿黄铜: an ink green dark with brass fittings.
func brassPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#0a1310"),
		sidebar:        ui.Hex("#10201a"),
		text:           ui.Hex("#e9f0e2"),
		muted:          ui.RGBA(215, 235, 205, 0.58),
		faint:          ui.RGBA(215, 235, 205, 0.30),
		hairline:       ui.RGBA(201, 162, 39, 0.16),
		hover:          ui.RGBA(201, 162, 39, 0.08),
		press:          ui.RGBA(201, 162, 39, 0.14),
		accent:         ui.Hex("#c9a227"),
		onAccent:       ui.Hex("#12160a"),
		cardShade:      ui.RGBA(0, 0, 0, 0.55),
		surface:        ui.Hex("#14241b"),
		surfaceHover:   ui.Hex("#1b3024"),
		surfacePressed: ui.Hex("#22392c"),
		border:         ui.Hex("#314a37"),
	}
}

// winePalette is 酒红米白: wine red against the cream of a tablecloth.
func winePalette() palette {
	return palette{
		bg:             ui.Hex("#fdf5f0"),
		sidebar:        ui.Hex("#f6e6df"),
		text:           ui.Hex("#3a2320"),
		muted:          ui.RGBA(90, 50, 45, 0.62),
		faint:          ui.RGBA(90, 50, 45, 0.34),
		hairline:       ui.RGBA(140, 50, 50, 0.12),
		hover:          ui.RGBA(150, 60, 60, 0.06),
		press:          ui.RGBA(150, 60, 60, 0.11),
		accent:         ui.Hex("#8c2f3d"),
		onAccent:       ui.Hex("#fff5f2"),
		cardShade:      ui.RGBA(90, 40, 40, 0.14),
		surface:        ui.Hex("#f9ebe6"),
		surfaceHover:   ui.Hex("#f2ded6"),
		surfacePressed: ui.Hex("#ead0c6"),
		border:         ui.Hex("#dcbdb2"),
	}
}
