package main

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// TestPreview renders the current View() to stdout for eyeballing layout/
// color changes, without needing a live terminal or a throwaway scratch
// file. It's a _test.go file, so `go build` never compiles it into the
// production binary — only `go test` sees it, and only when asked to run
// it explicitly (it's skipped otherwise, so it doesn't spam `go test ./...`
// output or fail CI on a missing terminal).
//
// Usage:
//
//	MN_PREVIEW=1 go test -run TestPreview -v .
//
// Optional env vars override the model's state (all default to New()'s
// defaults, stopped):
//
//	MN_WIDTH        terminal width, e.g. 80
//	MN_BPM          current BPM, e.g. 130
//	MN_PLAYING      "1" to mark the metronome as playing
//	MN_CURRENT_BEAT which beat (1-4) is currently struck, e.g. 2
//	MN_TEMPO_TRAINING "1" to turn tempo training on
//	MN_STEP_BPM, MN_STEP_INTERVAL, MN_TARGET_BPM tempo-training attributes
//	MN_FOCUSED      "0" to preview the unfocused (grayscaled) state
func TestPreview(t *testing.T) {
	if os.Getenv("MN_PREVIEW") == "" {
		t.Skip("set MN_PREVIEW=1 to render a preview, e.g. MN_PREVIEW=1 go test -run TestPreview -v .")
	}

	m := New()
	m.width = previewEnvInt("MN_WIDTH", 80)
	m.bpm = previewEnvInt("MN_BPM", m.bpm)
	m.playing = previewEnvBool("MN_PLAYING", false)
	m.currentBeat = previewEnvInt("MN_CURRENT_BEAT", 0)
	m.tempoTrainingOn = previewEnvBool("MN_TEMPO_TRAINING", false)
	m.stepBPM = previewEnvInt("MN_STEP_BPM", m.stepBPM)
	m.stepIntervalMeasures = previewEnvInt("MN_STEP_INTERVAL", m.stepIntervalMeasures)
	m.targetBPM = previewEnvInt("MN_TARGET_BPM", m.targetBPM)
	m.focused = previewEnvBool("MN_FOCUSED", true)

	fmt.Println(m.View().Content)
}

func previewEnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func previewEnvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v == "1"
}
