package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// settingsItem is one action in the settings panel.
type settingsItem struct {
	label string
	run   func()
}

// settingsOverlay holds the actions the native menu bar used to carry. It is a
// wide, three-column panel: the old single column made every label narrow and
// the dialog far taller than it needed to be.
func (a *app) settingsOverlay(c *ui.Context) {
	panel := ui.DialogBase(c, &a.settingsOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(880).
			Padding(28).
			Gap(18).
			Radius(20).
			Background(c.Theme().Background).
			MaxWidth(1120).
			MaxHeight(760)
		panel.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))

		ui.Text(c, "Settings").FontSize(32).FontWeight(200).TextColor(ink)

		ui.Grid(c).Columns(3).GapX(18).GapY(14).Children(func() {
			a.settingsGroup(c, "Library", []settingsItem{
				{"Add games folder…", func() { a.overlay = overlayAddFolder }},
				{"Your Steam library…", func() { a.overlay = overlaySteamKey }},
			})
			a.settingsGroup(c, "Go to", []settingsItem{
				{"All games", func() { a.goToSection(sectionHome) }},
				{"Favorites", func() { a.goToSection(sectionFavorites) }},
				{"Library", func() { a.goToSection(sectionLibrary) }},
			})
			ui.Column(c).Gap(14).Children(func() {
				a.settingsGroup(c, "Data", []settingsItem{
					{"Export data…", func() { a.exportData(); a.overlay = overlayNone }},
					{"Import data…", func() { a.importData(); a.overlay = overlayNone }},
				})
				a.settingsGroup(c, "App", []settingsItem{
					{"Quit", func() { mygo.App.Quit() }},
				})
			})
		})

		ui.Row(c).Justify(ui.End).Children(func() {
			ui.Button(c, "Close").Radius(999).Padding(13, 28).OnClick(func() {
				a.overlay = overlayNone
			})
		})
	})
	if panel.Dismissed() {
		a.overlay = overlayNone
	}
}

// settingsGroup is one labelled column of actions.
func (a *app) settingsGroup(c *ui.Context, label string, items []settingsItem) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(inkDim)
		for _, item := range items {
			item := item
			ui.Button(c, item.label).
				Radius(12).
				Padding(12, 18).
				OnClick(func() {
					a.playSound("tick")
					item.run()
				})
		}
	})
}

// goToSection leaves the open game and the settings panel behind.
func (a *app) goToSection(section int) {
	a.current = nil
	a.section = section
	a.lastKey = -1
	a.overlay = overlayNone
}

// openSettings shows the panel from the button or the S key.
func (a *app) openSettings() {
	a.playSound("tick")
	a.overlay = overlaySettings
}
