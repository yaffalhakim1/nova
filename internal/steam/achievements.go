package steam

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The two endpoints an account's achievements need: the player's own unlocks,
// which needs the Web API key, and how many players have each one, which needs
// nothing.
const (
	achievementsURL = "https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1/"
	rarityURL       = "https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/"
)

// achievementsCacheTTL is how long an account's unlocks are trusted. An unlock
// is permanent once it happens, but a session can earn a few at a time, so a
// few hours keeps the window honest without asking on every launch.
const achievementsCacheTTL = 6 * time.Hour

// rarityCacheTTL is how long the global percentages are trusted. They move a
// fraction of a percent a day.
const rarityCacheTTL = 24 * time.Hour

// Achievement is one achievement of one game, and whether this account has it.
type Achievement struct {
	// APIName is Steam's own name for it, which is the key the global
	// percentages are matched by. Name is what the player reads.
	APIName     string
	Name        string
	Description string
	Achieved    bool
	// Unlocked is when it was earned, or the zero time when it was not.
	Unlocked time.Time
}

// Achievements lists a game's achievements and which of them the account has.
//
// A game Steam has no stats for answers with success:false and no list; that is
// reported as no achievements rather than as a failure, because it is a true
// answer about the game. Only a request that could not be made or read fails.
func (c *Client) Achievements(ctx context.Context, key, steamID, appID string) ([]Achievement, bool) {
	if c == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(steamID) == "" || strings.TrimSpace(appID) == "" {
		return nil, false
	}

	if list, ok := c.cachedAchievements(steamID, appID); ok {
		return list, true
	}

	list, ok := c.fetchAchievements(ctx, key, steamID, appID)
	if !ok {
		return nil, false
	}
	// An empty list is not written: a game that gains stats later, or a profile
	// made public later, is then asked again rather than remembered as empty.
	if len(list) > 0 {
		_ = c.saveAchievements(steamID, appID, list)
	}
	return list, true
}

// fetchAchievements calls the API once.
func (c *Client) fetchAchievements(ctx context.Context, key, steamID, appID string) ([]Achievement, bool) {
	q := url.Values{}
	q.Set("key", key)
	q.Set("steamid", steamID)
	q.Set("appid", appID)
	q.Set("l", "english")
	q.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, achievementsURL+"?"+q.Encode(), nil)
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
	return parseAchievements(raw)
}

// parseAchievements reads the API's `playerstats` wrapper.
//
// `achieved` is 1 or 0 and `unlocktime` a unix second, 0 when it was never
// earned. Both are read through `jsonNumber`, which takes a number or a quoted
// one: this API family has served `percent` as a string, and a decoder that
// expects only one shape throws away the whole body when it meets the other.
//
// A hidden achievement's `description` is an empty string, never a missing key,
// so nothing here distinguishes the two.
//
// A game with no stats answers `{"playerstats":{"success":false,"error":...}}`:
// that is no achievements, which is true, not a failure. A body with no
// `playerstats` at all is junk, and fails.
func parseAchievements(raw []byte) ([]Achievement, bool) {
	var wrapper struct {
		PlayerStats *struct {
			Success      bool `json:"success"`
			Achievements []struct {
				APIName    string     `json:"apiname"`
				Achieved   jsonNumber `json:"achieved"`
				UnlockTime jsonNumber `json:"unlocktime"`
				Name       string     `json:"name"`
				Desc       string     `json:"description"`
			} `json:"achievements"`
		} `json:"playerstats"`
	}

	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.PlayerStats == nil {
		return nil, false
	}

	out := make([]Achievement, 0, len(wrapper.PlayerStats.Achievements))
	for _, a := range wrapper.PlayerStats.Achievements {
		if a.APIName == "" {
			continue
		}
		unlocked := time.Time{}
		if a.UnlockTime > 0 {
			unlocked = time.Unix(int64(a.UnlockTime), 0)
		}
		out = append(out, Achievement{
			APIName:     a.APIName,
			Name:        a.Name,
			Description: a.Desc,
			Achieved:    a.Achieved == 1,
			Unlocked:    unlocked,
		})
	}
	if len(out) == 0 {
		return nil, true
	}
	return out, true
}

// Rarity is how many players have each of a game's achievements, by the name
// the achievement carries, as a percentage from 0 to 100.
//
// It needs no key: Steam publishes it for everyone.
func (c *Client) Rarity(appID string) (map[string]float64, bool) {
	if c == nil || strings.TrimSpace(appID) == "" {
		return nil, false
	}

	if percents, ok := c.cachedRarity(appID); ok {
		return percents, true
	}

	percents, ok := c.fetchRarity(appID)
	if !ok {
		return nil, false
	}
	_ = c.saveRarity(appID, percents)
	return percents, true
}

// fetchRarity calls the API once.
func (c *Client) fetchRarity(appID string) (map[string]float64, bool) {
	q := url.Values{}
	q.Set("gameid", appID)

	res, err := c.http.Get(rarityURL + "?" + q.Encode())
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, false
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, false
	}
	return parseRarity(raw)
}

// parseRarity reads the API's `achievementpercentages` wrapper, keyed by the
// achievement's own name so it can be joined with the player's unlocks.
//
// `percent` is a **string** in the response Steam actually serves —
// `{"name":"LILAC","percent":"60.3"}` — although its documentation and its older
// answers show a number. `jsonNumber` reads either, because a decoder that
// expects one shape rejects the whole body when it gets the other.
func parseRarity(raw []byte) (map[string]float64, bool) {
	var wrapper struct {
		Percentages struct {
			Achievements []struct {
				Name    string     `json:"name"`
				Percent jsonNumber `json:"percent"`
			} `json:"achievements"`
		} `json:"achievementpercentages"`
	}

	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, false
	}

	out := make(map[string]float64, len(wrapper.Percentages.Achievements))
	for _, a := range wrapper.Percentages.Achievements {
		if a.Name == "" {
			continue
		}
		out[a.Name] = float64(a.Percent)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// jsonNumber is a number that Steam writes either bare or quoted, which is the
// difference between `"percent":60.3` and `"percent":"60.3"`. It reads both so a
// response of either shape decodes.
type jsonNumber float64

// UnmarshalJSON reads a number or a string holding one. A value that is neither
// is reported as malformed, so a missing percentage cannot silently become zero
// and land its achievement in the wrong tier.
func (n *jsonNumber) UnmarshalJSON(raw []byte) error {
	text := strings.TrimSpace(string(raw))
	if text == "null" || text == "" {
		return nil
	}
	text = strings.Trim(text, `"`)
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return err
	}
	*n = jsonNumber(v)
	return nil
}

// achievementsPath is where one account's unlocks of one game are cached.
func (c *Client) achievementsPath(steamID, appID string) string {
	return filepath.Join(c.dir, "achievements-"+sanitize(steamID)+"-"+sanitize(appID)+".json")
}

// achievementsFile is the cache's shape, with when it was written so it can
// expire.
type achievementsFile struct {
	Fetched      time.Time     `json:"fetched"`
	Achievements []Achievement `json:"achievements"`
}

// cachedAchievements reads a fresh cache, or reports that there is none.
func (c *Client) cachedAchievements(steamID, appID string) ([]Achievement, bool) {
	raw, err := os.ReadFile(c.achievementsPath(steamID, appID))
	if err != nil {
		return nil, false
	}

	var f achievementsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, false
	}
	if time.Since(f.Fetched) > achievementsCacheTTL {
		return nil, false
	}
	if len(f.Achievements) == 0 {
		return nil, false
	}
	return f.Achievements, true
}

// saveAchievements writes the cache whole, so a crash cannot leave half of it.
func (c *Client) saveAchievements(steamID, appID string, list []Achievement) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(achievementsFile{Fetched: time.Now(), Achievements: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.achievementsPath(steamID, appID), raw, 0o644)
}

// rarityPath is where one game's global percentages are cached.
func (c *Client) rarityPath(appID string) string {
	return filepath.Join(c.dir, "rarity-"+sanitize(appID)+".json")
}

// rarityFile is the cache's shape.
type rarityFile struct {
	Fetched   time.Time          `json:"fetched"`
	Percents  map[string]float64 `json:"percents"`
}

// cachedRarity reads a fresh cache, or reports that there is none.
func (c *Client) cachedRarity(appID string) (map[string]float64, bool) {
	raw, err := os.ReadFile(c.rarityPath(appID))
	if err != nil {
		return nil, false
	}

	var f rarityFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, false
	}
	if time.Since(f.Fetched) > rarityCacheTTL {
		return nil, false
	}
	if len(f.Percents) == 0 {
		return nil, false
	}
	return f.Percents, true
}

// saveRarity writes the cache whole.
func (c *Client) saveRarity(appID string, percents map[string]float64) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(rarityFile{Fetched: time.Now(), Percents: percents}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.rarityPath(appID), raw, 0o644)
}
