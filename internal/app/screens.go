package app

import (
	"github.com/egoist/mygo/ui"
)

// screenshots draws the game's real store screenshots, falling back to stills
// generated from its artwork when the store has none. Each is 16:10, the ratio
// of the design's gallery, kept by AspectRatio so the panel's column width
// decides the height rather than a fixed one cropping them.
func (a *app) screenshots(c *ui.Context, e entry) {
	var urls []string
	if d := a.storeFor(e); d != nil {
		urls = d.Screenshots
	}

	ui.Row(c).Gap(12).Children(func() {
		for i := 0; i < shotLimit; i++ {
			index := i
			url := ""
			if index < len(urls) {
				url = urls[index]
			}

			ui.Box(c).
				Grow(1).
				Basis(0).
				AspectRatio(16.0 / 10.0).
				Radius(12).
				Draw(func(p *ui.Painter, r ui.Rect) {
					if url != "" {
						if b := a.shots.get(url); b != nil {
							p.Clip(r, 12, func() { p.Image(b, r, ui.Cover) })
							return
						}
					}
					a.paintShot(p, r, 12, e, index)
				})
		}
	})
}

