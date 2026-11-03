package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// gutter is the screen's side margin.
const gutter = 24

// portraiture is a tile's height over its width. Steam's artwork is 600x900,
// so a 2:3 tile shows the whole cover with nothing cropped.
const portraiture = 1.5

// stageTop is the room between the filter bar and the carousel. The design
// sets margin-top: clamp(18px, 6vh, 64px) and then its own padding-block: 56px,
// so the gap a reader sees is both of them together.
func stageTop(h float32) float32 { return clamp(18, h*0.06, 64) + 24 }

// stageBottom is the room under the carousel, the design's padding-block end.
const stageBottom float32 = 44

// capsuleCenter is where the selected capsule sits, as fractions of the
// screen. The stage anchors its track a fixed distance from the left and
// centres the row, so the focused tile's middle is a fixed point of that
// layout: the backdrop's glow sits on the game's own artwork.
func capsuleCenter(c *ui.Context) ui.Point {
	w, h := c.Size()

	tile := clamp(120, w*0.15, 196)
	anchor := clamp(gutter, w*0.06, 96)

	// The top bar, the stage's own top padding, then half the square tile. The
	// enlargement is a transform, so it does not move this centre.
	x := anchor + tile/2
	y := 28 + stageTop(h) + tile/2
	return ui.Point{X: clamp01(x / w), Y: clamp01(y / h)}
}

// homePage is the console home screen: the bar, the carousel of games, the one
// in focus, and what lies below the fold.
func (a *app) homePage(c *ui.Context) {
	w, _ := c.Size()

	backdropArt := artOf("nova")
	accent := white
	center := ui.Point{X: 0.5, Y: 0.35}
	if sel := a.focused(); sel != nil {
		backdropArt = coverArt(*sel)
		accent = a.accentFor(*sel)
		center = capsuleCenter(c)
		a.bg.set(c, sel.slug(), backdropArt, accent, center)
	} else {
		a.bg.set(c, "", backdropArt, accent, center)
	}

	ui.Column(c).Fill().
		Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ui.Scroll(c).Grow(1).Children(func() {
					// The header and everything below it sit inside the gutter,
					// but the stage is full-bleed, as the design's negative
					// margins make it: it is built outside the padded column and
					// pulled back by the gutter's width.
					ui.Column(c).Padding(0, gutter).Children(func() {
						a.topBar(c)
					})

					if len(a.entries) == 0 {
						ui.Column(c).Padding(0, gutter).Children(func() {
							a.emptyLibrary(c)
							// A machine with no installed games is exactly where
							// the owned library is worth showing.
							a.steamLibrary(c)
						})
						return
					}

					a.stage(c, w)

					ui.Column(c).Padding(0, gutter).Children(func() {
						if sel := a.focused(); sel != nil {
							a.focusBlock(c, *sel, w)
						}
						a.below(c)
						a.steamLibrary(c)
					})
				})
				a.hintBar(c)
			})
		})
}

// emptyLibrary says where games were looked for when none were found.
func (a *app) emptyLibrary(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(10).Padding(80, 0).Children(func() {
		if a.scanning {
			ui.Text(c, "Looking for your games…").FontSize(20).FontWeight(200).TextColor(ink)
			ui.Spinner(c)
			return
		}
		ui.Text(c, "No games found").FontSize(34).FontWeight(200).TextColor(ink)
		ui.Text(c, "Looked in "+a.libCfg.Describe()+".").FontSize(15).TextColor(inkDim)
		ui.Text(c, "Press N to add a folder of games.").FontSize(13).TextColor(inkFaint)
	})
}

// trackOffset is how far the track slides for a selection: the selected game's
// own position, exactly as the design's `translateX(-tile.offsetLeft)` uses it.
//
// It is a pure function of the selection, and that is the point. An earlier
// version translated by the distance from the first *built* tile, and because
// the built window followed the selection, that distance stopped changing after
// the fifth tile and the carousel froze. Anything that reintroduces a window
// into this calculation re-breaks it.
func trackOffset(selected int, stride float32) float32 {
	return -(float32(selected) * stride)
}

// stage is the row of tiles, built to the design's structure:
//
//	.stage{overflow-x:clip; padding-block:56px 44px}   ← spans the window, clips
//	.track{width:max-content; transition:transform}   ← as wide as content, slides
//
// The stage clips and the track slides: keeping those on different elements is
// what stops the slide from dragging the clip edge with it.
//
// The capsule keeps Steam's own portrait thumbnail ratio rather than the
// design's square: Steam publishes no square asset, so a square tile would crop
// the art it ships. `portraiture` is the ratio of Steam's real files.
func (a *app) stage(c *ui.Context, w float32) {
	_, h := c.Size()

	tile := clamp(120, w*0.15, 196)
	height := tile * portraiture
	side := tile * 0.12
	stride := tile + side*2

	stagePadTop := stageTop(h)
	anchor := clamp(gutter, w*0.06, 96)

	stage := ui.Box(c).
		FillWidth().
		ClipX().
		Padding(stagePadTop, 0, stageBottom, 0)

	row := ui.Row(c).
		Gap(6).
		AlignItems(ui.Center).
		Padding(0, anchor)

	target := trackOffset(a.selected, stride)
	x := row.AnimateTo("track", target, ui.Animation{
		Duration: 550 * time.Millisecond,
		Ease:     ui.CubicBezier(0.22, 1, 0.36, 1),
	})
	row.Translate(x, 0)

	stage.Children(func() {
		row.Children(func() {
			for i := range a.entries {
				a.tile(c, i, a.entries[i], tile, height, side)
			}
			a.addTile(c, tile, height, side)
		})
	})
}


// tile is one game's artwork, enlarged and ringed while it is in focus.
func (a *app) tile(c *ui.Context, i int, e entry, size, height, side float32) {
	selected := i == a.selected

	scale := float32(1)
	opacity := float32(0.62)
	if selected {
		scale, opacity = 1.24, 1
	}

	box := ui.Box(c).
		Size(size, height).
		Radius(14).
		Key(e.slug()).
		Label(e.name()).
		Margin(0, side).
		Draw(func(p *ui.Painter, r ui.Rect) { a.paintGame(p, r, 14, e, false) })

	motion := ui.Animation{
		Duration: 450 * time.Millisecond,
		Ease:     ui.CubicBezier(0.22, 1.2, 0.36, 1),
	}
	s := box.AnimateTo("scale", scale, motion)
	box.Scale(s, s)
	box.Opacity(box.AnimateTo("dim", opacity, ui.Animation{Duration: 350 * time.Millisecond}))

	if selected {
		box.Border(3, ui.RGBA(255, 255, 255, 0.95))
		box.Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.6))
		box.Shadow(0, 0, 46, 0, ui.RGBA(255, 255, 255, 0.38))
	} else {
		// Every tile casts the design's resting shadow, and lifts under the
		// pointer.
		box.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.4))
		if box.Hovered() {
			box.Shadow(0, 14, 38, 0, ui.RGBA(0, 0, 0, 0.5))
		}
	}

	// A game never played is marked New, the design's corner badge. A game the
	// account owns but does not have installed says that instead, because it
	// cannot be played from here and the two badges share a corner.
	if !e.installed() {
		box.Children(func() { notInstalledBadge(c) })
	} else if a.lastOf(e).IsZero() {
		box.Children(func() { newBadge(c) })
	}
	if a.favorites[e.slug()] {
		box.Children(func() { a.favoriteBadge(c) })
	}

	box.OnClick(func() {
		if selected {
			a.playSound("select")
			a.open(e)
			return
		}
		a.selectGame(i)
		a.playSound("tick")
	})
}

// newBadge marks a game that has never been played, in the tile's top-left
// corner, absolutely placed so it overlays the artwork.
func newBadge(c *ui.Context) {
	ui.Row(c).Absolute().Top(0).Left(0).Padding(8).Children(func() {
		ui.Text(c, "New").
			FontSize(11).
			FontWeight(500).
			TextColor(success).
			LetterSpacing(0.06).
			Padding(3, 9).
			Radius(99).
			Background(ui.RGBA(0, 0, 0, 0.5))
	})
}

// notInstalledBadge marks a game the account owns but does not have on this
// machine, so a tile that cannot start never looks playable.
//
// It sits in the tile's top-left, inside its padding, and alone: a game that is
// not installed is not also marked New, because the two would overlap and the
// useful fact is that it cannot be played from here.
func notInstalledBadge(c *ui.Context) {
	ui.Row(c).Absolute().Top(8).Left(8).Children(func() {
		ui.Text(c, "Not installed").
			FontSize(11).
			FontWeight(500).
			TextColor(inkDim).
			Padding(3, 9).
			Radius(99).
			Background(ui.RGBA(0, 0, 0, 0.62))
	})
}

// focusBlock is the title, its facts, its blurb and its actions.
func (a *app) focusBlock(c *ui.Context, e entry, w float32) {
	// The stage above already leaves its own bottom gap; this is the design's
	// `.focus{padding-block:6px 40px}` plus the air its actions need, so the
	// buttons never sit against the capsule's glow.
	ui.Column(c).Gap(10).MaxWidth(760).Padding(28, 0, 40, 0).Children(func() {
		ui.Text(c, e.name()).
			FontSize(clamp(34, w*0.054, 64)).
			FontWeight(200).
			TextColor(ink).
			LineHeight(1.05)

		// The meta line, then at most two lines of blurb: enough to say what
		// the game is without pushing the actions off the screen.
		a.homeMeta(c, e)

		if blurb := a.homeBlurb(e); blurb != "" {
			ui.Text(c, blurb).
				FontSize(16).
				FontWeight(300).
				TextColor(inkDim).
				LineHeight(1.55).
				MaxWidth(560).
				MaxLines(2).
				Padding(2, 0, 0, 0)
		}

		ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(14, 0, 0, 0).Children(func() {
			a.playButtonStyled(c, "Play", func() { a.play(e) })

			// Details is the way to the game's own page, and it is a glass pill
			// rather than a second white button: one play action, one way in.
			details := glassSurface(ui.Row(c).Height(52).Padding(0, 26).AlignItems(ui.Center).Radius(999))
			details.Children(func() {
				ui.Text(c, "Details").FontSize(16).TextColor(ink)
			})
			details.OnClick(func() {
				a.playSound("select")
				a.open(e)
			})

			roundButton(c, "star", a.favorites[e.slug()], func() { a.toggleFavorite(e) })
		})
	})
}

// homeMeta is the meta line for a game: its genre, its play time and when it
// was last played. A game that has never run still shows its genre and the
// play time, which is "Not played yet" for one with no history at all.
func (a *app) homeMeta(c *ui.Context, e entry) {
	last := a.lastOf(e)
	parts := []string{a.genreLine(e), a.playedLine(e)}
	if !last.IsZero() {
		parts = append(parts, "Last played "+lastPlayed(last))
	}
	metaLine(c, parts...)
}

// homeBlurb is the short line under the meta: the store's description, cut to
// its opening sentence so two lines are enough.
func (a *app) homeBlurb(e entry) string {
	text := a.describe(e)
	const limit = 190
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

// recentRow lists the games played lately, newest first: the same `stats.Last`
// the cards print, so a card's timestamp is the reason it is in this row.
func (a *app) recentRow(c *ui.Context) {
	games := a.playedLately(recentCards)

	if len(games) == 0 {
		ui.Text(c, "Games you play show up here.").FontSize(14).TextColor(inkFaint)
		return
	}

	ui.Row(c).Gap(12).Wrap().Children(func() {
		for _, e := range games {
			a.recentCard(c, e)
		}
	})
}

// recentCards is how many games "Jump back in" lists, as the design's
// `.slice(0, 4)`.
const recentCards = 4

// playedLately returns the games with a play history, most recently played
// first. It is the one definition of "recent" the home row and the Recently
// opened filter share, and it reads stats.Last rather than Opened on purpose:
// "Jump back in" is about games you played, and a game you only looked at has
// no "last played" to print under its name.
func (a *app) playedLately(limit int) []entry {
	type used struct {
		entry
		at time.Time
	}

	var games []used
	for _, e := range a.allEntries() {
		if t := a.lastOf(e); !t.IsZero() {
			games = append(games, used{entry: e, at: t})
		}
	}

	sort.SliceStable(games, func(i, j int) bool {
		return games[i].at.After(games[j].at)
	})

	if limit > 0 && len(games) > limit {
		games = games[:limit]
	}

	out := make([]entry, 0, len(games))
	for _, g := range games {
		out = append(out, g.entry)
	}
	return out
}

// below is the row under the carousel: what was played lately, and the
// library's numbers.
func (a *app) below(c *ui.Context) {
	ui.Row(c).
		Gap(clamp(20, 48, 48)).
		AlignItems(ui.Start).
		Padding(8, 0, 52, 0).
		Children(func() {
			ui.Column(c).Grow(1.6).Gap(14).Children(func() {
				ui.Text(c, "Jump back in").FontSize(15).FontWeight(500).TextColor(inkDim)
				a.recentRow(c)
			})

			ui.Column(c).Grow(1).Gap(14).Children(func() {
				ui.Text(c, "Your library").FontSize(15).FontWeight(500).TextColor(inkDim)
				a.libraryStats(c)
			})
		})
}

// recentCard is one entry of the recent row: a small cover, its name, and when
// it was last played, as the design's cards show. It prints the same wording
// the game's own page does.
func (a *app) recentCard(c *ui.Context, e entry) {
	card := ui.Row(c).
		Basis(230).
		Grow(1).
		Gap(14).
		Padding(10).
		Radius(18).
		AlignItems(ui.Center).
		OnClick(func() { a.selectAndOpen(e) })
	card = raisedSurface(card)

	card.Children(func() {
		ui.Box(c).Size(54, 54).Radius(12).
			Draw(func(p *ui.Painter, r ui.Rect) { a.paintGame(p, r, 12, e, false) })

		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, e.name()).FontSize(16).TextColor(ink).SingleLine()
			ui.Text(c, lastPlayed(a.lastOf(e))).
				FontSize(13).TextColor(inkDim).SingleLine()
		})
	})
}

// libraryStats is the three numbers the design's summary shows: the games in
// the library, the time played, and how many are favourites.
func (a *app) libraryStats(c *ui.Context) {
	var total time.Duration
	favorites := 0
	all := a.allEntries()
	for _, e := range all {
		total += a.playedOf(e)
		if a.favorites[e.slug()] {
			favorites++
		}
	}

	ui.Row(c).
		Gap(14).
		Padding(16, 20).
		Radius(18).
		Background(ui.RGBA(10, 14, 30, 0.45)).
		Border(1, glassLine).
		Children(func() {
			for _, s := range []struct{ value, label string }{
				{itoa(len(all)), "Games"},
				{hours(total), "Time played"},
				{itoa(favorites), "Favorites"},
			} {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, s.value).FontSize(26).FontWeight(200).TextColor(ink)
					ui.Text(c, s.label).FontSize(13).TextColor(inkDim)
				})
			}
		})
}

// hours formats a total as the design does: whole hours with one decimal.
func hours(d time.Duration) string {
	h := float64(d) / float64(time.Hour)
	if h < 10 {
		return fmt.Sprintf("%.1fh", h)
	}
	return fmt.Sprintf("%.0fh", h)
}

// entryBySlug finds an installed game by its slug.
func (a *app) entryBySlug(slug string) (entry, bool) {
	for _, e := range a.allEntries() {
		if e.slug() == slug {
			return e, true
		}
	}
	return entry{}, false
}

// selectAndOpen focuses a game and opens it.
func (a *app) selectAndOpen(e entry) {
	for i, cur := range a.entries {
		if cur.slug() == e.slug() {
			a.selected = i
			break
		}
	}
	a.open(e)
}

// favoriteBadge is the star on a favorite tile.
// favoriteBadge is the star in a favourite tile's top-right corner. It is
// absolutely placed so it overlays the artwork instead of taking a share of the
// tile's layout, which is what two stacked Fill() rows would do.
func (a *app) favoriteBadge(c *ui.Context) {
	ui.Row(c).Absolute().Top(0).Right(0).Padding(8).Children(func() {
		ui.Box(c).Size(22, 22).Radius(11).
			Background(ui.RGBA(0, 0, 0, 0.45)).
			Center().
			Children(func() { icon(c, "star", 12).TextColor(favorite) })
	})
}

// toggleFavorite flips a game's star.
func (a *app) toggleFavorite(e entry) {
	slug := e.slug()
	a.favorites[slug] = !a.favorites[slug]

	if a.favorites[slug] {
		a.playSound("select")
		a.say(e.name(), "added to favorites")
		return
	}
	a.playSound("back")
	a.say(e.name(), "removed from favorites")
}

// addTile is the last tile: the way into the add form, as the design's tile is
// the way into "Add a game".
func (a *app) addTile(c *ui.Context, size, height, side float32) {
	box := ui.Box(c).
		Size(size, height).
		Radius(14).
		Key("add").
		Label("Add a game").
		Margin(0, side).
		Center().
		OnClick(func() {
			a.playSound("tick")
			a.openAddGame()
		})
	box = glassSurface(box)

	box.Children(func() {
		ui.Column(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "+").FontSize(44).FontWeight(200).TextColor(inkDim)
			ui.Text(c, "Add game").FontSize(13).TextColor(inkDim)
		})
	})
}
