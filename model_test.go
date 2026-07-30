package main

import (
	"fmt"
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
	return fmt.Sprintf("♩ = %d BPM", bpm)
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
	assertStatusBar(t, m, "◥ STOPPED ◥◥ mn ◥")
}

// @ft:2
func TestStartTheMetronome(t *testing.T) {
	m := New()
	m = press(m, "space")

	assertStatusBar(t, m, "◥ PLAYING ◥◥ mn ◥")
}

// @ft:3
func TestStopTheMetronome(t *testing.T) {
	m := New()
	m.playing = true

	m = press(m, "space")

	assertStatusBar(t, m, "◥ STOPPED ◥◥ mn ◥")
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

	view := m.View().Content
	if !strings.Contains(view, "^") {
		t.Fatalf("expected a \"^\" caret above beat 1 at all times, got:\n%s", view)
	}

	next, _ := m.Update(beatMsg{}) // beat 1 struck
	m = next.(Model)

	beat1Rendered := m.beatDotStyled(1)
	beat2Rendered := m.beatDotStyled(2) // unlit right now, but style differs when struck below

	next, _ = m.Update(beatMsg{}) // beat 2 struck
	m2 := next.(Model)
	beat2StruckRendered := m2.beatDotStyled(2)

	if beat1Rendered == beat2StruckRendered {
		t.Errorf("expected beat 1's struck style to differ from beats 2-4's struck style, both rendered as %q", beat1Rendered)
	}
	_ = beat2Rendered

	viewAfterBeat1 := m.View().Content
	if !strings.Contains(viewAfterBeat1, "^") {
		t.Errorf("expected the beat-1 caret to remain visible while beat 1 is struck, got:\n%s", viewAfterBeat1)
	}
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
