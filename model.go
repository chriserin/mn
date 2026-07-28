package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	minBPM          = 20
	maxBPM          = 300
	defaultBPM      = 120
	smallStep       = 1
	largeStep       = 10
	beatsPerMeasure = 4
)

var (
	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	plainStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// shapeStopped and shapePlaying are 3-row ASCII glyphs shown in the status
// banner, aspect-ratio-corrected so a terminal cell (roughly 2x taller than
// wide) still reads as a square/triangle. See design/03-status-banner-variations.md.
var shapeStopped = [3]string{
	"██████",
	"██████",
	"██████",
}

var shapePlaying = [3]string{
	"██◥   ",
	"██████",
	"██◢   ",
}

// letterStopped and letterPlaying are figlet `mini` renderings of the words,
// hardcoded so the app doesn't depend on the figlet binary at runtime. See
// design/03-status-banner-variations.md variant K.
var letterStopped = [3]string{
	` ______  _  _  _ _`,
	`(_  |/ \|_)|_)|_| \`,
	`__) |\_/|  |  |_|_/`,
}

var letterPlaying = [3]string{
	` _        ___     __`,
	`|_)|  /\\_/| |\ |/__`,
	`|  |_/--\|_|_| \|\_|`,
}

// beatMsg is emitted once per beat by the timing engine. Recomputing the
// next tick's duration from the current BPM each time (rather than using a
// persistent ticker) avoids cumulative drift and lets BPM changes take
// effect on the very next beat.
type beatMsg struct{}

// Model is the metronome's Bubble Tea model.
type Model struct {
	bpm         int
	playing     bool
	currentBeat int // 0 = no beat struck yet; otherwise 1..beatsPerMeasure
}

// New returns a Model with phase-1 defaults: stopped, 120 BPM.
func New() Model {
	return Model{bpm: defaultBPM}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func tickCmd(bpm int) tea.Cmd {
	interval := time.Minute / time.Duration(bpm)
	return tea.Tick(interval, func(time.Time) tea.Msg { return beatMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "space":
			m.playing = !m.playing
			if m.playing {
				return m, tickCmd(m.bpm)
			}
			m.currentBeat = 0
			return m, nil
		case "up":
			m.bpm = clamp(m.bpm+smallStep, minBPM, maxBPM)
		case "down":
			m.bpm = clamp(m.bpm-smallStep, minBPM, maxBPM)
		case "shift+up":
			m.bpm = clamp(m.bpm+largeStep, minBPM, maxBPM)
		case "shift+down":
			m.bpm = clamp(m.bpm-largeStep, minBPM, maxBPM)
		}
		return m, nil
	case beatMsg:
		if !m.playing {
			return m, nil
		}
		m.currentBeat = m.currentBeat%beatsPerMeasure + 1
		return m, tickCmd(m.bpm)
	}
	return m, nil
}

func (m Model) View() tea.View {
	content := strings.Join([]string{
		m.renderBanner(),
		"",
		fmt.Sprintf("♩ = %d BPM", m.bpm),
		"",
		m.renderBeats(),
	}, "\n")
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderBanner() string {
	shape, letters, word := shapeStopped, letterStopped, "STOPPED"
	if m.playing {
		shape, letters, word = shapePlaying, letterPlaying, "PLAYING"
	}
	_ = word // word is embedded in the letters glyph itself
	lines := make([]string, 3)
	for i := range lines {
		lines[i] = shape[i] + " " + letters[i]
	}
	return strings.Join(lines, "\n")
}

// beatDotStyled renders the dot for beat slot i (1-indexed), applying the
// accent style when beat 1 is currently struck and a plain style for any
// other struck beat, so beat 1 is visually distinguishable from beats 2-4.
func (m Model) beatDotStyled(i int) string {
	if i != m.currentBeat {
		return dimStyle.Render("○")
	}
	if i == 1 {
		return accentStyle.Render("●")
	}
	return plainStyle.Render("●")
}

const beatSeparator = "   "

func (m Model) renderBeats() string {
	dots := make([]string, beatsPerMeasure)
	for i := 1; i <= beatsPerMeasure; i++ {
		dots[i-1] = m.beatDotStyled(i)
	}
	dotsLine := strings.Join(dots, beatSeparator)
	// Plain width per slot is 1 rune, so the caret (aligned above beat 1)
	// is followed by enough spaces to span the remaining slots without
	// needing to measure the ANSI-styled dotsLine.
	plainWidth := beatsPerMeasure + (beatsPerMeasure-1)*len(beatSeparator)
	caretLine := "^" + strings.Repeat(" ", plainWidth-1)
	return caretLine + "\n" + dotsLine
}
