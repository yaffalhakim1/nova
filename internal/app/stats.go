package app

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"time"

	steamstore "github.com/yafit/nova/internal/steam"
)

// statsName is the file that holds how long each game has been played.
const statsName = "stats.json"

// gameStats is one game's play history.
type gameStats struct {
	// Played is the total time the game ran, from launches through this app.
	Played time.Duration `json:"played"`
	// Sessions counts how many times it was started.
	Sessions int `json:"sessions"`
	// Last is when it was last started: the one timestamp every "last played"
	// line and the recent row read.
	Last time.Time `json:"last"`
}

// stats is the play history of the whole library, keyed by slug.
type stats struct {
	Games map[string]*gameStats `json:"games"`
}

// loadStats reads the play history, or starts an empty one. A missing or
// damaged file is an empty history, never an error: the launcher still starts.
func loadStats(dir string) stats {
	s := stats{Games: map[string]*gameStats{}}

	raw, err := os.ReadFile(filepath.Join(dir, statsName))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(raw, &s); err != nil || s.Games == nil {
		return stats{Games: map[string]*gameStats{}}
	}
	return s
}

// save writes the play history.
func (s stats) save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, statsName), raw, 0o644)
}

// get returns a game's history, creating it when it has none.
//
// A nil Games map is initialised rather than bypassed: returning a throwaway
// would silently swallow every write, which is worse than a missing entry.
func (s *stats) get(slug string) *gameStats {
	if s.Games == nil {
		s.Games = map[string]*gameStats{}
	}
	if g, ok := s.Games[slug]; ok && g != nil {
		return g
	}
	g := &gameStats{}
	s.Games[slug] = g
	return g
}

// mergeSteam folds Steam's own record of each game into the play history,
// keeping the larger of the two numbers.
//
// Max rather than sum on purpose: Steam and this app are both counting the same
// sessions, so adding them would double every hour. Max is also idempotent, so
// importing again on every start changes nothing. Sessions stay nova's own,
// because Steam does not count them.
func (a *app) mergeSteam(steam map[string]steamstore.Entry) {
	if len(steam) == 0 {
		return
	}

	for _, e := range a.allEntries() {
		id := e.inst.AppID
		if id == "" {
			continue
		}
		entry, ok := steam[id]
		if !ok {
			continue
		}

		h := a.stat.get(e.slug())
		if entry.Played > h.Played {
			h.Played = entry.Played
		}
		if entry.Last.After(h.Last) {
			h.Last = entry.Last
		}
	}
}

// importSteam reads the play time of the Steam account used last and folds it
// in. A machine without Steam, or with an unreadable file, imports nothing and
// the launcher carries on.
func (a *app) importSteam() {
	root := steamstore.InstallRoot()
	if root == "" {
		return
	}
	account := steamstore.DefaultAccount(root)
	if account == nil {
		return
	}
	a.mergeSteam(steamstore.Playtime(root, account.ID))
}

// record adds one finished session to a game.
func (s *stats) record(slug string, ran time.Duration) {
	g := s.get(slug)
	g.Played += ran
	g.Sessions++
	g.Last = time.Now()
}

// played formats a duration as the design shows it: "3h 12m", "42m".
func played(d time.Duration) string {
	if d <= 0 {
		return "0h"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return itoaPad(m) + "m"
	}
	return itoaPad(h) + "h " + itoaPad(m) + "m"
}

// lastPlayed says when a game was last played, in words. It is the launcher's
// only such function: the home meta line, the recent cards and the detail page
// all call it, so they cannot disagree about a game.
//
// The thresholds round rather than truncate, as the design's `ago()` does: a
// hour and a half is "2 hr ago", not "1 hr ago" next to a card that says
// something else.
func lastPlayed(t time.Time) string {
	if t.IsZero() {
		return "Not played yet"
	}

	// A clock that ran backwards leaves a time in the future; reading that as
	// "Just now" is truer than a negative age.
	m := int(math.Round(time.Since(t).Minutes()))
	if m < 1 {
		return "Just now"
	}
	if m < 60 {
		return itoa(m) + " min ago"
	}

	h := int(math.Round(float64(m) / 60))
	if h < 24 {
		return itoa(h) + " hr ago"
	}

	d := int(math.Round(float64(h) / 24))
	if d == 1 {
		return "Yesterday"
	}
	return itoa(d) + " days ago"
}

// playedLine is the meta line's play-time entry, as the design shows it:
// "41h 30m played", or "Not played yet" for a game with no history.
func (a *app) playedLine(e entry) string {
	total := a.playedOf(e)
	if a.playing != nil && a.playing.slug == e.slug() {
		total += a.sessionElapsed()
	}
	if total <= 0 {
		return "Not played yet"
	}
	return played(total) + " played"
}

// itoaPad formats a number, padding a single digit with a leading zero.
func itoaPad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// itoa formats a non-negative number.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
