package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
)

// launcher builds an app over a set of installs, with its own cache and no
// network.
func launcher(t *testing.T, installs []library.Install) *app {
	t.Helper()

	dir := t.TempDir()
	a := &app{
		theme:     theme(),
		dataDir:   dir,
		installs:  installs,
		favorites: map[string]bool{},
		expanded:  map[string]bool{},
		appIDs:    map[string]string{},
		hidden:    map[string]bool{},
		lastKey:   -1,
		epoch:     time.Unix(0, 0),
	}
	a.covers = newCovers(dir)
	a.shots = newShots(dir)
	a.refresh()
	return a
}

// installsOf builds n games whose folders do not exist, which is enough for
// the views: nothing launches during a test.
func installsOf(n int) []library.Install {
	out := make([]library.Install, 0, n)
	for i := 0; i < n; i++ {
		name := "Game " + fmt.Sprint(i)
		out = append(out, library.Install{
			Name:   name,
			Dir:    `Y:\games\` + name,
			AppID:  fmt.Sprint(1000 + i),
			Source: library.Steam,
		})
	}
	return out
}

// TestHomeShowsInstalledGames checks the home screen paints only what is on
// disk, and none of the mod furniture.
func TestHomeShowsInstalledGames(t *testing.T) {
	a := launcher(t, installsOf(6))

	tester := ui.NewTester(a.view, 1280, 840)
	tester.Frame()

	for _, want := range []string{"Home", "Favorites", "Library", "Game 0", "Jump back in", "Your library", "Play"} {
		if !tester.HasText(want) {
			t.Fatalf("home does not show %q; texts = %q", want, tester.Texts())
		}
	}

	// The whole point of this build: no mod or trainer wording anywhere.
	for _, banned := range []string{"Mods", "mods", "cheat", "Trainer", "trainer"} {
		if tester.HasText(banned) {
			t.Fatalf("home still mentions %q", banned)
		}
	}
}

// TestDetailHasNoModsPanel checks the detail page is a launcher page: a cover,
// a title, Play, and facts about play time.
func TestDetailHasNoModsPanel(t *testing.T) {
	a := launcher(t, installsOf(3))
	a.open(a.entries[0])

	tester := ui.NewTester(a.view, 1280, 840)
	tester.Frame()

	for _, want := range []string{"Back to games", "Game 0", "About", "Play", "Time played", "Sessions"} {
		if !tester.HasText(want) {
			t.Fatalf("detail does not show %q; texts = %q", want, tester.Texts())
		}
	}
	for _, banned := range []string{"Mods", "Turn all off", "cheat"} {
		if tester.HasText(banned) {
			t.Fatalf("detail still mentions %q", banned)
		}
	}
}

// TestFilters checks the three tabs narrow the library.
func TestFilters(t *testing.T) {
	a := launcher(t, installsOf(5))

	a.favorites[entry{inst: a.installs[3]}.slug()] = true
	a.section = sectionFavorites
	a.lastKey = -1
	a.refresh()

	if len(a.entries) != 1 || a.entries[0].inst.AppID != a.installs[3].AppID {
		t.Fatalf("favorites = %d entries, want just one", len(a.entries))
	}
}

// TestArrowsChooseAndEnterPlays checks the keyboard the design lists: the
// arrows browse, Down opens the game's page and Escape goes back.
func TestArrowsChooseAndEnterPlays(t *testing.T) {
	a := launcher(t, installsOf(10))

	tester := ui.NewTester(a.view, 1280, 840)
	tester.Frame()

	tester.Key(0, ui.KeyRight)
	tester.Key(0, ui.KeyRight)
	tester.Frame()
	if a.selected != 2 {
		t.Fatalf("selected = %d, want 2", a.selected)
	}

	// Down is the design's way into the game's own page.
	tester.Key(0, ui.KeyDown)
	tester.Frame()
	if a.current == nil || a.current.name() != "Game 2" {
		t.Fatalf("Down opened %v, want Game 2", a.current)
	}

	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.current != nil {
		t.Fatal("Escape did not go back")
	}
}

// TestForgetHidesTheGame checks the trash button takes a game out of the
// library without deleting it from disk.
func TestForgetHidesTheGame(t *testing.T) {
	a := launcher(t, installsOf(3))
	first := a.entries[0]

	a.forget(first)

	if len(a.entries) != 2 {
		t.Fatalf("library = %d entries after forget, want 2", len(a.entries))
	}
	for _, e := range a.entries {
		if e.slug() == first.slug() {
			t.Fatal("the forgotten game is still listed")
		}
	}

	// The exclusion is remembered, and the config file carries it, so a later
	// scan agrees.
	if !a.isHidden(first.slug()) {
		t.Fatal("the forgotten game is not hidden")
	}
	if a.libCfg.Hidden == nil || len(a.libCfg.Hidden) == 0 {
		t.Fatal("the exclusion was not kept in the config")
	}
}

// TestPlayRecorsdASession checks a finished session adds to the play history.
func TestPlayRecordsASession(t *testing.T) {
	a := launcher(t, installsOf(2))
	e := a.entries[0]

	// A launcher records a session when the game is quit. A zero stats needs
	// its map before it can record: loadStats always provides one.
	a.stat = stats{Games: map[string]*gameStats{}}

	a.playing = &playingSession{slug: e.slug(), name: e.name(), start: time.Now().Add(-90 * time.Minute)}
	a.quitPlay()

	h := a.stat.get(e.slug())
	if h.Sessions != 1 {
		t.Fatalf("sessions = %d, want 1", h.Sessions)
	}
	if h.Played < 89*time.Minute {
		t.Fatalf("played = %v, want about 90m", h.Played)
	}
	if a.playing != nil {
		t.Fatal("quitPlay left the session open")
	}
}

// TestLaunchProblemNamesTheCause checks a game with no executable is refused
// with a reason rather than a silent no-op.
func TestLaunchProblemNamesTheCause(t *testing.T) {
	missing := entry{inst: library.Install{Name: "Gone", Dir: `Y:\nope`, Exe: `Y:\nope\gone.exe`}}
	if canLaunch(missing) {
		t.Fatal("a game with no executable should not be launchable")
	}
	if problem := launchProblem(missing); problem == "" {
		t.Fatal("a missing executable should be explained")
	}
}
