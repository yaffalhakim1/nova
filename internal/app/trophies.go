package app

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// trophiesModal is the design's `#ovTro`: the full achievement list for the open
// game, over the dimmed hub.
//
// It is the list the Trophies card no longer has room for. The card is the way
// in; this is where a description and the rarity of each achievement can be
// read, because a dialog can be taller than 224.
func (a *app) trophiesModal(c *ui.Context) {
	ui.Modal(c, &a.trophiesOpen, func() {
		if !a.trophiesOpen {
			return
		}
		e, ok := a.entryBySlug(a.current.slug())
		if !ok {
			a.trophiesOpen = false
			return
		}
		g := a.trophiesFor(e)
		if g == nil || len(g.list) == 0 {
			a.trophiesOpen = false
			return
		}

		ui.Text(c, "Trophies").FontSize(32).FontWeight(200).TextColor(ink)
		ui.Text(c, e.name()+" · "+itoa(g.earned())+" of "+itoa(len(g.list))+" earned").
			FontSize(13).
			TextColor(inkFaint)

		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			trophyRing(c, g)
			a.trophyTiers(c, g)
		})

		// The list scrolls: a game can have a hundred achievements, and the
		// dialog must keep its Done button reachable.
		ui.Scroll(c).
			MaxHeight(380).
			Children(func() {
				ui.Column(c).Children(func() {
					for _, row := range achievementRows(g) {
						a.trophyListRow(c, g, row)
					}
				})
			})

		ui.Row(c).Gap(12).Justify(ui.End).Children(func() {
			primaryButton(c, "Done", func() {
				a.playSound("back")
				a.trophiesOpen = false
			})
		})
	})
}

// achievementRows orders the list as the design does: the platinum-tier
// achievements first, then the rarest, then the newest. The earned ones and the
// locked ones are interleaved by that same rule, so the top of the list is what
// is worth looking at rather than a block of noise.
func achievementRows(g *gameTrophies) []steamstore.Achievement {
	rows := make([]steamstore.Achievement, len(g.list))
	copy(rows, g.list)

	rarity := func(a steamstore.Achievement) float64 {
		if p, ok := g.rarity[a.APIName]; ok {
			return p
		}
		return 100
	}

	sort.SliceStable(rows, func(i, j int) bool {
		ti, ki := g.tierOfAchievement(rows[i])
		tj, kj := g.tierOfAchievement(rows[j])
		if ki != kj {
			return ki
		}
		if ki && ti != tj {
			return tierRank(ti) < tierRank(tj)
		}
		return rarity(rows[i]) < rarity(rows[j])
	})
	return rows
}

// tierRank orders tiers rarest first, as the design's list does.
func tierRank(t trophyTier) int {
	for i, known := range tiersInOrder {
		if known == t {
			return i
		}
	}
	return len(tiersInOrder)
}

// rarityWord names how rare an achievement is, in the design's words, from the
// real share of players holding it.
func rarityWord(percent float64) string {
	switch {
	case percent < 5:
		return "Ultra Rare"
	case percent < 15:
		return "Very Rare"
	case percent < 50:
		return "Rare"
	default:
		return "Common"
	}
}

// rarityPercent formats the share of players, to one decimal so a 0.4% stays
// distinguishable from a 0%.
func rarityPercent(percent float64) string {
	whole := int(percent)
	tenths := int(percent*10) % 10
	if tenths == 0 {
		return itoa(whole) + "%"
	}
	return itoa(whole) + "." + itoa(tenths) + "%"
}

// trophyListRow is one achievement in the modal, as the design's `.tro`: its cup,
// its name and description, and its rarity on the right. A locked one is dimmed,
// as the design's `.tro.lock`.
func (a *app) trophyListRow(c *ui.Context, g *gameTrophies, ach steamstore.Achievement) {
	tier, known := g.tierOfAchievement(ach)
	color := inkDim
	if known {
		color = tier.color()
	}

	row := ui.Row(c).Gap(14).AlignItems(ui.Center).Padding(13, 2).MinWidth(0)
	row.BorderWidth(1, 0, 0, 0).BorderColor(glassLine)

	// The design dims a locked achievement rather than hiding it.
	if !ach.Achieved {
		row.Opacity(0.55)
	}

	row.Children(func() {
		ui.Box(c).Size(46, 46).Radius(13).
			Background(ui.RGBA(255, 255, 255, 0.06)).
			Center().
			Shrink(0).
			Children(func() { cup(c, 28, color) })

		ui.Column(c).Gap(2).Grow(1).Basis(0).MinWidth(0).Children(func() {
			ui.Text(c, achievementName(ach)).FontSize(16).TextColor(ink).SingleLine()

			// The description is why this dialog exists: the card has no room
			// for it. A hidden achievement has none, which is its own fact.
			desc := strings.TrimSpace(ach.Description)
			if desc == "" {
				desc = "Hidden until earned."
			}
			ui.Text(c, desc).FontSize(13).TextColor(inkDim).MaxLines(2)
		})

		ui.Column(c).Gap(2).AlignItems(ui.End).Shrink(0).Children(func() {
			if p, ok := g.rarity[ach.APIName]; ok {
				ui.Text(c, rarityWord(p)).FontSize(13).TextColor(inkDim)
				ui.Text(c, rarityPercent(p)+" of players").FontSize(12).TextColor(inkFaint)
				return
			}
			// No percentages came back: say whether it was earned, which is the
			// part that is known.
			if ach.Achieved {
				ui.Text(c, "Earned").FontSize(13).TextColor(inkDim)
			} else {
				ui.Text(c, "Locked").FontSize(13).TextColor(inkFaint)
			}
		})
	})
}
