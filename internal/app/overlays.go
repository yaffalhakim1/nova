package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
)

// The overlays the window can show above the home screen.
const (
	overlayNone = iota
	overlaySearch
	overlayPlaying
	overlayAddFolder
	overlayAddGame
	overlaySteamKey
)

// shortcuts wires the keyboard the design lists: the arrows browse, Enter
// plays, Down opens the game's page, Escape goes back, / searches, N adds and F
// favourites.
func (a *app) shortcuts(c *ui.Context) {
	if a.overlay == overlayNone && a.current == nil {
		if c.Shortcut(0, ui.KeyLeft) {
			a.moveSelection(-1)
		}
		if c.Shortcut(0, ui.KeyRight) {
			a.moveSelection(1)
		}
		if c.Shortcut(0, ui.KeyEnter) {
			if e := a.focused(); e != nil {
				a.playSound("select")
				a.play(*e)
			}
		}
		if c.Shortcut(0, ui.KeyDown) {
			if e := a.focused(); e != nil {
				a.playSound("select")
				a.open(*e)
			}
		}
		if c.Shortcut(0, ui.KeyEscape) {
			a.playSound("back")
			a.current = nil
		}
	}

	if c.Shortcut(0, ui.KeySlash) {
		a.playSound("tick")
		a.overlay = overlaySearch
		a.searchQuery = ""
	}

	// 1, 2 and 3 choose a section, as the design's tabs do by number.
	if a.overlay == overlayNone {
		if c.Shortcut(0, ui.Key1) {
			a.section = sectionHome
			a.lastKey = -1
			a.playSound("tick")
		}
		if c.Shortcut(0, ui.Key2) {
			a.section = sectionFavorites
			a.lastKey = -1
			a.playSound("tick")
		}
		if c.Shortcut(0, ui.Key3) {
			a.section = sectionLibrary
			a.lastKey = -1
			a.playSound("tick")
		}
	}

	// N adds a game by picking its folder.
	if c.Shortcut(0, ui.KeyN) {
		a.playSound("tick")
		a.openAddGame()
	}

	if a.current != nil && c.Shortcut(0, ui.KeyEscape) {
		a.playSound("back")
		a.current = nil
	}

	if e := a.focused(); e != nil && c.Shortcut(0, ui.KeyF) {
		a.toggleFavorite(*e)
	}
}

// moveSelection moves the focus and ticks, or bumps at either end, as the
// design's move() does.
func (a *app) moveSelection(delta int) {
	next := a.selected + delta
	if next < 0 || next >= len(a.entries) {
		a.playSound("bump")
		return
	}
	a.selected = next
	a.playSound("tick")
}

// openAddGame shows the add form on a clean draft.
func (a *app) openAddGame() {
	a.draft = addDraft{}
	a.overlay = overlayAddGame
	a.addGameOpen = true
}

// openURL opens an address in the user's browser.
func openURL(target string) error { return mygo.Shell.OpenExternal(target) }

// say shows a toast, once, on the frame after the message is set.
func (a *app) say(title, detail string) {
	a.toast = title + " · " + detail
}

// flushToast shows the pending message and clears it, so it is shown once.
func (a *app) flushToast(c *ui.Context) {
	if a.toast == "" {
		return
	}
	c.Toast(a.toast)
	a.toast = ""
}

// searchOverlay finds an installed game and takes the carousel to it.
func (a *app) searchOverlay(c *ui.Context) {
	ui.Modal(c, &a.searchOpen, func() {
		if !a.searchOpen {
			a.overlay = overlayNone
			return
		}

		ui.Text(c, "Find a game").FontSize(32).FontWeight(200).TextColor(ink)
		ui.TextInput(c, &a.searchQuery).
			Placeholder("Type a game name").
			AutoFocus().
			Label("Game name")

		matches := a.searchEntries()
		if a.searchQuery == "" {
			ui.Text(c, fmt.Sprintf("%d games installed.", len(a.installs))).FontSize(13).TextColor(inkFaint)
			a.closeRow(c)
			return
		}
		if len(matches) == 0 {
			ui.Text(c, "Nothing matches that.").FontSize(15).TextColor(inkDim)
			a.closeRow(c)
			return
		}

		ui.Column(c).Gap(2).Children(func() {
			for _, e := range matches {
				a.searchResult(c, e)
			}
		})
		a.closeRow(c)
	})
}

// searchEntries is the installed games the query names, best first.
func (a *app) searchEntries() []entry {
	if strings.TrimSpace(a.searchQuery) == "" {
		return nil
	}

	all := a.allEntries()
	matched := make([]entry, 0, len(all))
	for _, e := range all {
		if matchesQuery(e, a.searchQuery) {
			matched = append(matched, e)
		}
	}

	// A name that starts with the query comes before one that contains it.
	q := strings.ToLower(a.searchQuery)
	sort.SliceStable(matched, func(i, j int) bool {
		return strings.HasPrefix(strings.ToLower(matched[i].name()), q) &&
			!strings.HasPrefix(strings.ToLower(matched[j].name()), q)
	})

	if len(matched) > 8 {
		matched = matched[:8]
	}
	return matched
}

// searchResult is one row of the search overlay.
func (a *app) searchResult(c *ui.Context, e entry) {
	ui.Row(c).
		Padding(10, 12).
		Gap(14).
		Radius(14).
		AlignItems(ui.Center).
		Label("Open " + e.name()).
		OnClick(func() {
			a.overlay = overlayNone
			a.selectAndOpen(e)
		}).
		Children(func() {
			ui.Box(c).Size(44, 44).Radius(10).
				Draw(func(p *ui.Painter, r ui.Rect) { a.paintGame(p, r, 10, e, false) })
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, e.name()).FontSize(18).TextColor(ink).SingleLine()
				ui.Text(c, a.genreLine(e)).FontSize(13).TextColor(inkDim).SingleLine()
			})
		})
}

// closeRow is the overlay's Cancel, which the design pairs with its action.
func (a *app) closeRow(c *ui.Context) {
	ui.Row(c).Gap(12).Justify(ui.End).Children(func() {
		ui.Button(c, "Close").Radius(999).Padding(13, 28).OnClick(func() {
			a.overlay = overlayNone
		})
	})
}

// addFolderOverlay lets a games folder be added to the scan, which is what the
// design's "Add a game" does: the launcher finds the game in the folder rather
// than taking its name and path by hand.
func (a *app) addFolderOverlay(c *ui.Context) {
	ui.Modal(c, &a.addOpen, func() {
		if !a.addOpen {
			a.overlay = overlayNone
			return
		}

		ui.Text(c, "Add a games folder").FontSize(32).FontWeight(200).TextColor(ink)
		ui.Text(c, "Point the launcher at a folder whose children are games. It looks the art and description up itself.").
			FontSize(14).TextColor(inkDim)
		ui.TextInput(c, &a.folderInput).
			Placeholder(`Y:\games`).
			AutoFocus().
			Label("Games folder")

		ui.Text(c, "Already looking in: "+a.libCfg.Describe()).FontSize(13).TextColor(inkFaint)

		ui.Row(c).Gap(12).Justify(ui.End).Children(func() {
			primaryButton(c, "Add folder", func() { a.addFolder() })
			ui.Button(c, "Cancel").Radius(999).Padding(13, 28).OnClick(func() {
				a.overlay = overlayNone
			})
		})
	})
}

// addFolder saves the folder and rescans, so its games appear at once.
func (a *app) addFolder() {
	dir := strings.TrimSpace(a.folderInput)
	if dir == "" {
		return
	}

	for _, f := range a.libCfg.Folders {
		if strings.EqualFold(f, dir) {
			a.overlay = overlayNone
			a.say("Already added", dir)
			return
		}
	}

	a.libCfg.Folders = append(a.libCfg.Folders, dir)
	a.overlay = overlayNone
	a.say("Scanning", dir)

	go func() {
		_ = library.SaveConfig(a.dataDir, a.libCfg)
		a.update(func() { a.scanLibrary() })
	}()
}

// hintBar is the row of key hints at the foot of the window.
func (a *app) hintBar(c *ui.Context) {
	type hint struct{ keys, action string }
	hints := []hint{
		{"← →", "Browse"},
		{"Enter", "Play"},
		{"↓", "Details"},
		{"F", "Favorite"},
		{"/", "Search"},
		{"N", "Add"},
		{"S", "Settings"},
		{"Esc", "Back"},
	}

	ui.Row(c).Height(28).Justify(ui.Center).Gap(22).AlignItems(ui.Center).Children(func() {
		for _, h := range hints {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, h.keys).FontSize(12).TextColor(inkFaint).
					Padding(1, 7).Radius(6).Border(1, inkFaint)
				ui.Text(c, h.action).FontSize(12).TextColor(inkFaint)
			})
		}
	})
}
