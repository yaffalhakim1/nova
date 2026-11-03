package app

import (
	"github.com/egoist/mygo"
)

// setMenu gives the window an application menu whose first entry carries the
// app's mark, which is where Windows and Linux show it beside a menu bar.
func (a *app) setMenu() {
	icon, err := mygo.NewMenuIcon(iconData, false)

	app := &mygo.MenuItem{Label: "Wand"}
	if err == nil {
		app.Icon = icon
	}
	app.Submenu = []*mygo.MenuItem{
		{ID: "add-folder", Label: "Add games folder…", Accelerator: "CmdOrCtrl+O",
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.update(func() { a.overlay = overlayAddFolder })
			}},
		mygo.Separator(),
		{ID: "quit", Label: "Quit", Role: mygo.RoleQuit},
	}

	view := &mygo.MenuItem{Label: "View"}
	view.Submenu = []*mygo.MenuItem{
		{ID: "all", Label: "All games", Accelerator: "1",
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.update(func() { a.section = sectionHome; a.lastKey = -1 })
			}},
		{ID: "recent", Label: "Recently opened", Accelerator: "2",
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.update(func() { a.section = sectionHome; a.lastKey = -1 })
			}},
		{ID: "favorites", Label: "Favorites", Accelerator: "3",
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.update(func() { a.section = sectionFavorites; a.lastKey = -1 })
			}},
	}

	steam := &mygo.MenuItem{ID: "steam", Label: "Your Steam library…",
		Click: func(*mygo.MenuItem, *mygo.Window) {
			a.update(func() { a.openSteamKey() })
		}}

	a.win.SetMenu(mygo.NewMenu([]*mygo.MenuItem{app, view, steam}))
}
