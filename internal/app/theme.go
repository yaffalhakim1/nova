package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Palette of the launcher: a deep blue-black ground, white type at light
// weights, and glass surfaces. The only hue in the interface comes from the
// selected game's artwork, which is why nothing here is tinted.
var (
	deep      = ui.Hex("#05070f")
	ink       = ui.Hex("#f4f7ff")
	inkDim    = ui.RGBA(220, 230, 255, 0.62)
	inkFaint  = ui.RGBA(220, 230, 255, 0.34)
	glass     = ui.RGBA(255, 255, 255, 0.07)
	glassLine = ui.RGBA(255, 255, 255, 0.14)
	panelFill = ui.RGBA(10, 14, 30, 0.50)
	success   = ui.Hex("#6dffb0")
	favorite  = ui.Hex("#ffd36a")
	white     = ui.Hex("#ffffff")
)

// theme builds the window theme from those tokens.
func theme() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = deep
	t.Surface = panelFill
	t.SurfaceHover = ui.RGBA(255, 255, 255, 0.12)
	t.SurfacePressed = ui.RGBA(255, 255, 255, 0.18)
	t.Border = glassLine
	t.Text = ink
	t.TextMuted = inkDim
	t.Accent = white
	t.AccentHover = ui.RGBA(255, 255, 255, 0.90)
	t.AccentPressed = ui.RGBA(255, 255, 255, 0.75)
	t.AccentText = ui.Hex("#080c1a")
	t.Selection = ui.RGBA(255, 255, 255, 0.20)
	t.Focus = white
	t.Inverse = ink
	t.InverseText = deep
	t.Success = success
	t.Warning = favorite
	t.Danger = ui.Hex("#ff6b6b")
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.18)
	return t
}

// hsl builds a color from a hue in degrees and saturation and lightness from
// 0 to 1, as the artwork gradients are specified.
func hsl(h, s, l float32) ui.Color { return hsla(h, s, l, 1) }

// hsla is hsl with its own alpha.
func hsla(h, s, l, a float32) ui.Color {
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	c := (1 - abs(2*l-1)) * s
	x := c * (1 - abs(float32(math.Mod(float64(h/60), 2))-1))
	m := l - c/2

	var r, g, b float32
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return ui.RGBA(
		uint8(clamp01(r+m)*255),
		uint8(clamp01(g+m)*255),
		uint8(clamp01(b+m)*255),
		clamp01(a),
	)
}

// fade returns c with another alpha, for the transparent end of a gradient.
func fade(c ui.Color, a float32) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, clamp01(a))
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp(lo, v, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// fadeOf returns c with an alpha scaled by k, which is how a colour is drawn at
// part opacity.
func fadeOf(c ui.Color, k float32) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, clamp01(float32(c.A)/255*k))
}
