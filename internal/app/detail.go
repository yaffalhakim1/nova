package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
)

// overlayRise is how long the detail page takes to rise into place, and the
// curve it goes by: the design's `rise .45s cubic-bezier(.22,1,.36,1)`.
const overlayRise = 450 * time.Millisecond

// riseEase is that curve, applied by hand because the page's transform is set
// from elapsed time rather than by an ElementTransition.
var riseEase = ui.CubicBezier(0.22, 1, 0.36, 1)

// open selects a game and shows its page.
func (a *app) open(e entry) {
	a.current = &e
	a.openedAt = time.Now()
}

// detailPage is the open game's hub: its own artwork filling the window, and
// over it the title, the actions and the cards of `hub.go`.
func (a *app) detailPage(c *ui.Context) {
	e := *a.current
	// The glow follows the artwork: the page is full-bleed, so its colour is
	// the game's own rather than a corner of the screen.
	a.bg.set(c, e.slug(), coverArt(e), a.accentFor(e), ui.Point{X: 0.18, Y: 0.3})

	// The design opens the page with `rise`: it fades in and settles from 3%
	// larger, over 450ms on cubic-bezier(.22,1,.36,1). The elapsed time drives
	// it, so the page is not built at a stale scale when a frame is slow.
	t := float32(1)
	if !a.openedAt.IsZero() {
		t = clamp01(float32(c.Now().Sub(a.openedAt)) / float32(overlayRise))
		if t < 1 {
			c.AnimationFrame()
		}
	}

	// Only the page's content rises: the backdrop is the window's own and is
	// painted under both screens, so the transition never fades to bare
	// ground. The scale and the opacity are the design's `rise`.
	scale, opacity := riseAt(t)

	ui.Column(c).Fill().
		Scale(scale, scale).
		Opacity(opacity).
		Draw(func(p *ui.Painter, r ui.Rect) { a.paintHubArt(p, r, e) }).
		Children(func() {
			a.hub(c, e)
		})
}

// riseAt is the page's entrance at t through its 450ms: it starts 3% larger
// than its final size and invisible, and settles to both, as the design's
// `@keyframes rise` does.
func riseAt(t float32) (scale, opacity float32) {
	eased := riseEase(clamp01(t))
	return 1.03 + (1.0-1.03)*eased, eased
}

// playButton is the screen's one action: it starts the game.
func (a *app) playButton(c *ui.Context, e entry) {
	if !canLaunch(e) {
		primaryButton(c, "Play", nil).Opacity(0.5)
		return
	}
	a.playButtonStyled(c, "Play", func() { a.play(e) })
}

// forget hides a game from the library without touching the folder on disk.
// The exclusion is kept in the library config, so it stays gone across runs.
func (a *app) forget(e entry) {
	slug := e.slug()

	a.mu.Lock()
	if a.hidden == nil {
		a.hidden = map[string]bool{}
	}
	a.hidden[slug] = true
	keys := make([]string, 0, len(a.hidden))
	for k := range a.hidden {
		keys = append(keys, k)
	}
	a.mu.Unlock()

	a.current = nil
	a.lastKey = -1
	a.refresh()
	a.say(e.name(), "removed from your library")

	// Saved now, not in the background: the exclusion is one small file and
	// losing it would show the game again on the next run.
	cfg := a.libCfg
	cfg.Hidden = keys
	if err := library.SaveConfig(a.dataDir, cfg); err != nil {
		a.status = "could not save the library: " + err.Error()
	}
	a.libCfg = cfg
}

// genreLine is a game's genres: what the user typed when they added it by hand,
// then the store page's own.
func (a *app) genreLine(e entry) string {
	if e.inst.Genre != "" {
		return e.inst.Genre
	}
	if d := a.storeFor(e); d != nil && len(d.Genres) > 0 {
		return strings.Join(d.Genres, ", ")
	}
	return ""
}

// lastPlayedLine is the meta line's entry for when the game was last played:
// the same words the recent cards show, so the two never disagree. A game with
// no history says so once, rather than "Last played Never played".
//
// It reads lastOf rather than the local history alone, so a game the Steam
// account played elsewhere is dated here too.
func (a *app) lastPlayedLine(e entry) string {
	t := a.lastOf(e)
	if t.IsZero() {
		return "Never played"
	}
	return "Last played " + lastPlayed(t)
}

// describe is the game's blurb: what the user typed when they added the game by
// hand, then the store's own description, and otherwise what this app knows
// about the install.
func (a *app) describe(e entry) string {
	if e.inst.Description != "" {
		return e.inst.Description
	}

	if d := a.storeFor(e); d != nil && d.Description != "" {
		return blurb(d.Description)
	}

	if e.inst.Source == library.Manual {
		return "Added by hand. There is no store page for it, so the launcher knows only what is on disk."
	}

	if e.inst.Source == library.Steam {
		return "Installed through Steam. Its store page has not been reached yet."
	}
	return "Found in a games folder. Play starts it; there is no store page for it."
}

// blurb keeps the opening of a store description, cut at a sentence, so the
// About card reads as a paragraph rather than the whole store page.
func blurb(text string) string {
	const limit = 380
	if len(text) <= limit {
		return text
	}

	cut := text[:limit]
	if i := strings.LastIndexAny(cut, ".!?"); i > limit/2 {
		return cut[:i+1]
	}
	if i := strings.LastIndex(cut, " "); i > 0 {
		return cut[:i] + "…"
	}
	return cut + "…"
}
