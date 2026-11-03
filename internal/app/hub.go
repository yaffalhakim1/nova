package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// The game hub: the detail page the design draws as `.detail` over `.hub`. The
// game's artwork fills the window, and a column over it holds the way back, the
// title with its actions, and a row of cards. Each card is a fixed 224 tall so
// the row reads as one shelf, whatever it holds.
const (
	// hubCardHeight and hubCardRadius are the design's `.hcard` metrics, and
	// hubCardGap the design's `.cards` gap.
	hubCardHeight float32 = 224
	hubCardRadius float32 = 22
	hubCardGap    float32 = 16
	// hubCardWidth and hubCardWideWidth are the two widths the cards are built
	// at, and the floor and ceiling of the design's `clamp()` for each.
	hubCardWidth     = 320
	hubCardWideWidth = 440
	hubCardMin       = 250
	hubCardWideMin   = 300
)

// hubCardFill is the design's `rgba(10,14,30,.58)`: dark enough for white type,
// translucent enough that the artwork behind the row shows through.
var hubCardFill = ui.RGBA(10, 14, 30, 0.58)

// paintHubArt is the page's `.dart`: the game's artwork filling the window,
// scaled a little past its edges as the design's `transform:scale(1.12)` does,
// with the two scrims it lays over the artwork so the title and the cards keep
// their contrast.
func (a *app) paintHubArt(p *ui.Painter, r ui.Rect, e entry) {
	const over = 1.12
	w, h := r.W*over, r.H*over
	art := ui.Rect{X: r.X - (w-r.W)/2, Y: r.Y - (h-r.H)/2, W: w, H: h}

	p.Clip(r, 0, func() {
		// The ground first: a game whose art is missing still gets the page's
		// own dark, never the window's bare background through a gap.
		p.Fill(art, deep, 0)

		// The store's wide page background when it has arrived, which is the
		// shape this page is; the portrait cover until then.
		if b := a.backdropOf(e); b != nil {
			p.Image(b, art, ui.Cover)
		} else {
			a.paintGame(p, art, 0, e, false)
		}
	})

	// Darkest at the left, where the title and the cards start.
	p.FillGradient(r, ui.LinearGradient{
		Angle: 90,
		Stops: []ui.GradientStop{
			{Color: fade(deep, 0.86), At: 0},
			{Color: fade(deep, 0.20), At: 0.7},
			{Color: fade(deep, 0.20), At: 1},
		},
	}, 0)

	// And darkest at the foot, under the cards.
	p.FillGradient(r, ui.LinearGradient{
		Angle: 180,
		Stops: []ui.GradientStop{
			{Color: fade(deep, 0), At: 0},
			{Color: fade(deep, 0), At: 0.4},
			{Color: fade(deep, 0.96), At: 1},
		},
	}, 0)
}

// hub is the page's own column: the way back at the top, and the focus and the
// cards pinned to the foot of the window, as the design's `.hubmain` takes up
// the free space above it with `margin-top:auto`.
//
// The column is given the window's own height, which is the design's
// `min-height:100%` on `.hub`: the foot stays at the foot, and a window too
// short to hold the cards scrolls instead of clipping them, as the design's
// `overflow-y:auto` on `.detail` does.
func (a *app) hub(c *ui.Context, e entry) {
	_, h := c.Size()

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).
			MinHeight(h).
			Padding(24, 0, 28, 0).
			Children(func() {
				ui.Row(c).PaddingX(gutter).Children(func() {
					backButton(c, func() {
						a.playSound("back")
						a.current = nil
					})
				})

				ui.Spacer(c)

				ui.Column(c).MinWidth(0).Children(func() {
					a.hubFocus(c, e)
					a.hubCards(c, e)
				})
			})
	})
}

// hubFocus is the title, the meta line and the actions, as the design's
// `.hfocus`: no wider than 820 so the title never runs the whole window, and
// 28 of room under it before the cards. The gutter is on the row around it, so
// the 820 is the text's own width, as the design's max-width is measured inside
// the hub's padding.
func (a *app) hubFocus(c *ui.Context, e entry) {
	w, _ := c.Size()

	ui.Row(c).PaddingX(gutter).Children(func() {
		ui.Column(c).
			MaxWidth(820).
			Padding(48, 0, 28, 0).
			Gap(10).
			Children(func() {
				ui.Text(c, e.name()).
					FontSize(clamp(40, w*0.07, 84)).
					FontWeight(200).
					TextColor(ink).
					LineHeight(1.05)

				a.hubMeta(c, e)

				ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(14, 0, 0, 0).Children(func() {
					a.playButton(c, e)
					roundButton(c, "star", a.favorites[e.slug()], func() { a.toggleFavorite(e) })
					roundButton(c, "trash", false, func() { a.forget(e) })
				})
			})
	})
}

// hubMeta is the line under the title: what the game is, how long it has been
// played, and when. A game that has never run says that once, rather than
// "Not played yet · Never played".
func (a *app) hubMeta(c *ui.Context, e entry) {
	parts := []string{a.genreLine(e)}
	if a.lastOf(e).IsZero() {
		parts = append(parts, a.lastPlayedLine(e))
	} else {
		parts = append(parts, a.playedLine(e), a.lastPlayedLine(e))
	}
	metaLine(c, parts...)
}

// hubCards is the design's `.cards`: a row that scrolls sideways, built
// full-bleed. The page's gutter is left off the row and put inside it instead,
// so the first card starts where the title does and the row runs to the
// window's edges.
func (a *app) hubCards(c *ui.Context, e entry) {
	w, _ := c.Size()
	cardW := clamp(hubCardMin, w*0.24, hubCardWidth)
	wideW := clamp(hubCardWideMin, w*0.34, hubCardWideWidth)

	ui.ScrollHorizontal(c).
		FillWidth().
		Gap(hubCardGap).
		Padding(8, gutter, 14, gutter).
		Scrollbars(false).
		Children(func() {
			a.trophiesCard(c, e, cardW)
			a.activityCard(c, e, cardW)
			a.sessionsCard(c, e, cardW)
			a.infoCard(c, e, cardW)
			a.screenshotsCard(c, e, wideW)
			a.aboutCard(c, e, wideW)
		})
}

// hubCard is one card of the row, as the design's `.hcard`: a flat translucent
// panel with a hairline, which lifts four pixels under the pointer.
func hubCard(c *ui.Context, width float32, title, note string, content func()) {
	hubCardElement(c, width, title, note, content)
}

// hubCardElement is hubCard's body, returning the card so a caller can make it
// clickable: the design's `.hcard` is a button where it opens something.
func hubCardElement(c *ui.Context, width float32, title, note string, content func()) ui.Element {
	card := ui.Column(c).
		Key(title).
		Width(width).
		Height(hubCardHeight).
		Radius(hubCardRadius).
		Padding(20, hubCardPadding).
		Gap(12).
		Background(hubCardFill).
		Border(1, glassLine).
		// The design's `.hcard{overflow:hidden}`: a value that is wider than the
		// card is cut at its edge instead of running out over the artwork.
		Clip().
		Transition(hoverFade)

	lift := float32(0)
	if card.Hovered() {
		card.Background(ui.RGBA(255, 255, 255, 0.10)).
			Border(1, ui.RGBA(255, 255, 255, 0.30))
		lift = -4
	}
	y := card.AnimateTo("lift", lift, ui.Animation{
		Duration: 250 * time.Millisecond,
		Ease:     ui.EaseOut,
	})
	card.Translate(0, y)

	card.Children(func() {
		hubCardTitle(c, title, note)
		content()
	})
	return card
}

// hubCardTitle heads a card: a small upper-case word, and a note on its right.
//
// The design upper-cases the heading with `text-transform`, which the toolkit
// has no property for, so the case is written into the string. The label keeps
// the word in its own case for assistive technology.
func hubCardTitle(c *ui.Context, title, note string) {
	ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Text(c, strings.ToUpper(title)).
			Label(title).
			FontSize(12).
			FontWeight(500).
			// The design's `.08em` on 12px type, in the DIPs LetterSpacing
			// takes.
			LetterSpacing(0.96).
			TextColor(inkDim)

		if note == "" {
			return
		}
		ui.Spacer(c)
		ui.Text(c, note).
			FontSize(13).
			FontWeight(300).
			TextColor(inkFaint).
			SingleLine()
	})
}

// trophiesCard is the design's trophy card, filled from the account's real Steam
// achievements: a ring of how many of the game's achievements are earned, the
// four rarity tiers beside it, and the last one earned underneath.
//
// It degrades to the design's own empty line whenever there is nothing true to
// show — a game with no app id, no key, no achievements, or a load that has not
// answered yet — rather than inventing a ring.
func (a *app) trophiesCard(c *ui.Context, e entry, width float32) {
	g := a.trophiesFor(e)

	if g == nil || len(g.list) == 0 {
		hubCard(c, width, "Trophies", "", func() {
			ui.Text(c, "Not tracked for this game.").
				FontSize(14).
				TextColor(inkFaint).
				LineHeight(1.5)
		})
		return
	}

	note := itoa(g.earned()) + " of " + itoa(len(g.list))

	// The card is the way into the full list, as the design's own card is a
	// button. It is only clickable when there is something to open, which is
	// why the empty state above is not.
	card := hubCardElement(c, width, "Trophies", note, func() {
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			trophyRing(c, g)
			a.trophyTiers(c, g)
		})

		ui.Spacer(c)

		// The two newest fit in the room left under the ring; a third would
		// push the card's bottom row past its 224.
		if rows := g.latestRows(2); len(rows) > 0 {
			withDate := width-hubCardPadding*2 >= trophyDateMinWidth

			ui.Column(c).Gap(7).Children(func() {
				for _, r := range rows {
					trophyRow(c, g, r, withDate)
				}
			})
		}
	})
	card.Label("Trophies")
	card.OnClick(func() {
		a.playSound("select")
		a.trophiesOpen = true
	})
}

// trophyTiers is the design's `.tiers`: the four cups with how many earned
// achievements each holds, rarest first. Steam publishes no tier, so these count
// an achievement by how few players hold it.
func (a *app) trophyTiers(c *ui.Context, g *gameTrophies) {
	counts, ok := g.tierCounts()
	if !ok {
		// No percentages came back, so a count per tier would be a count of
		// nothing. The earned total is still true and is said instead.
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, itoa(g.earned())+" earned").FontSize(15).TextColor(ink)
			ui.Text(c, "rarity unknown").FontSize(12).TextColor(inkFaint)
		})
		return
	}

	// A two-by-two grid rather than a wrapping row: the design's `.tiers` breaks
	// four cups into three and an orphan at this width, which reads as broken.
	ui.Grid(c).Columns(2).GapX(14).GapY(8).JustifyItems(ui.Start).Children(func() {
		for _, tier := range tiersInOrder {
			ui.Row(c).Gap(7).AlignItems(ui.Center).Label(tier.name() + " " + itoa(counts[tier])).Children(func() {
				cup(c, 20, tier.color())
				ui.Text(c, itoa(counts[tier])).
					FontSize(15).
					TextColor(ink).
					FontFeatures("tnum")
			})
		}
	})
}

// trophyRow is the design's `.tro`, cut down to the two rows that fit in the
// card: the newest earned achievements, each with its cup, its name, and when it
// was earned in the design's right-hand `.rt` column when the card is wide
// enough to carry one.
//
// The row is one line because the card is 224 tall: a description under each
// name makes two rows taller than the room under the ring, and the card would
// clip the second. The description and the rarity word belong to the full list,
// which this card does not build.
func trophyRow(c *ui.Context, g *gameTrophies, a steamstore.Achievement, withDate bool) {
	tier, known := g.tierOfAchievement(a)
	color := inkDim
	if known {
		color = tier.color()
	}

	ui.Row(c).Gap(10).AlignItems(ui.Center).MinWidth(0).Children(func() {
		ui.Box(c).Size(trophyRowIcon, trophyRowIcon).Radius(8).
			Background(ui.RGBA(255, 255, 255, 0.06)).
			Center().
			Shrink(0).
			Children(func() {
				cup(c, trophyRowIcon-9, color)
			})

		ui.Text(c, achievementName(a)).
			FontSize(14).
			TextColor(ink).
			SingleLine().
			Grow(1).
			Basis(0).
			MinWidth(0)

		if !withDate {
			return
		}
		if when := dateOf(a.Unlocked); when != "" {
			ui.Text(c, when).
				FontSize(11).
				TextColor(inkFaint).
				NoWrap().
				Shrink(0)
		}
	})
}

// achievementName is what to call an achievement: its own name, or its API name
// when Steam left the name out, so a row never shows an empty line.
func achievementName(a steamstore.Achievement) string {
	if a.Name != "" {
		return a.Name
	}
	return a.APIName
}

// activityCard is the play time as one large number, then the sessions it came
// from. Completion is left at "Not tracked" rather than invented: nothing this
// app reads knows how far through a game anyone is.
func (a *app) activityCard(c *ui.Context, e entry, width float32) {
	total := a.playedOf(e)
	if a.playing != nil && a.playing.slug == e.slug() {
		total += a.sessionElapsed()
	}
	sessions := a.stat.get(e.slug()).Sessions

	per := "0m"
	if sessions > 0 {
		per = played(total / time.Duration(sessions))
	}

	hubCard(c, width, "Activity", "Time played", func() {
		ui.Text(c, played(total)).
			FontSize(40).
			FontWeight(200).
			TextColor(ink).
			LineHeight(1).
			FontFeatures("tnum")

		ui.Row(c).Gap(12).Children(func() {
			statPair(c, itoa(sessions), "Sessions")
			statPair(c, per, "Per session")
		})

		ui.Spacer(c)

		ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, "Completion").FontSize(13).TextColor(inkDim)
			ui.Text(c, "Not tracked").FontSize(13).TextColor(inkDim)
		})
	})
}

// statPair is one number and its word, as the design's `.pair` holds them.
func statPair(c *ui.Context, value, label string) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
		ui.Text(c, value).
			FontSize(22).
			FontWeight(200).
			TextColor(ink).
			SingleLine().
			FontFeatures("tnum")
		ui.Text(c, label).FontSize(13).TextColor(inkDim).SingleLine()
	})
}

// sessionsCard is the honest version of the design's session log: this app
// records how much a game has been played and when it last ran, and no list of
// individual sessions, so it says so rather than printing made-up rows.
func (a *app) sessionsCard(c *ui.Context, e entry, width float32) {
	sessions := a.stat.get(e.slug()).Sessions

	note := ""
	if last := a.lastOf(e); !last.IsZero() {
		note = lastPlayed(last)
	}

	body := "No sessions yet. Press Play to start the first one."
	if sessions > 0 {
		body = "Nova keeps the running total and the last time it ran, not a log of each session."
	}

	hubCard(c, width, "Recent sessions", note, func() {
		ui.Text(c, body).FontSize(14).TextColor(inkFaint).LineHeight(1.5)
	})
}

// infoCard is the game's facts: what Steam says about it, where it is installed,
// and the two things this app does not read (a version and a size) said plainly.
func (a *app) infoCard(c *ui.Context, e entry, width float32) {
	hubCard(c, width, "Game info", e.inst.Label(), func() {
		ui.Column(c).Gap(9).Children(func() {
			for _, row := range a.infoRows(e) {
				infoRow(c, row.label, row.value)
			}
		})
	})
}

// infoRows are the Game info card's rows, in the design's order.
func (a *app) infoRows(e entry) []struct{ label, value string } {
	developer, released := "Unknown", "Unknown"
	if d := a.storeFor(e); d != nil {
		if len(d.Developers) > 0 {
			developer = strings.Join(d.Developers, ", ")
		}
		if d.ReleaseDate != "" {
			released = d.ReleaseDate
		}
	}

	location := "Not set"
	if e.inst.Dir != "" {
		location = e.inst.Dir
	}

	return []struct{ label, value string }{
		{"Developer", developer},
		{"Released", released},
		{"Version", "Not recorded"},
		{"Install size", "Not recorded"},
		{"Location", location},
	}
}

// infoRow is one row of the card: the word on the left, and the value
// right-aligned and cut with an ellipsis when it does not fit.
// infoRow is one row of the Game info card: a dim label on the left and a value
// on the right.
//
// Both sides are bounded so a long value truncates with an ellipsis instead of
// running out of the card: the label keeps its own width (`Shrink(0)`), the
// value takes the rest and is the only side that can give (`Grow(1)`,
// `Basis(0)`, `MinWidth(0)`), which is what `SingleLine` needs to clip.
func infoRow(c *ui.Context, label, value string) {
	ui.Row(c).Gap(16).Children(func() {
		ui.Text(c, label).
			FontSize(14).
			TextColor(inkDim).
			NoWrap().
			Shrink(0)

		// The value takes what is left and can shrink below its text's width;
		// without MinWidth(0) a row child refuses to go narrower than its
		// content, which is what let a long developer name escape the card.
		ui.Text(c, value).
			FontSize(14).
			TextColor(ink).
			TextAlign(ui.End).
			SingleLine().
			Grow(1).
			Basis(0).
			MinWidth(0)
	})
}

// screenshotsCard is the store's own stills, three of them at 16:10.
func (a *app) screenshotsCard(c *ui.Context, e entry, width float32) {
	hubCard(c, width, "Screenshots", "", func() {
		a.screenshots(c, e)
	})
}

// aboutCard is the blurb, at most six lines, as the design clamps it.
func (a *app) aboutCard(c *ui.Context, e entry, width float32) {
	hubCard(c, width, "About", "", func() {
		ui.Text(c, a.describe(e)).
			FontSize(15).
			LineHeight(1.55).
			TextColor(ink).
			MaxLines(6)
	})
}
