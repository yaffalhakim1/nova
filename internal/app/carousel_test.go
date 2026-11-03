package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTrackOffsetMovesEverySelection is the regression test for the carousel
// freezing: the offset must change on every step, including past the fifth tile.
//
// The bug was translating by the distance from the first *built* tile while the
// built window followed the selection, so that distance pinned at 4 and the
// track stopped moving. Testing the offset directly is the only way to catch it:
// a pixel test still sees the tile's own scale animation and passes anyway.
func TestTrackOffsetMovesEverySelection(t *testing.T) {
	const stride float32 = 245

	prev := trackOffset(0, stride)
	for sel := 1; sel <= 20; sel++ {
		got := trackOffset(sel, stride)
		if got == prev {
			t.Fatalf("offset did not change from selection %d to %d: both %v", sel-1, sel, got)
		}
		if want := -(float32(sel) * stride); got != want {
			t.Fatalf("offset at %d = %v, want %v", sel, got, want)
		}
		prev = got
	}
}

// TestStageBuildsEveryTile checks the track is built from the whole library, so
// the offset above can name any of them.
func TestStageBuildsEveryTile(t *testing.T) {
	a := launcher(t, installsOf(12))

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	// The last game and the add tile are both painted, which they would not be
	// if only a window around the focus were built.
	if !tt.HasText("Game 11") {
		t.Fatalf("the last game is not built; texts = %q", tt.Texts())
	}
	if !tt.HasText("Add game") {
		t.Fatal("the add tile is not built")
	}
}

// TestCarouselLeavesRoomForTheGlow checks the stage keeps the design's own
// padding, so the enlarged tile's ring and glow are not cut off vertically.
func TestCarouselLeavesRoomForTheGlow(t *testing.T) {
	if stageBottom < 30 {
		t.Fatalf("stageBottom = %v, the glow needs at least 30", stageBottom)
	}
	if got := stageTop(840); got < 40 {
		t.Fatalf("stageTop at 840 = %v, want room for the glow", got)
	}
}