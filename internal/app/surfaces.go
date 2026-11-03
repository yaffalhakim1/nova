package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The motion of the interface, taken from the design's own numbers.
var (
	// hoverFade is how fast a surface takes its hover colour.
	hoverFade = ui.ElementTransition{
		Colors:   true,
		Duration: 250 * time.Millisecond,
	}
	// pressFade is the shorter one of a press, which should feel immediate.
	pressFade = ui.ElementTransition{
		Colors:   true,
		Duration: 120 * time.Millisecond,
	}
)

// glassSurface gives an element the design's glass look, and the hover and
// press states it lists: a lighter fill under the pointer, lighter still while
// held, both fading rather than snapping.
//
// The design writes this as a CSS rule with a transition; here it is the state
// the view reads, which is what MyGo has instead of selectors.
func glassSurface(e ui.Element) ui.Element {
	n := e.
		Background(glass).
		Border(1, glassLine).
		Transition(hoverFade)

	if n.Pressed() {
		n.Background(ui.RGBA(255, 255, 255, 0.20))
	} else if n.Hovered() {
		n.Background(ui.RGBA(255, 255, 255, 0.14))
	}
	return n
}

// raisedSurface is the panels' look: a translucent fill with a hairline, which
// lifts under the pointer as the design's cards do.
func raisedSurface(e ui.Element) ui.Element {
	n := e.
		Background(panelFill).
		Border(1, glassLine).
		Transition(hoverFade)

	if n.Hovered() {
		n.Background(ui.RGBA(16, 21, 40, 0.62))
	}
	return n
}
