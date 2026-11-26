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
	{"sepia", N("✤ Retro Brown")},
	{"brass", N("❦ Dark Green Brass")},
	{"wine", N("❧ Wine Cream")},
	{"neon", N("⚡ Neon Violet")},
	{"dusk", N("◑ Midnight Ember")},
	{"rose", N("✺ Electric Rose")},
	{"mist", N("◌ Mist Blue")},
	{"sage", N("❖ Sage")},
	{"oat", N("✧ Oat Apricot")},
	{"klein", N("◆ Klein Blue")},
	{"tomato", N("▲ Tomato Indigo")},
	{"sakura", N("❀ Sakura")},
	{"navy", N("⚓ Navy")},
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
	case "light", "green", "amber", "sky", "sakura", "sepia", "brass", "wine",
		"mist", "sage", "oat", "klein", "tomato":
		return false
	case "dark", "gold", "neon", "dusk", "rose", "navy":
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
	case "sepia":
		return sepiaPalette()
	case "brass":
		return brassPalette()
	case "wine":
		return winePalette()
	case "neon":
		return neonPalette()
	case "dusk":
		return duskPalette()
	case "rose":
		return rosePalette()
	case "mist":
		return mistPalette()
	case "sage":
		return sagePalette()
	case "oat":
		return oatPalette()
	case "klein":
		return kleinPalette()
	case "tomato":
		return tomatoPalette()
	case "sakura":
		return sakuraPalette()
	case "navy":
		return navyPalette()
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

// sepiaPalette is 暖褐复古: the brown of old paper and worn leather, a dot of
// brass in the accent.
func sepiaPalette() palette {
	return palette{
		bg:             ui.Hex("#f5efe6"),
		sidebar:        ui.Hex("#ede4d7"),
		text:           ui.Hex("#3e2c23"),
		muted:          ui.RGBA(62, 44, 35, 0.64),
		faint:          ui.RGBA(62, 44, 35, 0.36),
		hairline:       ui.RGBA(139, 94, 60, 0.14),
		hover:          ui.RGBA(139, 94, 60, 0.07),
		press:          ui.RGBA(139, 94, 60, 0.12),
		accent:         ui.Hex("#8b5e3c"),
		onAccent:       ui.Hex("#f5efe6"),
		cardShade:      ui.RGBA(90, 60, 40, 0.16),
		surface:        ui.Hex("#efe7db"),
		surfaceHover:   ui.Hex("#e6dccb"),
		surfacePressed: ui.Hex("#ddd1bc"),
		border:         ui.Hex("#d8c9b4"),
	}
}

// brassPalette is 墨绿黄铜: an ink green on aged paper, brass for the marks.
func brassPalette() palette {
	return palette{
		bg:             ui.Hex("#ede6d6"),
		sidebar:        ui.Hex("#e4dbc7"),
		text:           ui.Hex("#1f2a24"),
		muted:          ui.RGBA(31, 42, 36, 0.64),
		faint:          ui.RGBA(31, 42, 36, 0.36),
		hairline:       ui.RGBA(47, 79, 62, 0.16),
		hover:          ui.RGBA(176, 141, 87, 0.10),
		press:          ui.RGBA(176, 141, 87, 0.18),
		accent:         ui.Hex("#2f4f3e"),
		onAccent:       ui.Hex("#f2eee2"),
		cardShade:      ui.RGBA(47, 79, 62, 0.14),
		surface:        ui.Hex("#e7dfce"),
		surfaceHover:   ui.Hex("#ded4be"),
		surfacePressed: ui.Hex("#d3c7ac"),
		border:         ui.Hex("#c9bca0"),
	}
}

// winePalette is 酒红米白: wine red against the cream of a tablecloth.
func winePalette() palette {
	return palette{
		bg:             ui.Hex("#f7f1e8"),
		sidebar:        ui.Hex("#f0e4db"),
		text:           ui.Hex("#2b1b1f"),
		muted:          ui.RGBA(43, 27, 31, 0.64),
		faint:          ui.RGBA(43, 27, 31, 0.36),
		hairline:       ui.RGBA(123, 45, 59, 0.14),
		hover:          ui.RGBA(123, 45, 59, 0.07),
		press:          ui.RGBA(123, 45, 59, 0.12),
		accent:         ui.Hex("#7b2d3b"),
		onAccent:       ui.Hex("#f7f1e8"),
		cardShade:      ui.RGBA(90, 45, 55, 0.14),
		surface:        ui.Hex("#f1e7dd"),
		surfaceHover:   ui.Hex("#e9dbd1"),
		surfacePressed: ui.Hex("#e0cfc2"),
		border:         ui.Hex("#d9c4b5"),
	}
}

// neonPalette is 深紫霓虹: a near black violet night, lit by neon violet with a
// cyan edge.
func neonPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#0d0b1a"),
		sidebar:        ui.Hex("#131022"),
		text:           ui.Hex("#e8e6f0"),
		muted:          ui.RGBA(232, 230, 240, 0.58),
		faint:          ui.RGBA(232, 230, 240, 0.30),
		hairline:       ui.RGBA(0, 229, 255, 0.16),
		hover:          ui.RGBA(123, 47, 247, 0.10),
		press:          ui.RGBA(123, 47, 247, 0.18),
		accent:         ui.Hex("#7b2ff7"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(0, 0, 0, 0.50),
		surface:        ui.Hex("#191630"),
		surfaceHover:   ui.Hex("#221d3c"),
		surfacePressed: ui.Hex("#2a2447"),
		border:         ui.Hex("#3a3160"),
	}
}

// duskPalette is 午夜青橙: a midnight blue night with a warm orange, cold cyan
// against it.
func duskPalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#101827"),
		sidebar:        ui.Hex("#16202f"),
		text:           ui.Hex("#f2f4f8"),
		muted:          ui.RGBA(242, 244, 248, 0.58),
		faint:          ui.RGBA(242, 244, 248, 0.30),
		hairline:       ui.RGBA(57, 213, 229, 0.16),
		hover:          ui.RGBA(255, 107, 74, 0.09),
		press:          ui.RGBA(255, 107, 74, 0.16),
		accent:         ui.Hex("#ff6b4a"),
		onAccent:       ui.Hex("#20100b"),
		cardShade:      ui.RGBA(0, 0, 0, 0.50),
		surface:        ui.Hex("#1b2536"),
		surfaceHover:   ui.Hex("#242f42"),
		surfacePressed: ui.Hex("#2c394e"),
		border:         ui.Hex("#38465c"),
	}
}

// rosePalette is 电光玫红: a near black plum, lit by electric rose with an ice
// cyan edge.
func rosePalette() palette {
	return palette{
		dark:           true,
		bg:             ui.Hex("#141018"),
		sidebar:        ui.Hex("#1b1520"),
		text:           ui.Hex("#f5e9f0"),
		muted:          ui.RGBA(245, 233, 240, 0.58),
		faint:          ui.RGBA(245, 233, 240, 0.30),
		hairline:       ui.RGBA(125, 249, 255, 0.16),
		hover:          ui.RGBA(255, 46, 136, 0.10),
		press:          ui.RGBA(255, 46, 136, 0.18),
		accent:         ui.Hex("#ff2e88"),
		onAccent:       ui.Hex("#12060c"),
		cardShade:      ui.RGBA(0, 0, 0, 0.50),
		surface:        ui.Hex("#221a28"),
		surfaceHover:   ui.Hex("#2b2032"),
		surfacePressed: ui.Hex("#34273c"),
		border:         ui.Hex("#46354f"),
	}
}

// mistPalette is 雾蓝岩灰: a misty blue gray page, quiet for long reading, with
// a brass dot.
func mistPalette() palette {
	return palette{
		bg:             ui.Hex("#f4f6f8"),
		sidebar:        ui.Hex("#eaf0f5"),
		text:           ui.Hex("#2c3440"),
		muted:          ui.RGBA(44, 52, 64, 0.62),
		faint:          ui.RGBA(44, 52, 64, 0.34),
		hairline:       ui.RGBA(107, 140, 174, 0.18),
		hover:          ui.RGBA(107, 140, 174, 0.08),
		press:          ui.RGBA(107, 140, 174, 0.14),
		accent:         ui.Hex("#6b8cae"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(44, 60, 80, 0.14),
		surface:        ui.Hex("#edf1f5"),
		surfaceHover:   ui.Hex("#e3e9ef"),
		surfacePressed: ui.Hex("#d8e0e8"),
		border:         ui.Hex("#c9d4de"),
	}
}

// sagePalette is 鼠尾草绿: a low green for lists and switches, apricot for the
// dot.
func sagePalette() palette {
	return palette{
		bg:             ui.Hex("#f3f5f0"),
		sidebar:        ui.Hex("#e9eee3"),
		text:           ui.Hex("#2e3a30"),
		muted:          ui.RGBA(46, 58, 48, 0.62),
		faint:          ui.RGBA(46, 58, 48, 0.34),
		hairline:       ui.RGBA(122, 154, 126, 0.18),
		hover:          ui.RGBA(122, 154, 126, 0.08),
		press:          ui.RGBA(122, 154, 126, 0.14),
		accent:         ui.Hex("#7a9a7e"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(50, 70, 55, 0.13),
		surface:        ui.Hex("#ecf0e7"),
		surfaceHover:   ui.Hex("#e2e8dc"),
		surfacePressed: ui.Hex("#d7dfcf"),
		border:         ui.Hex("#c7d1c1"),
	}
}

// oatPalette is 燕麦米杏: oat cream with an apricot accent, the one that goes
// with anything.
func oatPalette() palette {
	return palette{
		bg:             ui.Hex("#faf6f0"),
		sidebar:        ui.Hex("#f4ede2"),
		text:           ui.Hex("#3a322b"),
		muted:          ui.RGBA(58, 50, 43, 0.62),
		faint:          ui.RGBA(58, 50, 43, 0.34),
		hairline:       ui.RGBA(200, 169, 138, 0.24),
		hover:          ui.RGBA(200, 169, 138, 0.12),
		press:          ui.RGBA(200, 169, 138, 0.20),
		accent:         ui.Hex("#c8a98a"),
		onAccent:       ui.Hex("#3a322b"),
		cardShade:      ui.RGBA(70, 55, 40, 0.13),
		surface:        ui.Hex("#f5efe6"),
		surfaceHover:   ui.Hex("#ece4d7"),
		surfacePressed: ui.Hex("#e2d8c8"),
		border:         ui.Hex("#d9ccb9"),
	}
}

// kleinPalette is 克莱因蓝: International Klein Blue on a paper white, citrus
// yellow for the dot.
func kleinPalette() palette {
	return palette{
		bg:             ui.Hex("#f7f7f2"),
		sidebar:        ui.Hex("#eeefe8"),
		text:           ui.Hex("#111111"),
		muted:          ui.RGBA(17, 17, 17, 0.60),
		faint:          ui.RGBA(17, 17, 17, 0.32),
		hairline:       ui.RGBA(0, 47, 167, 0.16),
		hover:          ui.RGBA(0, 47, 167, 0.06),
		press:          ui.RGBA(0, 47, 167, 0.11),
		accent:         ui.Hex("#002fa7"),
		onAccent:       ui.Hex("#ffffff"),
		cardShade:      ui.RGBA(20, 25, 40, 0.14),
		surface:        ui.Hex("#f0f1ea"),
		surfaceHover:   ui.Hex("#e7e9df"),
		surfacePressed: ui.Hex("#dde0d3"),
		border:         ui.Hex("#cbcebd"),
	}
}

// tomatoPalette is 番茄红+靛蓝: tomato on warm white, indigo for the text and a
// steel blue for the dot.
func tomatoPalette() palette {
	return palette{
		bg:             ui.Hex("#fff8f0"),
		sidebar:        ui.Hex("#f7ebdf"),
		text:           ui.Hex("#1d3557"),
		muted:          ui.RGBA(29, 53, 87, 0.60),
		faint:          ui.RGBA(29, 53, 87, 0.32),
		hairline:       ui.RGBA(230, 57, 70, 0.14),
		hover:          ui.RGBA(230, 57, 70, 0.06),
		press:          ui.RGBA(230, 57, 70, 0.11),
		accent:         ui.Hex("#e63946"),
		onAccent:       ui.Hex("#fff8f0"),
		cardShade:      ui.RGBA(90, 40, 40, 0.13),
		surface:        ui.Hex("#f8efe5"),
		surfaceHover:   ui.Hex("#f1e2d5"),
		surfacePressed: ui.Hex("#e8d5c5"),
		border:         ui.Hex("#dcc7b4"),
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
