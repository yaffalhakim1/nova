package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// libraryTileMin is the narrowest a library card gets, the design's
// `grid-template-columns:repeat(auto-fill,minmax(150px,1fr))`.
const libraryTileMin = 150

// libraryTileHeight is a card's height: the cover at its 2:3 ratio plus the two
// lines under it. A GridView cell is a fixed height, so this has to hold both:
// at the 150 floor the cover is 225 and the text needs the rest.
const libraryTileHeight = 297

// libraryPage is the design's Library section: a heading with a count, a sort
// select, genre chips, and a grid of every game.
//
// The grid is the last thing on the page: no summary strip and no "Jump back
// in" under it. The cards are the whole section, and every one is clickable —
// it opens the game's page.
func (a *app) libraryPage(c *ui.Context) {
	w, _ := c.Size()

	// The library has no capsule row, so its glow sits behind the heading.
	a.bg.set(c, "", artOf("nova"), white, ui.Point{X: 0.3, Y: 0.2})

	ui.Column(c).Fill().
		Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Column(c).Padding(0, gutter).Children(func() {
						a.topBar(c)
						ui.Column(c).
							Padding(32, 0, 64, 0).
							Children(func() {
								a.libraryHead(c)
								a.libraryChips(c)
								a.libraryGrid(c, w)
							})
					})
				})
				a.hintBar(c)
			})
		})
}

// libraryHead is the heading, the count and the sort select.
func (a *app) libraryHead(c *ui.Context) {
	w, _ := c.Size()
	total := len(a.allEntries())

	ui.Row(c).AlignItems(ui.Baseline).Gap(16).Wrap().Children(func() {
		ui.Text(c, "Library").
			FontSize(clamp(34, w*0.05, 56)).
			FontWeight(200).
			TextColor(ink).
			LetterSpacing(-0.01)

		label := itoa(len(a.entries)) + " games"
		if len(a.entries) == 1 {
			label = "1 game"
		}
		ui.Text(c, label).FontSize(15).TextColor(inkDim)

		// The count of everything, when a filter or genre is hiding some.
		if len(a.entries) != total {
			ui.Text(c, "of "+itoa(total)).FontSize(15).TextColor(inkFaint)
		}

		ui.Spacer(c)
		a.sortControl(c)
	})
}

// sortControl is the sort select, as the design's pill.
//
// ui.Select chooses one of its options into the pointer it is given, so the app
// keeps the chosen label in sortChoice and maps it back to the sort index on the
// frame that follows. The mapping happens before the list is built, so the order
// changes on the same click.
func (a *app) sortControl(c *ui.Context) {
	labels := []string{"Recently played", "A to Z", "Most played", "Recently added"}

	// A choice made last frame is applied here, before the list is drawn.
	for i, label := range labels {
		if a.sortChoice == label && a.sortBy != i {
			a.sortBy = i
			a.lastKey = -1
			a.playSound("tick")
		}
	}

	// First build, or a sort set from the menu: show what is in force.
	if a.sortChoice == "" {
		a.sortChoice = labels[a.sortBy]
	}

	ui.Select(c, &a.sortChoice, labels).
		Label("Sort by").
		Radius(99).
		Padding(9, 16).
		FontSize(15).
		Background(glass).
		Border(1, glassLine).
		Transition(hoverFade)
}

// sortChoice is the label the sort select shows, kept on the app so the select
// has a stable pointer. It is set from sortBy when the page opens.
func (a *app) syncSortChoice() {
	labels := []string{"Recently played", "A to Z", "Most played", "Recently added"}
	if a.sortBy >= 0 && a.sortBy < len(labels) {
		a.sortChoice = labels[a.sortBy]
	}
}

// libraryChips are the genre filters: one chip per genre, "All" first.
func (a *app) libraryChips(c *ui.Context) {
	genres := a.libraryGenres()

	ui.Row(c).Gap(8).Wrap().Padding(18, 0, 0, 0).Children(func() {
		for _, g := range genres {
			chip := ui.Box(c).
				Padding(7, 16).
				Radius(99).
				Label(g).
				Transition(hoverFade).
				OnClick(func() {
					a.genreFilter = g
					if g == "All" {
						a.genreFilter = ""
					}
					a.lastKey = -1
					a.playSound("tick")
				})

			selected := g == "All" && a.genreFilter == "" || g == a.genreFilter
			if selected {
				chip.Background(white)
				chip.Children(func() {
					ui.Text(c, g).FontSize(14).FontWeight(500).TextColor(ui.Hex("#080c1a"))
				})
				continue
			}

			chip = glassSurface(chip)
			colour := inkDim
			if chip.Hovered() {
				colour = white
			}
			chip.Children(func() {
				ui.Text(c, g).FontSize(14).TextColor(colour)
			})
		}
	})
}

// libraryGrid is the auto-filling grid of games, plus the add tile at the end.
func (a *app) libraryGrid(c *ui.Context, w float32) {
	n := len(a.entries) + 1

	ui.Column(c).Padding(28, 0, 0, 0).Children(func() {
		ui.GridView(c, nil, n, libraryTileMin, libraryTileHeight, func(i int) {
			if i == len(a.entries) {
				a.libraryAddTile(c)
				return
			}
			a.libraryTile(c, a.entries[i])
		}).Grow(1)
	})
}

// libraryTile is one card of the grid: the cover, the name, and the genre with
// its play time, as the design's `.lc`.
func (a *app) libraryTile(c *ui.Context, e entry) {
	card := ui.Column(c).
		Gap(10).
		MinWidth(0).
		Transition(hoverFade).
		OnClick(func() { a.open(e) })

	// The design lifts the whole card four pixels on hover.
	lift := float32(0)
	if card.Hovered() {
		lift = -4
	}
	y := card.AnimateTo("lift", lift, ui.Animation{
		Duration: 300 * time.Millisecond,
		Ease:     ui.CubicBezier(0.22, 1.2, 0.36, 1),
	})
	card.Translate(0, y)

	card.Children(func() {
		// The cover keeps the 2:3 ratio every Steam asset is, so nothing is
		// cropped. It takes the width the grid gives it; the height follows from
		// the ratio, which is what leaves room for the two lines under it.
		cover := ui.Box(c).
			FillWidth().
			AspectRatio(1/portraiture).
			Radius(18).
			Draw(func(p *ui.Painter, r ui.Rect) { a.paintGame(p, r, 18, e, false) })
		if card.Hovered() {
			cover.Border(3, ui.RGBA(255, 255, 255, 0.95))
			cover.Shadow(0, 18, 40, 0, ui.RGBA(0, 0, 0, 0.55))
		} else {
			cover.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.4))
		}
		if a.favorites[e.slug()] {
			cover.Children(func() { a.favoriteBadge(c) })
		}

		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, e.name()).FontSize(15).TextColor(ink).SingleLine()

			// The design's `.lc span.s`: "Genre · time", where a game with no
			// play time says so rather than showing "0m".
			ui.Text(c, a.librarySubtitle(e)).FontSize(13).TextColor(inkDim).SingleLine()
		})
	})
}

// librarySubtitle is the line under a library card's name.
func (a *app) librarySubtitle(e entry) string {
	genre := a.genreLine(e)
	if genre == "" {
		genre = "Game"
	}
	if a.playedOf(e) <= 0 {
		return genre + " · Not played"
	}
	return genre + " · " + played(a.playedOf(e))
}

// libraryAddTile is the last card: the way into the add form, as the design's
// `.lc.add`.
func (a *app) libraryAddTile(c *ui.Context) {
	card := ui.Column(c).
		Gap(10).
		MinWidth(0).
		Label("Add a game").
		Transition(hoverFade).
		OnClick(func() {
			a.playSound("tick")
			a.openAddGame()
		})

	lift := float32(0)
	if card.Hovered() {
		lift = -4
	}
	card.Translate(0, card.AnimateTo("lift", lift, ui.Animation{
		Duration: 300 * time.Millisecond,
		Ease:     ui.CubicBezier(0.22, 1.2, 0.36, 1),
	}))

	card.Children(func() {
		box := ui.Box(c).
			FillWidth().
			AspectRatio(1/portraiture).
			Radius(18).
			Center()
		box = glassSurface(box)
		box.Children(func() {
			ui.Text(c, "+").FontSize(44).FontWeight(200).TextColor(inkDim)
		})

		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "Add a game").FontSize(15).TextColor(ink).SingleLine()
			ui.Text(c, "From this PC").FontSize(13).TextColor(inkDim).SingleLine()
		})
	})
}
