package app

import (
	"testing"

	"github.com/yafit/nova/internal/library"
	steamstore "github.com/yafit/nova/internal/steam"
)

// TestBackdropPrefersTheWideStoreArt checks the detail page's full-bleed
// artwork comes from Steam's `page_bg_raw`, which is 1438x810, rather than from
// the 2:3 portrait cover. The page is wider than it is tall, so the portrait
// would have to crop most of the game away.
func TestBackdropPrefersTheWideStoreArt(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]

	a.mu.Lock()
	a.storeData = map[string]*steamstore.Details{
		e.slug(): {Background: "https://example/page_bg_raw.jpg"},
	}
	a.mu.Unlock()

	urls := a.backdropURLs(e)
	if len(urls) == 0 {
		t.Fatal("no backdrop urls")
	}
	if urls[0] != "https://example/page_bg_raw.jpg" {
		t.Fatalf("first backdrop = %q, want the wide store art", urls[0])
	}
}

// TestBackdropFallsBackToTheCover checks a game whose store page has not
// answered still paints something rather than a bare ground.
func TestBackdropFallsBackToTheCover(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]

	urls := a.backdropURLs(e)
	if len(urls) == 0 {
		t.Fatal("no backdrop urls at all")
	}
	// The cover is the last resort, so it must be somewhere in the list.
	covers := a.coverURLs(e)
	found := false
	for _, u := range urls {
		if u == covers[0] {
			found = true
		}
	}
	if !found {
		t.Fatal("the cover is not a fallback for the backdrop")
	}
}

// TestCoverURLsUseTheManifestAppID checks a Steam game whose id came from its
// manifest has cover urls immediately, without waiting for the background
// lookup: reading only the resolved map left it on generated art until then.
func TestCoverURLsUseTheManifestAppID(t *testing.T) {
	a := launcher(t, []library.Install{{
		Name: "X", Dir: `Y:\g\X`, AppID: "1091500", Source: library.Steam,
	}})

	// The resolved map is deliberately empty, as it is before `resolve` answers.
	if urls := a.coverURLs(a.entries[0]); len(urls) == 0 {
		t.Fatal("no cover urls for a game whose app id is in its manifest")
	}
}

// TestCoverStaysPortrait checks the tile artwork still comes from the portrait
// files, never the wide store art: a 2:3 tile needs a 2:3 source.
func TestCoverStaysPortrait(t *testing.T) {
	a := launcher(t, []library.Install{{
		Name: "X", Dir: `Y:\g\X`, AppID: "1091500", Source: library.Steam,
	}})
	e := a.entries[0]

	for _, u := range a.coverURLs(e) {
		if u == "https://example/page_bg_raw.jpg" {
			t.Fatal("the wide store art leaked into the tile covers")
		}
	}
}
