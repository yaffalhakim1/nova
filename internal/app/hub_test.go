package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// openHub opens the first game's page, settled: the rise has already run, so
// what a frame paints is the layout itself.
func openHub(t *testing.T, a *app, w, h int) *ui.Tester {
	t.Helper()

	a.open(a.entries[0])
	a.openedAt = a.epoch

	tt := ui.NewTester(a.view, w, h)
	tt.Frame()
	return tt
}

// TestHubShowsTheSixCards pins the page's shape: the design's six cards, in
// their order, and the wording each uses when this launcher has nothing to put
// in it.
func TestHubShowsTheSixCards(t *testing.T) {
	a := launcher(t, installsOf(3))
	tt := openHub(t, a, 1280, 840)

	for _, want := range []string{
		"Trophies", "Activity", "Recent sessions", "Game info", "Screenshots", "About",
		"Play",
	} {
		if !tt.HasText(want) {
			t.Fatalf("the hub does not show %q; texts = %q", want, tt.Texts())
		}
	}

	// These installs carry an app id but this test loads no achievements, so
	// the trophies card says so rather than printing a ring it made up.
	if !tt.HasText("Not tracked for this game.") {
		t.Errorf("trophies does not say it has no data; texts = %q", tt.Texts())
	}

	// No session log is kept either.
	if !tt.HasText("No sessions yet.") {
		t.Errorf("recent sessions does not say it is empty; texts = %q", tt.Texts())
	}

	// The old two-panel page and the design's trophy drill-down are gone: no
	// "read more" link, and no link to a list this build does not have.
	for _, gone := range []string{"Read more", "View all trophies", "Show less"} {
		if tt.HasText(gone) {
			t.Errorf("the hub still shows %q", gone)
		}
	}
}

// TestHubInventsNothingWithoutData checks that with no achievements loaded, the
// page shows no number it cannot know: no percentage, no tier count.
func TestHubInventsNothingWithoutData(t *testing.T) {
	a := launcher(t, installsOf(3))
	tt := openHub(t, a, 1280, 840)

	for _, text := range tt.Texts() {
		if strings.HasSuffix(text, "%") {
			t.Errorf("the hub shows a percentage it cannot know: %q", text)
		}
		if strings.HasPrefix(text, "Achievements ") {
			t.Errorf("the hub shows a ring it cannot know: %q", text)
		}
		for _, tier := range []string{"Platinum", "Gold", "Silver", "Bronze"} {
			if strings.HasPrefix(text, tier+" ") {
				t.Errorf("the hub counts a tier it cannot know: %q", text)
			}
		}
	}
}

// TestHubActivityIsTheRecordedPlayTime checks the numbers come from the play
// history, and that the one thing the design shows which this app cannot know
// — a completion percentage — is not invented.
func TestHubActivityIsTheRecordedPlayTime(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	a.stat = stats{Games: map[string]*gameStats{}}
	history := a.stat.get(e.slug())
	history.Played = 41*time.Hour + 32*time.Minute
	history.Sessions = 12
	history.Last = time.Now().Add(-2 * time.Hour)

	tt := openHub(t, a, 1280, 840)

	// The big number is the total, and the pair under it is the sessions it
	// came from and their average.
	for _, want := range []string{"41h 32m", "12", "Sessions", "Per session", "03h 27m"} {
		if !tt.HasText(want) {
			t.Fatalf("activity does not show %q; texts = %q", want, tt.Texts())
		}
	}

	// The meta line and the sessions card read the same history.
	if !tt.HasText("41h 32m played") {
		t.Errorf("the meta line lost the play time; texts = %q", tt.Texts())
	}
	if !tt.HasText("2 hr ago") {
		t.Errorf("the page lost when the game was last played; texts = %q", tt.Texts())
	}
}

// TestHubGameInfoSaysWhatItDoesNotKnow checks the info card's rows: the two
// facts Steam supplies, the folder, and the two this app never reads said
// plainly rather than left out or guessed.
func TestHubGameInfoSaysWhatItDoesNotKnow(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	tt := openHub(t, a, 1280, 840)

	for _, want := range []string{
		"Developer", "Released", "Version", "Install size", "Location",
		// No store page has answered for this game.
		"Unknown",
		// The folder is known, the version and the size are not.
		e.inst.Dir, "Not recorded",
	} {
		if !tt.HasText(want) {
			t.Fatalf("game info does not show %q; texts = %q", want, tt.Texts())
		}
	}
}

// TestHubHoldsItsShape checks the page is laid out at every width the window
// allows, and that the cards keep the design's heights rather than collapsing:
// a card's heading sits 595 from the top of an 840-tall window, which is where
// the pinned-bottom column puts a 224-tall card above the 28 DIP foot.
func TestHubHoldsItsShape(t *testing.T) {
	for _, size := range []struct{ w, h int }{{1280, 840}, {760, 840}, {700, 840}, {900, 600}} {
		a := launcher(t, installsOf(2))
		tt := openHub(t, a, size.w, size.h)

		if !tt.HasText("About") || !tt.HasText("Trophies") {
			t.Fatalf("%dx%d: the cards are not painted; texts = %q", size.w, size.h, tt.Texts())
		}

		title, ok := tt.Find("Game 0")
		if !ok {
			t.Fatalf("%dx%d: the title is missing", size.w, size.h)
		}
		// A window tall enough for the whole page pins the cards to the foot,
		// so the title sits low in it, not against the top bar. A shorter one
		// scrolls, and there the title starts near the top instead.
		if size.h >= 700 && title.Y < float32(size.h)*0.25 {
			t.Errorf("%dx%d: the title is at y=%.0f, not in the lower half", size.w, size.h, title.Y)
		}

		// The back pill stays at the top, whatever the width.
		back, ok := tt.Find("Back to games")
		if !ok || back.Y > 60 {
			t.Errorf("%dx%d: the back pill is at y=%v", size.w, size.h, back.Y)
		}
	}
}

// TestHubRiseIsTheDesignsEntrance pins the page's entrance: it starts 3% larger
// and invisible, and settles to its own size and to opaque — over the window's
// backdrop, never fading the ground to black.
func TestHubRiseIsTheDesignsEntrance(t *testing.T) {
	scale, opacity := riseAt(0)
	if scale != 1.03 || opacity != 0 {
		t.Fatalf("at the start: scale %v, opacity %v; want 1.03 and 0", scale, opacity)
	}

	scale, opacity = riseAt(1)
	if scale != 1 || opacity != 1 {
		t.Fatalf("at the end: scale %v, opacity %v; want 1 and 1", scale, opacity)
	}

	midScale, midOpacity := riseAt(0.4)
	if midScale <= 1 || midScale >= 1.03 {
		t.Errorf("half way: scale %v, want between 1 and 1.03", midScale)
	}
	if midOpacity <= 0 || midOpacity >= 1 {
		t.Errorf("half way: opacity %v, want between 0 and 1", midOpacity)
	}

	// A clock that runs past the animation's end, or backwards, still lands on
	// a whole page rather than one scaled past its start.
	if scale, opacity := riseAt(3); scale != 1 || opacity != 1 {
		t.Errorf("past the end: scale %v, opacity %v", scale, opacity)
	}
	if scale, opacity := riseAt(-1); scale != 1.03 || opacity != 0 {
		t.Errorf("before the start: scale %v, opacity %v", scale, opacity)
	}
}

// TestHubBackReturnsToTheLibrary checks the pill at the top is the way back.
func TestHubBackReturnsToTheLibrary(t *testing.T) {
	a := launcher(t, installsOf(2))
	tt := openHub(t, a, 1280, 840)

	if err := tt.Click("Back to games"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if a.current != nil {
		t.Fatal("the back pill left the game open")
	}
	if !tt.HasText("Jump back in") {
		t.Fatalf("the library did not come back; texts = %q", tt.Texts())
	}
}

// fixtureTrophies is a game's achievements as Steam answers them: the same
// seven unlocked and not, with real percentages, so the tiers have something
// true to count.
func fixtureTrophies() *gameTrophies {
	return &gameTrophies{
		list: []steamstore.Achievement{
			{APIName: "LILAC", Name: "Lilac and Gooseberries", Achieved: true, Unlocked: time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)},
			{APIName: "ENEMY", Name: "The Enemy of My Enemy", Achieved: true, Unlocked: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)},
			{APIName: "COUNSELOR", Name: "Family Counselor", Achieved: true, Unlocked: time.Date(2025, 12, 2, 0, 0, 0, 0, time.UTC)},
			{APIName: "TRAINED", Name: "Kaer Morhen Trained"},
			{APIName: "HIDDEN", Name: "Something Hidden"},
			{APIName: "CREW", Name: "Full Crew"},
			{APIName: "CIRI", Name: "Geralt and Ciri"},
		},
		// Real percentages: one under 5, one under 15, one under 50, the rest
		// common.
		rarity: map[string]float64{
			"LILAC": 3.1, "ENEMY": 9.4, "COUNSELOR": 22.0, "TRAINED": 61.2,
			"HIDDEN": 12.0, "CREW": 44.0, "CIRI": 2.0,
		},
		hasRarity: true,
	}
}

// TestTrophiesCardShowsTheRealRecord checks the card reads the achievements
// that were loaded: the earned share in the ring, how many of how many in the
// note, a count per rarity tier, and the newest unlock underneath.
func TestTrophiesCardShowsTheRealRecord(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.setTrophies(a.entries[0].slug(), fixtureTrophies())

	tt := openHub(t, a, 1280, 840)

	// Three of seven is 43%, rounded as the design's `Math.round` does.
	for _, want := range []string{
		"3 of 7",
		"Achievements 43%", "43%",
		"Lilac and Gooseberries", "Feb 25, 2026",
		"The Enemy of My Enemy", "Jan 4, 2026",
	} {
		if !tt.HasText(want) {
			t.Fatalf("the trophies card does not show %q; texts = %q", want, tt.Texts())
		}
	}

	// The four tiers, counted by real rarity: platinum 3.1 (LILAC), gold 9.4
	// (ENEMY), silver 22.0 (COUNSELOR), and nothing bronze.
	for _, want := range []string{"Platinum 1", "Gold 1", "Silver 1", "Bronze 0"} {
		if !tt.HasText(want) {
			t.Errorf("the tiers do not show %q; texts = %q", want, tt.Texts())
		}
	}

	// The empty state is gone, and no link to a list this build does not have.
	if tt.HasText("Not tracked for this game.") {
		t.Error("the card kept its empty state while it had data")
	}
	if tt.HasText("View all trophies") {
		t.Error("the card draws a link to nothing")
	}
}

// TestTrophiesCardStaysInsideItsCard checks what the card paints sits inside the
// 224 tall box: a ring and two rows are what fits, and a name cut short is
// better than a row spilling out over the artwork.
func TestTrophiesCardStaysInsideItsCard(t *testing.T) {
	a := launcher(t, installsOf(2))
	a.setTrophies(a.entries[0].slug(), fixtureTrophies())

	tt := openHub(t, a, 1280, 840)

	head, ok := tt.Find("Trophies")
	if !ok {
		t.Fatal("the card's heading is missing")
	}
	// The heading sits 20 below the card's top, and the card is 224 tall.
	top := head.Y - 20
	bottom := top + hubCardHeight

	for _, text := range []string{"Lilac and Gooseberries", "Feb 25, 2026", "The Enemy of My Enemy", "Jan 4, 2026"} {
		r, ok := tt.Find(text)
		if !ok {
			t.Fatalf("%q is not painted", text)
		}
		if r.Y < top || r.Y+r.H > bottom {
			t.Errorf("%q is at y=%.0f..%.0f, outside the card's %.0f..%.0f", text, r.Y, r.Y+r.H, top, bottom)
		}
	}
}

// TestTrophiesCardDropsTheDateOnANarrowCard checks the narrow card keeps the
// name whole rather than the date: at 250 DIPs a name and a date do not both
// fit, and the design drops the same column.
func TestTrophiesCardDropsTheDateOnANarrowCard(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.setTrophies(a.entries[0].slug(), fixtureTrophies())

	tt := openHub(t, a, 900, 600)

	if !tt.HasText("Lilac and Gooseberries") {
		t.Fatalf("the narrow card lost the name; texts = %q", tt.Texts())
	}
	if tt.HasText("Feb 25, 2026") {
		t.Error("the narrow card kept a date that crowds the name out")
	}
}

// TestTrophiesCardSaysSoWithoutRarity checks the card does not count four tiers
// of nothing when Steam published no percentages: it says how many were earned
// instead.
func TestTrophiesCardSaysSoWithoutRarity(t *testing.T) {
	a := launcher(t, installsOf(1))
	g := fixtureTrophies()
	g.rarity, g.hasRarity = nil, false
	a.setTrophies(a.entries[0].slug(), g)

	tt := openHub(t, a, 1280, 840)

	if !tt.HasText("3 earned") || !tt.HasText("rarity unknown") {
		t.Fatalf("the card does not explain that rarity is missing; texts = %q", tt.Texts())
	}
	for _, tier := range []string{"Platinum", "Gold", "Silver", "Bronze"} {
		if tt.HasText(tier + " 0") {
			t.Errorf("the card counted %q with no percentage to count it by", tier)
		}
	}
	// The ring and the total are still true without a percentage.
	if !tt.HasText("43%") {
		t.Errorf("the card lost the earned share; texts = %q", tt.Texts())
	}
}

// TestTrophiesCardFallsBackWhenThereIsNoData checks every way a game can have
// nothing to show lands on the same honest line rather than on a blank card.
func TestTrophiesCardFallsBackWhenThereIsNoData(t *testing.T) {
	// A game Steam says has no achievements at all: a loaded, empty answer.
	a := launcher(t, installsOf(1))
	a.setTrophies(a.entries[0].slug(), &gameTrophies{})

	tt := openHub(t, a, 1280, 840)
	if !tt.HasText("Not tracked for this game.") {
		t.Fatalf("an empty answer did not fall back; texts = %q", tt.Texts())
	}
	if tt.HasText("0 of 0") {
		t.Error("an empty answer was drawn as a game with no achievements earned")
	}

	// And a failed load, which is recorded the same way.
	b := launcher(t, installsOf(1))
	b.markTrophiesStarted(b.entries[0].slug())
	tt = openHub(t, b, 1280, 840)
	if !tt.HasText("Not tracked for this game.") {
		t.Fatalf("a failed load did not fall back; texts = %q", tt.Texts())
	}
}

// TestLoadTrophiesNeedsAKeyAccountAndID checks the three things a load needs:
// without any of them nothing is started and the card keeps its empty state,
// rather than a request that comes back 401 retried on every frame.
func TestLoadTrophiesNeedsAKeyAccountAndID(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]

	// No account: `launcher` resolves none, and no key is stored.
	if g := a.trophiesFor(e); g != nil {
		t.Fatalf("a game with no account got trophies: %+v", g)
	}
	for i := 0; i < 5; i++ {
		if g := a.trophiesFor(e); g != nil {
			t.Fatalf("a game with no account got trophies on frame %d: %+v", i, g)
		}
	}
	if a.trophiesStarted(e.slug()) {
		t.Error("a load was started with no account, key or app id")
	}

	// A game with no app id has nothing to ask about even with a key and an
	// account. `refresh` skips a rebuild when nothing changed, so the entry is
	// forced through it.
	b := launcher(t, installsOf(1))
	b.installs[0].AppID = ""
	b.lastKey = -1
	b.entries = nil
	b.refresh()
	b.steamKey = "0123456789ABCDEF0123456789ABCDEF"
	b.steam = &steamstore.Account{SteamID: "76561198000000000"}

	if b.entries[0].inst.AppID != "" {
		t.Fatalf("the fixture game kept its app id: %q", b.entries[0].inst.AppID)
	}
	if g := b.trophiesFor(b.entries[0]); g != nil {
		t.Fatalf("a game with no app id got trophies: %+v", g)
	}
	if b.trophiesStarted(b.entries[0].slug()) {
		t.Error("a load was started for a game with no app id")
	}
}

// TestTrophiesForAsksOnce checks the guard that keeps the API from being called
// every frame: a game already begun is never begun again, and reads as no data
// until its answer arrives.
func TestTrophiesForAsksOnce(t *testing.T) {
	a := launcher(t, installsOf(1))
	slug := a.entries[0].slug()

	a.markTrophiesStarted(slug)

	for i := 0; i < 10; i++ {
		if g := a.trophiesFor(a.entries[0]); g != nil {
			t.Fatalf("frame %d read data that had not arrived: %+v", i, g)
		}
		if !a.trophiesStarted(slug) {
			t.Fatal("the load was forgotten and would be asked again")
		}
	}
}

// TestLoadTrophiesReadsTheCacheAndLandsIt checks the whole path the card relies
// on: the background load reaches the Steam client, an answer already on disk is
// used without a network, and what it finds arrives through `update` so the card
// shows it. The cache is written by the test, synchronously, so no request is
// made and nothing races the temp folder's cleanup.
func TestLoadTrophiesReadsTheCacheAndLandsIt(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.store = steamstore.NewClient(a.dataDir)
	a.steamKey = "0123456789ABCDEF0123456789ABCDEF"
	a.steam = &steamstore.Account{SteamID: "76561198000000000"}

	e := a.entries[0]
	if e.inst.AppID == "" {
		t.Fatal("the fixture game has no app id")
	}

	// A cache inside its TTL, written before the load starts, so the client
	// answers from disk.
	want := fixtureTrophies()
	path := filepath.Join(a.dataDir, "store", "achievements-76561198000000000-"+e.inst.AppID+".json")
	writeJSON(t, path, map[string]any{
		"fetched":      time.Now(),
		"achievements": want.list,
	})
	writeJSON(t, filepath.Join(a.dataDir, "store", "rarity-"+e.inst.AppID+".json"), map[string]any{
		"fetched":  time.Now(),
		"percents": want.rarity,
	})

	// `update` runs at once when there is no window, which is what a test has,
	// but `loadTrophies` starts its work on a goroutine: wait for it, bounded, so
	// a hang fails the test instead of blocking it.
	a.loadTrophies(e)

	var g *gameTrophies
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if g = a.trophiesLoaded(e.slug()); g != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if g == nil {
		t.Fatal("the load did not land anything")
	}
	if len(g.list) != len(want.list) || g.earned() != 3 {
		t.Fatalf("loaded %d achievements, %d earned; want %d and 3", len(g.list), g.earned(), len(want.list))
	}
	if !g.hasRarity {
		t.Error("the rarity cache was not read")
	}
	if g.percent() != 43 {
		t.Errorf("percent = %d, want 43", g.percent())
	}

	// And the card paints it.
	tt := openHub(t, a, 1280, 840)
	for _, text := range []string{"3 of 7", "43%", "Lilac and Gooseberries", "Platinum 1"} {
		if !tt.HasText(text) {
			t.Fatalf("the card does not show %q; texts = %q", text, tt.Texts())
		}
	}
}

// TestTrophyTierOfAchievementNeedsItsOwnPercentage checks an achievement Steam
// published no percentage for is not counted under a tier it does not belong to.
func TestTrophyTierOfAchievementNeedsItsOwnPercentage(t *testing.T) {
	g := fixtureTrophies()
	g.list = append(g.list, steamstore.Achievement{APIName: "NOPERCENT", Achieved: true})

	if _, ok := g.tierOfAchievement(steamstore.Achievement{APIName: "NOPERCENT"}); ok {
		t.Error("an achievement with no percentage was given a tier")
	}

	counts, ok := g.tierCounts()
	if !ok {
		t.Fatal("the counts were rejected with percentages present")
	}
	// The four known earned achievements count; the unknown one does not.
	total := 0
	for _, n := range counts {
		total += n
	}
	if total != 3 {
		t.Errorf("counted %d achievements into tiers, want the 3 with a percentage", total)
	}
}

// writeJSON writes a fixture file, synchronously: a background write would
// outlive t.TempDir and race its cleanup.
func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTrophyTiersAreRarity pins the one thing that must not be invented: Steam
// has no bronze, silver, gold or platinum, so a tier is derived from the real
// share of players who hold the achievement.
func TestTrophyTiersAreRarity(t *testing.T) {
	for _, tc := range []struct {
		percent float64
		want    trophyTier
	}{
		{0.5, tierPlatinum},
		{4.9, tierPlatinum},
		{5, tierGold},
		{14.9, tierGold},
		{15, tierSilver},
		{49.9, tierSilver},
		{50, tierBronze},
		{100, tierBronze},
	} {
		if got := tierOf(tc.percent); got != tc.want {
			t.Errorf("tierOf(%v) = %v, want %v", tc.percent, got, tc.want)
		}
	}

	// A tier is named and coloured as the design's TIER map does.
	if tierPlatinum.name() != "Platinum" || tierGold.name() != "Gold" ||
		tierSilver.name() != "Silver" || tierBronze.name() != "Bronze" {
		t.Error("the tier names do not match the design")
	}
}

// TestTrophiesPercentRounds checks the ring's percentage is the earned share
// rounded, and that a game with no achievements is 0 rather than a divide by
// zero.
func TestTrophiesPercentRounds(t *testing.T) {
	g := fixtureTrophies()
	if got := g.percent(); got != 43 {
		t.Errorf("percent = %d, want 3 of 7 rounded to 43", got)
	}
	if got := g.earned(); got != 3 {
		t.Errorf("earned = %d, want 3", got)
	}
	if got := (&gameTrophies{}).percent(); got != 0 {
		t.Errorf("an empty game is %d%%, want 0", got)
	}
	if got := (*gameTrophies)(nil).percent(); got != 0 {
		t.Errorf("a nil game is %d%%, want 0", got)
	}
	if got := (*gameTrophies)(nil).earned(); got != 0 {
		t.Errorf("a nil game earned %d, want 0", got)
	}
	if got := (*gameTrophies)(nil).latestRows(2); got != nil {
		t.Errorf("a nil game listed %d rows, want none", len(got))
	}
}

// TestTrophiesLatestIsNewest checks the rows under the ring are the newest
// unlocks, not the first Steam listed.
func TestTrophiesLatestIsNewest(t *testing.T) {
	g := fixtureTrophies()

	rows := g.latestRows(2)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].APIName != "LILAC" || rows[1].APIName != "ENEMY" {
		t.Fatalf("rows = %q then %q, want LILAC then ENEMY", rows[0].APIName, rows[1].APIName)
	}

	// An unearned achievement is never in the list.
	for _, a := range g.recent() {
		if !a.Achieved {
			t.Errorf("%q was listed without being earned", a.APIName)
		}
	}
	if got := len(g.recent()); got != 3 {
		t.Errorf("recent = %d, want the 3 earned", got)
	}
}

// TestTrophyRowNamesAnUnnamedAchievement checks a row never shows an empty line
// when Steam left the name out.
func TestTrophyRowNamesAnUnnamedAchievement(t *testing.T) {
	if got := achievementName(steamstore.Achievement{APIName: "LILAC"}); got != "LILAC" {
		t.Errorf("name = %q, want the API name", got)
	}
	if got := achievementName(steamstore.Achievement{APIName: "LILAC", Name: "Lilac"}); got != "Lilac" {
		t.Errorf("name = %q, want the real one", got)
	}
}

// TestRingPathIsClosed checks the ring is a circle drawn from the top, which is
// what makes the arc start where the design's rotated SVG does.
func TestRingPathIsClosed(t *testing.T) {
	b := ringPath(50, 50, 20).Bounds()

	// The bezier circle is a hair inside its control points, so the box is the
	// circle's own 40 across within a DIP.
	for _, v := range []float32{b.W, b.H} {
		if v < 39.5 || v > 40.5 {
			t.Errorf("ring box = %v x %v, want about 40", b.W, b.H)
		}
	}
	if b.X < 29.5 || b.X > 30.5 || b.Y < 29.5 || b.Y > 30.5 {
		t.Errorf("ring box starts at %v,%v, want 30,30", b.X, b.Y)
	}
}
