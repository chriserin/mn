package main

import "testing"

// testEngine uses a 1000Hz "sample rate" purely so tempo-to-frame math
// comes out in round numbers; the engine doesn't care what the number
// means. accentWaveform/plainWaveform are single distinguishable marker
// frames (2 channels), not real audio, so tests can tell which one Fill
// placed without decoding actual synthesized sound. Playing is set true
// (matching what beginPlayback always does before scheduling ever
// matters) since most tests here are about scheduling, not the
// play/stop gate itself — see TestFillProducesNoClicksWhileNotPlaying.
func testEngine() *metronomeEngine {
	accent := []float32{1, 1}
	plain := []float32{2, 2}
	e := newMetronomeEngine(1000, accent, plain)
	e.SetPlaying(true)
	return e
}

func fill(e *metronomeEngine, startFrame int64, frameCount int) (frames []float32, offset int, clickIndex int, hasClick bool) {
	frames = make([]float32, frameCount*2)
	offset, clickIndex, hasClick = e.Fill(startFrame, frames, 2)
	return frames, offset, clickIndex, hasClick
}

// @ft:71 (superseding the old playClick-spy version — see audio_test.go)
func TestFillCyclesBeatsOneThroughFour(t *testing.T) {
	e := testEngine()
	e.SetTempo(60) // 1000 samples/beat at 1000 "Hz"

	var beats []int
	frame := int64(0)
	for i := 0; i < 6; i++ {
		_, _, beat, hasClick := fill(e, frame, 1000)
		if !hasClick {
			t.Fatalf("iteration %d: expected a click", i)
		}
		beats = append(beats, beat)
		frame += 1000
	}

	want := []int{1, 2, 3, 4, 1, 2}
	for i, w := range want {
		if beats[i] != w {
			t.Errorf("call %d: got beat %d, want %d", i, beats[i], w)
		}
	}
}

// @ft:72 / @ft:73 (superseding the old playClick-spy versions)
func TestFillUsesAccentWaveformOnlyOnBeatOne(t *testing.T) {
	e := testEngine()
	e.SetTempo(60)

	frame := int64(0)
	for i := 0; i < 4; i++ {
		frames, offset, clickIndex, hasClick := fill(e, frame, 1000)
		if !hasClick {
			t.Fatalf("call %d: expected a click", i)
		}
		want := e.plainWaveform[0]
		if clickIndex == 1 {
			want = e.accentWaveform[0]
		}
		if got := frames[offset*2]; got != want {
			t.Errorf("call %d (beat %d): got sample %v, want %v", i, clickIndex, got, want)
		}
		frame += 1000
	}
}

func TestFillNoClickInABufferThatDoesntReachTheNextBeat(t *testing.T) {
	e := testEngine()
	e.SetTempo(60) // 1000 samples/beat

	if _, _, _, hasClick := fill(e, 0, 1000); !hasClick {
		t.Fatal("expected the first beat immediately at frame 0")
	}
	if _, _, _, hasClick := fill(e, 1000, 500); !hasClick {
		t.Fatal("expected the second beat exactly at the start of this buffer")
	}
	if _, _, _, hasClick := fill(e, 1500, 100); hasClick {
		t.Fatal("expected no click: the next beat (frame 2000) isn't in [1500,1600)")
	}
}

func TestResetPlacesTheNextClickImmediatelyAsBeatOne(t *testing.T) {
	e := testEngine()
	e.SetTempo(60)

	fill(e, 0, 1000)    // beat 1
	fill(e, 1000, 1000) // beat 2

	e.Reset()

	// After Reset, Fill should place a click right away at whatever
	// startFrame it's given next — not continue the old schedule — and
	// it should be beat 1 again, matching mn's "strike beat 1 immediately"
	// behavior on every Start, not just the very first one.
	_, _, clickIndex, hasClick := fill(e, 5000, 100)
	if !hasClick {
		t.Fatal("expected an immediate click after Reset")
	}
	if clickIndex != 1 {
		t.Fatalf("expected beat 1 after Reset, got %d", clickIndex)
	}
}

func TestSetTempoChangesTheNextInterval(t *testing.T) {
	e := testEngine()
	e.SetTempo(60) // 1000 samples/beat

	fill(e, 0, 1000) // beat 1 at frame 0

	e.SetTempo(120) // 500 samples/beat

	if _, _, _, hasClick := fill(e, 500, 1); !hasClick {
		t.Fatal("expected the next beat at the new, shorter interval (frame 500)")
	}
}

// A freshly constructed engine (playing defaults to false) and an engine
// that's just been stopped should both produce silence — no click, no
// error — rather than scheduling anything. This is what lets Stop/Start
// avoid pausing the underlying AudioQueue at all (see timing.go's playing
// field doc comment): the queue keeps running through calls like these,
// and it's Fill's job alone to make that silence rather than clicks.
func TestFillProducesNoClicksWhileNotPlaying(t *testing.T) {
	accent := []float32{1, 1}
	plain := []float32{2, 2}
	e := newMetronomeEngine(1000, accent, plain) // playing: false (zero value)
	e.SetTempo(60)

	frames, _, _, hasClick := fill(e, 0, 1000)
	if hasClick {
		t.Fatal("expected no click from a freshly constructed (not-playing) engine")
	}
	for i, s := range frames {
		if s != 0 {
			t.Fatalf("frame %d: expected silence, got %v", i, s)
		}
	}

	e.SetPlaying(true)
	if _, _, _, hasClick := fill(e, 1000, 1000); !hasClick {
		t.Fatal("expected a click once playing is set true")
	}

	e.SetPlaying(false)
	if _, _, _, hasClick := fill(e, 2000, 1000); hasClick {
		t.Fatal("expected no click after playing is set back to false")
	}
}
