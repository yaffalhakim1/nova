package app

import (
	"math"
	"testing"
)

// TestParticleFieldKeepsTheSameSpeeds checks the stateful drift moves at the
// speeds the clock-derived version used, so switching models did not change how
// fast the dots go.
func TestParticleFieldKeepsTheSameSpeeds(t *testing.T) {
	var f particleField
	f.seedParticles()

	if len(f.dots) != particleCount {
		t.Fatalf("dots = %d, want %d", len(f.dots), particleCount)
	}

	for i, d := range f.dots {
		// The old formula, verbatim, on the same index.
		seed := float64(i+1) * 2.399963
		want := float64(particleSpeed) * (0.5 + frac(seed*0.917))
		if math.Abs(d.speed-want) > 1e-9 {
			t.Errorf("dot %d speed = %v, want the old %v", i, d.speed, want)
		}
		if d.speed < float64(particleSpeed)*0.5 || d.speed > float64(particleSpeed)*1.5 {
			t.Errorf("dot %d speed = %v, outside the design's 0.5x..1.5x spread", i, d.speed)
		}
	}
}

// TestParticleFieldRisesAtTheRightRate checks a dot moves exactly its speed in
// pixels for one second of time, which is the whole contract of the model.
func TestParticleFieldRisesAtTheRightRate(t *testing.T) {
	var f particleField
	f.advance(0, 920) // seeds, and records the clock
	f.dots[0].y = 0   // start a known dot at the foot

	speed := f.dots[0].speed
	f.move(1.0, 920) // one second of motion

	if math.Abs(f.dots[0].y-speed) > 1e-9 {
		t.Fatalf("after 1s the dot moved %v DIPs, want its speed %v", f.dots[0].y, speed)
	}
}

// TestParticleFieldWrapsAtTheTop checks a dot that leaves the top reappears at
// the foot rather than vanishing or flying off.
func TestParticleFieldWrapsAtTheTop(t *testing.T) {
	var f particleField
	f.advance(0, 500)
	f.dots[0].y = 490
	f.dots[0].speed = 100

	f.move(1.0, 500) // would reach 590, past the 500 travel

	if f.dots[0].y < 0 || f.dots[0].y > 500 {
		t.Fatalf("dot y = %v, outside the 0..500 cycle", f.dots[0].y)
	}
	if f.dots[0].y != 90 {
		t.Fatalf("dot y = %v, want 90 after wrapping", f.dots[0].y)
	}
}

// TestParticleFieldClampsAGap checks a long pause does not teleport the dots:
// a window that was hidden for a minute must not jump the whole field.
func TestParticleFieldClampsAGap(t *testing.T) {
	var f particleField
	f.advance(0, 920)
	f.dots[0].y = 0
	speed := f.dots[0].speed

	f.advance(60, 920) // a minute later

	// The clamp is a quarter second, so the dot moves a quarter of its speed.
	want := speed * 0.25
	if math.Abs(f.dots[0].y-want) > 1e-9 {
		t.Fatalf("after a long gap the dot moved %v, want the clamped %v", f.dots[0].y, want)
	}
}

// TestParticleFieldIsDeterministic checks two fields seeded from the same
// indices are identical, so the drift looks the same on every run.
func TestParticleFieldIsDeterministic(t *testing.T) {
	var a, b particleField
	a.seedParticles()
	b.seedParticles()

	for i := range a.dots {
		if a.dots[i] != b.dots[i] {
			t.Fatalf("dot %d differs between two seeds: %+v vs %+v", i, a.dots[i], b.dots[i])
		}
	}
}

// TestParticleSeedHasNoZeroDot checks the tidy-up: index 0 used to hash to zero
// on every property, pinning one dot to the left edge at the minimum size.
func TestParticleSeedHasNoZeroDot(t *testing.T) {
	d := particleSeed(0)
	if d.x == 0 && d.y == 0 && d.size == 2.0 {
		t.Fatal("the first dot is still all zeroes: the seed offset is missing")
	}
}
