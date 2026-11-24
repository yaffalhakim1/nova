package app

import (
	"testing"
	"time"
)

// TestCheckpointSavesAnActiveSession checks that time is written while a game
// runs. A forced exit then loses only the last checkpoint, not the whole
// session, and quitting counts the session once.
func TestCheckpointSavesAnActiveSession(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	now := time.Now()

	a.playing = &playingSession{
		slug:      e.slug(),
		name:      e.name(),
		start:     now.Add(-25 * time.Second),
		lastCheck: now.Add(-20 * time.Second),
	}
	a.checkpointPlay()

	h := a.stat.get(e.slug())
	if h.Played < 15*time.Second {
		t.Fatalf("checkpoint saved %v, want about 20s", h.Played)
	}
	if h.Sessions != 0 {
		t.Fatalf("a checkpoint counted a session: %d", h.Sessions)
	}
	if a.playing.recorded < 15*time.Second {
		t.Fatalf("recorded = %v, want about 20s", a.playing.recorded)
	}

	// A fresh read is what a restart sees. The file was saved, not just held
	// in memory.
	reloaded := loadStats(a.dataDir)
	if got := reloaded.Games[e.slug()].Played; got < 15*time.Second {
		t.Fatalf("reloaded play time = %v, want about 20s", got)
	}

	// Quitting records only the time since the checkpoint, and one session.
	a.playing.start = now.Add(-30 * time.Second)
	a.quitPlay()
	h = a.stat.get(e.slug())
	if h.Sessions != 1 {
		t.Fatalf("sessions = %d, want 1", h.Sessions)
	}
	if h.Played < 25*time.Second {
		t.Fatalf("play time = %v, want at least 25s", h.Played)
	}
}

// TestCheckpointDoesNotRunTooOften pins the interval: two calls in one frame
// must not write twice.
func TestCheckpointDoesNotRunTooOften(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	now := time.Now()

	a.playing = &playingSession{
		slug:      e.slug(),
		start:     now.Add(-5 * time.Minute),
		lastCheck: now,
	}
	a.checkpointPlay()
	a.checkpointPlay()

	if got := a.stat.get(e.slug()).Played; got != 0 {
		t.Fatalf("play time = %v, want no write inside the interval", got)
	}
}
