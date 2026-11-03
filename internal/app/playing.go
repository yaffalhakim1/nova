package app

import (
	"github.com/egoist/mygo/ui"
)

// playingOverlay is the session window the design shows once a game starts:
// its cover, the running timer, and Quit.
func (a *app) playingOverlay(c *ui.Context) {
	ui.Modal(c, &a.playingOpen, func() {
		if !a.playingOpen {
			a.overlay = overlayNone
			return
		}
		if a.playing == nil {
			a.overlay = overlayNone
			return
		}

		e, ok := a.entryBySlug(a.playing.slug)
		if !ok {
			a.overlay = overlayNone
			return
		}

		ui.Column(c).AlignItems(ui.Center).Gap(14).Children(func() {
			ui.Box(c).Size(150, 150).Radius(26).
				Draw(func(p *ui.Painter, r ui.Rect) { a.paintGame(p, r, 26, e, true) })

			ui.Text(c, e.name()).FontSize(32).FontWeight(200).TextColor(ink)

			ui.Text(c, clock(a.sessionElapsed())).
				FontSize(44).
				FontWeight(200).
				TextColor(ink)

			ui.Text(c, a.playNote).FontSize(13).TextColor(inkDim)

			ui.Row(c).Gap(12).Children(func() {
				primaryButton(c, "Quit game", func() { a.quitPlay() })
				ui.Button(c, "Close").Radius(999).Padding(13, 28).OnClick(func() {
					a.overlay = overlayNone
				})
			})
		})

		// The timer is a moving value, so the frame must be asked for again.
		c.AnimationFrame()
	})
}
