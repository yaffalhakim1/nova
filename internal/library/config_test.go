package library

import (
	"os"
	"path/filepath"
	"testing"
)

// TestManualGamesSurviveAReload checks the manual list is written and read back
// in full, which is what makes an added game stay added.
func TestManualGamesSurviveAReload(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		Steam:   []string{},
		Folders: []string{},
		Manual: []ManualGame{{
			Dir:         `Y:\games\Hollow Grove`,
			Exe:         `bin\HollowGrove.exe`,
			Name:        "Hollow Grove",
			Genre:       "Exploration",
			Description: "A quiet forest.",
			Hue:         150,
		}},
	}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	got := LoadConfig(dir)
	if len(got.Manual) != 1 {
		t.Fatalf("manual = %d entries, want 1", len(got.Manual))
	}
	if got.Manual[0] != cfg.Manual[0] {
		t.Fatalf("manual = %+v, want %+v", got.Manual[0], cfg.Manual[0])
	}
}

// TestRootsCarriesManual checks Detect receives the manual list through the one
// struct it already takes, so the scan stays a pure function of its input.
func TestRootsCarriesManual(t *testing.T) {
	cfg := Config{Manual: []ManualGame{{Dir: `Y:\games\X`, Name: "X"}}}
	if len(cfg.Roots().Manual) != 1 {
		t.Fatal("Roots did not carry the manual games")
	}
}

// TestManualExeIsRelativeToDir is the invariant that keeps a manual game working
// after the folder is moved: never an absolute path.
func TestManualExeIsRelativeToDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Nested", "Game.exe"))

	m, ok := AddManual(root)
	if !ok {
		t.Fatal("the folder was not accepted")
	}
	if filepath.IsAbs(m.Exe) {
		t.Fatalf("exe = %q, want relative", m.Exe)
	}

	if _, err := os.Stat(filepath.Join(m.Dir, m.Exe)); err != nil {
		t.Fatalf("exe does not resolve under the game folder: %v", err)
	}
}
