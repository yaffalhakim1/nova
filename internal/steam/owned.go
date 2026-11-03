package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ownedURL is the Web API endpoint that lists the games an account owns.
const ownedURL = "https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/"

// ErrNoKey is returned when a call needs a Steam Web API key and has none.
var ErrNoKey = errors.New("a Steam API key is needed")

// ownedCacheTTL is how long a fetched library is trusted. It changes only when
// the user buys or refunds a game, so a day is generous and keeps the API calls
// few.
const ownedCacheTTL = 24 * time.Hour

// OwnedGame is one game an account owns, installed or not.
type OwnedGame struct {
	// AppID is the Steam app id, as a string so it compares with Install.AppID.
	AppID string
	// Name is the game's name, from the store.
	Name string
	// Played is the lifetime play time Steam has recorded.
	Played time.Duration
	// Recent is the play time of the last two weeks, or zero.
	Recent time.Duration
	// Last is the last time Steam recorded the game running, or zero.
	Last time.Time
}

// OwnedGames lists the games an account owns, from the cache when it is fresh
// and the API otherwise.
//
// It is the one call that needs a key. Everything else this app reads comes from
// Steam's own files, so a machine with no key still works: only this list is
// missing, and the window says so.
func (c *Client) OwnedGames(ctx context.Context, key, steamID string) ([]OwnedGame, bool) {
	if c == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(steamID) == "" {
		return nil, false
	}

	if games, ok := c.cachedOwned(steamID); ok {
		return games, true
	}

	games, ok := c.fetchOwned(ctx, key, steamID)
	if !ok {
		return nil, false
	}
	_ = c.saveOwned(steamID, games)
	return games, true
}

// fetchOwned calls the Web API once.
func (c *Client) fetchOwned(ctx context.Context, key, steamID string) ([]OwnedGame, bool) {
	q := url.Values{}
	q.Set("key", key)
	q.Set("steamid", steamID)
	q.Set("include_appinfo", "1")
	q.Set("include_played_free_games", "1")
	q.Set("include_family_licenses", "1")
	q.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ownedURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, false
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, false
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, false
	}
	return parseOwnedGames(raw)
}

// parseOwnedGames reads the API's wrapper.
//
// `{"response":{}}` is what a private profile answers with, so it is reported as
// a failure rather than as a library of no games: the two look the same to a
// reader and only one of them is worth showing.
func parseOwnedGames(raw []byte) ([]OwnedGame, bool) {
	var wrapper struct {
		Response struct {
			Count int `json:"game_count"`
			Games []struct {
				AppID    int    `json:"appid"`
				Name     string `json:"name"`
				Forever  int    `json:"playtime_forever"`
				TwoWeeks int    `json:"playtime_2weeks"`
				LastPlay int64  `json:"rtime_last_played"`
			} `json:"games"`
		} `json:"response"`
	}

	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, false
	}
	if len(wrapper.Response.Games) == 0 {
		return nil, false
	}

	out := make([]OwnedGame, 0, len(wrapper.Response.Games))
	for _, g := range wrapper.Response.Games {
		if g.AppID == 0 {
			continue
		}
		last := time.Time{}
		if g.LastPlay > 0 {
			last = time.Unix(g.LastPlay, 0)
		}
		out = append(out, OwnedGame{
			AppID:  fmt.Sprint(g.AppID),
			Name:   g.Name,
			Played: time.Duration(g.Forever) * time.Minute,
			Recent: time.Duration(g.TwoWeeks) * time.Minute,
			Last:   last,
		})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// ownedPath is where one account's library is cached.
func (c *Client) ownedPath(steamID string) string {
	return filepath.Join(c.dir, "owned-"+sanitize(steamID)+".json")
}

// ownedFile is the cache's shape, with when it was written so it can expire.
type ownedFile struct {
	Fetched time.Time   `json:"fetched"`
	Games   []OwnedGame `json:"games"`
}

// cachedOwned reads a fresh cache, or reports that there is none.
func (c *Client) cachedOwned(steamID string) ([]OwnedGame, bool) {
	raw, err := os.ReadFile(c.ownedPath(steamID))
	if err != nil {
		return nil, false
	}

	var f ownedFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, false
	}
	if time.Since(f.Fetched) > ownedCacheTTL {
		return nil, false
	}
	if len(f.Games) == 0 {
		return nil, false
	}
	return f.Games, true
}

// saveOwned writes the cache, written whole so a crash cannot leave half of it.
func (c *Client) saveOwned(steamID string, games []OwnedGame) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ownedFile{Fetched: time.Now(), Games: games}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.ownedPath(steamID), raw, 0o644)
}
