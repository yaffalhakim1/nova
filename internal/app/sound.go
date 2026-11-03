package app

import (
	"math"

	"github.com/egoist/mygo/audio"
)

// The interface's sounds, as the design plays them: short tones made from a
// sine or triangle, with an exponential decay so they tick rather than ring.
//
// The design synthesises them with WebAudio at run time. The samples are made
// once here, at the same frequencies and lengths, so the app ships no audio
// files and stays a single binary.
const sampleRate = 44100

type tone struct {
	freq float64
	// to is where the pitch slides to by the end, or 0 to hold it.
	to     float64
	length float64
	volume float64
	// wave is "triangle" or "sine".
	wave  string
	delay float64
}

// The tones the design plays, by name.
var tones = map[string][]tone{
	// Navigate: a short high tick.
	"tick": {{freq: 1180, to: 900, length: 0.07, volume: 0.05, wave: "sine"}},
	// The end of a list: a low, dull bump.
	"bump": {{freq: 220, to: 180, length: 0.09, volume: 0.05, wave: "triangle"}},
	// Select: two rising tones.
	"select": {
		{freq: 660, to: 990, length: 0.12, volume: 0.06, wave: "sine"},
		{freq: 990, length: 0.18, volume: 0.05, wave: "sine", delay: 0.07},
	},
	// Back: a short falling tone.
	"back": {{freq: 520, to: 340, length: 0.12, volume: 0.05, wave: "sine"}},
}

// sounds holds the decoded tones, built on first use.
var sounds = buildSounds()

// buildSounds renders every tone once. A failure leaves that sound nil, which
// play treats as silence: no sound is worth failing the window for.
func buildSounds() map[string]*audio.Sound {
	out := make(map[string]*audio.Sound, len(tones))
	for name, ts := range tones {
		out[name] = renderTones(ts)
	}
	return out
}

// renderTones mixes a sound's tones into one buffer.
func renderTones(ts []tone) *audio.Sound {
	// The buffer lasts as long as the last tone finishes.
	end := 0.0
	for _, t := range ts {
		if t.delay+t.length > end {
			end = t.delay + t.length
		}
	}
	if end <= 0 {
		return nil
	}

	frames := int(end * sampleRate)
	samples := make([]float32, frames)

	for _, t := range ts {
		mixTone(samples, t)
	}
	return audio.NewSound(sampleRate, 1, samples)
}

// mixTone adds one tone into the buffer.
func mixTone(samples []float32, t tone) {
	start := int(t.delay * sampleRate)
	n := int(t.length * sampleRate)

	phase := 0.0
	for i := 0; i < n; i++ {
		at := start + i
		if at < 0 || at >= len(samples) {
			continue
		}

		progress := float64(i) / float64(n)

		freq := t.freq
		if t.to > 0 {
			// An exponential slide, as the design's exponentialRampToValueAtTime.
			freq = t.freq * math.Pow(t.to/t.freq, progress)
		}
		phase += 2 * math.Pi * freq / sampleRate

		var v float64
		if t.wave == "triangle" {
			v = 2 / math.Pi * math.Asin(math.Sin(phase))
		} else {
			v = math.Sin(phase)
		}

		// A quick attack and an exponential decay, so the sound has an edge
		// and dies away instead of clicking off.
		env := 1.0
		if progress < 0.06 {
			env = progress / 0.06
		} else {
			env = math.Exp(-5 * (progress - 0.06) / 0.94)
		}

		samples[at] += float32(v * env * t.volume)
	}
}

// playSound plays one of the named sounds, unless sound is off.
func (a *app) playSound(name string) {
	if a.muted {
		return
	}
	if s := sounds[name]; s != nil {
		s.Play()
	}
}
