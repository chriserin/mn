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

	defaultStepBPM   = 10
	minStepBPM       = 1
	maxStepBPM       = 20
	defaultInterval  = 8
	minInterval      = 1
	maxInterval      = 32
	defaultTargetBPM = 180
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

	tempoTrainingOn      bool
	stepBPM              int
	stepIntervalMeasures int
	targetBPM            int
	startBPM             int // BPM captured when the current run started playing
	measuresSinceStep    int
	pendingTempoStep     bool // a step landed on beat 4; apply it when beat 1 lands
}

// New returns a Model with phase-1/phase-2 defaults: stopped, 120 BPM,
// tempo training off with default step/interval/target.
func New() Model {
	return Model{
		bpm:                  defaultBPM,
		stepBPM:              defaultStepBPM,
		stepIntervalMeasures: defaultInterval,
		targetBPM:            defaultTargetBPM,
		startBPM:             defaultBPM,
	}
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
				m.startBPM = m.bpm
				return m, tickCmd(m.bpm)
			}
			// Revert any drift tempo training caused this run, rather than
			// letting Start mirror wherever BPM ended up.
			m.bpm = m.startBPM
			m.currentBeat = 0
			return m, nil
		case "up", "k":
			m.bpm = clamp(m.bpm+smallStep, minBPM, maxBPM)
		case "down", "j":
			m.bpm = clamp(m.bpm-smallStep, minBPM, maxBPM)
		case "shift+up", "K":
			m.bpm = clamp(m.bpm+largeStep, minBPM, maxBPM)
		case "shift+down", "J":
			m.bpm = clamp(m.bpm-largeStep, minBPM, maxBPM)
		case "t":
			m.tempoTrainingOn = !m.tempoTrainingOn
			if m.tempoTrainingOn {
				// Start counting a fresh full interval from the moment
				// training is (re-)enabled, rather than resuming a count
				// that may have accrued while it was off.
				m.measuresSinceStep = 0
			}
		case "[":
			m.stepBPM = clamp(m.stepBPM-1, minStepBPM, maxStepBPM)
		case "]":
			m.stepBPM = clamp(m.stepBPM+1, minStepBPM, maxStepBPM)
		case "{":
			m.stepIntervalMeasures = clamp(m.stepIntervalMeasures-1, minInterval, maxInterval)
		case "}":
			m.stepIntervalMeasures = clamp(m.stepIntervalMeasures+1, minInterval, maxInterval)
		case "n":
			m.targetBPM = clamp(m.targetBPM-smallStep, minBPM, maxBPM)
		case "N":
			m.targetBPM = clamp(m.targetBPM-largeStep, minBPM, maxBPM)
		case "m":
			m.targetBPM = clamp(m.targetBPM+smallStep, minBPM, maxBPM)
		case "M":
			m.targetBPM = clamp(m.targetBPM+largeStep, minBPM, maxBPM)
		}
		return m, nil
	case beatMsg:
		if !m.playing {
			return m, nil
		}
		m, tickBPM := m.advanceBeat()
		return m, tickCmd(tickBPM)
	}
	return m, nil
}

// advanceBeat moves to the next beat. A tempo-training step that lands on
// beat 4 (the last beat of the measure, and the natural "measure complete"
// signal) is not applied immediately — it's marked pending and only applied
// once beat 1 of the next measure lands. This means the BPM readout itself,
// and the tick interval that follows, don't change until beat 1: a tempo
// change takes effect starting at the first beat of the new measure, not
// the last beat of the old one.
func (m Model) advanceBeat() (Model, int) {
	m.currentBeat = m.currentBeat%beatsPerMeasure + 1
	switch {
	case m.currentBeat == beatsPerMeasure && m.tempoTrainingOn:
		m.measuresSinceStep++
		if m.measuresSinceStep >= m.stepIntervalMeasures {
			m.pendingTempoStep = true
			m.measuresSinceStep = 0
		}
	case m.currentBeat == 1 && m.pendingTempoStep:
		m.stepTempoTraining()
		m.pendingTempoStep = false
	}
	return m, m.bpm
}

// stepTempoTraining moves bpm by stepBPM toward targetBPM, clamping so it
// never overshoots. If bpm already equals targetBPM, it's a no-op.
func (m *Model) stepTempoTraining() {
	switch {
	case m.targetBPM > m.bpm:
		m.bpm += m.stepBPM
		if m.bpm > m.targetBPM {
			m.bpm = m.targetBPM
		}
	case m.targetBPM < m.bpm:
		m.bpm -= m.stepBPM
		if m.bpm < m.targetBPM {
			m.bpm = m.targetBPM
		}
	}
}

func (m Model) View() tea.View {
	lines := []string{
		m.renderBanner(),
		"",
		fmt.Sprintf("♩ = %d BPM", m.bpm),
		"",
		m.renderBeats(),
		"",
		m.tempoTrainingHeader(),
	}
	if m.tempoTrainingOn {
		lines = append(lines, m.renderTempoTrainingTable())
	}
	v := tea.NewView(strings.Join(lines, "\n"))
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

// tempoRow is one label/value pair in the tempo-training table.
type tempoRow struct {
	label string
	value string
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// tempoTrainingState returns "off", "on", or "target reached (N bpm)".
func (m Model) tempoTrainingState() string {
	if !m.tempoTrainingOn {
		return "off"
	}
	if m.bpm == m.targetBPM {
		return fmt.Sprintf("target reached (%d bpm)", m.bpm)
	}
	return "on"
}

// tempoTrainingHeader is always rendered, regardless of whether tempo
// training is on, so on/off/target-reached status is visible at a glance.
// See design/06-tempo-training-table.md option E.
func (m Model) tempoTrainingHeader() string {
	return "Tempo Training: " + m.tempoTrainingState()
}

// tempoTrainingRows returns the tempo-training table's rows in display
// order: Start, Step, Interval, Target. Only rendered while tempo training
// is on (see tempoTrainingHeader for the always-visible on/off status).
func (m Model) tempoTrainingRows() []tempoRow {
	startBPM := m.bpm
	if m.playing {
		startBPM = m.startBPM
	}

	return []tempoRow{
		{"Start", fmt.Sprintf("%d bpm", startBPM)},
		{"Step", fmt.Sprintf("%d bpm", m.stepBPM)},
		{"Interval", fmt.Sprintf("%d %s", m.stepIntervalMeasures, pluralize(m.stepIntervalMeasures, "measure", "measures"))},
		{"Target", fmt.Sprintf("%d bpm", m.targetBPM)},
	}
}

// tempoTrainingRow looks up a single row's value by label, for tests.
func (m Model) tempoTrainingRow(label string) string {
	for _, r := range m.tempoTrainingRows() {
		if r.label == label {
			return r.value
		}
	}
	return ""
}

func (m Model) renderTempoTrainingTable() string {
	rows := m.tempoTrainingRows()
	labelWidth, valueWidth := 0, 0
	for _, r := range rows {
		labelWidth = max(labelWidth, len(r.label))
		valueWidth = max(valueWidth, len(r.value))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "┌%s┬%s┐\n", strings.Repeat("─", labelWidth+2), strings.Repeat("─", valueWidth+2))
	for _, r := range rows {
		fmt.Fprintf(&b, "│ %-*s │ %-*s │\n", labelWidth, r.label, valueWidth, r.value)
	}
	fmt.Fprintf(&b, "└%s┴%s┘", strings.Repeat("─", labelWidth+2), strings.Repeat("─", valueWidth+2))
	return b.String()
}
