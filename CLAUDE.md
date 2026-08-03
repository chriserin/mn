# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`mn` is a metronome TUI built with Bubble Tea (`charm.land/bubbletea/v2`) and
Lip Gloss (`charm.land/lipgloss/v2`). Single-binary Go app; `go run .` or
`go build -o mn .` then `./mn`.

## Commands

```
go build -o mn .          # build
go run .                  # run directly
go test ./...             # run all tests
go test -run TestName -v .   # run a single test
ft status                 # check design-scenario implementation status (see Phased workflow below)
```

Rendering/layout changes are hard to judge from test assertions alone — use
the preview harnesses below to eyeball actual output:

```
MN_PREVIEW=1 go test -run TestPreview -v .    # render Model.View() to stdout
```
Optional env vars for `TestPreview` (see preview_test.go for the full list):
`MN_WIDTH`, `MN_HEIGHT`, `MN_BPM`, `MN_PLAYING`, `MN_CURRENT_BEAT`,
`MN_TEMPO_TRAINING`, `MN_STEP_BPM`, `MN_STEP_INTERVAL`, `MN_TARGET_BPM`,
`MN_FOCUSED`.

```
go run ./cmd/bignumpreview [-n 120] [-scale 1,2,3]   # preview big-digit banner glyphs/scales
```

## Phased workflow

This project is built in explicit phases, and features are not implemented
ahead of their phase without discussion:

1. **Design** — a doc in `design/NN-title.md` is written and discussed before
   any implementation.
2. **Scenarios** — Gherkin scenarios are added to `.ft` files in `fts/`,
   tracked with the [`ft`](https://github.com/chriserin/ft) CLI (each
   scenario tagged `@ft:N`).
3. **Implementation** — scenarios marked `ready` in `ft list`/`ft status` are
   implemented test-first (write the failing test from the scenario, then
   make it pass).

`phases/phase-N.md` defines each phase's in-scope/out-of-scope boundary:
- Phase 1 (`phases/phase-1.md`): core metronome — BPM, start/stop, status
  bar, beat pulse, no audio/tempo-training/persistence.
- Phase 2 (`phases/phase-2.md`): tempo training — auto BPM stepping toward a
  target over time.
- Phase 3 (`phases/phase-3.md`): audio click, not yet implemented.
- Phase 4 (`phases/phase-4.md`): settings persistence, not yet implemented.

Check which phase a change belongs to before adding scope; out-of-scope items
listed in a phase doc are deferred deliberately, not oversights. `design/`
holds the rationale/mockups referenced from each phase doc (e.g.
`design/07-status-bar.md` for the status bar layout, `design/06-tempo-training-table.md`
for the tempo-training block layout).

## Architecture

Everything is one Bubble Tea `Model` (`model.go`) following the standard
Elm-style `Init`/`Update`/`View` loop; there is no other package for app
logic. Supporting packages exist only to factor out reusable, independently
testable rendering data:

- `internal/bignum` — ASCII-art digit glyphs (three hand-tuned scale tiers)
  and the `Render(n, scale)` compositor used for the big tempo-number banner.
  Kept separate so both the app and `cmd/bignumpreview` (a standalone preview
  CLI) can render identically without duplicating glyph data.
- `bigdigits.go` — thin wrapper (`renderBigNumber`) adapting `internal/bignum`
  to the main package.

### Timing engine

There is no persistent ticker. Each beat schedules the *next* tick by
recomputing `time.Minute / bpm` from the current BPM (`tickCmd` in
`model.go`), so BPM changes take effect on the very next beat and there's no
cumulative drift. `beatMsg` triggers `Model.advanceBeat`, which cycles
`currentBeat` through 1..4 (`beatsPerMeasure`) and, when tempo training is on,
tracks measures elapsed.

Tempo-training steps that land on beat 4 are deferred: `pendingTempoStep` is
set on beat 4 and only applied when beat 1 of the next measure lands
(`advanceBeat`), so the BPM readout and tick interval change at the start of
a measure, not its last beat. `stepTempoTraining` moves `bpm` by `stepBPM`
toward `targetBPM`, clamping so it can't overshoot.

`startBPM` captures `bpm` when play starts, and stopping reverts `bpm` to
`startBPM` — this undoes tempo-training drift from that run rather than
letting the next Start mirror wherever BPM drifted to.

### Focus-aware color model

The terminal's focus state (`tea.FocusMsg`/`BlurMsg`, tracked in
`Model.focused`) drives a global grayscale substitution: `Model.mutedColor`
and `Model.prominentColor` swap any hue-bearing color for one of two gray
tiers (`grayMuted`/`grayProminent`) whenever unfocused, so an unfocused `mn`
reads as a plain, non-attention-seeking UI. Every color used in `View()` goes
through one of these two functions rather than being applied directly — when
adding new colored UI, route it through `mutedColor`/`prominentColor` rather
than hardcoding a `lipgloss.Color`. Colors that are already gray/black/white
need no substitution.

Status bar and beat-pill colors deliberately use the 4-bit ANSI palette
(`lipgloss.Black`, `lipgloss.BrightBlack`, etc.), not fixed 256-color/hex
values, so they render using whatever the user's terminal theme assigns to
each slot instead of clashing with it.

### Layout

`View()` composes three vertically stacked blocks: the status bar (+ tempo
training panel if on), the big-digit BPM banner (centered in whatever space
is left, sized by `Model.bannerScale` to the largest glyph tier — `1`/`2`/`3`
from `internal/bignum` — that still fits `m.width`/`m.height`), and the 4
beat pills anchored to the bottom row. Most render helpers are pure functions
of `Model` fields (no side effects), which is what makes them straightforward
to unit test by constructing a `Model` and calling e.g. `m.renderStatusBar()`
or `m.tempoTrainingRow("Step")` directly.

## Testing conventions

- `model_test.go` drives behavior through `Update()` with a `key(name)` /
  `press(m, name)` helper pair that builds `tea.KeyPressMsg` values, then
  asserts against `View()` output or exported query helpers (e.g.
  `tempoIndicatorText`, `tempoTrainingRow`, `measureCounterText`) — prefer
  extending these helpers over re-deriving key messages inline.
- Gherkin scenarios in `fts/*.ft` (tagged `@ft:N`) are the source of truth
  for expected behavior; a new test should trace back to a `ready` scenario
  rather than inventing new behavior ahead of the phase/design docs.
- `preview_test.go`'s `TestPreview` and `cmd/bignumpreview` are visual-only
  harnesses, not correctness tests — skipped by default (`go test ./...`
  never triggers them) so they don't affect CI.
