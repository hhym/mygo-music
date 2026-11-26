package app

import (
	"hash/fnv"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/local"
	"mygo-music/internal/plugin"
)

// The Plugins page has two tabs, and a + to install a plugin by hand: the plugins
// installed, each with what it asks of the listener; and those others made, as
// the website's plugins page lists them: public repositories on GitHub with the
// plugins' topic. Install on one asks about it, as a link to install it does.

// communityTab is the id of the Plugins page's route on its Community tab; on
// Installed, it has none. In the route, back goes to the tab it was on.
const communityTab = "community"

// fetchCommunity lists a page of the plugins on GitHub; tests replace it.
var fetchCommunity = plugin.Community

// communityFresh is how long a list stays good. GitHub's search allows few
// requests a minute without the listener's token.
const communityFresh = time.Hour

// communityState is what the Plugins page has of the plugins on GitHub: those a
// search found, or all of them, a page at a time.
type communityState struct {
	query    string    // what's written in the search field
	searched string    // what the list is of: "" for all
	at       time.Time // when its first page came, zero until it has
	list     []plugin.Listing
	pages    int // how many pages of it came
	paged    paged
	gen      int       // counts the lists asked for, for an answer about one before to be dropped
	box      ui.Handle // the search field's input
}

// loadCommunity shows the list of the Community tab, asking GitHub for it anew
// unless it did a while ago.
func (a *app) loadCommunity() {
	d := &a.community
	if d.paged.loading || !d.at.IsZero() && time.Since(d.at) < communityFresh {
		return
	}
	a.searchCommunity(d.searched)
}

// searchCommunity lists the plugins query finds, all of them for "", from their
// first page.
func (a *app) searchCommunity(query string) {
	d := &a.community
	d.searched, d.list, d.pages, d.paged, d.at = strings.TrimSpace(query), nil, 0, paged{}, time.Time{}
	d.gen++
	a.moreCommunity()
}

// moreCommunity asks GitHub for the next page of the list.
func (a *app) moreCommunity() {
	d := &a.community
	if d.paged.loading {
		return
	}
	d.paged.loading, d.paged.err = true, ""
	query, gen := d.searched, d.gen
	page := d.pages + 1
	fetch := fetchCommunity // here, for a test may change it as one before it waits
	go func() {
		got, err := fetch(query, page)
		a.post(func() {
			if d.gen != gen {
				return // the list asked for since takes its place
			}
			d.paged.loading = false
			if err != nil {
				log.Printf("listing plugins: %v", err)
				d.paged.err = T("Couldn't load the plugins")
				return
			}
			if page == 1 {
				d.at = time.Now()
			}
			// The stars, which order the list, may have moved one already listed
			// onto this page.
			for _, l := range got.Listings {
				if !slices.ContainsFunc(d.list, func(m plugin.Listing) bool { return strings.EqualFold(m.Repo, l.Repo) }) {
					d.list = append(d.list, l)
				}
			}
			d.pages = page
			d.paged.loaded, d.paged.more = true, got.More && len(got.Listings) > 0
		})
	}()
}

// installedRepos are the repositories the plugins installed came from, in lower
// case, as GitHub takes owner/repo in any.
func (a *app) installedRepos() map[string]bool {
	repos := map[string]bool{}
	for _, lib := range a.allLibs {
		if p := pluginOf(lib); p != nil {
			if o := p.Origin(); o != nil {
				repos[strings.ToLower(o.Repo)] = true
			}
		}
	}
	return repos
}

// pluginsPage is the plugins installed, or those on GitHub, by its tab.
func (a *app) pluginsPage(c *ui.Context) {
	p := a.pal
	top, bottom := a.insets(c)
	// Keyed by the page, not its tab, for the tabs' capsule to slide over.
	ui.Scroll(c.Key("plugins")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		ui.Row(c).AlignItems(ui.Center).Gap(12).Margin(0, 0, 18, 0).Children(func() {
			ui.Text(c, T("Plugins")).FontSize(30).FontWeight(700).LetterSpacing(-0.6).Grow(1).Shrink(1).MinWidth(0).SingleLine()
			tab := 0
			if a.route.id == communityTab {
				tab = 1
			}
			if segmented(c.Key("tab"), &tab, T("Installed"), T("Community")) {
				a.choosePluginsTab(tab == 1)
			}
			ui.Box(c).Margin(0, -8, 0, 0).Children(func() { a.installMenu(c, p) })
		})
		if a.route.id == communityTab {
			a.communityPlugins(c, p)
		} else {
			a.installedPlugins(c, p)
		}
	})
}

// choosePluginsTab shows the Plugins page's Community tab, or else Installed, in
// place of the one showing: back goes to the page before, not the other tab.
func (a *app) choosePluginsTab(community bool) {
	r := a.route
	r.id = ""
	if community {
		r.id = communityTab
	}
	a.route = r
	a.installer.marked = ""
	a.load(r)
}

// communityPlugins is the plugins on GitHub, or those a search found, as cards
// with a button to install each, more loading as the page nears its end.
func (a *app) communityPlugins(c *ui.Context, p palette) {
	d := &a.community
	ui.Row(c).Gap(16).AlignItems(ui.Center).Margin(0, 0, 18, 0).Children(func() {
		ui.Text(c, T("Plugins made by others bring music from more places. Install only those you trust.")).
			FontSize(13).TextColor(p.muted).LineHeight(1.45).Grow(1).Shrink(1).MinWidth(0)
		a.communitySearch(c, p)
	})
	cols := a.columns(c, 260)
	switch {
	case !d.paged.loaded && d.paged.loading:
		s := a.pageShimmer(c)
		ui.Grid(c).Columns(cols).GapX(16).GapY(16).Children(func() {
			for i := range 6 {
				s.bone(c.Key(i), 14).Height(150)
			}
		})
	case !d.paged.loaded && d.paged.err != "":
		ui.Column(c).Padding(40, 0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, d.paged.err).FontSize(13).TextColor(p.muted)
			if ui.Button(c.Key("retry"), T("Retry")).Clicked() {
				a.moreCommunity()
			}
		})
	case !d.paged.loaded:
	case len(d.list) == 0 && d.searched != "":
		a.empty(c, Tf("No plugins found for “%s”", d.searched))
	case len(d.list) == 0:
		a.empty(c, T("No plugins yet"))
	default:
		installed := a.installedRepos()
		ui.Grid(c.Key(d.gen)).Columns(cols).GapX(16).GapY(16).Children(func() {
			for i := range d.list {
				l := &d.list[i]
				a.listingCard(c.Key(l.Repo), l, installed[strings.ToLower(l.Repo)])
			}
		})
		a.pageFoot(c, &d.paged, a.moreCommunity)
	}
}

// communitySearch is the field that searches the plugins on GitHub, as Enter
// is pressed, its few searches a minute not going on each letter. Emptied, by
// its button or by hand, the list is of them all again.
func (a *app) communitySearch(c *ui.Context, p palette) {
	d := &a.community
	// The input fills the field, under its icon, so a click anywhere on the field
	// focuses it.
	field := ui.Row(c).Width(240).Height(30).Radius(8).AlignItems(ui.Stretch).Shrink(0)
	if c.Focused(d.box) {
		field.Background(p.bg).Shadow(0, 0, 0, 3, p.accent.Alpha(0.25)).Border(1, p.accent.Alpha(0.6))
	} else {
		field.Background(p.hover).Border(1, p.hairline)
	}
	clearable := d.query != ""
	end := float32(10)
	if clearable {
		end = 28
	}
	submitted := false
	field.Children(func() {
		in := ui.TextInputBase(c.Key("search-plugins"), &d.query).Bind(&d.box)
		submitted = in.Submitted()
		in.Grow(1).MinWidth(0).PaddingStart(29).PaddingEnd(end).Placeholder(T("Search Plugins")).FontSize(13).Label(T("Search Plugins"))
		ui.Icon(c, iconSearch).Size(13, 13).TextColor(p.muted).Absolute().Left(9).Top(8.5).IgnoreInput(true)
		if !clearable {
			return
		}
		x := ui.ButtonBase(c.Key("clear")).Size(22, 22).Center().Absolute().Right(4).Top(4).
			KeepFocus().FocusRing(false).Label(T("Clear Search"))
		fill := p.faint
		if x.Hovered() {
			fill = p.muted
		}
		x.Children(func() {
			ui.Box(c).Size(14, 14).Radius(7).Center().Background(fill).Children(func() {
				ui.Icon(c, iconX).Size(8, 8).TextColor(p.bg)
			})
		})
		if x.Clicked() {
			d.query = ""
		}
	})
	q := strings.TrimSpace(d.query)
	switch {
	case submitted && q != d.searched:
		a.searchCommunity(q)
	case q == "" && d.searched != "":
		a.searchCommunity("")
	}
}

// installedPlugins are the plugins installed, a card each, in the order the
// source switcher has them: the built-in source's first.
func (a *app) installedPlugins(c *ui.Context, p palette) {
	ui.Column(c.Key("installed")).Gap(12).Children(func() {
		for _, lib := range a.allLibs {
			if lib.id == local.ID {
				if a.local != nil {
					a.localCard(c.Key(lib.id), lib, p)
				}
				continue
			}
			a.pluginCard(c.Key(lib.id), lib, p)
		}
	})
}

// pluginTileSize is the size of the square at the start of a plugin's card.
const pluginTileSize = 40

// pluginCard is a plugin: what it is, where it came from and who's signed in, and
// its controls, in its head; and under it, while it's on, what it can reach and
// what it asks of the listener.
func (a *app) pluginCard(c *ui.Context, lib *library, p palette) {
	info := lib.src.Info()
	enabled := a.pluginEnabled(lib)
	g, guards := lib.src.(guarded)
	asks := guards && (len(g.Permissions().Network) > 0 || len(g.Permissions().Process) > 0)
	var body func()
	if enabled && (asks || len(info.Settings) > 0) {
		body = func() {
			if asks {
				a.pluginPermissions(c.Key("permissions"), lib, g, p)
			}
			if len(info.Settings) > 0 {
				ui.Grid(c.Key("settings")).Columns(a.settingColumns(c)).GapX(24).GapY(14).Children(func() {
					for _, st := range info.Settings {
						a.sourceSetting(c.Key(st.Key), lib, st, p)
					}
				})
			}
		}
	}
	a.pluginFrame(c, p, lib, enabled, func() {
		a.pluginTile(c, lib)
	}, func() {
		ui.Row(c).Gap(6).AlignItems(ui.Baseline).MinWidth(0).Children(func() {
			ui.Text(c, lib.name()).FontSize(14.5).FontWeight(600).SingleLine().Shrink(1).MinWidth(0)
			if info.Version != "" {
				ui.Text(c, info.Version).FontSize(12).TextColor(p.faint).SingleLine().Shrink(0)
			}
		})
		// Where it came from, a link to its repository, and who's signed in.
		from, url := pluginFrom(lib)
		signedIn := ""
		if enabled && !lib.needsSetup && lib.account != nil {
			signedIn = Tf("Signed in as %s", lib.account.Name)
		}
		if from != "" || signedIn != "" {
			ui.Row(c).MinWidth(0).Children(func() {
				if url != "" {
					ui.Link(c, from, url).FontSize(12).TextColor(p.muted).SingleLine().Shrink(0).Tooltip(T("Open on GitHub"))
				} else if from != "" {
					ui.Text(c, from).FontSize(12).TextColor(p.muted).SingleLine().Shrink(0)
				}
				if signedIn != "" {
					if from != "" {
						signedIn = " · " + signedIn
					}
					ui.Text(c, signedIn).FontSize(12).TextColor(p.muted).SingleLine().Shrink(1).MinWidth(0)
				}
			})
		}
		if enabled && lib.needsSetup {
			notice(c, p, setupNotice(lib, info))
		}
		a.pluginUpdateError(c, lib, p)
	}, func() {
		a.pluginUpdateButton(c, lib)
		switch {
		case !enabled:
		case lib.account == nil && lib.accountUnknown():
			// A source not yet used hasn't said who's signed in: asked now, its
			// card offers no sign-in to someone who already is.
			a.loadAccount(lib)
		case lib.account != nil && lib.can("signOut"):
			if ui.Button(c.Key("sign-out"), T("Sign Out")).Clicked() {
				a.signOut(lib)
			}
		case lib.account == nil && lib.can("signIn"):
			if ui.PrimaryButton(c.Key("sign-in"), T("Sign In")).Clicked() {
				a.promptSignIn(lib)
			}
		}
		a.pluginMenu(c, lib, p)
		a.pluginSwitch(c, lib)
	}, body)
}

// localCard is the built-in source: how much music it found, and the folders it
// looks in, each to take away.
func (a *app) localCard(c *ui.Context, lib *library, p palette) {
	folders := a.local.Folders()
	enabled := a.pluginEnabled(lib)
	status := T("Add the folders where your music lives")
	if a.local.Scanning() {
		status = T("Looking for music…")
	} else if len(folders) > 0 {
		status = Tf("%d songs found", a.local.TrackCount())
	}
	var body func()
	if enabled && len(folders) > 0 {
		body = func() {
			ui.Column(c).Gap(2).Margin(-6, -8).Children(func() {
				for _, dir := range folders {
					row := ui.Row(c.Key(dir)).Height(36).PaddingX(8).Gap(10).Radius(8).AlignItems(ui.Center)
					if row.Hovered() {
						row.Background(p.hover)
					}
					row.Children(func() {
						ui.Icon(c, iconFolder).Size(15, 15).TextColor(p.muted).Shrink(0)
						ui.Text(c, dir).FontSize(12.5).SingleLine().Grow(1).Shrink(1).MinWidth(0)
						if iconButton(c.Key("remove"), iconX, 13, p.muted, Tf("Remove %s", dir)).Tooltip(T("Remove")).Clicked() {
							a.local.RemoveFolder(dir)
						}
					})
				}
			})
		}
	}
	a.pluginFrame(c, p, lib, enabled, func() {
		ui.Box(c).Size(pluginTileSize, pluginTileSize).Radius(10).Background(p.accent).Center().Shrink(0).Children(func() {
			ui.Icon(c, iconNote).Size(20, 20).TextColor(p.onAccent)
		})
	}, func() {
		ui.Text(c, T("Local Files")).FontSize(14.5).FontWeight(600).SingleLine()
		ui.Text(c, T("Built-in")+" · "+status).FontSize(12).TextColor(p.muted).SingleLine()
	}, func() {
		if enabled && len(folders) > 0 && ui.Button(c.Key("rescan"), T("Rescan")).Disabled(a.local.Scanning()).Clicked() {
			a.local.Rescan()
		}
		if enabled && ui.Button(c.Key("add-folder"), T("Add Folder…")).Clicked() {
			a.addMusicFolder()
		}
		a.pluginSwitch(c, lib)
	}, body)
}

// pluginFrame is a plugin's card: in its head, its tile, what's said of it, and
// its controls at the end; and under a hairline, body, if there's one, lined up
// with what's said. One that's off fades, but for its controls. One just
// installed is ringed in the accent, and scrolled to as it first shows.
func (a *app) pluginFrame(c *ui.Context, p palette, lib *library, enabled bool, tile, about, controls, body func()) {
	fade := float32(1)
	if !enabled {
		fade = 0.5
	}
	in := &a.installer
	card := ui.Column(c).Radius(14).Background(p.bg)
	marked := card.AnimateTo("marked", boolTo(in.marked == lib.id), ui.Animation{Duration: 240 * time.Millisecond})
	card.Border(1, p.hairline.Mix(p.accent, marked)).
		Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.03)).
		Shadow(0, 0, 0, 3*marked, p.accent.Alpha(0.18*marked))
	if in.marked == lib.id && in.reveal {
		in.reveal = false
		card.ScrollIntoView(ui.ScrollIntoViewOptions{Block: ui.ScrollCenter, Animation: &ui.Smooth})
	}
	card.Children(func() {
		ui.Row(c).Padding(14, 16).Gap(12).AlignItems(ui.Center).MinHeight(pluginTileSize + 28).Children(func() {
			ui.Box(c).Shrink(0).Opacity(fade).Children(tile)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Opacity(fade).Children(about)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(controls)
		})
		if body == nil {
			return
		}
		ui.Box(c).Height(1).Background(p.hairline)
		ui.Column(c).Padding(14, 16, 16, 16+pluginTileSize+12).Gap(14).Children(body)
	})
}

// pluginPermissions says what a plugin can reach, and asks before it may start
// programs, which can do anything the listener can. It may not until they say.
func (a *app) pluginPermissions(c *ui.Context, lib *library, g guarded, p palette) {
	perms := g.Permissions()
	ui.Column(c).Gap(10).Children(func() {
		if len(perms.Network) > 0 {
			reach := Tf("Can reach %s", strings.Join(perms.Network, ", "))
			if slices.Contains(perms.Network, "*") {
				reach = T("Can reach any website")
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconGlobe).Size(14, 14).TextColor(p.muted).Shrink(0)
				ui.Text(c, reach).FontSize(12.5).TextColor(p.muted).LineHeight(1.4).Shrink(1).MinWidth(0)
			})
		}
		if len(perms.Process) == 0 {
			return
		}
		// Allowed, it's a line as the network's is; not yet, it asks, in the
		// accent's tint, for it may be what the plugin needs to work.
		granted := g.Granted()
		box := ui.Row(c).Gap(10).AlignItems(ui.Center)
		color := p.muted
		if !granted {
			box.Padding(10, 12).Radius(10).Background(p.accent.Alpha(0.08)).Border(1, p.accent.Alpha(0.18)).Margin(2, 0, 0, 0)
			color = p.text
		}
		box.Children(func() {
			ui.Icon(c, iconTerminal).Size(14, 14).TextColor(color).Shrink(0)
			text := programsWarning(perms.Process)
			if granted {
				text = T("Allowed to start programs")
			}
			ui.Text(c, text).FontSize(12.5).TextColor(color).LineHeight(1.4).Grow(1).Shrink(1).MinWidth(0)
			label := T("Allow Programs")
			if granted {
				label = T("Stop Allowing")
			}
			if ui.Button(c.Key("grant"), label).Clicked() {
				a.grantPrograms(lib, g, !granted)
			}
		})
	})
}

// boolTo is 1 for true and 0 for false, as an animation's target.
func boolTo(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

// settingColumns is how many of a plugin's settings fit side by side in its card.
func (a *app) settingColumns(c *ui.Context) int {
	w, _ := c.Size()
	inner := w - sidebarWidth - 2*pagePad - (16 + pluginTileSize + 12) - 16
	if inner >= 2*320+24 {
		return 2
	}
	return 1
}

// ownerAvatar is the picture of an account on GitHub, at the size of a plugin's
// tile; tests replace it, to reach no network.
var ownerAvatar = func(owner string) string {
	return "https://github.com/" + url.PathEscape(owner) + ".png?size=120"
}

// pluginIcon is the picture on a plugin's tile: the icon it names, or else, for
// one installed from GitHub, its owner's picture; or "" for none.
func pluginIcon(lib *library) string {
	plug := pluginOf(lib)
	if plug == nil {
		return ""
	}
	if icon := plug.Icon(); icon != "" {
		return icon
	}
	if o := plug.Origin(); o != nil {
		owner, _, _ := strings.Cut(o.Repo, "/")
		return ownerAvatar(owner)
	}
	return ""
}

// pluginTile is the square at the start of a plugin's card: its icon, fading in
// over its monogram, which stays where there's none, or it doesn't load.
func (a *app) pluginTile(c *ui.Context, lib *library) {
	var bmp *ui.Bitmap
	if icon := pluginIcon(lib); icon != "" {
		bmp = bitmap(a.art.get(sized(icon, pluginTileSize*3), false))
	}
	box := ui.Box(c).Size(pluginTileSize, pluginTileSize).Radius(10).Shrink(0)
	in := artIn(box, bmp)
	box.Children(func() {
		if in < 1 {
			monogram(c, lib.name(), lib.id)
		}
		artImage(c, bmp, 10, in)
	})
}

// monogram is a plugin's tile: the first letter of its name, on a color of its
// own, which its ID chooses, to stay the same as it's renamed.
func monogram(c *ui.Context, name, id string) {
	letter := "?"
	for _, r := range strings.TrimSpace(name) {
		letter = strings.ToUpper(string(r))
		break
	}
	h := fnv.New32a()
	h.Write([]byte(id))
	hue := float32(h.Sum32() % 360)
	top, bottom := ui.Oklch(0.7, 0.14, hue), ui.Oklch(0.56, 0.16, hue+20)
	ui.Box(c).Size(pluginTileSize, pluginTileSize).Radius(10).Shrink(0).Center().
		Gradient(top, bottom, 180).
		Children(func() {
			ui.Text(c, letter).FontSize(18).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
		})
}

// notice says what a plugin wants before it works, in the accent's color, by a
// dot of it.
func notice(c *ui.Context, p palette, text string) {
	ui.Row(c).Gap(6).AlignItems(ui.Start).Margin(2, 0, 0, 0).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(p.accent).Margin(5, 0, 0, 0).Shrink(0)
		ui.Text(c, text).FontSize(12).TextColor(p.accent).LineHeight(1.4).Shrink(1).MinWidth(0)
	})
}

// listingCard is a plugin on GitHub: who made it and what it does, a link to its
// repository, and Install, or that it is.
func (a *app) listingCard(c *ui.Context, l *plugin.Listing, installed bool) {
	p := a.pal
	ui.Column(c).Padding(16).Gap(10).Radius(14).Background(p.hover).Border(0.5, p.hairline).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			a.avatar(c, l.Avatar, 36, p.press)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
				ui.Link(c, l.Name, l.URL).FontSize(14).FontWeight(600).TextColor(p.text).SingleLine().Tooltip(T("Open on GitHub"))
				ui.Text(c, l.Owner).FontSize(12).TextColor(p.muted).SingleLine()
			})
		})
		desc := l.Description
		if desc == "" {
			desc = l.Repo
		}
		ui.Text(c, desc).FontSize(12.5).TextColor(p.muted).LineHeight(1.4).Lines(2, 2).Grow(1)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconStar).Size(13, 13).TextColor(p.faint)
			ui.Text(c, count(l.Stars)).FontSize(12).TextColor(p.muted).Grow(1)
			if installed {
				ui.Row(c).Gap(4).AlignItems(ui.Center).Height(28).Children(func() {
					ui.Icon(c, iconCheck).Size(13, 13).TextColor(p.muted)
					ui.Text(c, T("Installed")).FontSize(12.5).FontWeight(500).TextColor(p.muted)
				})
				return
			}
			busy := a.installer.busy && strings.EqualFold(a.installer.repo, l.Repo)
			label := T("Install")
			if busy {
				label = T("Downloading…")
			}
			b := ui.Button(c.Key("install"), label).Label(Tf("Install %s", l.Name)).Disabled(busy)
			if b.Clicked() && !busy {
				a.installRepo(l.Repo)
			}
		})
	})
}
