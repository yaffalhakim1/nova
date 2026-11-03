package app

import (
	"testing"
	"time"
)

// TestLastPlayedIsOneWording pins the rule the surfaces share: the home meta
// line and a recent card say the same thing about the same game, because both
// read one function over one field.
func TestLastPlayedIsOneWording(t *testing.T) {
	a := launcher(t, installsOf(2))
	e := a.entries[0]
	a.stat = stats{Games: map[string]*gameStats{}}

	ages := []time.Duration{
		30 * time.Second,
		5 * time.Minute,
		90 * time.Minute,
		23 * time.Hour,
		47 * time.Hour,
		10 * 24 * time.Hour,
	}

	for _, age := range ages {
		a.stat.get(e.slug()).Last = time.Now().Add(-age)

		card := lastPlayed(a.stat.get(e.slug()).Last)
		meta := a.lastPlayedLine(e)

		want := "Last played " + card
		if meta != want {
			t.Errorf("age %v: meta = %q, card = %q", age, meta, card)
		}
	}

	// With no history at all the meta line says it once, not twice.
	a.stat.get(e.slug()).Last = time.Time{}
	if got := a.lastPlayedLine(e); got != "Never played" {
		t.Errorf("no history: meta = %q, want %q", got, "Never played")
	}
}

// TestStatsGetRetainsOnANilMap guards a trap that silently loses data: with no
// map, get used to return a throwaway, so every write to it vanished.
func TestStatsGetRetainsOnANilMap(t *testing.T) {
	s := stats{}

	g := s.get("a")
	g.Played = 3 * time.Hour
	g.Last = time.Now()

	if again := s.get("a"); again.Played != 3*time.Hour {
		t.Fatalf("the write was lost: %v", again.Played)
	}
	if len(s.Games) != 1 {
		t.Fatalf("games = %d, want the entry created", len(s.Games))
	}
}

// TestRecordLandsInTheHistory checks a finished session is written to the map
// the app keeps, not to a copy of it.
func TestRecordLandsInTheHistory(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	a.stat = stats{}

	a.stat.record(e.slug(), 90*time.Minute)

	if got := a.stat.get(e.slug()).Played; got != 90*time.Minute {
		t.Fatalf("played = %v, want 90m", got)
	}
	if a.stat.get(e.slug()).Sessions != 1 {
		t.Fatalf("sessions = %d, want 1", a.stat.get(e.slug()).Sessions)
	}
}

// TestLastPlayedRounds checks the wording rounds rather than truncates, so a
// game played an hour and a half ago does not read "1 hr ago".
func TestLastPlayedRounds(t *testing.T) {
	now := time.Now()

	cases := []struct {
		age  time.Duration
		want string
	}{
		{20 * time.Second, "Just now"},
		{5 * time.Minute, "5 min ago"},
		{90 * time.Minute, "2 hr ago"},
		{119 * time.Minute, "2 hr ago"},
		{23 * time.Hour, "23 hr ago"},
		{47 * time.Hour, "2 days ago"},
		{41 * 24 * time.Hour, "41 days ago"},
		// A clock that ran ahead must not read as a negative age.
		{-2 * time.Hour, "Just now"},
	}

	for _, c := range cases {
		if got := lastPlayed(now.Add(-c.age)); got != c.want {
			t.Errorf("age %v = %q, want %q", c.age, got, c.want)
		}
	}

	if got := lastPlayed(time.Time{}); got != "Not played yet" {
		t.Errorf("zero time = %q, want %q", got, "Not played yet")
	}
}

// TestRecentRowFollowsStats checks "Jump back in" is built from the play
// history, newest first, and survives a restart — it is not a list of what was
// clicked this session.
func TestRecentRowFollowsStats(t *testing.T) {
	a := launcher(t, installsOf(3))

	dir := a.dataDir
	a.stat = stats{Games: map[string]*gameStats{}}
	a.stat.get(a.entries[0].slug()).Last = time.Now().Add(-2 * time.Hour)
	a.stat.get(a.entries[1].slug()).Last = time.Now().Add(-5 * time.Minute)
	if err := a.stat.save(dir); err != nil {
		t.Fatal(err)
	}

	reloaded := launcher(t, installsOf(3))
	reloaded.dataDir = dir
	reloaded.stat = loadStats(dir)

	got := reloaded.playedLately(recentCards)
	if len(got) != 2 {
		t.Fatalf("recent = %d games, want 2", len(got))
	}
	if got[0].name() != "Game 1" || got[1].name() != "Game 0" {
		t.Fatalf("recent order = %q, %q; want Game 1 then Game 0",
			got[0].name(), got[1].name())
	}
}

// TestHomeOrderPutsTheLastPlayedFirst checks the carousel opens on the game
// played last, as the design's homeOrder() does, and that the row above it
// agrees: the first tile is the first card.
func TestHomeOrderPutsTheLastPlayedFirst(t *testing.T) {
	a := launcher(t, installsOf(3))
	a.stat = stats{Games: map[string]*gameStats{}}
	a.stat.get(a.entries[2].slug()).Last = time.Now().Add(-time.Minute)

	a.section = sectionHome
	a.lastKey = -1
	a.refresh()

	row := a.playedLately(recentCards)
	if len(row) == 0 {
		t.Fatal("the recent row is empty")
	}
	if len(a.entries) == 0 || a.entries[0].slug() != row[0].slug() {
		t.Fatalf("the carousel does not open on the last played game: %v vs %v",
			a.entries, row)
	}
	// Every installed game is listed: Home is not filtered by play history.
	if len(a.entries) != 3 {
		t.Fatalf("home shows %d games, want all 3", len(a.entries))
	}
}
