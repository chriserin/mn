package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func key(name string) tea.KeyPressMsg {
	switch name {
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "shift+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	case "shift+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	case "t", "n", "m", "j", "k", "[", "]", "{", "}":
		r := rune(name[0])
		return tea.KeyPressMsg{Code: r, Text: name}
	case "shift+n":
		return tea.KeyPressMsg{Code: 'N', Text: "N", Mod: tea.ModShift}
	case "shift+m":
		return tea.KeyPressMsg{Code: 'M', Text: "M", Mod: tea.ModShift}
	case "shift+j":
		return tea.KeyPressMsg{Code: 'J', Text: "J", Mod: tea.ModShift}
	case "shift+k":
		return tea.KeyPressMsg{Code: 'K', Text: "K", Mod: tea.ModShift}
	}
	panic("test helper: unknown key " + name)
}

func press(m Model, keyName string) Model {
	next, _ := m.Update(key(keyName))
	return next.(Model)
}

func bpmLine(bpm int) string {
	return renderBigNumber(bpm)
}

// assertStatusBar compares the status bar with its ANSI styling stripped,
// since the powerline segments are colored (see design/07-status-bar.md)
// and tests only need to assert on the visible text and wedge glyphs.
func assertStatusBar(t *testing.T, m Model, want string) {
	t.Helper()
	got := ansi.Strip(m.renderStatusBar())
	if got != want {
		t.Errorf("expected status bar %q, got %q", want, got)
	}
}

// @ft:1
func TestDefaultBPMOnStartup(t *testing.T) {
	m := New()
	view := m.View().Content

	if !strings.Contains(view, bpmLine(120)) {
		t.Errorf("expected BPM readout %q, got:\n%s", bpmLine(120), view)
	}
	assertStatusBar(t, m, "◥ mn ◥◥ STOPPED ◥◥ ♩ 120 bpm ◥◤ tempo training (t) ◤")
}

// @ft:74
func TestStatusBarTempoIndicatorTracksBPM(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "up")

	assertStatusBar(t, m, "◥ mn ◥◥ STOPPED ◥◥ ♩ 121 bpm ◥◤ tempo training (t) ◤")
}

// @ft:2
func TestStartTheMetronome(t *testing.T) {
	m := New()
	m = press(m, "space")

	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ tempo training (t) ◤")
}

// @ft:73
func TestBeatOneLightsImmediatelyOnStart(t *testing.T) {
	m := New()
	m = press(m, "space")

	if m.currentBeat != 1 {
		t.Fatalf("expected beat 1 to be lit the instant playback starts, got currentBeat=%d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 1)
}

// @ft:3
func TestStopTheMetronome(t *testing.T) {
	m := New()
	m.playing = true

	m = press(m, "space")

	assertStatusBar(t, m, "◥ mn ◥◥ STOPPED ◥◥ ♩ 120 bpm ◥◤ tempo training (t) ◤")
}

// @ft:4
func TestIncreaseBPMBySmallIncrement(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "up")

	assertBPM(t, m, 121)
}

// @ft:5
func TestDecreaseBPMBySmallIncrement(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "down")

	assertBPM(t, m, 119)
}

// @ft:6
func TestIncreaseBPMByLargeIncrement(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "shift+up")

	assertBPM(t, m, 130)
}

// @ft:7
func TestDecreaseBPMByLargeIncrement(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "shift+down")

	assertBPM(t, m, 110)
}

// @ft:8
func TestBPMCannotGoBelowMinimum(t *testing.T) {
	m := New()
	m.bpm = minBPM

	m = press(m, "down")

	assertBPM(t, m, minBPM)
}

// @ft:9
func TestBPMCannotGoAboveMaximum(t *testing.T) {
	m := New()
	m.bpm = maxBPM

	m = press(m, "up")

	assertBPM(t, m, maxBPM)
}

func assertBPM(t *testing.T, m Model, want int) {
	t.Helper()
	view := m.View().Content
	if !strings.Contains(view, bpmLine(want)) {
		t.Errorf("expected BPM readout %q, got:\n%s", bpmLine(want), view)
	}
}

// @ft:10
func TestBeatPulseAdvancesAndWrapsAfterBeatFour(t *testing.T) {
	m := New()
	m.bpm = 120
	m.playing = true

	next, _ := m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 1 {
		t.Fatalf("after beat 1 struck, expected currentBeat 1, got %d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 1)

	next, _ = m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 2 {
		t.Fatalf("after beat 2 struck, expected currentBeat 2, got %d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 2)

	next, _ = m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 3 {
		t.Fatalf("after beat 3 struck, expected currentBeat 3, got %d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 3)

	next, _ = m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 4 {
		t.Fatalf("after beat 4 struck, expected currentBeat 4, got %d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 4)

	next, _ = m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 1 {
		t.Fatalf("after the beat following beat 4, expected the lit dot to jump back to 1 (not a 5th slot), got %d", m.currentBeat)
	}
	assertOnlyBeatLit(t, m, 1)
}

func assertOnlyBeatLit(t *testing.T, m Model, lit int) {
	t.Helper()
	for i := 1; i <= beatsPerMeasure; i++ {
		want := i == lit
		got := i == m.currentBeat
		if got != want {
			t.Errorf("beat slot %d: expected lit=%v, got lit=%v (currentBeat=%d)", i, want, got, m.currentBeat)
		}
	}
}

// @ft:11
func TestBeat1IsVisuallyAccented(t *testing.T) {
	m := New()
	m.bpm = 120
	m.playing = true

	// Beat 1 is indicated as accented independent of color (currently: a
	// solid border, vs. beats 2-4's dashed border), so the accent survives
	// in a no-color/colorblind terminal.
	if pillBorderFor(1) == pillBorderFor(2) {
		t.Fatalf("expected beat 1 to be indicated as accented independent of color, but its border matches beats 2-4's")
	}
	for i := 2; i <= beatsPerMeasure; i++ {
		if pillBorderFor(i) != pillBorderFor(2) {
			t.Errorf("expected beats 2-4 to share the same (non-accented) border, beat %d differs", i)
		}
	}

	next, _ := m.Update(beatMsg{}) // beat 1 struck
	m = next.(Model)

	beat1Bg := m.beatPillBg(1)
	beat2Bg := m.beatPillBg(2) // unlit right now, but color differs when struck below

	next, _ = m.Update(beatMsg{}) // beat 2 struck
	m2 := next.(Model)
	beat2StruckBg := m2.beatPillBg(2)

	if beat1Bg == beat2StruckBg {
		t.Errorf("expected beat 1's struck color to differ from beats 2-4's struck color, both rendered as %v", beat1Bg)
	}
	_ = beat2Bg
}

// @ft:12
func TestJMirrorsDownForBPM(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "j")

	assertBPM(t, m, 119)
}

// @ft:13
func TestKMirrorsUpForBPM(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "k")

	assertBPM(t, m, 121)
}

// @ft:14
func TestShiftJMirrorsShiftDownForBPM(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "shift+j")

	assertBPM(t, m, 110)
}

// @ft:15
func TestShiftKMirrorsShiftUpForBPM(t *testing.T) {
	m := New()
	m.bpm = 120

	m = press(m, "shift+k")

	assertBPM(t, m, 130)
}
