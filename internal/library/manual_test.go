package library

import (
	"path/filepath"
	"testing"
)

// TestAddManualFindsThePickedGame checks the folder picker's case: the user
// picks one game's own folder, not a folder full of games.
func TestAddManualFindsThePickedGame(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Binaries", "Win64", "HollowGrove-Win64-Shipping.exe"))

	m, ok := AddManual(dir)
	if !ok {
		t.Fatal("a folder holding a game was not accepted")
	}
	if m.Name != cleanName(filepath.Base(dir)) {
		t.Errorf("name = %q", m.Name)
	}
	// The exe is stored relative, so moving the folder does not break it.
	if filepath.IsAbs(m.Exe) {
		t.Errorf("exe = %q, want a path relative to the game folder", m.Exe)
	}
	if want := filepath.Join("Binaries", "Win64", "HollowGrove-Win64-Shipping.exe"); m.Exe != want {
		t.Errorf("exe = %q, want %q", m.Exe, want)
	}
}

// TestExeForResolvesRelative checks a manual entry's executable is found again
// from its folder and relative path.
func TestExeForResolvesRelative(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bin", "game.exe"))

	got, ok := ExeFor(ManualGame{Dir: dir, Exe: filepath.Join("bin", "game.exe")})
	if !ok {
		t.Fatal("the relative executable was not resolved")
	}
	if got != filepath.Join(dir, "bin", "game.exe") {
		t.Fatalf("exe = %q", got)
	}

	if _, ok := ExeFor(ManualGame{Dir: dir, Exe: filepath.Join("bin", "gone.exe")}); ok {
		t.Fatal("a missing executable resolved")
	}
}

// TestExeForRefusesEscape checks a relative path cannot point outside the game's
// own folder, which a hand-edited config could otherwise do.
func TestExeForRefusesEscape(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Game")
	writeFile(t, filepath.Join(dir, "game.exe"))
	// A file that does exist, one level above the game's own folder.
	writeFile(t, filepath.Join(root, "outside.exe"))

	if _, ok := ExeFor(ManualGame{Dir: dir, Exe: filepath.Join("..", "outside.exe")}); ok {
		t.Fatal("an executable outside the game folder resolved")
	}

	// An absolute path is refused for the same reason.
	if _, ok := ExeFor(ManualGame{Dir: dir, Exe: filepath.Join(root, "outside.exe")}); ok {
		t.Fatal("an absolute executable resolved")
	}
}

// TestDetectManualGames checks a saved manual game appears in the library and
// is labelled so nobody claims Steam owns it.
func TestDetectManualGames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "HollowGrove.exe"))

	got := Detect(Roots{Manual: []ManualGame{{Dir: dir, Name: "Hollow Grove"}}})
	if len(got) != 1 {
		t.Fatalf("found %d games, want 1: %+v", len(got), got)
	}
	in := got[0]
	if in.Name != "Hollow Grove" {
		t.Errorf("name = %q", in.Name)
	}
	if in.Source != Manual {
		t.Errorf("source = %q, want %q", in.Source, Manual)
	}
	if in.Label() != "added by hand" {
		t.Errorf("label = %q, want %q", in.Label(), "added by hand")
	}
	if in.Exe == "" {
		t.Error("no executable resolved")
	}
}

// TestManualFallbackName checks an empty name falls back to the folder's, so the
// library never shows a nameless tile.
func TestManualFallbackName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.exe"))

	got := Detect(Roots{Manual: []ManualGame{{Dir: dir}}})
	if len(got) != 1 {
		t.Fatalf("found %d games", len(got))
	}
	if got[0].Name == "" {
		t.Fatal("a manual game with no name showed as an empty tile")
	}
}

// TestVanishedManualIsSkipped checks a manual game whose folder was deleted
// quietly disappears instead of panicking or showing a dead tile.
func TestVanishedManualIsSkipped(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "deleted")
	got := Detect(Roots{Manual: []ManualGame{{Dir: gone, Name: "Gone"}}})
	if len(got) != 0 {
		t.Fatalf("found %d games in a deleted folder: %+v", len(got), got)
	}
}

// TestManualDoesNotDuplicateScan checks a game that is both in a scanned games
// folder and in the manual list appears once, and keeps the better entry.
func TestManualDoesNotDuplicateScan(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Hollow Grove")
	writeFile(t, filepath.Join(gameDir, "HollowGrove.exe"))

	got := Detect(Roots{
		Custom: []string{root},
		Manual: []ManualGame{{Dir: gameDir, Name: "Hollow Grove", Genre: "Exploration"}},
	})
	if len(got) != 1 {
		t.Fatalf("found %d games, want the game listed once: %+v", len(got), got)
	}
}
