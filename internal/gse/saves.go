// Package gse reads the save files written by the GSE Steam emulator. It only
// reads local files; it never writes to a game or attaches to its process.
package gse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// appIDFile is the small text file an emulated game keeps beside its exe.
const appIDFile = "steam_appid.txt"

// unlocksFile is where GSE records a game's achievements.
const unlocksFile = "achievements.json"

// Record is one achievement as the emulator stores it.
type Record struct {
	APIName  string    `json:"apiName"`
	Achieved bool      `json:"earned"`
	Unlocked time.Time `json:"unlocked,omitempty"`
}

// unlock is one entry of the emulator's JSON file.
type unlock struct {
	Earned     bool  `json:"earned"`
	EarnedTime int64 `json:"earned_time"`
}

// Root returns the folder where GSE keeps per-game saves. A machine without a
// user config folder has no root, and every read then reports no data.
func Root() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return ""
	}
	return filepath.Join(base, "GSE Saves")
}

// AppID reads the app id a repack ships with its executable. Goldberg puts it
// beside the exe or in steam_settings, and a library entry may point at a
// parent folder, so every likely folder is checked.
func AppID(dirs ...string) string {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		for _, path := range []string{
			filepath.Join(dir, appIDFile),
			filepath.Join(dir, "steam_settings", appIDFile),
		} {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			id := strings.TrimSpace(string(raw))
			if validID(id) {
				return id
			}
		}
	}
	return ""
}

// Read loads one game's unlocks. A missing, empty or damaged file is no data,
// not a game with zero achievements.
func Read(root, appID string) ([]Record, bool) {
	if root == "" || !validID(appID) {
		return nil, false
	}

	raw, err := os.ReadFile(filepath.Join(root, appID, unlocksFile))
	if err != nil {
		return nil, false
	}
	var stored map[string]unlock
	if err := json.Unmarshal(raw, &stored); err != nil || len(stored) == 0 {
		return nil, false
	}

	names := make([]string, 0, len(stored))
	for name := range stored {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Record, 0, len(names))
	for _, name := range names {
		u := stored[name]
		rec := Record{APIName: name, Achieved: u.Earned}
		if u.EarnedTime > 0 {
			rec.Unlocked = time.Unix(u.EarnedTime, 0)
		}
		out = append(out, rec)
	}
	return out, true
}

// All reads every game's unlocks under a saves root. A folder that is not a
// decimal app id, or that holds no readable file, is skipped.
func All(root string) map[string][]Record {
	out := map[string][]Record{}
	if root == "" {
		return out
	}
	dirs, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	for _, d := range dirs {
		if !d.IsDir() || !validID(d.Name()) {
			continue
		}
		if records, ok := Read(root, d.Name()); ok {
			out[d.Name()] = records
		}
	}
	return out
}

// Save writes one game's unlocks whole, so a crash cannot leave half a file.
func Save(root, appID string, records []Record) error {
	if root == "" || !validID(appID) {
		return nil
	}
	dir := filepath.Join(root, appID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// The emulator writes a map keyed by API name, not an array.
	stored := make(map[string]unlock, len(records))
	for _, r := range records {
		entry := unlock{Earned: r.Achieved}
		if !r.Unlocked.IsZero() {
			entry.EarnedTime = r.Unlocked.Unix()
		}
		stored[r.APIName] = entry
	}
	return writeJSON(filepath.Join(dir, unlocksFile), stored)
}

// Restore merges imported unlocks with what is already on disk. An unlock is
// permanent, so an older backup cannot turn it off.
func Restore(root string, imported map[string][]Record) error {
	for appID, records := range imported {
		if !validID(appID) {
			continue
		}
		merged := map[string]Record{}
		if current, ok := Read(root, appID); ok {
			for _, r := range current {
				merged[r.APIName] = r
			}
		}
		for _, r := range records {
			old, seen := merged[r.APIName]
			if !seen || (r.Achieved && (!old.Achieved || r.Unlocked.After(old.Unlocked))) {
				merged[r.APIName] = r
			}
		}
		out := make([]Record, 0, len(merged))
		for _, name := range sortedKeys(merged) {
			out = append(out, merged[name])
		}
		if err := Save(root, appID, out); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON writes a whole file through a temporary name in the same folder.
func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", raw, 0o644); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// sortedKeys returns a map's keys in a stable order.
func sortedKeys(records map[string]Record) []string {
	names := make([]string, 0, len(records))
	for name := range records {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// validID accepts only a decimal app id, so a value from a game folder can
// never climb out of the saves root.
func validID(id string) bool {
	if id == "" || id == "0" {
		return false
	}
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return false
	}
	return true
}
