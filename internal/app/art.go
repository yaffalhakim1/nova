package app

import (
	"hash/fnv"
	"image"
	"image/color"

	"github.com/egoist/mygo/ui"
)

// art is the generated cover of one game: two hues and a pattern, all derived
// from its slug so a game always looks the same. Real artwork replaces it on
// the same square crop; nothing else in the interface is tinted by it.
type art struct {
	hue, hue2 float32
	pattern   int
}

// artOf derives a game's artwork from its slug.
func artOf(slug string) art {
	h := fnv.New32a()
	_, _ = h.Write([]byte(slug))
	sum := h.Sum32()

	hue := float32(sum % 360)
	return art{
		hue:     hue,
		hue2:    float32((int(hue) + 55 + int(sum>>9)%70) % 360),
		pattern: int(sum>>3) % 6,
	}
}

// artWithHue derives a game's artwork from its slug, but with the cover colour
// the user chose when they added it by hand. A hue of 0 keeps the derived one.
func artWithHue(slug string, hue int) art {
	a := artOf(slug)
	if hue <= 0 {
		return a
	}
	h := float32((hue%360 + 360) % 360)
	a.hue = h
	a.hue2 = float32((int(h) + 55) % 360)
	return a
}

// coverArt is the artwork a game shows: the colour the user picked for a game
// they added by hand, and the derived one otherwise.
func coverArt(e entry) art { return artWithHue(e.slug(), e.inst.Hue) }

// accent is the color the game lends the switch and its glow.
func (a art) accent() ui.Color { return hsl(a.hue, 0.85, 0.62) }

// paintCover fills a rounded rectangle with the game's artwork: a diagonal
// gradient, one of six shape patterns, and the fade that carries the title.
func paintCover(p *ui.Painter, r ui.Rect, radius float32, a art) {
	p.Clip(r, radius, func() {
		p.FillGradient(r, ui.LinearGradient{
			Angle: 135,
			Stops: []ui.GradientStop{
				ui.GradientStop{Color: hsl(a.hue, 0.75, 0.42), At: 0},
				ui.GradientStop{Color: hsl(a.hue2, 0.70, 0.16), At: 1},
			},
		}, 0)

		paintPattern(p, r, a)

		// The bottom fade behind the title.
		p.FillGradient(r, ui.LinearGradient{
			Angle: 180,
			Stops: []ui.GradientStop{
				ui.GradientStop{Color: fade(deep, 0), At: 0.55},
				ui.GradientStop{Color: ui.RGBA(5, 7, 15, 0.88), At: 1},
			},
		}, 0)
	})
}

// paintPattern draws the game's one shape motif over its gradient.
func paintPattern(p *ui.Painter, r ui.Rect, a art) {
	light := ui.RGBA(255, 255, 255, 0.16)
	dark := ui.RGBA(0, 0, 0, 0.22)

	switch a.pattern {
	case 0: // concentric rings
		for i := 1; i <= 3; i++ {
			d := r.W * float32(i) / 3.2
			p.Fill(ui.Rect{X: r.X + r.W/2 - d/2, Y: r.Y + r.H*0.32 - d/2, W: d, H: d}, light, d/2)
		}
	case 1: // diagonal bands
		for i := 0; i < 5; i++ {
			x := r.X + r.W*float32(i)/4
			p.Line(x, r.Y+r.H, x+r.W*0.55, r.Y, r.W*0.06, dark)
		}
	case 2: // corner blocks
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * 0.34, H: r.H * 0.34}, light, 0)
		p.Fill(ui.Rect{X: r.X + r.W*0.66, Y: r.Y + r.H*0.66, W: r.W * 0.34, H: r.H * 0.34}, dark, 0)
	case 3: // a large arc
		p.Fill(ui.Rect{X: r.X + r.W*0.52, Y: r.Y - r.H*0.30, W: r.W * 0.9, H: r.W * 0.9}, light, r.W*0.45)
	case 4: // a grid
		step := r.W / 5
		for i := float32(1); i < 5; i++ {
			p.Line(r.X+i*step, r.Y, r.X+i*step, r.Y+r.H, 1, light)
			p.Line(r.X, r.Y+i*step, r.X+r.W, r.Y+i*step, 1, light)
		}
	case 5: // a diagonal wedge
		var path ui.Path
		path.MoveTo(r.X, r.Y+r.H)
		path.LineTo(r.X+r.W, r.Y+r.H*0.35)
		path.LineTo(r.X+r.W, r.Y+r.H)
		path.Close()
		p.FillPath(&path, light)
	}
}

// paintShot draws one of a game's generated stills: the same artwork with its
// hues shifted and a different motif, as the design's gallery does.
func paintShot(p *ui.Painter, r ui.Rect, radius float32, a art, index int) {
	shot := a
	shot.hue = float32((int(a.hue) + index*26) % 360)
	shot.hue2 = float32((int(a.hue2) + index*18) % 360)
	shot.pattern = (a.pattern + index + 1) % 6

	paintCover(p, r, radius, shot)

	// A little light in the top corner, so a still reads as a photograph of
	// the game rather than a smaller cover.
	p.FillRadialGradient(r, ui.RadialGradient{
		Center: ui.Point{X: 0.72, Y: 0.18},
		Radius: r.W * 0.7,
		Stops: []ui.GradientStop{
			{Color: ui.RGBA(255, 255, 255, 0.10), At: 0},
			{Color: ui.RGBA(255, 255, 255, 0), At: 1},
		},
	}, 0)
}

// paintGame draws a game's artwork in a rounded box: its real cover when it
// has loaded, and the generated art until then or when Steam has none.
// withTitle draws the fade that keeps a caption legible over a photo.
func (a *app) paintGame(p *ui.Painter, r ui.Rect, radius float32, e entry, withTitle bool) {
	if b := a.coverOf(e); b != nil {
		p.Clip(r, radius, func() {
			p.Image(b, r, ui.Cover)
			if withTitle {
				p.FillGradient(r, ui.LinearGradient{
					Angle: 180,
					Stops: []ui.GradientStop{
						{Color: fade(deep, 0), At: 0.55},
						{Color: ui.RGBA(5, 7, 15, 0.88), At: 1},
					},
				}, 0)
			}
		})
		return
	}
	paintCover(p, r, radius, coverArt(e))
}

// accentOf samples the dominant hue of a cover, so the window takes the game's
// own colour: Cyberpunk's yellow, Control's red. It skips the dull and the
// near-white, which would make every game look the same, and weights what is
// left by how saturated it is.
func accentOf(img image.Image) ui.Color {
	if img == nil {
		return white
	}

	b := img.Bounds()
	step := 5
	var r, g, bl, weight float32

	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			cr, cg, cb := rgbOf(img.At(x, y))

			h, s, l := rgbToHSL(cr, cg, cb)
			// Drop the dull, the blown-out and the near-black: they carry no
			// colour worth adopting.
			if s < 0.28 || l < 0.16 || l > 0.86 {
				continue
			}

			w := s * (1 - abs(l-0.5)*0.6)
			if h >= 60 && h <= 180 {
				// Greens dominate photographs; weight them down so a jungle
				// does not turn every game green.
				w *= 0.55
			}

			r += cr * w
			g += cg * w
			bl += cb * w
			weight += w
		}
	}

	if weight <= 0 {
		return white
	}

	r, g, bl = r/weight, g/weight, bl/weight

	// Lift the result so it reads as an accent on the dark ground.
	max := r
	if g > max {
		max = g
	}
	if bl > max {
		max = bl
	}
	if max > 0 {
		gain := float32(0.92) / max
		gain = clamp(1.0, gain, 1.9)
		r, g, bl = clamp01(r*gain), clamp01(g*gain), clamp01(bl*gain)
	}

	return ui.RGBA(uint8(r*255), uint8(g*255), uint8(bl*255), 255)
}

// rgbOf reads a color's non-premultiplied channels from 0 to 1.
func rgbOf(c color.Color) (r, g, b float32) {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return float32(n.R) / 255, float32(n.G) / 255, float32(n.B) / 255
}

// rgbToHSL converts to hue in degrees and saturation and lightness from 0 to 1.
func rgbToHSL(r, g, b float32) (h, s, l float32) {
	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}

	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

// paintShot draws one of a game's stills: its real cover shifted and framed
// when there is one, and generated art otherwise.
func (a *app) paintShot(p *ui.Painter, r ui.Rect, radius float32, e entry, index int) {
	if b := a.coverOf(e); b != nil {
		p.Clip(r, radius, func() {
			// Each still shows a different part of the cover, so the gallery
			// reads as three shots of one game.
			zoom := 1.0 + 0.18*float32(index)
			w := r.W * zoom
			h := r.H * zoom
			x := r.X - (w-r.W)*float32(index)/2
			y := r.Y - (h-r.H)/2*float32(index+1)/2
			p.Image(b, ui.Rect{X: x, Y: y, W: w, H: h}, ui.Cover)
		})
		return
	}
	paintShot(p, r, radius, coverArt(e), index)
}

// paintBackdrop paints the screen behind everything: the selected game's two
// radial glows over the deep ground, and the vignette that darkens the edges.
// accent is the colour the game's own artwork lends the screen, which keeps the
// mood matching the game even when its cover has not loaded.
// paintBackdropAt paints one game's glows, centred on the capsule that lent
// them, at the given opacity: the cross-fade paints the outgoing game at a
// falling opacity and the incoming one at a rising one, so where they overlap
// they mix rather than one covering the other.
func paintBackdropAt(p *ui.Painter, r ui.Rect, a art, accent ui.Color, center ui.Point, fade float32) {
	if fade <= 0 {
		return
	}

	glowA := accent
	glowB := fadeOf(accent, 1)
	if glowA == white {
		glowA = hsl(a.hue, 0.70, 0.38)
		glowB = hsl(a.hue2, 0.65, 0.26)
	}

	// The design centres its glows on the screen; centring them on the
	// selected capsule makes the colour read as coming from the game itself.
	p.FillRadialGradient(r, ui.RadialGradient{
		Center: center,
		Radius: r.W * 1.05,
		Stops: []ui.GradientStop{
			{Color: fadeOf(glowA, 0.60*fade), At: 0},
			{Color: fadeOf(glowA, 0), At: 1},
		},
	}, 0)

	p.FillRadialGradient(r, ui.RadialGradient{
		Center: ui.Point{X: clamp01(center.X + 0.16), Y: clamp01(center.Y + 0.18)},
		Radius: r.W * 0.82,
		Stops: []ui.GradientStop{
			{Color: fadeOf(glowB, 0.42*fade), At: 0},
			{Color: fadeOf(glowB, 0), At: 1},
		},
	}, 0)
}

// paintBackdrop paints one game's colours over the ground, with the vignette
// and scrim the design puts on every backdrop.
func paintBackdrop(p *ui.Painter, r ui.Rect, a art, accent ui.Color) {
	p.Fill(r, deep, 0)
	paintBackdropAt(p, r, a, accent, ui.Point{X: 0.5, Y: 0.5}, 1)
	paintVignette(p, r)
}

// paintVignette darkens the bottom and the top of the screen, so the title and
// the bar keep their contrast over any artwork.
func paintVignette(p *ui.Painter, r ui.Rect) {

	// The bottom vignette, and a top scrim so the bar keeps its contrast.
	p.FillRadialGradient(r, ui.RadialGradient{
		Center: ui.Point{X: 0.5, Y: 1.1},
		Radius: r.H * 0.9,
		Stops: []ui.GradientStop{
			{Color: ui.RGBA(5, 7, 15, 0.92), At: 0},
			{Color: ui.RGBA(5, 7, 15, 0), At: 1},
		},
	}, 0)
	p.FillGradient(r, ui.LinearGradient{
		Angle: 180,
		Stops: []ui.GradientStop{
			ui.GradientStop{Color: ui.RGBA(5, 7, 15, 0.35), At: 0},
			ui.GradientStop{Color: ui.RGBA(5, 7, 15, 0.65), At: 1},
		},
	}, 0)
}
