package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestPlayingOverlayHasOneAction checks that the running-session window offers
// only End session. Closing the window is done by Escape or a click outside,
// and the session ends on its own when the game exits.
func TestPlayingOverlayHasOneAction(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	a.playing = &playingSession{slug: e.slug(), name: e.name(), start: time.Now()}
	a.overlay = overlayPlaying

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	if !tt.HasText("End session") {
		t.Fatalf("the session action is missing; texts = %q", tt.Texts())
	}
	for _, gone := range []string{"Quit game", "Close"} {
		if tt.HasText(gone) {
			t.Fatalf("the session window still shows %q", gone)
		}
	}
}
