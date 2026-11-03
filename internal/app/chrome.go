package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// The bar's icons, parsed once from their outlines.
var iconBodies = map[string]string{
	"search": `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><circle cx="11" cy="11" r="6.5"/><path d="M16 16l4.5 4.5"/></svg>`,
	"back":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l-7 7 7 7"/></svg>`,
	"play":   `<svg viewBox="0 0 16 16" fill="currentColor"><path d="M3 1.5v13l11-6.5z"/></svg>`,
	"next":   `<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 2l6 6-6 6"/></svg>`,
	"star":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"><path d="M12 3.5l2.6 5.5 6 .8-4.4 4.2 1.1 6-5.3-2.9-5.3 2.9 1.1-6L3.4 9.8l6-.8z"/></svg>`,
	"volume": `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9.5v5h3.5L12 18.5v-13L7.5 9.5H4z"/><path d="M15.5 9a4 4 0 010 6M18 6.5a7.5 7.5 0 010 11"/></svg>`,
	"muted":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9.5v5h3.5L12 18.5v-13L7.5 9.5H4z"/><path d="M16 9.5l5 5M21 9.5l-5 5"/></svg>`,
	"trash":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 7h14M10 7V4.5h4V7M7 7l.8 12.5h8.4L17 7"/></svg>`,
	"plus":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>`,
	"check":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>`,
	// The design draws the gear as two rails with their knobs rather than a
	// cog, which stays legible at 20px.
	"settings":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7h10M18 7h2M4 17h2M10 17h10"/><circle cx="16" cy="7" r="2"/><circle cx="8" cy="17" r="2"/></svg>`,
	"controller": `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M7 8h10a4 4 0 014 4v2a3 3 0 01-5.2 2L14.5 14h-5l-1.3 2A3 3 0 013 14v-2a4 4 0 014-4z"/><path d="M8 10.5v3M6.5 12h3"/></svg>`,
}

// icons holds the parsed outlines, by name.
var icons = parseIcons()

// parseIcons parses every icon once; a malformed outline is skipped rather
// than taking the window down.
func parseIcons() map[string]*ui.SVG {
	out := make(map[string]*ui.SVG, len(iconBodies))
	for name, body := range iconBodies {
		if v, err := ui.ParseSVG([]byte(body)); err == nil {
			out[name] = v
		}
	}
	return out
}

// icon builds one icon at a size.
func icon(c *ui.Context, name string, size float32) ui.Element {
	svg := icons[name]
	if svg == nil {
		return ui.Box(c).Size(size, size)
	}
	return ui.Icon(c, svg).Size(size, size)
}

// sectionTabs is the design's tab bar: three sections, the chosen one in a
// glass pill with a hairline. MyGo's Segmented has a look of its own, so the
// parts are built without one and styled here.
func (a *app) sectionTabs(c *ui.Context) {
	labels := []string{"Home", "Favorites", "Library"}

	parts := ui.SegmentedBase(c, &a.section, len(labels))
	parts.Track.Gap(6).AlignItems(ui.Center)

	parts.Track.Children(func() {
		for i, label := range labels {
			seg := parts.Segment(i).
				Padding(8, 18).
				Radius(999).
				Transition(hoverFade).
				Label(label)
			if i == a.section {
				seg.Background(glass).Border(1, glassLine)
			} else {
				seg.Background(ui.RGBA(0, 0, 0, 0))
			}

			color := inkDim
			if i == a.section {
				color = white
			} else if seg.Hovered() {
				color = ink
			}

			seg.Children(func() {
				ui.Text(c, label).FontSize(17).FontWeight(400).TextColor(color)
			})
		}
	})
}

// topBar is the console's header: the sections, the system icons, the clock and
// the profile. The design has no inline search field: search is an overlay,
// opened from the magnifier or "/".
func (a *app) topBar(c *ui.Context) {
	ui.Row(c).
		Padding(28, 0, 0).
		Gap(16).
		AlignItems(ui.Center).
		Children(func() {
			a.sectionTabs(c)

			ui.Spacer(c)

			// The system cluster sits closer together than the rest of the bar,
			// as the design's `.sys` does.
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				iconButton(c, "search", false, func() {
					a.playSound("tick")
					a.overlay = overlaySearch
					a.searchQuery = ""
				})

				iconButton(c, "plus", false, func() {
					a.playSound("tick")
					a.openAddGame()
				})

				iconButton(c, "settings", false, func() {
					a.playSound("tick")
					a.say("Settings", "not built yet")
				})

				// The controller is a status light, not a button: it lights when
				// a pad is connected.
				ui.Box(c).Width(34).Height(40).Center().Label("Controller").
					Children(func() {
						color := inkFaint
						if a.controller {
							color = success
						}
						icon(c, "controller", 20).TextColor(color)
					})

				ui.Text(c, c.Now().Format("15:04")).
					FontSize(18).FontWeight(400).TextColor(ink).
					Margin(0, 0, 0, 4)

				ui.Avatar(c, a.profileInitial(), nil)
			})
		})
}

// profileInitial is the letter in the avatar. A launcher has no account, so it
// is the app's own initial.
func (a *app) profileInitial() string { return "N" }

// primaryButton is the white pill the screen's one action sits in.
func primaryButton(c *ui.Context, label string, onClick func()) ui.Element {
	b := ui.PrimaryButton(c, label).
		Radius(999).
		Padding(15, 44).
		FontSize(18).
		FontWeight(500).
		Transition(hoverFade)
	if onClick != nil {
		b.OnClick(onClick)
	}
	return b
}

// playButton is the primary action with the design's hover: it grows a little
// and lights up, so the screen's one control answers the pointer.
func (a *app) playButtonStyled(c *ui.Context, label string, onClick func()) ui.Element {
	b := primaryButton(c, label, onClick)

	scale := float32(1)
	if b.Hovered() {
		scale = 1.05
	}
	// The scale is animated by the element's own transition, from whatever it
	// showed last frame.
	s := b.AnimateTo("grow", scale, ui.Animation{
		Duration: 200 * time.Millisecond,
		Ease:     ui.EaseOut,
	})
	b.Scale(s, s)

	if b.Hovered() {
		b.Shadow(0, 0, 36, 0, ui.RGBA(255, 255, 255, 0.45))
	}
	return b
}

// iconButton is one of the bar's small glass circles.
func iconButton(c *ui.Context, name string, active bool, onClick func()) ui.Element {
	b := ui.Box(c).
		Size(40, 40).
		Radius(999).
		Center().
		Label(name)
	if onClick != nil {
		b.OnClick(onClick)
	}
	b.Children(func() {
		color := inkDim
		if active {
			color = ink
		}
		icon(c, name, 20).TextColor(color)
	})
	return b
}

// roundButton is one of the glass circles beside it.
func roundButton(c *ui.Context, name string, pressed bool, onClick func()) ui.Element {
	b := ui.Box(c).Size(52, 52).Radius(999).Center()
	b = glassSurface(b)
	if onClick != nil {
		b.OnClick(onClick)
	}
	b.Children(func() {
		color := ink
		if name == "star" && pressed {
			color = favorite
		}
		icon(c, name, 20).TextColor(color)
	})
	return b
}

// backButton leaves the detail page.
func backButton(c *ui.Context, onClick func()) ui.Element {
	b := ui.Row(c).
		Height(40).
		Padding(0, 18, 0, 12).
		Gap(8).
		Radius(999).
		AlignItems(ui.Center).
		OnClick(onClick)
	b.Children(func() {
		icon(c, "back", 18).TextColor(inkDim)
		ui.Text(c, "Back to games").FontSize(16).TextColor(inkDim)
	})
	return b
}

// metaLine is the dim row under a title: the genre, how long it has been played
// and when, each entry its own element so a chip can sit among them.
func metaLine(c *ui.Context, parts ...string) {
	filled := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filled = append(filled, p)
		}
	}
	ui.Text(c, strings.Join(filled, "   ·   ")).
		FontSize(15).
		FontWeight(300).
		TextColor(inkDim)
}


