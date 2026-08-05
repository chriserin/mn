package main

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestMain stubs ensureAudioInit for the whole test binary: no test in
// this package should touch a real audio device (design/08-audio.md
// Testing). This used to be automatic — playClick only ever ran inside a
// tea.Cmd tests didn't execute — but ensureAudioInit now runs synchronously
// inside Update's "space" case (see beginPlayback), so every test that
// presses space needs this stub, not just audio-specific ones. Leaving
// ensureAudioInit stubbed to a no-op keeps audioAvailable at its zero
// value (false) for the whole run, so all "space"-triggered playback in
// every test takes the wall-clock tickCmd fallback path.
func TestMain(m *testing.M) {
	ensureAudioInit = func() {}
	os.Exit(m.Run())
}

// @ft:74
func TestNoBeatAdvanceWhileStopped(t *testing.T) {
	m := New()
	next, _ := m.Update(beatMsg{})
	m = next.(Model)
	if m.currentBeat != 0 {
		t.Fatalf("expected currentBeat to stay 0 while stopped, got %d", m.currentBeat)
	}
}

// @ft:75
func TestEnsureAudioInitDegradesGracefullyWithoutADevice(t *testing.T) {
	// Not exercising the real ensureAudioInit here (see TestMain) — this
	// asserts the fallback path itself (beginPlayback/nextBeatCmd when
	// audioAvailable is false) doesn't panic and returns a usable Cmd,
	// which is what "no audio device" actually degrades to at runtime.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("beginPlayback panicked: %v", r)
		}
	}()
	origTickCmd := tickCmd
	tickCmd = func(int) tea.Cmd { return func() tea.Msg { return nil } }
	defer func() { tickCmd = origTickCmd }()

	if cmd := beginPlayback(120); cmd == nil {
		t.Fatal("expected a non-nil fallback Cmd")
	}
	if cmd := nextBeatCmd(120); cmd == nil {
		t.Fatal("expected a non-nil fallback Cmd")
	}
}
