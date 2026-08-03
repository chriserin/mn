package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/chriserin/mn/internal/bignum"
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

	tempoSegBg = lipgloss.Black
	tempoSegFg = lipgloss.Cyan

	counterSegBg = lipgloss.Blue
	counterSegFg = lipgloss.White

	// tempoHintFg is the "tempo training (t)" hint's lettering: darker than
	// tempoSegFg so the hint reads as a low-key nudge rather than
	// competing with the tempo readout for attention, even though it
	// shares the same background.
	tempoHintFg = lipgloss.BrightBlack
)

// grayMuted and grayProminent are the two substitute colors used in place
// of any hue-bearing color while the terminal is unfocused (see
// Model.mutedColor/prominentColor): everything that would normally draw
// attention with color instead reads as a plain gray UI, so an unfocused
// mn doesn't compete for attention with whatever the user switched to.
// Colors that are already gray/black/white (e.g. appSegBg, tempoHintFg)
// need no substitution and are left alone regardless of focus.
var (
	grayMuted     = lipgloss.BrightBlack
	grayProminent = lipgloss.BrightWhite
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
	currentBeat int  // 0 = no beat struck yet; otherwise 1..beatsPerMeasure
	width       int  // terminal width, used to stretch the status bar edge to edge
	height      int  // terminal height, used to size the big-digit tempo banner; see Model.bannerScale
	focused     bool // whether the terminal currently has focus; see tea.FocusMsg/BlurMsg

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
		// Assumed focused until a BlurMsg says otherwise, since most
		// terminals/multiplexers report an initial FocusMsg only on actual
		// focus changes, not on startup.
		focused: true,
	}
}

// mutedColor returns c while focused, or grayMuted while unfocused: for
// colors that play a secondary/idle role (e.g. beats 2-4, the tempo
// training Start/Step/Interval values), so they read as the dimmer of the
// two gray tiers.
func (m Model) mutedColor(c color.Color) color.Color {
	if m.focused {
		return c
	}
	return grayMuted
}

// prominentColor returns c while focused, or grayProminent while
// unfocused: for colors that play a primary/attention-grabbing role (e.g.
// beat 1, the tempo readout, the Target value), so they read as the
// brighter of the two gray tiers even with their hue removed.
func (m Model) prominentColor(c color.Color) color.Color {
	if m.focused {
		return c
	}
	return grayProminent
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

// tickCmd is a package-level var, not a plain func, so tests can substitute
// an instant stub: its real implementation schedules a beatMsg after a
// real time.Duration, which would make tests that execute the returned
// tea.Cmd (e.g. to observe audio triggering) block for that duration.
var tickCmd = func(bpm int) tea.Cmd {
	interval := time.Minute / time.Duration(bpm)
	return tea.Tick(interval, func(time.Time) tea.Msg { return beatMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.FocusMsg:
		m.focused = true
		return m, nil
	case tea.BlurMsg:
		m.focused = false
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
				// first tick to elapse, so the beat indicator (and its
				// accented click) lights up the instant playback starts,
				// not a full beat late.
				m.currentBeat = 1
				return m, tea.Batch(tickCmd(m.bpm), clickCmd(true))
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
		return m, tea.Batch(tickCmd(tickBPM), clickCmd(m.clickAccented()))
	}
	return m, nil
}

// clickAccented reports whether the beat currently lit should play the
// accented (beat 1) click rather than the plain (beats 2-4) one.
func (m Model) clickAccented() bool {
	return m.currentBeat == 1
}

// clickCmd plays one beat's click as a side effect, reusing the same
// beatMsg that drives the visual pulse so audio and animation share one
// clock (see design/08-audio.md).
func clickCmd(accented bool) tea.Cmd {
	return func() tea.Msg {
		playClick(accented)
		return nil
	}
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
	top := []string{m.renderStatusBar()}
	if m.tempoTrainingOn {
		top = append(top, "", m.renderTempoTrainingBlocks())
	}
	topBlock := strings.Join(top, "\n")
	beats := m.renderBeats()

	// The banner sits between topBlock and beats, each separated by a
	// blank line, so what's left over vertically is m.height minus those
	// two fixed blocks and the two blank-line separators.
	availHeight := m.height - lineCount(topBlock) - 2 - lineCount(beats)
	scale := m.bannerScale(m.width, availHeight)
	banner := renderBigNumber(m.bpm, scale)

	lines := append([]string{}, top...)
	lines = append(lines, "")

	// Vertically center the banner between topBlock and beats, and anchor
	// beats to the bottom edge: any vertical slack left after sizing the
	// banner (availHeight minus what the banner actually used) is split
	// into blank lines above and below it, rather than left entirely below
	// — so the banner sits centered in the gap instead of hugging its top,
	// and beats still land on the last row rather than floating just under
	// the banner. Skipped when m.height is unknown (0, e.g. before the
	// first WindowSizeMsg or in tests that never send one), since there's
	// nothing to center or anchor against yet.
	if m.height > 0 {
		if slack := availHeight - lineCount(banner); slack > 0 {
			above := slack / 2
			lines = append(lines, make([]string, above)...)
		}
	}

	lines = append(lines, lipgloss.PlaceHorizontal(m.width, lipgloss.Center, banner))

	if m.height > 0 {
		if slack := availHeight - lineCount(banner); slack > 0 {
			below := slack - slack/2
			lines = append(lines, make([]string, below)...)
		}
	}

	lines = append(lines, "", beats)
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.ReportFocus = true
	return v
}

// lineCount returns the number of display rows in a possibly multi-line
// string.
func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

// bannerScale returns the largest banner tier (1..bignum.MaxScale) whose
// rendered digits still fit within availWidth x availHeight, so the tempo
// readout grows to fill extra room on a larger terminal but never overflows
// a small one. availHeight <= 0 (e.g. before the first WindowSizeMsg, or in
// tests that never send one) always falls back to the native 1x size.
func (m Model) bannerScale(availWidth, availHeight int) int {
	scale := 1
	for s := 2; s <= bignum.MaxScale; s++ {
		banner := renderBigNumber(m.bpm, s)
		w, h := lipgloss.Width(banner), lineCount(banner)
		if w > availWidth || h > availHeight {
			break
		}
		scale = s
	}
	return scale
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

// tempoIndicatorText returns the current-tempo readout shown in the middle
// of the status bar, e.g. "♩ 120 bpm".
func (m Model) tempoIndicatorText() string {
	return fmt.Sprintf("♩ %d bpm", m.bpm)
}

// renderStatusBar renders a vim-airline-style status bar spanning the full
// terminal width: a colored mode block (PLAYING/STOPPED) and app-name block
// on the left, the current-tempo readout centered in the middle, and a
// block pinned to the right edge — the measure counter while tempo training
// is on, or a "tempo training (t)" hint while it's off, so the toggle key
// is discoverable even before it's ever been used. The bar's background
// color fills the remaining gaps. See design/07-status-bar.md.
func (m Model) renderStatusBar() string {
	modeBg := m.mutedColor(stoppedBg)
	if m.playing {
		modeBg = m.prominentColor(playingBg)
	}

	left :=
		statusSegment(appName, appSegBg, appSegFg, wedgeAfter) +
			statusSegment(m.playingStatus(), modeBg, modeFg, wedgeAfter)

	var middle = ""
	if m.focused {
		middle = statusSegment(m.tempoIndicatorText(), tempoSegBg, m.prominentColor(tempoSegFg), wedgeAfter)
	}

	right := ""
	if m.tempoTrainingOn {
		right = statusSegment(m.measureCounterText(), appSegBg, counterSegFg, wedgeBefore)
	} else {
		right = statusSegment("tempo training (t)", tempoSegBg, tempoHintFg, wedgeBefore)
	}

	leftW, middleW, rightW := lipgloss.Width(left), lipgloss.Width(middle), lipgloss.Width(right)

	// Center the middle segment across the full bar width, but never let it
	// creep left of where the left group ends.
	middleStart := max((m.width-middleW)/2, leftW)
	fillBefore := max(middleStart-leftW, 0)
	fillAfter := max(m.width-leftW-fillBefore-middleW-rightW, 0)

	fillStyle := lipgloss.NewStyle().Background(statusBarBg)
	return left +
		fillStyle.Render(strings.Repeat(" ", fillBefore)) +
		middle +
		fillStyle.Render(strings.Repeat(" ", fillAfter)) +
		right
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

// Beat-pill colors: dim gray when idle, blue when struck, and a distinct
// orange for beat 1 when struck so the downbeat reads differently from
// beats 2-4 even by color alone.
//
// beatDimBg uses the 4-bit ANSI palette (lipgloss.BrightBlack) rather than
// a fixed 256-color value, matching the status bar's approach (see
// design/07-status-bar.md): it renders using whatever gray the user's
// terminal theme assigns to that slot instead of a fixed shade that may be
// washed out on light themes or too dark on others.
var (
	beatDimBg    = lipgloss.BrightBlack
	beatPlainBg  = lipgloss.Color("39")
	beatAccentBg = lipgloss.Color("208")
)

// beatPillBg returns the background color for beat slot i (1-indexed).
func (m Model) beatPillBg(i int) color.Color {
	if i != m.currentBeat {
		return beatDimBg // already gray; no substitution needed when unfocused
	}
	if i == 1 {
		return m.prominentColor(beatAccentBg)
	}
	// prominentColor, not mutedColor: while focused this still renders in
	// beatPlainBg (its own distinct color, unchanged from before), but
	// while unfocused it converges on the same grayProminent tier as beat
	// 1's struck color instead of a separate muted gray. Struck vs. idle
	// needs to stay a big brightness jump in grayscale, or a struck
	// beat 2-4 reads as indistinguishable from an idle one (both would
	// otherwise land on a similar dark gray).
	return m.prominentColor(beatPlainBg)
}

const (
	beatPillGap      = 2 // space left between (and around) beat pills
	minBeatPillWidth = 5 // smallest a pill can shrink to before losing its rounded shape
)

// beatPillLayout divides the terminal width evenly across beatsPerMeasure
// pills (with beatPillGap of space between and around them), so the pills
// stretch across the full width. Any remainder from the division is spread
// across the first few pills rather than left as slack on one side.
func (m Model) beatPillLayout() (pillWidths [beatsPerMeasure]int) {
	totalGap := beatPillGap * (beatsPerMeasure + 1)
	avail := m.width - totalGap
	if minTotal := beatsPerMeasure * minBeatPillWidth; avail < minTotal {
		avail = minTotal
	}
	base, extra := avail/beatsPerMeasure, avail%beatsPerMeasure
	for i := range pillWidths {
		pillWidths[i] = base
		if i < extra {
			pillWidths[i]++
		}
	}
	return pillWidths
}

// pillDashedBorder is pillBorderFor's border for the unaccented beats
// (2-4): the same "─"/"│" strokes lipgloss.RoundedBorder already uses for
// the solid accented pill (so the line weight matches and reads just as
// clearly at beatDimBg's low contrast), alternated with a blank space to
// open up visible gaps. The rounded corners (╭╮╰╯) are kept as-is so the
// unaccented pills stay the same silhouette as the accented one.
var pillDashedBorder = lipgloss.Border{
	Top:         "─ ",
	Bottom:      "─ ",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}

// pillBorderFor returns the border style for beat slot i: a solid rounded
// border for beat 1 (the accented downbeat), a dashed rounded border for
// beats 2-4. This is a fixed property of the beat's position, not its
// struck state — beat 1's border stays the one visually unique thing in
// the row (see pillStruckTexture for how a struck beat 2-4 is indicated
// instead).
func pillBorderFor(i int) lipgloss.Border {
	if i == 1 {
		return lipgloss.RoundedBorder()
	}
	return pillDashedBorder
}

// pillStruckTexture is the fill used for beats 2-4 the instant they're
// struck: a dotted/shaded block (not a flat solid fill) rendered in the
// terminal's default foreground over the pill's own background color, so
// struck reads as a distinct texture rather than only a brighter color —
// this is what keeps a struck beat 2-4 from looking merely like a slightly
// brighter idle beat once color is desaturated in grayscale (unfocused)
// mode. Beat 1 never uses this: its permanently-solid border already makes
// it unique, so its fill stays flat.
const pillStruckTexture = "░"

// renderBeatPill renders beat slot i as a solid-colored capsule using
// pillBorderFor's border, with the border colored to match the fill.
// BorderBackground is deliberately left unset: setting it paints the space
// around each corner glyph too, which turns every corner cell into a
// solid-colored square and hides the rounding (see cmd/pillpreview's
// boxDrawingRounded, which omits it for the same reason).
//
// width is passed straight to Style.Width, not reduced by the border's 2
// columns first: lipgloss's Width sets the total rendered block size
// (border included), so subtracting the border first ends up rendering 2
// columns narrower than requested, which is what was making the pills fall
// short of the full terminal width.
func (m Model) renderBeatPill(i, width int) string {
	bg := m.beatPillBg(i)

	fill := ""
	if i != 1 && i == m.currentBeat {
		// width-2, not width: width is the total block size border
		// included (see the Style.Width note below), but the fill only
		// occupies the inner content area, between the left and right
		// border columns. Passing the full width here wrapped the ░ fill
		// onto a second line, corrupting the row's height and alignment.
		fill = strings.Repeat(pillStruckTexture, max(width-2, 1))
	}

	return lipgloss.NewStyle().
		Width(width).
		Background(bg).
		BorderStyle(pillBorderFor(i)).
		BorderForeground(bg).
		Render(fill)
}

// renderBeats renders the 4 beat pills. Beat 1's solid border vs. beats
// 2-4's dashed border (see pillBorderFor) is itself a color-agnostic accent
// marker, so a separate "^" caret above beat 1 is no longer needed.
func (m Model) renderBeats() string {
	pillWidths := m.beatPillLayout()
	gap := strings.Repeat(" ", beatPillGap)

	pieces := make([]string, 0, beatsPerMeasure*2+1)
	pieces = append(pieces, gap)
	for i := 1; i <= beatsPerMeasure; i++ {
		pieces = append(pieces, m.renderBeatPill(i, pillWidths[i-1]), gap)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, pieces...)
}

// tempoRow is one attribute in the tempo-training display: a label/value/
// unit triple, plus the keys that adjust it (e.g. "k/j/K/J"), shown next to
// the label so the controls are visible without a separate legend.
type tempoRow struct {
	label string
	value string
	unit  string
	keys  string
}

// displayLabel returns the row's label with its adjustment keys appended,
// e.g. "Step [/]", for measuring the label line's plain-text width.
func (r tempoRow) displayLabel() string {
	return fmt.Sprintf("%s %s", r.label, r.keys)
}

// renderLabelLine centers the row's label+keys text within width, styling
// the label and the keys hint in separate colors (tempoBlockLabelStyle,
// tempoBlockKeysStyle) so the keybinding reads as distinct from the label
// rather than blending into it.
func (r tempoRow) renderLabelLine(width int) string {
	text := r.displayLabel()
	pad := width - len(text)
	if pad < 0 {
		pad = 0
	}
	left, right := pad/2, pad-pad/2
	return strings.Repeat(" ", left) +
		tempoBlockLabelStyle.Render(r.label) + " " + tempoBlockKeysStyle.Render(r.keys) +
		strings.Repeat(" ", right)
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

// tempoTrainingHeader summarizes on/off/target-reached status, for tests.
func (m Model) tempoTrainingHeader() string {
	return "Tempo Training: " + m.tempoTrainingState()
}

// tempoTrainingRows returns the tempo-training attributes in display order:
// Start, Step, Interval, Target. Only rendered while tempo training is on.
func (m Model) tempoTrainingRows() []tempoRow {
	startBPM := m.bpm
	if m.playing {
		startBPM = m.startBPM
	}

	return []tempoRow{
		{"Start", fmt.Sprintf("%d", startBPM), "bpm", "j/k/J/K"},
		{"Step", fmt.Sprintf("%d", m.stepBPM), "bpm", "[/]"},
		{"Interval", fmt.Sprintf("%d", m.stepIntervalMeasures), pluralize(m.stepIntervalMeasures, "measure", "measures"), "{/}"},
		{"Target", fmt.Sprintf("%d", m.targetBPM), "bpm", "n/m/N/M"},
	}
}

// tempoTrainingRow looks up a single row's "value unit" text by label, for
// tests.
func (m Model) tempoTrainingRow(label string) string {
	for _, r := range m.tempoTrainingRows() {
		if r.label == label {
			return strings.TrimSpace(r.value + " " + r.unit)
		}
	}
	return ""
}

// tempoTrainingBlockGap is the space left between (and around) attribute
// blocks when the terminal is too narrow to spread them across its full
// width.
const tempoTrainingBlockGap = 4

// centerPad centers s within width by padding both sides with spaces,
// leaving s untouched if it's already at or beyond width.
func centerPad(s string, width int) string {
	pad := width - len(s)
	if pad <= 0 {
		return s
	}
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// tempoBlockLabelStyle styles a block's label text brightly, since it's the
// primary thing being named. tempoBlockKeysStyle styles the adjustment-keys
// hint next to it in a muted gray, so the keybinding reads as secondary
// reference material rather than competing with the label for attention.
// tempoBlockValueStyle styles the value+unit row for every attribute except
// Target, which uses accentStyle instead to draw the eye to the goal tempo
// training is working toward.
var tempoBlockLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightWhite)
var tempoBlockKeysStyle = dimStyle
var tempoBlockValueStyle = plainStyle

// tempoBlockValueStyleFor returns the value-row style for the attribute
// with the given label, substituting a gray tier for its color while
// unfocused (Target's accent color is prominent, the rest are muted).
func (m Model) tempoBlockValueStyleFor(label string) lipgloss.Style {
	if label == "Target" {
		return accentStyle.Foreground(m.prominentColor(lipgloss.Color("208")))
	}
	return tempoBlockValueStyle.Foreground(m.mutedColor(lipgloss.Color("39")))
}

// renderTempoTrainingBlocks renders each tempo-training attribute (Start,
// Step, Interval, Target) as a block of 2 centered rows (label, then value
// and unit together), spaced evenly across the terminal width. Only
// rendered while tempo training is on.
func (m Model) renderTempoTrainingBlocks() string {
	rows := m.tempoTrainingRows()

	valueUnit := make([]string, len(rows))
	blockWidth := 0
	for i, r := range rows {
		valueUnit[i] = strings.TrimSpace(r.value + " " + r.unit)
		blockWidth = max(blockWidth, len(r.displayLabel()), len(valueUnit[i]))
	}

	blocks := make([]string, len(rows))
	for i, r := range rows {
		blocks[i] = r.renderLabelLine(blockWidth) + "\n" +
			m.tempoBlockValueStyleFor(r.label).Render(centerPad(valueUnit[i], blockWidth))
	}

	totalBlocksWidth := blockWidth * len(blocks)
	gapWidth := tempoTrainingBlockGap
	if m.width > totalBlocksWidth {
		gapWidth = (m.width - totalBlocksWidth) / (len(blocks) + 1)
	}
	gap := strings.Repeat(" ", gapWidth)

	pieces := make([]string, 0, len(blocks)*2+1)
	pieces = append(pieces, gap)
	for _, b := range blocks {
		pieces = append(pieces, b, gap)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, pieces...)
}
