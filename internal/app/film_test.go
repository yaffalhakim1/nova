package app

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// The clip's size and pace: the same 1440x900 the README screenshots use, at
// the 30 fps every browser plays smoothly.
const (
	filmWidth  = 1440
	filmHeight = 900
	filmRate   = 30
)

// filmClock is how long one captured frame advances the app by.
var filmClock = time.Second / filmRate

// TestFilmTour renders a short tour of the real launcher and encodes it as the
// landing page's clip. It is a capture tool, not a test: without NOVA_FILM it
// skips, so `go test ./...` never spends minutes encoding video.
//
// The frames come from the app itself, through ui.NewTester: the carousel
// really slides, the page really rises, and the trophies dialog really opens.
// Nothing here is a mock-up, and nothing is drawn by hand.
//
// The achievements need a Steam Web API key and account, as the app does:
//
//	$env:STEAM_API_KEY="..."; $env:STEAM_ID="..."
//	$env:NOVA_FILM="$env:TEMP\tour.mp4"
//	go test ./internal/app -run TestFilmTour -v -timeout 900s
func TestFilmTour(t *testing.T) {
	out := os.Getenv("NOVA_FILM")
	if out == "" {
		t.Skip("set NOVA_FILM=<file.mp4> to capture the tour")
	}

	a := filmLauncher(t)

	tt := ui.NewTester(a.view, filmWidth, filmHeight)
	tt.Frame()

	// The drift's own clock starts where the Tester's does, so a capture always
	// opens on the same particles.
	a.epoch = tt.Now()

	filmTour(t, a, tt, filmStart(t, out))
}

// filmStart opens the encoder and returns a function that writes one frame.
func filmStart(t *testing.T, out string) func(*image.RGBA) {
	t.Helper()

	cmd := exec.Command("ffmpeg",
		"-y",
		"-loglevel", "error",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", filmWidth, filmHeight),
		"-r", fmt.Sprint(filmRate),
		"-i", "-",
		"-an",
		"-c:v", "libx264",
		"-preset", "slow",
		"-crf", "21",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		out,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("ffmpeg stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start ffmpeg: %v", err)
	}

	frames := 0
	t.Cleanup(func() {
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("ffmpeg: %v", err)
		}
		if frames == 0 {
			t.Error("the clip has no frames")
		}
	})

	return func(f *image.RGBA) {
		// The whole frame buffer is written as it is: a 1440x900 RGBA frame has
		// no padding, because the tester's own stride matches its width.
		if _, err := stdin.Write(f.Pix); err != nil {
			t.Fatalf("write frame %d: %v", frames, err)
		}
		frames++
	}
}

// filmLauncher builds the launcher over this machine's real data, as Run does,
// so the clip shows the actual library, covers and achievements.
func filmLauncher(t *testing.T) *app {
	t.Helper()

	a := &app{
		favorites: map[string]bool{},
		expanded:  map[string]bool{},
		lastKey:   -1,
		epoch:     time.Now(),
	}
	if err := a.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// The capture runs as a test binary, whose app identifier differs from the
	// released one, so the key stored in the credential store is not visible
	// here. Take it from the environment instead, as the caller sets it:
	//
	//	$env:STEAM_API_KEY="..."; $env:STEAM_ID="..."
	//
	// Nothing is written to disk, and the key never reaches the source tree.
	if key := strings.TrimSpace(os.Getenv("STEAM_API_KEY")); key != "" {
		a.steamKey, a.keyStored = key, true
	}
	if id := strings.TrimSpace(os.Getenv("STEAM_ID")); id != "" {
		// The SteamID64 alone is enough: the achievement and owned-game calls
		// take it, and the folder id only matters for reading local play time.
		a.steam = &steamstore.Account{SteamID: id}
	}

	// `load` read the key before these were set, so the owned library it began
	// was a no-op. Ask again now that there is a key to ask with.
	if a.steamKey != "" {
		a.loadOwnedGames()
	}

	// A nil window runs every update inline, which is what a capture wants: the
	// state is settled before the next frame is asked for.
	waitFor(t, 60*time.Second, "the library scan", func() bool { return !a.scanning })
	waitFor(t, 20*time.Second, "the owned library", func() bool { return len(a.owned) > 0 })

	registerFonts()
	return a
}

// filmTour is the clip itself: a few real inputs, with the frames between them.
func filmTour(t *testing.T, a *app, tt *ui.Tester, write func(*image.RGBA)) {
	t.Helper()

	// The clip starts on the game before its subject, so the first thing it shows
	// is the carousel moving onto it.
	subject := filmSubject(t, a)
	a.lastKey = -1
	a.refresh()

	idx := -1
	for i, e := range a.entries {
		if e.slug() == subject {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("%s is not in the settled entry list", subject)
	}

	// The arrow that lands on the subject: from the left when there is a game
	// before it, and from the right when it is first.
	step := ui.KeyRight
	switch {
	case idx > 0:
		a.selected = idx - 1
	default:
		a.selected = idx + 1
		step = ui.KeyLeft
	}
	tt.Frame()

	// Wait for the subject's own art before recording, so the clip never opens
	// on the generated stand-in that fills in a moment later.
	waitFor(t, 30*time.Second, "the subject's cover and store page", func() bool {
		e, ok := a.entryBySlug(subject)
		return ok && a.coverOf(e) != nil && a.storeFor(e) != nil
	})

	hold := func(d time.Duration) {
		for elapsed := time.Duration(0); elapsed < d; elapsed += filmClock {
			tt.Advance(filmClock)
			write(tt.Image())
		}
	}

	hold(1400 * time.Millisecond)

	// The carousel slides one game along, onto the subject.
	tt.Key(0, step)
	hold(1200 * time.Millisecond)

	if e := a.focused(); e == nil || e.slug() != subject {
		t.Fatalf("the carousel did not land on %s", subject)
	}

	// The subject's page: full-bleed art, rising into place.
	tt.Key(0, ui.KeyDown)
	// The page's entrance is timed from the clock the frame is painted with.
	// `open` stamps it with the wall clock, which a capture has run past while
	// it waited for the artwork; pinning it to the tester's clock is what the
	// two being the same clock does in the running app, and it is what makes the
	// rise visible here instead of already finished before the first frame.
	if a.current != nil {
		a.openedAt = tt.Now()
	}
	hold(1900 * time.Millisecond)

	open := a.current
	if open == nil {
		t.Fatal("the clip never opened a game")
	}

	// The trophies card is the way into the full list, so the data behind it has
	// to be there before the click.
	waitFor(t, 20*time.Second, "the achievements", func() bool {
		g := a.trophiesLoaded(open.slug())
		return g != nil && len(g.list) > 0
	})
	hold(600 * time.Millisecond)

	if err := tt.Click("Trophies"); err != nil {
		t.Fatalf("the trophies card is not clickable: %v", err)
	}
	hold(1300 * time.Millisecond)

	// Scroll the list, so the clip shows more than its first rows.
	tt.Wheel(float32(filmWidth)/2, float32(filmHeight)*0.72, 0, 260)
	hold(1100 * time.Millisecond)

	// Leave the way a visitor would: Done, then back out to the games. The clip
	// ends where it began, so playing it on a loop does not cut mid-dialog.
	if err := tt.Click("Done"); err != nil {
		t.Fatalf("the Done button is not clickable: %v", err)
	}
	hold(700 * time.Millisecond)

	tt.Key(0, ui.KeyEscape)
	hold(1100 * time.Millisecond)
}

// filmSubject is the slug of the game the clip opens.
//
// It prefers a game installed on this machine: its Play button is live, and a
// clip whose only action is greyed out reads as broken. Among those it wants one
// partly finished, because a full set draws a 100% ring and an empty one draws
// nothing. The last tiebreak is the longest list, so the dialog has something to
// scroll.
//
// `trophiesFor` starts the load and answers nil on the frame it starts it, so
// this asks for every installed game, waits for the answers, then chooses. A
// slug, not an index: the list re-sorts once the owned library arrives, and an
// index kept across that sort named the wrong game.
func filmSubject(t *testing.T, a *app) string {
	t.Helper()

	if a.steamKey == "" || a.steamAccountID() == "" {
		t.Fatal("the clip needs STEAM_API_KEY and STEAM_ID in the environment: " +
			"the achievements are real, so there is nothing to film without them")
	}

	var candidates []entry
	for _, e := range a.entries {
		if e.inst.AppID != "" {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		t.Fatal("nothing in the library has a Steam app id to look achievements up by")
	}

	// Ask for every game's achievements once, then wait: the loads run in the
	// background, and a game that is still loading reads as no data.
	for _, e := range candidates {
		a.trophiesFor(e)
	}
	waitFor(t, 60*time.Second, "the achievements of every installed game", func() bool {
		for _, e := range candidates {
			if a.trophiesLoaded(e.slug()) == nil {
				return false
			}
		}
		return true
	})

	// Choose by these passes, in order: a game on disk that is partly finished
	// is the ideal clip, because its Play button is live and its ring shows an
	// arc. A game on disk that is finished is second, since a live button beats
	// an arc. A game Steam owns but this machine does not have is last, and only
	// if nothing is installed can be filmed.
	passes := []struct {
		installed bool
		partial   bool
		what      string
	}{
		{true, true, "installed and partly finished"},
		{true, false, "installed"},
		{false, true, "owned, not installed, and partly finished"},
	}

	for _, pass := range passes {
		best, bestCount := "", -1
		for _, e := range candidates {
			if e.installed() != pass.installed {
				continue
			}
			g := a.trophiesLoaded(e.slug())
			if g == nil || len(g.list) < 10 {
				continue
			}
			if pass.partial {
				if p := g.percent(); p < 15 || p > 85 {
					continue
				}
			}
			if len(g.list) > bestCount {
				best, bestCount = e.slug(), len(g.list)
			}
		}
		if best == "" {
			continue
		}

		subject, _ := a.entryBySlug(best)
		g := a.trophiesLoaded(best)
		t.Logf("the clip opens %s (%s), %s: %d of %d achievements, %d%%",
			subject.name(), subject.inst.AppID, pass.what, g.earned(), len(g.list), g.percent())
		return best
	}

	t.Fatal("no game has enough achievements to film the trophy list with")
	return ""
}


// waitFor blocks until cond holds, bounded by d, and fails the test with what
// was being waited for when it does not.
func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("waited %v for %s in vain", d, what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
