package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDetailRiseDoesNotBlinkBlack guards the transition from the home screen to
// a game: the backdrop belongs to the window, so the first frame of the rise
// shows the game's colour rather than the bare ground, and the page's own
// opacity only fades its content in.
func TestDetailRiseDoesNotBlinkBlack(t *testing.T) {
	a := launcher(t, installsOf(3))
	a.stat = stats{Games: map[string]*gameStats{}}

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	a.open(a.entries[0])
	a.openedAt = tt.Now()
	tt.Frame()

	img := tt.Image()
	b := img.Bounds()

	// Sample the middle of the screen, where the backdrop glows are, at the
	// very first frame of the 450ms rise.
	var sum float64
	var n int
	for y := b.Min.Y + b.Dy()/3; y < b.Min.Y+b.Dy()*2/3; y += 11 {
		for x := b.Min.X + b.Dx()/3; x < b.Min.X+b.Dx()*2/3; x += 11 {
			r, g, bl, _ := img.At(x, y).RGBA()
			sum += float64(r>>8+g>>8+bl>>8) / 3
			n++
		}
	}
	got := sum / float64(n)

	// The deep ground is #05070f, about 7/255. Anything near that means the
	// backdrop was faded out with the content, which is the black blink.
	if got < 20 {
		t.Fatalf("first frame of the rise averages %.1f/255, which is the bare ground: "+
			"the backdrop is being faded with the content", got)
	}
}
