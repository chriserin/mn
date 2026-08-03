package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// runCmd executes cmd (and, recursively, any tea.Batch'd sub-commands) for
// its side effects, without routing resulting messages back through
// Update() — these tests only care about which clicks playClick recorded.
func runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			runCmd(c)
		}
	}
}

// fireBeat sends a beatMsg (as the timing engine would on the next tick)
// and runs the resulting commands, so any click triggered by that beat is
// recorded by the currently-installed playClick spy.
func fireBeat(m Model) Model {
	next, cmd := m.Update(beatMsg{})
	runCmd(cmd)
	return next.(Model)
}

// pressAndRun is press() (model_test.go) plus running the resulting
// command, needed here (unlike other tests) because starting playback's
// immediate beat 1 triggers a click via the returned tea.Cmd.
func pressAndRun(m Model, keyName string) Model {
	next, cmd := m.Update(key(keyName))
	runCmd(cmd)
	return next.(Model)
}

// stubPlayClick installs a spy in place of the real playClick for the
// duration of a test, recording each call's accented flag, and restores
// the original afterward. It also stubs tickCmd to an instant no-op, since
// fireBeat executes the tea.Cmd returned from Update (including tickCmd)
// to trigger the click side effect, and the real tickCmd blocks for a
// full beat interval.
func stubPlayClick(t *testing.T) *[]bool {
	t.Helper()
	var calls []bool
	origPlayClick := playClick
	playClick = func(accented bool) {
		calls = append(calls, accented)
	}
	origTickCmd := tickCmd
	tickCmd = func(int) tea.Cmd { return func() tea.Msg { return nil } }
	t.Cleanup(func() {
		playClick = origPlayClick
		tickCmd = origTickCmd
	})
	return &calls
}

// @ft:71
func TestClickTriggeredOnEveryBeat(t *testing.T) {
	calls := stubPlayClick(t)
	m := New()
	m = pressAndRun(m, "space") // beat 1
	m = fireBeat(m)             // beat 2
	m = fireBeat(m)             // beat 3
	if len(*calls) != 3 {
		t.Fatalf("expected 3 clicks triggered, got %d", len(*calls))
	}
}

// @ft:72
func TestBeatOneClickIsAccented(t *testing.T) {
	calls := stubPlayClick(t)
	m := New()
	m = pressAndRun(m, "space") // beat 1, struck immediately
	if len(*calls) != 1 || !(*calls)[0] {
		t.Fatalf("expected beat 1's immediate click to be accented, got %v", *calls)
	}

	m = fireBeat(m) // beat 2
	m = fireBeat(m) // beat 3
	m = fireBeat(m) // beat 4
	m = fireBeat(m) // wraps to beat 1

	last := (*calls)[len(*calls)-1]
	if !last {
		t.Fatalf("expected beat 1 (after wrap) to be accented, got %v", *calls)
	}
}

// @ft:73
func TestBeatsTwoThroughFourClicksAreNotAccented(t *testing.T) {
	calls := stubPlayClick(t)
	m := New()
	m = pressAndRun(m, "space") // beat 1
	*calls = nil                // ignore beat 1's accented click

	m = fireBeat(m) // beat 2
	m = fireBeat(m) // beat 3
	m = fireBeat(m) // beat 4

	if len(*calls) != 3 {
		t.Fatalf("expected 3 clicks, got %d", len(*calls))
	}
	for i, accented := range *calls {
		if accented {
			t.Errorf("beat %d: expected non-accented click, got accented", i+2)
		}
	}
}

// @ft:74
func TestNoClickTriggeredWhileStopped(t *testing.T) {
	calls := stubPlayClick(t)
	m := New()
	m = fireBeat(m) // stopped: beatMsg is a no-op
	if len(*calls) != 0 {
		t.Fatalf("expected no clicks while stopped, got %v", *calls)
	}
}

// @ft:75
func TestPlayClickIsSilentWithoutAnAudioDevice(t *testing.T) {
	// No audio device is expected in the test environment; playClick must
	// degrade to a silent no-op (via ensureAudioInit/audioAvailable)
	// rather than erroring or panicking.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("playClick panicked: %v", r)
		}
	}()
	playClick(true)
	playClick(false)
}

func TestClickAccentedTracksBeatOne(t *testing.T) {
	m := New()
	m.currentBeat = 1
	if !m.clickAccented() {
		t.Errorf("expected clickAccented true on beat 1")
	}
	for _, b := range []int{2, 3, 4} {
		m.currentBeat = b
		if m.clickAccented() {
			t.Errorf("expected clickAccented false on beat %d", b)
		}
	}
}
