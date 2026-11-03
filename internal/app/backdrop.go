package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// backdropFade is how long the screen takes to change from one game's colours
// to another's, as the design's 1s opacity transition does.
const backdropFade = time.Second

// backdrop is the colour behind every screen. It cross-fades the way the
// design's two stacked layers do: the outgoing game's glows fade out while the
// incoming game's fade in, so the two mix instead of one replacing the other.
type backdrop struct {
	// key identifies the game whose colours are showing, so a frame that has
	// not changed game does not restart the fade.
	key string
	art art
	// accent is the colour sampled from the cover, or white before it loads.
	accent ui.Color
	// center is where the glow sits, as fractions of the screen: the selected
	// capsule's own middle.
	center ui.Point
	// previous is the colour being faded out, kept only during a fade.
	previous *backdropState
	// started is when the fade began.
	started time.Time
}

// backdropState is one game's colours, kept while they fade out.
type backdropState struct {
	art    art
	accent ui.Color
	// center is where the glow sits, as fractions of the screen: the selected
	// capsule's own middle, so the colour reads as coming from the game.
	center ui.Point
}

// set chooses the colours for a game, starting a cross-fade when they differ.
func (b *backdrop) set(c *ui.Context, key string, a art, accent ui.Color, center ui.Point) {
	if b.key == key && b.accent == accent && b.center == center {
		return
	}

	if b.key != "" {
		// The layer that was showing is now the one fading out, from where its
		// glow sat, so a change mid-fade does not jump across the screen.
		b.previous = &backdropState{art: b.art, accent: b.accent, center: b.center}
	}
	b.key = key
	b.art = a
	b.accent = accent
	b.center = center
	b.started = c.Now()
}

// progress is how far the fade has gone, 0 to 1.
func (b *backdrop) progress(c *ui.Context) float32 {
	if b.started.IsZero() {
		return 1
	}
	elapsed := c.Now().Sub(b.started)
	if elapsed < 0 {
		return 1
	}
	if elapsed >= backdropFade {
		b.previous = nil
		return 1
	}
	return clamp01(float32(elapsed) / float32(backdropFade))
}

// paint draws the ground, then both games' glows: the old at 1-t, the new at t.
func (b *backdrop) paint(p *ui.Painter, r ui.Rect, c *ui.Context) {
	t := b.progress(c)

	p.Fill(r, deep, 0)

	if b.previous != nil {
		paintBackdropAt(p, r, b.previous.art, b.previous.accent, b.previous.center, 1-t)
	}
	paintBackdropAt(p, r, b.art, b.accent, b.center, t)

	paintVignette(p, r)

	if t < 1 {
		p.AnimationFrame()
	}
}
