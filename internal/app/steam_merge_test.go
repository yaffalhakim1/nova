package app

import (
	"testing"
	"time"

	"github.com/yafit/nova/internal/library"
	steamstore "github.com/yafit/nova/internal/steam"
)

// TestMergeSteamTakesTheLarger checks the chosen merge policy: of the two
// records of the same game, the larger time wins, so nothing played is lost.
func TestMergeSteamTakesTheLarger(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.stat = stats{Games: map[string]*gameStats{}}
	slug := a.entries[0].slug()

	local := time.Now().Add(-48 * time.Hour)
	a.stat.get(slug).Played = 41 * time.Hour
	a.stat.get(slug).Sessions = 38
	a.stat.get(slug).Last = local

	a.mergeSteam(map[string]steamstore.Entry{
		"1000": {Played: 100 * time.Hour, Last: time.Now().Add(-time.Hour)},
	})

	h := a.stat.get(slug)
	if h.Played != 100*time.Hour {
		t.Fatalf("played = %v, want Steam's larger 100h", h.Played)
	}
	if h.Sessions != 38 {
		t.Fatalf("sessions = %d: Steam has no session count, nova's must stay", h.Sessions)
	}
	if !h.Last.After(local) {
		t.Fatalf("last = %v, want Steam's more recent one", h.Last)
	}
}

// TestMergeSteamKeepsTheLargerLocal checks the other direction: nova's own time
// is not thrown away when Steam has less.
func TestMergeSteamKeepsTheLargerLocal(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.stat = stats{Games: map[string]*gameStats{}}
	slug := a.entries[0].slug()

	recent := time.Now()
	a.stat.get(slug).Played = 100 * time.Hour
	a.stat.get(slug).Last = recent

	a.mergeSteam(map[string]steamstore.Entry{
		"1000": {Played: 41 * time.Hour, Last: recent.Add(-time.Hour)},
	})

	h := a.stat.get(slug)
	if h.Played != 100*time.Hour {
		t.Fatalf("played = %v, want nova's larger 100h", h.Played)
	}
	if !h.Last.Equal(recent) {
		t.Fatalf("last = %v, want nova's more recent one", h.Last)
	}
}

// TestMergeSteamIsIdempotent is the property that makes importing on every start
// safe: merging the same numbers twice changes nothing.
func TestMergeSteamIsIdempotent(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.stat = stats{Games: map[string]*gameStats{}}
	slug := a.entries[0].slug()

	steam := map[string]steamstore.Entry{
		"1000": {Played: 100 * time.Hour, Last: time.Now().Add(-2 * time.Hour)},
	}

	a.mergeSteam(steam)
	once := *a.stat.get(slug)
	a.mergeSteam(steam)
	twice := *a.stat.get(slug)

	if once.Played != twice.Played || !once.Last.Equal(twice.Last) {
		t.Fatalf("merging twice changed the history: %v then %v", once, twice)
	}
}

// TestMergeSteamIgnoresGamesItCannotName checks a Steam app that is not in the
// library is skipped, and a game with no app id is never matched.
func TestMergeSteamIgnoresGamesItCannotName(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.stat = stats{Games: map[string]*gameStats{}}

	// A manual game has no app id, so nothing can be merged onto it.
	a.installs = append(a.installs, library.Install{
		Name:   "Repack",
		Dir:    `Y:\games\Repack`,
		Exe:    `Y:\games\Repack\game.exe`,
		Source: library.Manual,
	})
	a.refresh()

	a.mergeSteam(map[string]steamstore.Entry{
		"999999": {Played: 10 * time.Hour},
	})

	for _, e := range a.allEntries() {
		if got := a.stat.get(e.slug()).Played; got != 0 {
			t.Errorf("%q took a stranger's history: %v", e.name(), got)
		}
	}
}

// TestMergeSteamThenANewSessionAdds checks importing does not stop nova from
// recording its own sessions on top afterwards.
func TestMergeSteamThenANewSessionAdds(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.stat = stats{Games: map[string]*gameStats{}}
	e := a.entries[0]

	a.mergeSteam(map[string]steamstore.Entry{
		"1000": {Played: 10 * time.Hour},
	})

	a.playing = &playingSession{slug: e.slug(), name: e.name(), start: time.Now().Add(-30 * time.Minute)}
	a.quitPlay()

	if got := a.stat.get(e.slug()).Played; got < 10*time.Hour+29*time.Minute {
		t.Fatalf("played = %v, want the import plus the new session", got)
	}
}
