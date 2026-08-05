package main

import (
	"math"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/chriserin/mn/internal/clickqueue"
)

// Click synthesis parameters. See design/08-audio.md: a short sine burst
// shaped by a fast exponential decay (not a sustained tone), pitched higher
// on the accented beat 1 click than beats 2-4, mirroring the visual accent.
const (
	sampleRate    = 44100
	channelCount  = 2
	clickDuration = 25 * time.Millisecond
	accentFreqHz  = 1600.0
	plainFreqHz   = 1000.0
	// decayRate controls how quickly the envelope fades across
	// clickDuration; exp(-decayRate) at the end of the burst is small
	// enough that the tail is inaudible rather than cut off abruptly.
	decayRate = 6.0
)

// synthesizeClick renders a single click waveform: a sine wave at freqHz
// shaped by an exponentially decaying amplitude envelope, for
// clickDuration, interleaved for channelCount channels. Computed once (see
// ensureAudioInit), never per beat — metronomeEngine.Fill (timing.go) must
// not synthesize anything in real time, only place precomputed waveforms.
func synthesizeClick(freqHz float64) []float32 {
	frames := int(float64(sampleRate) * clickDuration.Seconds())
	out := make([]float32, frames*channelCount)
	for i := 0; i < frames; i++ {
		t := float64(i) / float64(sampleRate)
		envelope := math.Exp(-decayRate * t / clickDuration.Seconds())
		v := float32(math.Sin(2*math.Pi*freqHz*t) * envelope)
		for c := 0; c < channelCount; c++ {
			out[i*channelCount+c] = v
		}
	}
	return out
}

// audioInitOnce/audioAvailable lazily open the audio device on the first
// Start press, rather than at startup, and remember whether it succeeded:
// on environments without audio output (SSH, CI, etc.) clickqueue.NewContext
// fails, and playback falls back to the wall-clock tick (see
// beginPlayback/nextBeatCmd below) for the rest of the run instead of
// retrying per beat. See design/08-audio.md "No audio output".
var (
	audioInitOnce  sync.Once
	audioAvailable bool
	audioEngine    *metronomeEngine
	audioContext   *clickqueue.Context
)

// ensureAudioInit is a package-level var, not a plain func, so tests can
// substitute a no-op (see audio_test.go's TestMain) — the real
// implementation runs synchronously inside Update's "space" case (via
// beginPlayback), not deferred into a tea.Cmd, so unlike playClick's old
// spy seam, even tests that never execute the returned Cmd would still
// touch a real audio device without this being substitutable. No test in
// this package should depend on one being present (design/08-audio.md
// Testing).
var ensureAudioInit = func() {
	audioInitOnce.Do(func() {
		engine := newMetronomeEngine(sampleRate, synthesizeClick(accentFreqHz), synthesizeClick(plainFreqHz))

		ctx, ready, err := clickqueue.NewContext(&clickqueue.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: channelCount,
			BufferCount:  2, // minimal latency; see ../mndriverpoc's findings on (bufferCount-1)*bufferInterval
			// Rounds up to the hardware's actual I/O quantum via
			// context.go's alignment (see hwinfo_darwin.go); 2ms is just
			// small enough to always round up rather than accidentally
			// requesting something larger.
			BufferSize: 2 * time.Millisecond,
			Fill:       engine.Fill,
			// PredictionOffset intentionally left at 0 for now: the ~40ms
			// UI-ahead-of-click offset mndriverpoc measured on its
			// development machine isn't something CoreAudio exposes as a
			// queryable property, so it isn't safe to bake in as a
			// cross-machine default without re-measuring here. See
			// timing_plan_integration.md's open questions.
		})
		if err != nil {
			return
		}
		<-ready

		audioEngine = engine
		audioContext = ctx
		audioAvailable = true
		// NewContext starts the AudioQueue running (and calling Fill)
		// immediately, but engine.playing defaults to false, so Fill
		// produces silence until Start actually flips it — see
		// metronomeEngine's playing field for why this runs continuously
		// rather than being paused/resumed like a real AudioQueuePause
		// cycle would be.
	})
}

// beginPlayback lazily initializes audio on first use, then either starts
// the audio engine (resetting its schedule so beat 1 lands essentially
// immediately, same as model.go's Start handler already does visually) or
// falls back to the wall-clock tick if no audio device is available.
func beginPlayback(bpm int) tea.Cmd {
	ensureAudioInit()
	if !audioAvailable {
		return tickCmd(bpm)
	}
	audioEngine.Reset()
	audioEngine.SetTempo(float64(bpm))
	audioEngine.SetPlaying(true)
	return waitForBeat()
}

// nextBeatCmd returns the Cmd that waits for the next beat after one has
// already struck: the audio engine's own schedule if available, or the
// wall-clock fallback (recomputed from the current bpm each time, same
// drift-avoidance reasoning tickCmd has always used) otherwise.
func nextBeatCmd(bpm int) tea.Cmd {
	if audioAvailable {
		return waitForBeat()
	}
	return tickCmd(bpm)
}

// setEngineTempo forwards a BPM change to the audio engine if one is
// running; a harmless no-op otherwise (e.g. BPM changed before Start was
// ever pressed, or audio is unavailable).
func setEngineTempo(bpm int) {
	if audioAvailable {
		audioEngine.SetTempo(float64(bpm))
	}
}

// waitForBeat is a package-level var, not a plain func, so tests can
// substitute an instant stub the same way tickCmd's tests do (see
// audio_test.go) — its real implementation blocks on a channel receive
// and a time.Sleep, neither of which a test should wait through. It reads
// one ClickEvent from the audio engine's events channel and resolves at
// its PredictedAudible instant rather than on receipt — see
// timing_plan_integration.md and ../mndriverpoc for why that's what
// actually drives the UI from the audio clock rather than just reacting
// to it late.
var waitForBeat = func() tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-audioContext.Events()
		if !ok {
			return nil
		}
		if d := time.Until(ev.PredictedAudible); d > 0 {
			time.Sleep(d)
		}
		return beatMsg{beat: ev.ClickIndex}
	}
}
