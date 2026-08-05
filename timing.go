package main

import (
	"math"
	"sync"
)

// metronomeEngine places accent/plain click waveforms at exact sample
// positions derived from a BPM, implementing oto.FillFunc via Fill. Ported
// from mndriverpoc/internal/metronome (see ../mndriverpoc and
// timing_plan_integration.md), extended in two ways that POC never needed:
// choosing between two waveforms per mn's existing accent design (see
// audio.go), and Reset, needed because mn stops and restarts playback
// within one long-lived audio context — the POC only ever ran one
// continuous session and never needed to re-anchor its schedule.
type metronomeEngine struct {
	sampleRate     int
	accentWaveform []float32
	plainWaveform  []float32

	// mu guards everything below: Fill runs on the audio driver's own fill
	// goroutine, while SetTempo/Reset are called from Bubble Tea's Update
	// goroutine. Locked for the duration of each Fill call (not just a
	// tempo read, like the POC's version) since Reset touches the same
	// scheduling state Fill does.
	mu             sync.Mutex
	samplesPerBeat float64
	beatInMeasure  int // 0 before the first beat of a run; cycles 1..beatsPerMeasure, mirrors Model.currentBeat
	haveClicked    bool
	lastClickFrame int64
	pendingTail    []float32

	// playing gates whether Fill schedules new clicks. False is the
	// zero-value default, so a freshly created engine is silent until
	// SetPlaying(true). Deliberately NOT tied to whether the underlying
	// AudioQueue is running: the queue stays running continuously the
	// whole time (see audio.go), and stopping/starting mn's playback only
	// ever toggles this flag. That's what lets Start avoid the latency a
	// real AudioQueuePause/Start cycle would introduce — buffers already
	// enqueued before a real pause stay queued and have to drain before a
	// fresh schedule can take effect, which isn't a concern here since
	// there's never a pause to begin with.
	playing bool
}

func newMetronomeEngine(sampleRate int, accentWaveform, plainWaveform []float32) *metronomeEngine {
	return &metronomeEngine{
		sampleRate:     sampleRate,
		accentWaveform: accentWaveform,
		plainWaveform:  plainWaveform,
	}
}

// SetTempo changes the tempo used for beats from here on. The next beat is
// scheduled relative to the last one actually placed (see Fill), so this
// takes effect on whichever beat hasn't been placed into a buffer yet —
// it doesn't retroactively touch anything already committed.
func (e *metronomeEngine) SetTempo(bpm float64) {
	e.mu.Lock()
	e.samplesPerBeat = float64(e.sampleRate) * 60 / bpm
	e.mu.Unlock()
}

// Reset clears scheduling state so the next Fill call places a click as
// soon as possible, rather than continuing a schedule anchored to wherever
// playback left off before the previous Stop. Call this on every Start
// press (not just the first), so beat 1 always lands essentially
// immediately — the same "strike beat 1 immediately" behavior model.go's
// Start handler already gives the visual side, achieved here through the
// same mechanism every other beat uses rather than a special case.
func (e *metronomeEngine) Reset() {
	e.mu.Lock()
	e.haveClicked = false
	e.beatInMeasure = 0
	e.pendingTail = nil
	e.mu.Unlock()
}

// SetPlaying starts or stops click scheduling — see the playing field's
// doc comment for why this is what Start/Stop actually toggle, rather
// than pausing the underlying AudioQueue.
func (e *metronomeEngine) SetPlaying(playing bool) {
	e.mu.Lock()
	e.playing = playing
	e.mu.Unlock()
}

// Fill implements oto.FillFunc. It's called from the audio driver's single
// fill goroutine, strictly in playback order.
func (e *metronomeEngine) Fill(startFrame int64, frames []float32, channelCount int) (offsetFrames int, clickIndex int, hasClick bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	frameCount := len(frames) / channelCount

	if len(e.pendingTail) > 0 {
		n := copy(frames, e.pendingTail)
		e.pendingTail = e.pendingTail[n:]
	}

	if !e.playing {
		// Still drains any in-flight click's tail above (so stopping
		// mid-click doesn't cut it off abruptly), just schedules nothing
		// new — the buffer is otherwise silence, which appendBuffer
		// already zeroed before calling Fill.
		return 0, 0, false
	}

	var nextClickFrame int64
	if e.haveClicked {
		nextClickFrame = e.lastClickFrame + int64(math.Round(e.samplesPerBeat))
	} else {
		// Land as soon as possible rather than at a fixed origin (frame
		// 0), so this works the same way for the very first click of the
		// process and for every subsequent Reset+Start after a Stop.
		nextClickFrame = startFrame
	}

	endFrame := startFrame + int64(frameCount)
	if nextClickFrame < startFrame || nextClickFrame >= endFrame {
		return 0, 0, false
	}

	offset := int(nextClickFrame - startFrame)
	e.beatInMeasure = e.beatInMeasure%beatsPerMeasure + 1
	thisBeat := e.beatInMeasure
	e.lastClickFrame = nextClickFrame
	e.haveClicked = true

	waveform := e.plainWaveform
	if thisBeat == 1 {
		waveform = e.accentWaveform
	}

	n := copy(frames[offset*channelCount:], waveform)
	if n < len(waveform) {
		e.pendingTail = append([]float32(nil), waveform[n:]...)
	}

	return offset, thisBeat, true
}
