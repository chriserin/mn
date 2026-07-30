package main

import (
	"fmt"
	"image/color"
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

// Status bar colors, powerline-style. Each segment is a solid color block;
// the arrow between two segments is drawn in the outgoing segment's
// background color so it reads as a seamless wedge rather than a gap.
//
// Colors are the 4-bit ANSI palette (0-15), not fixed 256-color/hex values,
// so they render using whatever colors the user's terminal theme assigns to
// each slot rather than a fixed set that can clash with it.
// See design/07-status-bar.md.
var (
	statusBarBg = lipgloss.Black // fill between the left and right segment groups

	playingBg = lipgloss.Green
	stoppedBg = lipgloss.Red
	modeFg    = lipgloss.Black

	appSegBg = lipgloss.BrightBlack
	appSegFg = lipgloss.White

	counterSegBg = lipgloss.Blue
	counterSegFg = lipgloss.White
)

// Right-angle triangle wedges (Unicode Geometric Shapes block), not the
// Nerd Font powerline arrows, so the bar renders correctly without a
// patched font.
const (
	powerlineRight = "◥" // trails left-aligned segments, pointing right
	powerlineLeft  = "◤" // leads right-aligned segments, pointing left
)

// appName is shown at the start of the status bar. See design/07-status-bar.md.
const appName = "mn"

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
	width       int // terminal width, used to stretch the status bar edge to edge

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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "space":
			m.playing = !m.playing
			if m.playing {
				m.startBPM = m.bpm
				// Strike beat 1 immediately rather than waiting for the
				// first tick to elapse, so the beat indicator lights up
				// the instant playback starts, not a full beat late.
				m.currentBeat = 1
				return m, tickCmd(m.bpm)
			}
			// Revert any drift tempo training caused this run, rather than
			// letting Start mirror wherever BPM ended up.
			m.bpm = m.startBPM
			m.currentBeat = 0
			m.measuresSinceStep = 0
			m.pendingTempoStep = false
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
		}
	case m.currentBeat == 1 && m.pendingTempoStep:
		m.stepTempoTraining()
		m.pendingTempoStep = false
		m.measuresSinceStep = 0
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
		m.renderStatusBar(),
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

// playingStatus returns "PLAYING" or "STOPPED".
func (m Model) playingStatus() string {
	if m.playing {
		return "PLAYING"
	}
	return "STOPPED"
}

// statusSegment renders one powerline block: padded text on a solid
// background, followed by a wedge in that same background color (so it
// bleeds into whatever comes next). dir controls which side the wedge is
// drawn on and which glyph is used, so the same helper builds both the
// left-aligned and right-aligned segment groups.
type wedgeDir int

const (
	wedgeAfter  wedgeDir = iota // left-hand group: text, then a right-pointing wedge
	wedgeBefore                 // right-hand group: a left-pointing wedge, then text
)

func statusSegment(text string, bg, fg color.Color, dir wedgeDir) string {
	block := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1).Render(text)
	// The wedge takes the segment's own color as its background, not its
	// foreground, so it renders as a solid-colored triangle (matching the
	// block it belongs to) rather than a colored glyph on the terminal's
	// default background.
	if dir == wedgeAfter {
		wedgeStyle := lipgloss.NewStyle().Background(bg).Foreground(statusBarBg)
		return wedgeStyle.Reverse(true).Render(powerlineRight) + block + wedgeStyle.Render(powerlineRight)
	} else {
		wedgeStyle := lipgloss.NewStyle().Background(bg).Foreground(statusBarBg)
		return wedgeStyle.Reverse(false).Render(powerlineLeft) + block + wedgeStyle.Reverse(true).Render(powerlineLeft)
	}
}

// renderStatusBar renders a vim-airline-style status bar spanning the full
// terminal width: a colored mode block (PLAYING/STOPPED) and app-name block
// on the left, a measure counter block pinned to the right edge (while
// tempo training is on), and the bar's background color filling the gap
// between them. See design/07-status-bar.md.
func (m Model) renderStatusBar() string {
	modeBg := stoppedBg
	if m.playing {
		modeBg = playingBg
	}

	left := statusSegment(m.playingStatus(), modeBg, modeFg, wedgeAfter) +
		statusSegment(appName, appSegBg, appSegFg, wedgeAfter)

	right := ""
	if m.tempoTrainingOn {
		right = statusSegment(m.measureCounterText(), appSegBg, counterSegFg, wedgeBefore)
	}

	fillWidth := max(m.width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	fill := lipgloss.NewStyle().Background(statusBarBg).Render(strings.Repeat(" ", fillWidth))

	return left + fill + right
}

// measureCounterText returns the "N/M" measure counter shown while tempo
// training is on. A measure completing (beat 4) bumps measuresSinceStep
// right away so the interval threshold can be checked, but the displayed
// count shouldn't advance until beat 1 of the next measure actually lands.
func (m Model) measureCounterText() string {
	display := m.measuresSinceStep + 1
	if m.currentBeat == beatsPerMeasure {
		display = m.measuresSinceStep
	}
	return fmt.Sprintf("%d/%d", min(display, m.stepIntervalMeasures), m.stepIntervalMeasures)
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
