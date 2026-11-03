package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// particleSpeed is how fast the drift moves, in pixels a second, which the
// per-dot value spreads across. The design creeps (about 1px a second); this is
// lifted so the drift reads as motion rather than as a still image.
const particleSpeed = 46

// particlePulse is how fast a dot's opacity breathes, in radians a second. It
// scales with the speed, as the design's own fade does.
const particlePulse = 1.7

// particleCount is how many dots drift up the screen.
const particleCount = 46

// particle is one dot of the ambient drift, with its own place and pace.
//
// This is the plain, readable model: a dot has a position, and every frame moves
// it. The fields are seeded once (particleSeed) from the dot's index, so the
// drift looks the same on every run.
type particle struct {
	// x is the dot's column, as a fraction of the painter's width.
	x float64
	// y is its place on the rise, in DIPs, counted from below the foot of the
	// screen. It grows until it passes the top and wraps.
	y float64
	// speed is how many DIPs a second it rises.
	speed float64
	// size is its radius, in DIPs.
	size float32
	// phase shifts its fade, so the dots do not all breathe together.
	phase float64
}

// particleField is the drift's state: the dots, and the clock reading they were
// last moved to.
//
// last is what makes the motion frame-rate independent: the field advances by
// the time that actually passed, not by one step a frame, so a window that
// drops frames does not slow the drift down.
type particleField struct {
	dots  []particle
	last  float64
	moved bool
}

// particleSeed builds a dot from its index alone, so the field is identical on
// every run and on every resize.
//
// i+1, not i, because index 0 would make seed 0: every hash of 0 is 0, which
// pinned one dot to the left edge at the minimum speed and size.
func particleSeed(i int) particle {
	seed := float64(i+1) * 2.399963 // the golden angle: spreads the dots out
	return particle{
		x:     frac(seed * 0.7351),
		y:     frac(seed * 1.3819) * 920, // 920 is the tallest travel, below
		speed: float64(particleSpeed) * (0.5 + frac(seed*0.917)),
		size:  2.0 + 3.5*float32(frac(seed*1.553)),
		phase: seed,
	}
}

// seedParticles fills the field on its first frame. The dots are seeded from
// their index, so nothing has to be saved between runs.
func (f *particleField) seedParticles() {
	if f.moved {
		return
	}
	f.dots = make([]particle, particleCount)
	for i := range f.dots {
		f.dots[i] = particleSeed(i)
	}
	f.moved = true
}

// advance moves every dot by the time that passed since the last frame.
//
// dt is clamped: a window that was hidden for a minute must not teleport the
// dots, and the clamp keeps a slow first frame from starting them mid-flight.
//
// The first call seeds the field and records the clock, then moves nothing: a
// dot's place has not changed because the app just started.
func (f *particleField) advance(now float64, travel float64) {
	if !f.moved {
		f.seedParticles()
		f.last = now
		return
	}

	dt := now - f.last
	f.last = now
	if dt <= 0 {
		return
	}
	if dt > 0.25 {
		dt = 0.25
	}

	f.move(dt, travel)
}

// move advances the dots by dt seconds. It is separate from advance so the time
// stepping and the seeding do not hide each other.
func (f *particleField) move(dt, travel float64) {
	for i := range f.dots {
		d := &f.dots[i]
		d.y += d.speed * dt
		// The rise wraps: a dot that passes the top starts again below the foot.
		for d.y > travel {
			d.y -= travel
		}
	}
}

// paintParticlesStateful draws the ambient drift from the field's own state.
//
// This is the readable model: every dot has a position, and the frame moves it.
// It looks the same as the clock-derived version in paintParticles, at the same
// speeds, but the motion is a running total rather than a formula.
func (a *app) paintParticlesStateful(p *ui.Painter, r ui.Rect) {
	now := float64(p.Now().Sub(a.epoch).Seconds())
	travel := float64(r.H) + 80

	a.drift.advance(now, travel)

	const sway = 12.0
	for i := range a.drift.dots {
		d := &a.drift.dots[i]

		// The dot rises from the foot and wraps; y counts up from there.
		y := r.Y + r.H + 40 - float32(d.y)

		// A shallow sideways sway keeps the drift from looking mechanical.
		x := r.X + float32(d.x)*r.W + float32(sway*math.Sin(now*0.35+d.phase))

		// Opacity breathes between 15% and 75% as a dot rises.
		wave := 0.5 + 0.5*math.Sin(now*particlePulse+d.phase)
		alpha := 0.15 + 0.6*wave

		dot := ui.Rect{X: x - d.size/2, Y: y - d.size/2, W: d.size, H: d.size}
		p.FillRadialGradient(dot, ui.RadialGradient{
			Center: ui.Point{X: 0.5, Y: 0.5},
			Radius: d.size / 2,
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(255, 255, 255, float32(alpha)*0.55), At: 0},
				{Color: ui.RGBA(255, 255, 255, 0), At: 1},
			},
		}, 0)
	}

	// Ask for the next frame: the drift is never finished.
	p.AnimationFrame()
}

// paintParticles draws the ambient drift: soft dots rising slowly behind
// everything, each fading as it goes. They are a function of the clock, not of
// any state, so a frame at any moment looks the same whichever way it was
// reached.
//
// The app now paints the drift from state instead (paintParticlesStateful); this
// is kept because it is the simpler thing to reason about, and because a frame
// derived from the clock cannot drift out of step with itself. Switch `view`
// back to this call to use it again.
func (a *app) paintParticles(p *ui.Painter, r ui.Rect) {
	if a.epoch.IsZero() {
		return
	}
	t := float32(p.Now().Sub(a.epoch).Seconds())

	for i := 0; i < particleCount; i++ {
		seed := float64(i)*2.399963 // the golden angle: spreads the dots out
		rx := frac(seed * 0.7351)
		ry := frac(seed * 1.3819)
		// Each dot gets its own pace, from half the base speed to one and a
		// half times it, so the drift never moves as one sheet.
		speed := float64(particleSpeed) * (0.5 + frac(seed*0.917))
		size := 2.0 + 3.5*float32(frac(seed*1.553))

		// A shallow sideways sway keeps the drift from looking mechanical.
		x := r.X + float32(rx)*r.W + 12*float32(math.Sin(float64(t)*0.35+seed))
		travel := float64(r.H) + 80
		y := r.Y + r.H + 40 - float32(math.Mod(ry*travel+float64(t)*speed, travel))

		// Opacity breathes between 15% and 75% as a dot rises. The pulse is
		// scaled with the speed, so a faster drift does not look like dots
		// sliding through a motionless fog.
		wave := 0.5 + 0.5*float32(math.Sin(float64(t)*particlePulse+seed))
		alpha := 0.15 + 0.6*wave

		dot := ui.Rect{X: x - size/2, Y: y - size/2, W: size, H: size}
		p.FillRadialGradient(dot, ui.RadialGradient{
			Center: ui.Point{X: 0.5, Y: 0.5},
			Radius: size / 2,
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(255, 255, 255, alpha*0.55), At: 0},
				{Color: ui.RGBA(255, 255, 255, 0), At: 1},
			},
		}, 0)
	}

	// Ask for the next frame: the drift is never finished.
	p.AnimationFrame()
}

// frac is the fractional part of a float, the cheap hash the dots are spread
// with.
func frac(v float64) float64 {
	return v - math.Floor(v)
}
