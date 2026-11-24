package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSettingsHoldsTheMenuActions checks that every action from the removed
// menu bar is reachable in the panel behind the settings button.
func TestSettingsHoldsTheMenuActions(t *testing.T) {
	a := launcher(t, installsOf(2))
	a.overlay = overlaySettings

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	for _, want := range []string{
		"Settings",
		"Add games folder…",
		"Your Steam library…",
		"All games",
		"Favorites",
		"Library",
		"Export data…",
		"Import data…",
		"Quit",
	} {
		if !tt.HasText(want) {
			t.Fatalf("settings does not show %q; texts = %q", want, tt.Texts())
		}
	}

	// The actions sit in columns, not one tall strip.
	first, ok := tt.Find("Add games folder…")
	if !ok {
		t.Fatal("the first action has no rect")
	}
	last, ok := tt.Find("Quit")
	if !ok {
		t.Fatal("the last action has no rect")
	}
	if last.X-first.X < 300 {
		t.Fatalf("actions span only %.0f DIPs; the panel is still narrow", last.X-first.X)
	}
}
