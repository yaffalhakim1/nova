package gse

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadRejectsUnreadableFiles pins every way a save can fail to be data.
// An empty or damaged file must read as no data, never as a game with zero
// achievements, so the card keeps its honest empty state.
func TestReadRejectsUnreadableFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
		body string
		id   string
	}{
		{"missing file", "", "", "4115450"},
		{"bad json", "achievements.json", "{", "4115450"},
		{"wrong shape", "achievements.json", `["A"]`, "4115450"},
		{"wrong field type", "achievements.json", `{"A":{"earned":"yes","earned_time":1}}`, "4115450"},
		{"no records", "achievements.json", `{}`, "4115450"},
		{"empty id", "achievements.json", `{}`, ""},
		{"id with separator", "achievements.json", `{}`, `..\4115450`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.file != "" && validID(tt.id) {
				dir := filepath.Join(root, tt.id)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, tt.file), []byte(tt.body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got, ok := Read(root, tt.id); ok || len(got) != 0 {
				t.Fatalf("Read = %+v, %v; want no data", got, ok)
			}
		})
	}
}

// TestReadParsesUnlocks checks the fields the Trophies card uses: the API name,
// the earned flag and the unlock time. Keys are sorted so the card is stable.
func TestReadParsesUnlocks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "4115450")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
		"Z_Last": {"earned": false, "earned_time": 0},
		"A_First": {"earned": true, "earned_time": 1781932638},
		"M_Middle": {"earned": true, "earned_time": 0}
	}`
	if err := os.WriteFile(filepath.Join(dir, "achievements.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := Read(root, "4115450")
	if !ok {
		t.Fatal("Read reported no data")
	}
	if len(got) != 3 {
		t.Fatalf("records = %d, want 3", len(got))
	}
	if got[0].APIName != "A_First" || !got[0].Achieved || got[0].Unlocked != time.Unix(1781932638, 0) {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].APIName != "M_Middle" || !got[1].Achieved || !got[1].Unlocked.IsZero() {
		t.Errorf("middle = %+v", got[1])
	}
	if got[2].APIName != "Z_Last" || got[2].Achieved {
		t.Errorf("last = %+v", got[2])
	}
}

// TestSaveAllAndRestore pins the on-disk shape. The emulator reads a map keyed
// by API name, so a backup import must write that shape, not an array.
func TestSaveAllAndRestore(t *testing.T) {
	root := t.TempDir()
	when := time.Unix(1781932638, 0)
	original := []Record{{APIName: "A_FIRST", Achieved: true, Unlocked: when}}
	if err := Save(root, "4115450", original); err != nil {
		t.Fatal(err)
	}

	all := All(root)
	if len(all["4115450"]) != 1 || !all["4115450"][0].Achieved {
		t.Fatalf("All = %+v", all)
	}
	if got, ok := Read(root, "4115450"); !ok || len(got) != 1 || !got[0].Unlocked.Equal(when) {
		t.Fatalf("Read after Save = %+v, %v", got, ok)
	}

	// An older import cannot un-earn what the emulator already recorded.
	if err := Restore(root, map[string][]Record{
		"4115450": {{APIName: "A_FIRST", Achieved: false}},
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := Read(root, "4115450"); !got[0].Achieved {
		t.Fatal("an old backup un-earned an achievement")
	}
}

// TestAppID checks the one file that names the game for a repack, and the bad
// values a hand-edited folder must not turn into a path.
func TestAppID(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "steam_appid.txt")

	for _, bad := range []string{"", "abc", "0", "..\\other", "-1"} {
		if err := os.WriteFile(file, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := AppID(dir); got != "" {
			t.Errorf("%q: AppID = %q, want empty", bad, got)
		}
	}
	if err := os.WriteFile(file, []byte(" 4115450\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AppID(dir); got != "4115450" {
		t.Errorf("AppID = %q, want 4115450", got)
	}
	if got := AppID(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("missing folder: AppID = %q, want empty", got)
	}

	// A repack may keep the id in steam_settings, one level below the exe.
	nested := t.TempDir()
	if err := os.MkdirAll(filepath.Join(nested, "steam_settings"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "steam_settings", appIDFile), []byte("4115450"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AppID(filepath.Join(nested, "..", "parent"), nested); got != "4115450" {
		t.Errorf("AppID across folders = %q, want 4115450", got)
	}
}
