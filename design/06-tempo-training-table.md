# Tempo Training Display — Tabular Options

Replacing the single status line (`Tempo Training: off, step 10 bpm / 8
measures, target 180 bpm`) with a table, since it's grown to four distinct
attributes (state, step, interval, target) that are awkward to scan as one
run-on sentence. `lipgloss/v2` ships a `table` package we can use directly
for implementation once a layout is picked.

## A. Column-per-attribute, single data row

```
┌────────┬────────┬──────────┬──────────┐
│ State  │ Step   │ Interval │ Target   │
├────────┼────────┼──────────┼──────────┤
│ off    │ 10 bpm │ 8 measures│ 180 bpm  │
└────────┴────────┴──────────┴──────────┘
```

While playing, at 130 bpm with target reached:

```
┌────────────────┬────────┬──────────┬──────────┐
│ State          │ Step   │ Interval │ Target   │
├────────────────┼────────┼──────────┼──────────┤
│ target reached │ 10 bpm │ 8 measures│ 130 bpm  │
└────────────────┴────────┴──────────┴──────────┘
```

Reads naturally left-to-right; the State column widens/shifts the whole
table depending on content length ("off" vs "target reached"), which means
the table's width isn't fixed across states — a minor layout-stability
concern (compare to phase 1's fixed-height banner problem).

## B. Key/value rows (label column + value column) — chosen

```
┌──────────┬───────────┐
│ Start    │ 120 bpm   │
│ State    │ off       │
│ Step     │ 10 bpm    │
│ Interval │ 8 measures│
│ Target   │ 180 bpm   │
└──────────┴───────────┘
```

Fixed-width label column keeps this table's width constant regardless of
state — solves option A's resizing concern. Taller (6 rows vs 2), but in
a TUI with vertical room to spare that's a reasonable trade.

`Interval` reads the full word "measures" (not abbreviated "meas") for
clarity, e.g. "8 measures".

A `Start` row was added: the BPM the current tempo-training run began at.
While the metronome is stopped, `Start` mirrors the live BPM readout — so
adjusting BPM with `up`/`down`/`shift+up`/`shift+down` while stopped updates
`Start` too, since there's no run in progress yet and `Start` just shows
"the tempo you'd begin from." The moment the metronome starts playing,
`Start` is captured and held fixed at that value for the rest of the run,
even as tempo training steps the live BPM away from it — giving a fixed
reference point to see how far training has moved you from where you began.

The relationship is one-directional: `Start` never reverts to match a
drifted BPM. Instead, when the metronome *stops*, the live BPM reverts to
`Start` — undoing whatever tempo training did during that run, so the next
run begins from the same tempo the previous one did unless the user
deliberately changes BPM (or `Start` itself, indirectly, via BPM keys) while
stopped. Stopping and restarting without touching BPM in between reproduces
the same `Start` value; adjusting BPM while stopped sets a new `Start` for
the next run, per the mirroring rule above.

## E. Header + conditional table (final)

Splitting **B** further: `State` moves out of the table into a standalone
header line that is *always* rendered (so on/off status is visible at a
glance without needing to enable training first), while the rest of the
table (`Start`, `Step`, `Interval`, `Target`) is only rendered when tempo
training is on:

```
Tempo Training: off
```

```
Tempo Training: on
┌──────────┬───────────┐
│ Start    │ 120 bpm   │
│ Step     │ 10 bpm    │
│ Interval │ 8 measures│
│ Target   │ 180 bpm   │
└──────────┴───────────┘
```

```
Tempo Training: target reached (130 bpm)
┌──────────┬───────────┐
│ Start    │ 120 bpm   │
│ Step     │ 10 bpm    │
│ Interval │ 8 measures│
│ Target   │ 180 bpm   │
└──────────┴───────────┘
```

Rationale: the four detail rows (`Start`/`Step`/`Interval`/`Target`) are
only meaningful once training is actually running or about to run — showing
an empty-feeling table by default added visual noise for a feature that's
off most of the time. The header is cheap (one line) and gives the at-a-
glance on/off/target-reached status B was designed for, while the table
only appears when there's something to look at. Note: `Start`/`Step`/
`Interval`/`Target` remain adjustable via their keys (`n`/`m`/`[`/`]`/`{`/`}`)
even while training is off, in memory — they just aren't rendered until
training is turned on, when the table appears showing their current values.

## C. Compact single-line, column-aligned (no borders)

Keeps the one-line footprint of the original status text but aligns values
into fixed-width fields instead of a comma-separated sentence:

```
State: off              Step: 10 bpm   Interval: 8 meas   Target: 180 bpm
```

Least visually distinct as "a table," but cheapest on vertical space and
simplest to implement (just fixed-width `fmt.Sprintf` fields, no `lipgloss/table`
dependency needed).

## D. Key/value rows, progress bar folded in

Extends **B** with the existing "next bump in" progress indicator (from
`design/02-mockups.md` #4) as an extra row, so all tempo-training state
lives in one table instead of a table plus a separate progress line:

```
┌──────────┬──────────────────────┐
│ State    │ on                   │
│ Step     │ 10 bpm               │
│ Interval │ 8 meas (5 of 8)      │
│ Target   │ 180 bpm              │
└──────────┴──────────────────────┘
```

## Recommendation

**E** (final): header line always shown with `State`'s content folded into
it ("Tempo Training: off/on/target reached (N bpm)"), table with `Start`/
`Step`/`Interval`/`Target` only rendered while training is on. Supersedes
the earlier recommendation of B alone. **D**'s progress-in-interval-row idea
is still a candidate to fold into the table once the "measures elapsed"
counter is designed.
