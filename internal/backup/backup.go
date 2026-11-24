// Package backup reads and writes one portable JSON file with the data Nova
// keeps for its user: play time, library entries, favorites and local unlocks.
// Cached store pages and artwork are left out because Steam can rebuild them.
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yafit/nova/internal/gse"
	"github.com/yafit/nova/internal/library"
)

// currentVersion is the shape of the file this build understands.
const currentVersion = 1

// Game is one game's stored history.
type Game struct {
	Played   time.Duration `json:"played"`
	Sessions int           `json:"sessions"`
	Last     time.Time     `json:"last"`
}

// Bundle is the whole export. Maps are always written, even when empty, so an
// import can tell an empty section from one a newer build left out.
type Bundle struct {
	Version      int                     `json:"version"`
	ExportedAt   time.Time               `json:"exportedAt"`
	Library      library.Config          `json:"library"`
	Favorites    map[string]bool         `json:"favorites"`
	Games        map[string]Game         `json:"games"`
	Achievements map[string][]gse.Record `json:"achievements"`
}

// Write saves a bundle through a temporary name, so a crash cannot leave a
// half-written file that looks like a real backup.
func Write(path string, b Bundle) error {
	b.Version = currentVersion
	if b.ExportedAt.IsZero() {
		b.ExportedAt = time.Now()
	}
	b.fill()

	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", raw, 0o644); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// Read loads a backup. A damaged file or an unknown version is an error: it
// must never be applied as an empty one.
func Read(path string) (Bundle, error) {
	var b Bundle
	raw, err := os.ReadFile(path)
	if err != nil {
		return b, fmt.Errorf("read backup: %w", err)
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, fmt.Errorf("read backup: %w", err)
	}
	if b.Version != currentVersion {
		return b, fmt.Errorf("unsupported backup version %d", b.Version)
	}
	b.fill()
	return b, nil
}

// fill makes every map safe to read and write.
func (b *Bundle) fill() {
	if b.Favorites == nil {
		b.Favorites = map[string]bool{}
	}
	if b.Games == nil {
		b.Games = map[string]Game{}
	}
	if b.Achievements == nil {
		b.Achievements = map[string][]gse.Record{}
	}
}

// Merge folds an older backup into the current data. The greater value wins,
// so importing an old file cannot lose new play time or un-earn an unlock.
func Merge(fresh, old Bundle) Bundle {
	out := fresh
	out.fill()
	old.fill()

	out.Games = mergeGames(out.Games, old.Games)
	out.Favorites = mergeFavorites(out.Favorites, old.Favorites)
	out.Library = mergeLibrary(out.Library, old.Library)
	out.Achievements = mergeAchievements(out.Achievements, old.Achievements)
	return out
}

func mergeGames(fresh, old map[string]Game) map[string]Game {
	out := map[string]Game{}
	for key, g := range fresh {
		out[key] = g
	}
	for key, g := range old {
		cur, seen := out[key]
		if !seen || g.Played > cur.Played {
			cur.Played = g.Played
		}
		if g.Sessions > cur.Sessions {
			cur.Sessions = g.Sessions
		}
		if g.Last.After(cur.Last) {
			cur.Last = g.Last
		}
		out[key] = cur
	}
	return out
}

func mergeFavorites(fresh, old map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, on := range fresh {
		out[key] = on
	}
	for key, on := range old {
		if on {
			out[key] = true
		}
	}
	return out
}

func mergeAchievements(fresh, old map[string][]gse.Record) map[string][]gse.Record {
	out := map[string][]gse.Record{}
	for appID, records := range fresh {
		out[appID] = records
	}
	for appID, records := range old {
		byName := map[string]gse.Record{}
		for _, r := range out[appID] {
			byName[r.APIName] = r
		}
		for _, r := range records {
			cur, seen := byName[r.APIName]
			if !seen || (r.Achieved && (!cur.Achieved || r.Unlocked.After(cur.Unlocked))) {
				byName[r.APIName] = r
			}
		}
		merged := make([]gse.Record, 0, len(byName))
		for _, r := range byName {
			merged = append(merged, r)
		}
		out[appID] = merged
	}
	return out
}
