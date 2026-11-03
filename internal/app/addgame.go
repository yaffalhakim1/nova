package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
)

// addDraft is the half-typed form of "Add a game", kept on the app so a frame
// never resets what the user is in the middle of typing.
type addDraft struct {
	dir   string
	exe   string
	name  string
	genre string
	about string
	// hue is the cover colour, 0 meaning "derive it from the name".
	hue int
	// problem says why the last choice could not be used, instead of a silent
	// no-op.
	problem string
	// picking is true while the system's folder dialog is open, so the button
	// cannot start a second one.
	picking bool
}

// hueChoices are the design's cover colours, with the names it gives them.
var hueChoices = []struct {
	name string
	hue  int
}{
	{"Blue", 215},
	{"Violet", 275},
	{"Pink", 335},
	{"Red", 12},
	{"Amber", 38},
	{"Green", 150},
	{"Teal", 185},
}

// addGameOverlay adds one game by picking its folder, which is the case a scan
// of a whole games folder cannot help with: a single repack, in its own folder.
func (a *app) addGameOverlay(c *ui.Context) {
	ui.Modal(c, &a.addGameOpen, func() {
		if !a.addGameOpen {
			a.overlay = overlayNone
			return
		}

		ui.Text(c, "Add a game").FontSize(32).FontWeight(200).TextColor(ink)
		ui.Text(c, "Pick the game's own folder. It starts from there, and Steam's store page is still looked up by name.").
			FontSize(14).TextColor(inkDim)

		a.pickFolderRow(c)

		if a.draft.problem != "" {
			ui.Text(c, a.draft.problem).FontSize(13).TextColor(c.Theme().Danger)
		}

		if a.draft.dir != "" {
			a.addGameFields(c)
		}

		ui.Row(c).Gap(12).Justify(ui.End).Children(func() {
			add := primaryButton(c, "Add game", func() { a.commitAddGame() })
			if a.draft.dir == "" {
				// Nothing to add yet: the action is visibly unavailable rather
				// than a button that does nothing.
				add.Opacity(0.5)
			}
			ui.Button(c, "Cancel").Radius(999).Padding(13, 28).OnClick(func() {
				a.closeAddGame()
			})
		})
	})
}

// pickFolderRow is the way in: the system's own folder dialog, never a typed
// path, so a mistyped folder cannot become a game that will not start.
func (a *app) pickFolderRow(c *ui.Context) {
	label := "Choose folder…"
	if a.draft.picking {
		label = "Choosing…"
	}
	// AlignSelf(Start) keeps the button its own width: a row's child stretches
	// to fill by default, which would make this button full-width.
	b := ui.Button(c, label).Radius(999).Padding(13, 28).AlignSelf(ui.Start)
	if !a.draft.picking {
		b.OnClick(func() { a.chooseGameFolder() })
	}
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		_ = b
		if a.draft.dir != "" {
			ui.Text(c, trimPath(a.draft.dir)).FontSize(13).TextColor(inkFaint).SingleLine()
		}
	})
}

// addGameFields is the rest of the form: what cannot be inferred.
func (a *app) addGameFields(c *ui.Context) {
	ui.TextInput(c, &a.draft.name).
		Placeholder("Game name").
		Label("Name")

	ui.Row(c).Gap(12).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.TextInput(c, &a.draft.genre).
				Placeholder("Action").
				Label("Genre")
		})

		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, "Cover colour").FontSize(13).TextColor(inkDim)
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for _, choice := range hueChoices {
					a.hueSwatch(c, choice.name, choice.hue)
				}
			})
		})
	})

	ui.TextArea(c, &a.draft.about).
		Placeholder("What is this game about?").
		Label("Description")

	if a.draft.exe != "" {
		ui.Text(c, "Starts "+a.draft.exe).FontSize(12).TextColor(inkFaint).SingleLine()
	}
}

// hueSwatch is one cover colour, chosen by name so the choice is not colour
// alone: the chosen one is ringed and checked.
func (a *app) hueSwatch(c *ui.Context, name string, hue int) {
	on := a.draft.hue == hue
	b := ui.Box(c).
		Size(26, 26).
		Radius(999).
		Background(hsl(float32(hue), 0.75, 0.45)).
		Center().
		Label(name).
		OnClick(func() { a.draft.hue = hue })
	if on {
		b.Border(2, white)
		b.Children(func() {
			icon(c, "check", 14).TextColor(white)
		})
	}
}

// chooseGameFolder opens the system's folder dialog and reads what it returns.
//
// Dialog.Open blocks until the user answers, so it runs on its own goroutine and
// comes back through a.update: calling it on the view's goroutine would freeze
// the window for as long as the dialog is open.
func (a *app) chooseGameFolder() {
	if a.draft.picking {
		return
	}
	a.draft.picking = true
	a.draft.problem = ""

	win := a.win
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       "Choose the game's folder",
			ButtonLabel: "Add this game",
		})
		a.update(func() { a.folderChosen(paths, err) })
	}()
}

// folderChosen applies the dialog's answer. Cancelling is a normal outcome and
// leaves the form as it was.
func (a *app) folderChosen(paths []string, err error) {
	a.draft.picking = false

	if err != nil {
		a.draft.problem = "Could not open the folder: " + err.Error()
		return
	}
	if len(paths) == 0 {
		return
	}

	m, ok := library.AddManual(paths[0])
	if !ok {
		a.draft.problem = "That folder has no game in it. Pick the folder the game is installed in, " +
			"with its executable inside."
		return
	}

	a.draft.dir = m.Dir
	a.draft.exe = m.Exe
	a.draft.name = m.Name
	a.draft.problem = ""
}

// commitAddGame saves the form as a manual game and rescans, so it appears at
// once.
func (a *app) commitAddGame() {
	if a.draft.dir == "" {
		return
	}

	name := strings.TrimSpace(a.draft.name)
	if name == "" {
		name = cleanFolderName(a.draft.dir)
	}
	if name == "" {
		a.draft.problem = "Give the game a name."
		return
	}

	game := library.ManualGame{
		Dir:         a.draft.dir,
		Exe:         a.draft.exe,
		Name:        name,
		Genre:       strings.TrimSpace(a.draft.genre),
		Description: strings.TrimSpace(a.draft.about),
		Hue:         a.draft.hue,
	}

	a.libCfg.AddManual(game)

	// Saved before the scan reads it, and on this goroutine: a background write
	// could outlive the window and lose the game it just added.
	if err := library.SaveConfig(a.dataDir, a.libCfg); err != nil {
		a.draft.problem = "Could not save the game: " + err.Error()
		return
	}

	a.closeAddGame()
	a.lastKey = -1
	a.refresh()
	a.say(name, "added to your library")
}

// closeAddGame shuts the overlay and forgets the draft, so the next add starts
// clean.
func (a *app) closeAddGame() {
	a.addGameOpen = false
	a.overlay = overlayNone
	a.draft = addDraft{}
}

// cleanFolderName is the last-resort name for a picked folder.
func cleanFolderName(dir string) string {
	return strings.TrimSpace(filepath.Base(filepath.Clean(dir)))
}
