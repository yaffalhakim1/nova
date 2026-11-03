package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
)

// writeGame makes a folder with a game executable in it, for a manual add.
func writeGame(t *testing.T, dir, exe string) {
	t.Helper()
	path := filepath.Join(dir, exe)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// addGameTo runs the add flow the way the folder dialog does, without a dialog.
func addGameTo(t *testing.T, a *app, dir string) {
	t.Helper()
	a.openAddGame()
	a.folderChosen([]string{dir}, nil)
}

// TestAddGameFlow checks a picked folder becomes a game in the library, with the
// metadata the user typed, and that the config really holds it.
func TestAddGameFlow(t *testing.T) {
	a := launcher(t, nil)
	dir := t.TempDir()
	writeGame(t, filepath.Join(dir, "Hollow Grove"), `Binaries\HollowGrove.exe`)

	addGameTo(t, a, filepath.Join(dir, "Hollow Grove"))
	if a.draft.problem != "" {
		t.Fatalf("adding failed: %s", a.draft.problem)
	}

	a.draft.name = "Hollow Grove"
	a.draft.genre = "Exploration"
	a.draft.about = "A quiet forest."
	a.draft.hue = 150
	a.commitAddGame()

	if a.addGameOpen {
		t.Fatal("the form stayed open after adding")
	}

	// The library rescans, so the game is there at once.
	a.installs = library.Detect(a.libCfg.Roots())
	a.lastKey = -1
	a.refresh()

	var found *entry
	for i := range a.entries {
		if a.entries[i].name() == "Hollow Grove" {
			found = &a.entries[i]
		}
	}
	if found == nil {
		t.Fatalf("the added game is not in the library: %+v", a.entries)
	}
	if found.inst.Source != library.Manual {
		t.Errorf("source = %q", found.inst.Source)
	}
	if found.inst.Genre != "Exploration" {
		t.Errorf("genre = %q", found.inst.Genre)
	}
	if a.genreLine(*found) != "Exploration" {
		t.Errorf("genre line = %q", a.genreLine(*found))
	}
	if a.describe(*found) != "A quiet forest." {
		t.Errorf("description = %q", a.describe(*found))
	}

	// The choice survives a restart.
	reloaded := library.LoadConfig(a.dataDir)
	if _, ok := reloaded.ManualFor(found.inst.Dir); !ok {
		t.Fatal("the added game was not saved to the config")
	}
}

// TestAddGameRejectsAFolderWithNoGame checks the user is told why a folder was
// refused, rather than the form silently doing nothing.
func TestAddGameRejectsAFolderWithNoGame(t *testing.T) {
	a := launcher(t, nil)
	a.openAddGame()
	a.folderChosen([]string{t.TempDir()}, nil)

	if a.draft.problem == "" {
		t.Fatal("an empty folder was accepted without a word")
	}
	if a.draft.dir != "" {
		t.Fatal("an empty folder became a game")
	}
}

// TestAddGameHandlesCancel checks the dialog's cancel leaves the form usable.
func TestAddGameHandlesCancel(t *testing.T) {
	a := launcher(t, nil)
	a.openAddGame()
	a.folderChosen(nil, nil)

	if a.draft.problem != "" {
		t.Errorf("cancelling reported a problem: %s", a.draft.problem)
	}
	if a.draft.picking {
		t.Error("the form is stuck in the picking state")
	}
	if !a.addGameOpen {
		t.Error("cancelling the folder dialog closed the form")
	}
}

// TestAddGameFormPaints checks the overlay shows its fields and its one action,
// and that a game with no folder cannot be added yet.
func TestAddGameFormPaints(t *testing.T) {
	a := launcher(t, installsOf(2))
	a.openAddGame()

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	for _, want := range []string{"Add a game", "Choose folder…", "Add game", "Cancel"} {
		if !tt.HasText(want) {
			t.Fatalf("the form does not show %q; texts = %q", want, tt.Texts())
		}
	}
	// The fields appear only once there is a folder to describe.
	if tt.HasText("Game name") {
		t.Error("the name field showed before a folder was chosen")
	}
}

// TestManualGameUsesItsHue checks a chosen cover colour reaches the artwork,
// which is the whole point of the swatches.
func TestManualGameUsesItsHue(t *testing.T) {
	plain := coverArt(entry{inst: library.Install{Dir: `Y:\games\X`, Name: "X"}})
	picked := coverArt(entry{inst: library.Install{Dir: `Y:\games\X`, Name: "X", Hue: 150}})

	if plain.hue == picked.hue {
		t.Fatalf("the chosen hue did not change the artwork (both %v)", plain.hue)
	}
	if picked.hue != 150 {
		t.Fatalf("hue = %v, want 150", picked.hue)
	}

	// Zero means "derive it", so an unset choice changes nothing.
	unset := coverArt(entry{inst: library.Install{Dir: `Y:\games\X`, Name: "X", Hue: 0}})
	if unset.hue != plain.hue {
		t.Fatalf("an unset hue changed the artwork: %v vs %v", unset.hue, plain.hue)
	}
}

// TestAddGameShortcut checks N opens the add form rather than the old folder
// scan, which the menu still reaches.
func TestAddGameShortcut(t *testing.T) {
	a := launcher(t, installsOf(2))

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()
	tt.Key(0, ui.KeyN)
	tt.Frame()

	if !a.addGameOpen {
		t.Fatal("N did not open the add form")
	}
}
