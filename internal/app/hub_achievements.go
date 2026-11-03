package app

import (
	"context"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// The Trophies card reads an account's real Steam achievements, which is the one
// thing on this page the launcher did not have before. Everything it shows is
// either Steam's own answer or said plainly as missing.

// trophyRingSize is the design's `.hcard .ring`: 64 DIPs inside the card.
const trophyRingSize float32 = 64

// trophyRowIcon is the cup square of an achievement row, smaller than the
// design's 46 because two rows share what the ring leaves of a 224 card.
const trophyRowIcon float32 = 30

// hubCardPadding is the design's `.hcard` horizontal padding, 22 on both sides.
// `hubCard` itself uses it; it is named here so the trophies card can measure how
// much room a row really has.
const hubCardPadding float32 = 22

// trophyDateMinWidth is the narrowest row that still carries the date: below it
// the card drops the column and gives the room to the name, as the design's own
// `.tro` media query does under 420 DIPs.
const trophyDateMinWidth float32 = 246

// trophyBoard is the achievements loaded so far, by the game's slug: the card is
// built every frame and must never wait for the network, so a game's data is
// cached here once its background load answers.
type trophyBoard struct {
	// games holds an entry for every game whose load has finished, including one
	// Steam has no achievements for: an empty list is an answer, and asking
	// again on every frame would hammer the API.
	games map[string]*gameTrophies
	// started marks the games whose load was begun, so a game is asked for once.
	started map[string]bool
}

// gameTrophies is one game's achievements, with how rare each is.
type gameTrophies struct {
	list []steamstore.Achievement
	// rarity is how many players hold each achievement, by its API name, when
	// Steam answered.
	rarity    map[string]float64
	hasRarity bool
}

// earned is how many of the game's achievements this account has.
func (g *gameTrophies) earned() int {
	if g == nil {
		return 0
	}
	n := 0
	for _, a := range g.list {
		if a.Achieved {
			n++
		}
	}
	return n
}

// percent is the share of the game's achievements earned, 0 to 100.
func (g *gameTrophies) percent() int {
	if g == nil || len(g.list) == 0 {
		return 0
	}
	return (g.earned()*100 + len(g.list)/2) / len(g.list)
}

// recent is the achievements this account earned, newest first, which is what
// the card lists under the ring.
func (g *gameTrophies) recent() []steamstore.Achievement {
	if g == nil {
		return nil
	}
	var out []steamstore.Achievement
	for _, a := range g.list {
		if a.Achieved {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Unlocked.After(out[j].Unlocked) })
	return out
}

// latestRows are the newest achievements, at most n, for the rows under the
// ring.
func (g *gameTrophies) latestRows(n int) []steamstore.Achievement {
	earned := g.recent()
	if len(earned) > n {
		return earned[:n]
	}
	return earned
}

// trophiesFor is a game's achievements as far as they have loaded, starting the
// background load the first time the card asks. It never blocks: the frame that
// starts a load draws the honest empty state, and the data arrives on a later
// one.
//
// The board is read under the app's mutex because a load finishes on its own
// goroutine between frames, as `storeData` and `appIDs` are read.
func (a *app) trophiesFor(e entry) *gameTrophies {
	slug := e.slug()

	if g := a.trophiesLoaded(slug); g != nil {
		return g
	}

	a.loadTrophies(e)
	return a.trophiesLoaded(slug)
}

// loadTrophies fetches a game's achievements and their rarity in the background,
// as the owned library does: without a key, an account or an app id it does
// nothing at all, and the card keeps its empty state.
func (a *app) loadTrophies(e entry) {
	slug := e.slug()

	// A repack has no manifest id, so the id Steam was asked for by name is the
	// one to use. Until it is known, nothing is started, and a later frame that
	// has it begins the load.
	id := e.inst.AppID
	if id == "" {
		id = a.appIDString(e)
	}
	key := a.steamKey
	steamID := a.steamAccountID()
	if id == "" || key == "" || steamID == "" {
		return
	}

	if !a.markTrophiesStarted(slug) {
		return
	}

	go func() {
		list, ok := a.store.Achievements(context.Background(), key, steamID, id)
		if !ok {
			a.update(func() {
				// Recorded as answered so the card does not retry every frame;
				// an empty game reads as no data, which is what the card says.
				a.putTrophies(slug, nil, nil, false)
			})
			return
		}
		rarity, hasRarity := a.store.Rarity(id)
		a.update(func() { a.putTrophies(slug, list, rarity, hasRarity) })
	}()
}

// putTrophies records a finished load. It runs on the window's own goroutine,
// through update, under the app's mutex, so the next frame's read sees it whole
// rather than a half-written map.
func (a *app) putTrophies(slug string, list []steamstore.Achievement, rarity map[string]float64, hasRarity bool) {
	a.setTrophies(slug, &gameTrophies{list: list, rarity: rarity, hasRarity: hasRarity})
}

// setTrophies records one game's answer under the mutex, which is the only way
// the board is written.
func (a *app) setTrophies(slug string, g *gameTrophies) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.achievements.games == nil {
		a.achievements.games = map[string]*gameTrophies{}
	}
	a.achievements.games[slug] = g
}

// markTrophiesStarted records that a game's load was begun, and reports whether
// it had not been: the caller starts the load only when it gets true.
func (a *app) markTrophiesStarted(slug string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.achievements.started == nil {
		a.achievements.started = map[string]bool{}
	}
	if a.achievements.started[slug] {
		return false
	}
	a.achievements.started[slug] = true
	return true
}

// trophiesStarted reports whether a game's load was begun.
func (a *app) trophiesStarted(slug string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.achievements.started[slug]
}

// trophiesLoaded is a game's answer as the board holds it, under the mutex, or
// nil when its load has not finished.
func (a *app) trophiesLoaded(slug string) *gameTrophies {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.achievements.games[slug]
}

// trophyTier is how rare an achievement is, in the four steps the design draws.
//
// Steam has no bronze, silver, gold or platinum: that is a PlayStation concept,
// and inventing it would be a lie. The tier here is derived from the real
// percentage of players who hold the achievement, so "platinum" means "fewer
// than 5% of players have it", not "Steam calls it platinum".
type trophyTier int

const (
	tierBronze trophyTier = iota
	tierSilver
	tierGold
	tierPlatinum
)

// tiersInOrder is the design's order, rarest first.
var tiersInOrder = []trophyTier{tierPlatinum, tierGold, tierSilver, tierBronze}

// tierOf maps a percentage of players holding an achievement to its tier.
func tierOf(percent float64) trophyTier {
	switch {
	case percent < 5:
		return tierPlatinum
	case percent < 15:
		return tierGold
	case percent < 50:
		return tierSilver
	default:
		return tierBronze
	}
}

// name is the tier's word, as the design's `TIER[k].n` spells it.
func (t trophyTier) name() string {
	switch t {
	case tierPlatinum:
		return "Platinum"
	case tierGold:
		return "Gold"
	case tierSilver:
		return "Silver"
	default:
		return "Bronze"
	}
}

// color is the tier's cup, as the design's `TIER[k].c` sets it.
func (t trophyTier) color() ui.Color {
	switch t {
	case tierPlatinum:
		return ui.Hex("#9fd8ff")
	case tierGold:
		return ui.Hex("#ffd36a")
	case tierSilver:
		return ui.Hex("#d3dbe8")
	default:
		return ui.Hex("#d99468")
	}
}

// tierOfAchievement is one achievement's tier, and whether it is known: an
// achievement Steam published no percentage for has none, and is not counted
// under a tier it does not belong to.
func (g *gameTrophies) tierOfAchievement(a steamstore.Achievement) (trophyTier, bool) {
	if g == nil || !g.hasRarity {
		return tierBronze, false
	}
	percent, ok := g.rarity[a.APIName]
	if !ok {
		return tierBronze, false
	}
	return tierOf(percent), true
}

// tierCounts is how many earned achievements each tier holds, and whether every
// one of them has a known tier. When Steam gave no percentages at all the counts
// are not shown, rather than shown as four zeroes.
func (g *gameTrophies) tierCounts() (map[trophyTier]int, bool) {
	counts := map[trophyTier]int{}
	if g == nil || !g.hasRarity {
		return counts, false
	}

	known := 0
	for _, a := range g.list {
		if !a.Achieved {
			continue
		}
		t, ok := g.tierOfAchievement(a)
		if !ok {
			continue
		}
		counts[t]++
		known++
	}
	return counts, known > 0 || g.earned() == 0
}

// cupIcon is the design's trophy cup, as an outline that takes the colour of the
// element it is drawn on.
var cupIcon = parseCupIcon()

// parseCupIcon parses the cup once. A parse failure leaves a nil icon, which the
// card draws as a plain gap rather than taking the page down.
func parseCupIcon() *ui.SVG {
	const body = `<svg viewBox="0 0 24 24">` +
		`<path d="M7 3h10v5a5 5 0 01-10 0z" fill="currentColor"/>` +
		`<path d="M7 5H4v1.5A3.5 3.5 0 007.6 10M17 5h3v1.5A3.5 3.5 0 0116.4 10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>` +
		`<path d="M12 13v4M9.5 17h5v4h-5z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>` +
		`</svg>`

	svg, err := ui.ParseSVG([]byte(body))
	if err != nil {
		return nil
	}
	return svg
}

// cup draws the trophy cup at a size in a colour.
func cup(c *ui.Context, size float32, color ui.Color) ui.Element {
	if cupIcon == nil {
		return ui.Box(c).Size(size, size)
	}
	return ui.Icon(c, cupIcon).Size(size, size).TextColor(color)
}

// ringPath is a circle that starts at twelve o'clock and runs clockwise, so
// trimming it draws the design's arc from the top as its `stroke-dashoffset`
// does.
func ringPath(cx, cy, r float32) *ui.Path {
	const k = 0.5522847498
	return (&ui.Path{}).
		MoveTo(cx, cy-r).
		CubeTo(cx+r*k, cy-r, cx+r, cy-r*k, cx+r, cy).
		CubeTo(cx+r, cy+r*k, cx+r*k, cy+r, cx, cy+r).
		CubeTo(cx-r*k, cy+r, cx-r, cy+r*k, cx-r, cy).
		CubeTo(cx-r, cy-r*k, cx-r*k, cy-r, cx, cy-r)
}

// trophyRing is the design's `.ring`: a 64 DIP ring with the earned share
// stroked from the top, and the percentage centred in it as a real text element
// so the tester and a screen reader both see it.
func trophyRing(c *ui.Context, g *gameTrophies) ui.Element {
	percent := g.percent()

	return ui.Box(c).
		Size(trophyRingSize, trophyRingSize).
		Shrink(0).
		Label("Achievements " + itoa(percent) + "%").
		Draw(func(p *ui.Painter, r ui.Rect) {
			paintTrophyRing(p, r, percent, ink)
		}).
		Center().
		Children(func() {
			ui.Text(c, itoa(percent)+"%").
				FontSize(17).
				FontWeight(300).
				TextColor(ink).
				FontFeatures("tnum")
		})
}

// paintTrophyRing draws the ring and its percentage inside the box it is given.
// The design's `.ring` is 64 across with a 5 wide stroke at 76; at 64 that is a
// radius of 27 and a stroke of 4.5.
func paintTrophyRing(p *ui.Painter, r ui.Rect, percent int, color ui.Color) {
	const stroke float32 = 4.5

	cx, cy := r.X+r.W/2, r.Y+r.H/2
	radius := min(r.W, r.H)/2 - stroke/2
	if radius <= 0 {
		return
	}

	// The full circle first, so the part not yet earned still reads as a ring.
	if path := ringPath(cx, cy, radius); path != nil {
		p.StrokePath(path, stroke, ui.RGBA(255, 255, 255, 0.12))
	}

	fraction := clamp01(float32(percent) / 100)
	if fraction <= 0 {
		return
	}
	arc := ringPath(cx, cy, radius).Trim(0, fraction)
	p.StrokePath(arc, stroke, color)
}

// dateOf is when an achievement was earned, as the design's
// `toLocaleDateString` writes it: "Mar 12, 2025".
func dateOf(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}
