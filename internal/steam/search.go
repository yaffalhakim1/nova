package steam

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// searchURL finds an app by name, for the games whose id the catalogue does
// not carry: a repack installed from a folder.
const searchURL = "https://store.steampowered.com/api/storesearch/"

// Search resolves a game's name to a Steam app id. It returns the app whose
// name matches best, and false when the store knows nothing close.
func (c *Client) Search(name string) (string, bool) {
	if c == nil || strings.TrimSpace(name) == "" {
		return "", false
	}

	q := url.Values{}
	q.Set("term", name)
	q.Set("cc", "us")
	q.Set("l", "english")

	res, err := c.http.Get(searchURL + "?" + q.Encode())
	if err != nil {
		return "", false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", false
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", false
	}
	return pickApp(raw, name)
}

// pickApp takes the best app out of a store search.
func pickApp(raw []byte, want string) (string, bool) {
	var out struct {
		Items []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			ID   int    `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false
	}

	wantKey := key(want)
	var fallback string
	var fallbackName string
	for _, it := range out.Items {
		if it.Type != "app" || it.ID == 0 {
			continue
		}
		id := itoa(it.ID)
		if fallback == "" {
			fallback, fallbackName = id, it.Name
		}
		if key(it.Name) == wantKey {
			return id, true
		}
	}

	// Nothing spelled the same: accept the first hit when its name contains
	// the wanted one, so "Control" finds "CONTROL Ultimate Edition" but not
	// "Port Control".
	if fallback != "" {
		got := key(fallbackName)
		if len(wantKey) >= 5 && strings.Contains(got, wantKey) {
			return fallback, true
		}
	}
	return "", false
}

// key folds a game's name for comparison.
func key(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// itoa formats a positive id.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
