package steam

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestParseAchievements reads the shape the API really answers with, quoted from
// a live call for The Witcher 3: an `apiname`, `achieved` as 1 or 0,
// `unlocktime` as a unix second that is 0 when it was never earned, and a
// description that is an empty string on a hidden achievement.
func TestParseAchievements(t *testing.T) {
	raw := []byte(`{"playerstats":{
		"steamID":"76561198000000000","gameName":"The Witcher 3",
		"achievements":[
			{"apiname":"LILAC","achieved":1,"unlocktime":1771980824,"name":"Lilac and Gooseberries","description":"Find and kill a panicked horse."},
			{"apiname":"HIDDEN","achieved":0,"unlocktime":0,"name":"Something Hidden","description":""}
		]}}`)

	list, ok := parseAchievements(raw)
	if !ok {
		t.Fatal("parse rejected a valid response")
	}
	if len(list) != 2 {
		t.Fatalf("achievements = %d, want 2", len(list))
	}

	got := list[0]
	if got.APIName != "LILAC" || got.Name != "Lilac and Gooseberries" {
		t.Fatalf("first = %+v", got)
	}
	if !got.Achieved {
		t.Error("an achievement with achieved=1 was read as unearned")
	}
	if got.Unlocked.Unix() != 1771980824 {
		t.Errorf("unlocked = %v, want the unix second it carries", got.Unlocked)
	}

	// An unearned achievement has no unlock time, and the zero time is how that
	// is said rather than 1970.
	if list[1].Achieved {
		t.Error("an achievement with achieved=0 was read as earned")
	}
	if !list[1].Unlocked.IsZero() {
		t.Errorf("unearned unlocked = %v, want the zero time", list[1].Unlocked)
	}
	// A hidden achievement's description is an empty string, never missing; it
	// must not be confused with a parse failure.
	if list[1].Description != "" {
		t.Errorf("hidden description = %q, want empty", list[1].Description)
	}
}

// TestParseAchievementsQuotedNumbers checks the same body with its numbers as
// strings, which is how this API family has served them elsewhere.
func TestParseAchievementsQuotedNumbers(t *testing.T) {
	raw := []byte(`{"playerstats":{"achievements":[
		{"apiname":"LILAC","achieved":"1","unlocktime":"1771980824","name":"Lilac and Gooseberries","description":""}
	]}}`)

	list, ok := parseAchievements(raw)
	if !ok {
		t.Fatal("parse rejected quoted numbers")
	}
	if len(list) != 1 || !list[0].Achieved {
		t.Fatalf("list = %+v, want one earned achievement", list)
	}
	if list[0].Unlocked.Unix() != 1771980824 {
		t.Errorf("unlocked = %v, want the unix second it carries", list[0].Unlocked)
	}
}

// TestParseAchievementsNoStats pins the answer a game with no achievements
// gives: `success:false` and an error, which is a true answer about the game,
// not a failure to ask.
func TestParseAchievementsNoStats(t *testing.T) {
	raw := []byte(`{"playerstats":{"success":false,"error":"Requested app has no stats"}}`)

	list, ok := parseAchievements(raw)
	if !ok {
		t.Fatal("a game with no stats was reported as a failure")
	}
	if len(list) != 0 {
		t.Fatalf("achievements = %d, want none", len(list))
	}
}

// TestParseAchievementsJunk rejects a body that is not the API's shape at all,
// so junk is not read as "this game has no achievements".
func TestParseAchievementsJunk(t *testing.T) {
	if _, ok := parseAchievements([]byte(`not json`)); ok {
		t.Error("junk was read as a response")
	}
	if _, ok := parseAchievements([]byte(`{"response":{}}`)); ok {
		t.Error("a body with no playerstats was read as a response")
	}
}

// TestParseRarity reads the global percentages and keys them by the API name
// the player's own list carries, which is how the two are joined.
//
// The body is a slice of a live answer for The Witcher 3, kept exactly as Steam
// served it — `percent` as a **quoted string**, which is not what the endpoint's
// documentation shows.
func TestParseRarity(t *testing.T) {
	raw := []byte(`{"achievementpercentages":{"achievements":[` +
		`{"name":"LILAC","percent":"60.3"},` +
		`{"name":"BUTCHER_OF_BLAVIKEN","percent":"51.1"},` +
		`{"name":"ENEMY_OF_MY_ENEMY","percent":"2.6"}` +
		`]}}`)

	percents, ok := parseRarity(raw)
	if !ok {
		t.Fatal("parse rejected a valid response")
	}
	if got := percents["LILAC"]; got != 60.3 {
		t.Errorf("LILAC = %v, want 60.3", got)
	}
	if got := percents["ENEMY_OF_MY_ENEMY"]; got != 2.6 {
		t.Errorf("ENEMY_OF_MY_ENEMY = %v, want 2.6", got)
	}
	if len(percents) != 3 {
		t.Errorf("percents = %d, want 3", len(percents))
	}
}

// TestParseRarityAlsoReadsNumbers checks the other shape Steam has answered with:
// a bare number. The decoder takes both, because a body of the other one must not
// be thrown away whole.
func TestParseRarityAlsoReadsNumbers(t *testing.T) {
	raw := []byte(`{"achievementpercentages":{"achievements":[{"name":"LILAC","percent":60.3}]}}`)

	percents, ok := parseRarity(raw)
	if !ok {
		t.Fatal("parse rejected a numeric percentage")
	}
	if got := percents["LILAC"]; got != 60.3 {
		t.Errorf("LILAC = %v, want 60.3", got)
	}
}

// TestParseRarityRejectsAWordedPercent checks a percentage that is not a number
// fails rather than becoming zero, which would file the achievement as common.
func TestParseRarityRejectsAWordedPercent(t *testing.T) {
	raw := []byte(`{"achievementpercentages":{"achievements":[{"name":"LILAC","percent":"common"}]}}`)

	if _, ok := parseRarity(raw); ok {
		t.Error("a percentage that is not a number was read as one")
	}
}

// TestParseRarityJunk rejects what is not the API's shape.
func TestParseRarityJunk(t *testing.T) {
	if _, ok := parseRarity([]byte(`not json`)); ok {
		t.Error("junk was read as a response")
	}
	if _, ok := parseRarity([]byte(`{"achievementpercentages":{"achievements":[]}}`)); ok {
		t.Error("an empty list was read as a response")
	}
}

// TestAchievementsNeedsKeyAndIDs checks the client refuses to call the API
// without the three things it needs, rather than making a request that comes
// back 401.
func TestAchievementsNeedsKeyAndIDs(t *testing.T) {
	c := NewClient(t.TempDir())
	const id = "76561198000000000"

	if _, ok := c.Achievements(t.Context(), "", id, "292030"); ok {
		t.Error("a request with no key was attempted")
	}
	if _, ok := c.Achievements(t.Context(), "key", "", "292030"); ok {
		t.Error("a request with no steam id was attempted")
	}
	if _, ok := c.Achievements(t.Context(), "key", id, ""); ok {
		t.Error("a request with no app id was attempted")
	}
	if _, ok := c.Rarity(""); ok {
		t.Error("a rarity request with no app id was attempted")
	}
}

// TestAchievementsCache checks a fetched list is served from disk afterwards, so
// opening a game does not call the API every time, and that one account's cache
// is not served for another.
func TestAchievementsCache(t *testing.T) {
	c := NewClient(t.TempDir())
	const id, app = "76561198000000000", "292030"

	list := []Achievement{
		{APIName: "LILAC", Name: "Lilac and Gooseberries", Achieved: true, Unlocked: time.Unix(1771980824, 0)},
		{APIName: "HIDDEN", Name: "Something Hidden"},
	}
	if err := c.saveAchievements(id, app, list); err != nil {
		t.Fatal(err)
	}

	got, ok := c.cachedAchievements(id, app)
	if !ok {
		t.Fatal("the cache was not read back")
	}
	if len(got) != 2 || got[0].APIName != "LILAC" || !got[0].Achieved {
		t.Fatalf("cache = %+v", got)
	}
	if got[0].Unlocked.Unix() != 1771980824 {
		t.Errorf("the cached unlock time was lost: %v", got[0].Unlocked)
	}

	if _, ok := c.cachedAchievements("76561198000000002", app); ok {
		t.Error("another account was served from this account's cache")
	}
	if _, ok := c.cachedAchievements(id, "999"); ok {
		t.Error("another game was served from this game's cache")
	}
}

// TestRarityCache checks the global percentages are cached too.
func TestRarityCache(t *testing.T) {
	c := NewClient(t.TempDir())

	percents := map[string]float64{"LILAC": 60.3}
	if err := c.saveRarity("292030", percents); err != nil {
		t.Fatal(err)
	}

	got, ok := c.cachedRarity("292030")
	if !ok {
		t.Fatal("the cache was not read back")
	}
	if got["LILAC"] != 60.3 {
		t.Fatalf("cache = %+v", got)
	}
	if _, ok := c.cachedRarity("999"); ok {
		t.Error("another game was served from this game's cache")
	}
}

// TestAchievementsCacheExpires checks a stale cache is not served, so an
// achievement earned today shows up after the TTL rather than next install.
func TestAchievementsCacheExpires(t *testing.T) {
	c := NewClient(t.TempDir())

	if err := c.saveAchievements("1", "2", []Achievement{{APIName: "X"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.cachedAchievements("1", "2"); !ok {
		t.Fatal("a fresh cache was rejected")
	}

	// Backdate the fetch in the file it was written to, which is how the cache
	// sees a TTL pass without a clock the test controls.
	path := c.achievementsPath("1", "2")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f achievementsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	f.Fetched = time.Now().Add(-achievementsCacheTTL - time.Hour)
	raw, err = json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, ok := c.cachedAchievements("1", "2"); ok {
		t.Error("an expired cache was served")
	}
}
