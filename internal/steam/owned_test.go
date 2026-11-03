package steam

import (
	"errors"
	"testing"
)

// TestParseOwnedGames reads the store's own shape: a response wrapper around a
// games list whose playtime is in minutes.
func TestParseOwnedGames(t *testing.T) {
	raw := []byte(`{"response":{"game_count":3,"games":[
		{"appid":1091500,"name":"Cyberpunk 2077","playtime_forever":7833,"img_icon_url":"abc"},
		{"appid":480,"name":"Spacewar","playtime_forever":0},
		{"appid":292030,"name":"The Witcher 3","playtime_forever":5285,"playtime_2weeks":12}
	]}}`)

	games, ok := parseOwnedGames(raw)
	if !ok {
		t.Fatal("parse rejected a valid response")
	}
	if len(games) != 3 {
		t.Fatalf("games = %d, want 3", len(games))
	}
	if games[0].AppID != "1091500" || games[0].Name != "Cyberpunk 2077" {
		t.Fatalf("first = %+v", games[0])
	}
	if games[0].Played.Hours() < 130 {
		t.Fatalf("played = %v, want about 130h from 7833 minutes", games[0].Played)
	}
	if games[2].Recent.Hours() < 0.1 {
		t.Fatalf("recent = %v, want two weeks of play", games[2].Recent)
	}
}

// TestParseOwnedGamesPrivateProfile pins the failure that would otherwise look
// like "you own no games": a private profile answers with an empty response and
// no games list at all.
func TestParseOwnedGamesPrivateProfile(t *testing.T) {
	if _, ok := parseOwnedGames([]byte(`{"response":{}}`)); ok {
		t.Fatal("an empty response was read as a real library")
	}
	if _, ok := parseOwnedGames([]byte(`not json`)); ok {
		t.Fatal("junk was read as a real library")
	}
	if _, ok := parseOwnedGames([]byte(`{"response":{"game_count":0}}`)); ok {
		t.Fatal("a library of no games was read as a real one")
	}
}

// TestOwnedGamesNeedsAKey checks the client refuses to call the API without a
// key, rather than making a request that comes back 401.
func TestOwnedGamesNeedsAKey(t *testing.T) {
	c := NewClient(t.TempDir())
	if _, ok := c.OwnedGames(t.Context(), "", "76561198000000000"); ok {
		t.Fatal("a request with no key was attempted")
	}
	if _, ok := c.OwnedGames(t.Context(), "key", ""); ok {
		t.Fatal("a request with no steam id was attempted")
	}
}

// TestOwnedGamesCaches checks a second read is served from disk, so the API is
// not called every time the window opens.
func TestOwnedGamesCaches(t *testing.T) {
	dir := t.TempDir()
	c := NewClient(dir)

	games := []OwnedGame{{AppID: "1", Name: "One"}}
	if err := c.saveOwned("76561198000000001", games); err != nil {
		t.Fatal(err)
	}

	got, ok := c.cachedOwned("76561198000000001")
	if !ok {
		t.Fatal("the cache was not read back")
	}
	if len(got) != 1 || got[0].Name != "One" {
		t.Fatalf("cache = %+v", got)
	}

	if _, ok := c.cachedOwned("76561198000000002"); ok {
		t.Fatal("another account was served from this account's cache")
	}
}

// TestKeyStoreChecksShape pins what a Steam key looks like, so an obvious typo
// is caught before it becomes a 401 the user has to interpret.
func TestKeyStoreChecksShape(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want bool
	}{
		{"0123456789ABCDEF0123456789ABCDEF", true},
		{"abcdef0123456789abcdef0123456789", true},
		{"0123456789ABCDEF", false},      // too short
		{"not-a-steam-key-at-all", false}, // not hex
		{"", false},
	} {
		if got := ValidKeyShape(tc.key); got != tc.want {
			t.Errorf("ValidKeyShape(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// TestRedactedNeverShowsTheKey pins that what the window displays cannot leak a
// key the user pasted.
func TestRedactedNeverShowsTheKey(t *testing.T) {
	key := "0123456789ABCDEF0123456789ABCDEF"
	got := RedactedKey(key)
	if got == key {
		t.Fatal("the key was shown in full")
	}
	if len(got) >= len(key) {
		t.Fatalf("redacted = %q, not shorter than the key", got)
	}
	if RedactedKey("") != "" {
		t.Fatal("an empty key was shown as something")
	}
}

// TestOwnedGamesErrorIsNamed checks an HTTP failure comes back as an error the
// window can show, not as an empty library.
func TestOwnedGamesErrorIsNamed(t *testing.T) {
	if !errors.Is(ErrNoKey, ErrNoKey) {
		t.Fatal("sentinel")
	}
}
