// Package steam reads a game's public store data: its description, its
// screenshots and its wide artwork, by Steam app id.
package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// detailsURL is the store endpoint that answers for one app.
const detailsURL = "https://store.steampowered.com/api/appdetails"

// Details is the part of a game's store page this app shows.
type Details struct {
	AppID       string   `json:"appId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Developers  []string `json:"developers"`
	Publishers  []string `json:"publishers"`
	ReleaseDate string   `json:"releaseDate"`
	Genres      []string `json:"genres"`
	HeaderImage string   `json:"headerImage"`
	Background  string   `json:"background"`
	Screenshots []string `json:"screenshots"`
	Fetched     time.Time `json:"fetched"`
}

// Client fetches and caches store data.
type Client struct {
	dir  string
	http *http.Client
}

// NewClient returns a client caching under dir.
func NewClient(dir string) *Client {
	d := filepath.Join(dir, "store")
	_ = os.MkdirAll(d, 0o755)
	return &Client{dir: d, http: &http.Client{Timeout: 25 * time.Second}}
}

// Get returns an app's store data: from cache when it has it, otherwise from
// the store. A store that does not know the app is not an error: ok is false.
func (c *Client) Get(appID string) (d *Details, ok bool) {
	if c == nil || appID == "" {
		return nil, false
	}
	if d, ok := c.fromDisk(appID); ok {
		return d, true
	}

	d, ok = c.fetch(appID)
	if !ok {
		return nil, false
	}
	_ = c.toDisk(d)
	return d, true
}

// fromDisk reads cached store data.
func (c *Client) fromDisk(appID string) (*Details, bool) {
	raw, err := os.ReadFile(c.path(appID))
	if err != nil {
		return nil, false
	}
	var d Details
	if err := json.Unmarshal(raw, &d); err != nil || d.Name == "" {
		return nil, false
	}
	return &d, true
}

// toDisk writes store data.
func (c *Client) toDisk(d *Details) error {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(d.AppID), raw, 0o644)
}

// path is where one app's store data is cached.
func (c *Client) path(appID string) string {
	return filepath.Join(c.dir, sanitize(appID)+".json")
}

// fetch reads one app from the store, in English so the description is the
// one the design's copy is written for.
func (c *Client) fetch(appID string) (*Details, bool) {
	q := url.Values{}
	q.Set("appids", appID)
	q.Set("l", "english")
	q.Set("cc", "us")

	res, err := c.http.Get(detailsURL + "?" + q.Encode())
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
	return parse(raw, appID)
}

// parse decodes the store's wrapper: {"<appid>":{"success":true,"data":{...}}}.
func parse(raw []byte, appID string) (*Details, bool) {
	var wrapper map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Name            string `json:"name"`
			ShortDesc       string `json:"short_description"`
			DetailedDesc    string `json:"detailed_description"`
			HeaderImage     string `json:"header_image"`
			BackgroundRaw   string `json:"background_raw"`
			ReleaseDate     struct{ Date string `json:"date"` } `json:"release_date"`
			Developers      []string `json:"developers"`
			Publishers      []string `json:"publishers"`
			Genres          []struct{ Description string `json:"description"` } `json:"genres"`
			Screenshots     []struct{ PathFull string `json:"path_full"` } `json:"screenshots"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, false
	}
	entry, ok := wrapper[appID]
	if !ok || !entry.Success || entry.Data.Name == "" {
		return nil, false
	}

	d := &Details{
		AppID:       appID,
		Name:        entry.Data.Name,
		Description: cleanText(entry.Data.DetailedDesc, entry.Data.ShortDesc),
		Developers:  entry.Data.Developers,
		Publishers:  entry.Data.Publishers,
		ReleaseDate: entry.Data.ReleaseDate.Date,
		HeaderImage: entry.Data.HeaderImage,
		Background:  entry.Data.BackgroundRaw,
		Fetched:     time.Now().UTC(),
	}
	for _, g := range entry.Data.Genres {
		if g.Description != "" {
			d.Genres = append(d.Genres, g.Description)
		}
	}
	for _, s := range entry.Data.Screenshots {
		if s.PathFull != "" {
			d.Screenshots = append(d.Screenshots, s.PathFull)
		}
	}
	return d, true
}

// cleanText turns the store's HTML into a paragraph: its tags dropped, its
// entities decoded and its whitespace folded.
//
// The store's `detailed_description` is NOT prose. On a real page it is a
// picture gallery interleaved with `<h1>` slogans — The Witcher 3's is two
// slogans and six images, and not one sentence of description.
//
// The `short_description` is real writing, but a store page often opens it with
// marketing instead of the game: "Ultimate Edition", "Deluxe Edition", "EARLY
// DEVELOPMENT BUILD", "Version 2.5 - Out Now", "Reviews and Accolades". Those
// openers are dropped, so the blurb starts where the writing about the game
// does.
func cleanText(detailed, short string) string {
	if isProse(short) {
		return tidyDescription(stripHeadings(short))
	}
	if isProse(detailed) {
		return tidyDescription(stripHeadings(detailed))
	}
	// Neither is a paragraph: keep whatever words there are rather than nothing.
	if text := tidyDescription(stripHeadings(short)); text != "" {
		return text
	}
	return stripHeadings(detailed)
}

// tidyDescription removes the marketing openers a store page puts before it says
// anything about the game.
//
// Real pages make this clean: The Witcher 3, Red Dead Redemption 2 and Monster
// Hunter: World all put the description after an "About the Game" marker, behind
// an edition pitch, a DLC notice or a version banner. Cutting to that marker is
// exact where the heuristics below are not.
func tidyDescription(s string) string {
	const minRest = 120

	if i := strings.Index(strings.ToLower(s), "about the game"); i > 0 {
		if rest := strings.TrimSpace(s[i+len("about the game"):]); len(rest) >= minRest {
			return rest
		}
	}

	s = trimBoilerplate(s)
	s = trimSoftOpeners(s)
	return trimBoilerplate(s)
}


// trimBoilerplate removes a marketing label from the front of a description.
//
// A store page often opens with a label rather than the game: "EARLY
// DEVELOPMENT BUILD Deadlock is still…", "MAJOR UPDATE - OUT NOW About the
// Game You are no fighter…". Those labels are written in CAPITALS, so a leading
// run of all-capital words is dropped, along with a following "About the Game".
//
// Nothing else is guessed at. An opener in sentence case ("Ultimate Edition The
// Red Dead…") is left alone: telling a label from the game's own name needs
// more than a heuristic, and cutting the wrong words loses the description.
func trimBoilerplate(s string) string {
	const minRest = 60

	for pass := 0; pass < 4; pass++ {
		changed := false

		// A leading run of capitalised label words. The run may contain short
		// joining words ("- OUT NOW"), so they are skipped while a capital
		// label still follows.
		//
		// A run of ONE capital word is never dropped: that is usually the game's
		// own name, as "PEAK is a co-op climbing game…", and cutting it loses the
		// subject of the sentence.
		words := strings.Fields(s)
		drop := 0
		for drop < len(words) {
			w := strings.Trim(words[drop], " .:–—-—*\"'!?")
			if isCapsWord(w) {
				drop++
				continue
			}
			// A joiner inside the run, as "UPDATE - OUT NOW".
			if drop > 0 && len(w) <= 2 {
				drop++
				continue
			}
			break
		}
		// The run ends on a joiner that has no label after it; give it back.
		for drop > 0 {
			w := strings.Trim(words[drop-1], " .:–—-—*\"'!?")
			if len(w) <= 2 && !isCapsWord(w) {
				drop--
				continue
			}
			break
		}
		if drop >= 2 && drop < len(words) {
			rest := strings.TrimSpace(strings.Join(words[drop:], " "))
			rest = strings.TrimLeft(rest, "-–— :,.")
			if len(rest) >= minRest {
				s, changed = rest, true
			}
		}

		// "About the Game" is a section heading, not a sentence.
		if i := strings.Index(strings.ToLower(s), "about the game"); i == 0 {
			rest := strings.TrimSpace(s[len("about the game"):])
			if len(rest) >= minRest {
				s, changed = rest, true
			}
		}

		if !changed {
			break
		}
	}
	return strings.TrimSpace(s)
}

// softOpeners are label phrases written in sentence case, which the capital-run
// rule cannot see. Each is dropped with everything up to the next sentence end,
// because they are labels rather than sentences: "Notice regarding the DLC
// lineup… Commencing in December…" is an announcement, and the game's own
// sentence follows it.
var softOpeners = []string{
	"notice regarding",
	"version ",
	"out now",
	"about the game",
	"reviews and accolades",
	"reviews & accolades",
	"game of the year -",
	"game of the year award",
	"winner of ",
	"awards - ",
}

// trimSoftOpeners drops a leading label that is not written in capitals.
//
// It cuts to the end of the label's sentence. An award line that runs straight
// into the description with no full stop ("…Best Action Game of 2019 - IGN Carve
// your own clever path…") is left as it is: telling that boundary from the game's
// own words needs more than a rule, and cutting the wrong place loses the
// description entirely.
func trimSoftOpeners(s string) string {
	const minRest = 60

	for pass := 0; pass < 3; pass++ {
		lower := strings.ToLower(s)
		matched := false
		for _, phrase := range softOpeners {
			if !strings.HasPrefix(lower, phrase) {
				continue
			}
			rest := s
			if i := strings.IndexAny(s, ".!?"); i >= 0 && i < len(s)-1 {
				rest = strings.TrimSpace(s[i+1:])
			} else if i := strings.Index(lower, " about the game"); i > 0 {
				rest = strings.TrimSpace(s[i+len(" about the game"):])
			}
			if len(rest) >= minRest && rest != s {
				s, matched = rest, true
			}
			break
		}
		if !matched {
			break
		}
	}
	return strings.TrimSpace(s)
}

// isCapsWord reports whether a word is written in capitals, which is how these
// labels are set: "EARLY", "DLC", "OUT". A single letter is not, so "I" or "A"
// starting a real sentence is left in place.
func isCapsWord(w string) bool {
	if len(w) < 2 {
		return false
	}
	for _, r := range w {
		if r >= 'a' && r <= 'z' {
			return false
		}
	}
	return true
}

// isProse reports whether text reads as a sentence or two rather than a caption:
// long enough to be a description, and made of more than one word.
func isProse(s string) bool {
	t := stripHeadings(s)
	return len(t) >= 40 && strings.Count(t, " ") >= 8
}

// headingTags are the blocks the store uses for slogans and section titles. A
// heading's text is dropped with it: "<h1>About the Game</h1>" belongs in a
// layout, not in a paragraph.
var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

// stripHeadings removes heading elements and their contents, then strips what is
// left as ordinary markup.
//
// A heading's text is dropped with it: "<h1>About the Game</h1>" belongs in a
// layout, not in a paragraph. Anything else is left for stripTags.
func stripHeadings(s string) string {
	var b strings.Builder
	// inside is how deeply nested the reader is within a heading whose text must
	// be dropped.
	inside := 0

	for i := 0; i < len(s); {
		if s[i] != '<' {
			if inside == 0 {
				b.WriteByte(s[i])
			}
			i++
			continue
		}

		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			break
		}
		raw := strings.ToLower(strings.TrimSpace(s[i+1 : i+end]))
		closing := strings.HasPrefix(raw, "/")
		name := strings.TrimPrefix(raw, "/")
		if f := strings.IndexAny(name, " \t\r\n"); f >= 0 {
			name = name[:f]
		}
		name = strings.TrimSuffix(name, "/")

		switch {
		case headingTags[name] && !closing:
			inside++
		case headingTags[name] && closing && inside > 0:
			inside--
		case inside == 0:
			b.WriteString(s[i : i+end+1])
		}
		i += end + 1
	}

	return stripTags(b.String())
}

// stripTags removes HTML tags and folds whitespace. A closing tag becomes a
// space, so two paragraphs never run together.
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				b.WriteByte(' ')
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(decodeEntities(b.String())), " ")
}

// decodeEntities handles the handful of entities a store description uses.
func decodeEntities(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
	)
	return r.Replace(s)
}

// sanitize keeps an id safe as a file name.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return fmt.Sprintf("app-%d", time.Now().UnixNano())
	}
	return b.String()
}
