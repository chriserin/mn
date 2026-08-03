package main

import (
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// Click synthesis parameters. See design/08-audio.md: a short sine burst
// shaped by a fast exponential decay (not a sustained tone), pitched higher
// on the accented beat 1 click than beats 2-4, mirroring the visual accent.
const (
	clickSampleRate = beep.SampleRate(44100)
	clickDuration   = 25 * time.Millisecond
	accentFreqHz    = 1600.0
	plainFreqHz     = 1000.0
	// decayRate controls how quickly the envelope fades across
	// clickDuration; exp(-decayRate) at the end of the burst is small
	// enough that the tail is inaudible rather than cut off abruptly.
	decayRate = 6.0
)

// clickStreamer generates a single synthesized click: a sine wave at freqHz
// with an exponentially decaying amplitude envelope, for clickDuration.
type clickStreamer struct {
	freqHz       float64
	pos, samples int
}

func newClickStreamer(freqHz float64) *clickStreamer {
	return &clickStreamer{
		freqHz:  freqHz,
		samples: clickSampleRate.N(clickDuration),
	}
}

func (c *clickStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if c.pos >= c.samples {
		return 0, false
	}
	for n < len(samples) && c.pos < c.samples {
		t := float64(c.pos) / float64(clickSampleRate)
		envelope := math.Exp(-decayRate * t / clickDuration.Seconds())
		v := math.Sin(2*math.Pi*c.freqHz*t) * envelope
		samples[n][0] = v
		samples[n][1] = v
		c.pos++
		n++
	}
	return n, true
}

func (c *clickStreamer) Err() error { return nil }

// audioInitOnce/audioAvailable lazily open the audio device on the first
// click, rather than at startup, and remember whether it succeeded: on
// environments without audio output (SSH, CI, etc.) speaker.Init fails, and
// every click thereafter silently no-ops instead of retrying per beat. See
// design/08-audio.md "No audio output".
var (
	audioInitOnce  sync.Once
	audioAvailable bool
)

func ensureAudioInit() {
	audioInitOnce.Do(func() {
		// 100ms, matching beep's own documented Init example
		// (sr.N(time.Second/10)). A much smaller buffer (originally 20ms
		// here) underruns almost immediately under normal goroutine
		// scheduling/GC jitter, and some CoreAudio backends stop invoking
		// their data callback after an underrun and never resume — audible
		// as "plays once, then permanently silent until the app restarts".
		bufferSize := clickSampleRate.N(time.Second / 10)
		audioAvailable = speaker.Init(clickSampleRate, bufferSize) == nil
	})
}

// playClick plays one synthesized click, pitched for an accented (beat 1)
// or plain (beats 2-4) beat. It's a package-level var, not a plain func, so
// tests can substitute a spy and assert triggering/accent without a real
// audio device (see design/08-audio.md Testing).
var playClick = func(accented bool) {
	ensureAudioInit()
	if !audioAvailable {
		return
	}
	freq := plainFreqHz
	if accented {
		freq = accentFreqHz
	}
	speaker.Play(newClickStreamer(freq))
}
