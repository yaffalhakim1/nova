package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yafit/nova/internal/gse"
	"github.com/yafit/nova/internal/library"
)

// fixture returns a bundle with every kind of data the file carries.
func fixture() Bundle {
	return Bundle{
		Version:    1,
		ExportedAt: time.Unix(1790000000, 0),
		Library:    library.Config{Folders: []string{`Y:\games`}},
		Favorites:  map[string]bool{"steam-730": true},
		Games: map[string]Game{
			"steam-730": {Played: 90 * time.Minute, Sessions: 3, Last: time.Unix(1790000000, 0)},
		},
		Achievements: map[string][]gse.Record{
			"4115450": {{APIName: "A_FIRST", Achieved: true, Unlocked: time.Unix(1781932638, 0)}},
		},
	}
}

// TestReadRejectsBadFiles pins the failures that must never be mistaken for
// an empty, valid backup.
func TestReadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		body string
	}{
		{"bad json", "{"},
		{"wrong version", `{"version":99}`},
		{"no version", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".json")
			if err := os.WriteFile(path, []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Read(path); err == nil {
				t.Fatal("Read accepted a bad backup")
			}
		})
	}
	if _, err := Read(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("Read accepted a missing file")
	}
}

// TestWriteReadRoundTrip checks that a file keeps every field, including the
// local GSE unlock times that Nova cannot rebuild from Steam.
func TestWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nova-backup.json")
	want := fixture()
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Games["steam-730"].Played != 90*time.Minute {
		t.Errorf("play time = %v", got.Games["steam-730"].Played)
	}
	if !got.Favorites["steam-730"] {
		t.Error("favorite was lost")
	}
	rec := got.Achievements["4115450"]
	if len(rec) != 1 || !rec[0].Achieved || rec[0].Unlocked != time.Unix(1781932638, 0) {
		t.Errorf("achievements = %+v", rec)
	}
	if len(got.Library.Folders) != 1 || got.Library.Folders[0] != `Y:\games` {
		t.Errorf("library = %+v", got.Library.Folders)
	}
}

// TestMergeKeepsTheBetterValue checks that importing an old backup never
// lowers play time, drops a manually added game or unearns an achievement.
func TestMergeKeepsTheBetterValue(t *testing.T) {
	old := fixture()
	fresh := Bundle{
		Version:   1,
		Library:   library.Config{Folders: []string{`Z:\games`}},
		Favorites: map[string]bool{"steam-620": true},
		Games: map[string]Game{
			"steam-730":     {Played: 30 * time.Minute, Sessions: 1},
			"steam-1091500": {Played: 10 * time.Minute, Sessions: 1},
		},
		Achievements: map[string][]gse.Record{
			"4115450": {{APIName: "A_FIRST", Achieved: false}},
		},
	}

	got := Merge(fresh, old)

	if g := got.Games["steam-730"]; g.Played != 90*time.Minute || g.Sessions != 3 {
		t.Errorf("merged game = %+v", g)
	}
	if got.Games["steam-1091500"].Played != 10*time.Minute {
		t.Error("a new game was dropped")
	}
	if !got.Favorites["steam-730"] || !got.Favorites["steam-620"] {
		t.Errorf("favorites = %+v", got.Favorites)
	}
	if len(got.Library.Folders) != 2 {
		t.Errorf("folders = %+v", got.Library.Folders)
	}
	rec := got.Achievements["4115450"]
	if len(rec) != 1 || !rec[0].Achieved {
		t.Errorf("an achievement was unearned by the import: %+v", rec)
	}
}
