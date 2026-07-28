# Metronome TUI — Design Overview

## Goal

A terminal metronome built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Go). It must:

- Keep accurate tempo (BPM) and produce an audible click.
- Support tempo controls (set BPM directly, nudge up/down).
- Support "tempo training": automatically increase BPM at regular time intervals, by a chosen BPM increment, up to a chosen ceiling.
- Feel fun and visually engaging even though the feature set is small — this is a TUI, so "fun" comes from animation, color, and a satisfying pulse/beat visualization, not screenshots or media.

## Non-goals (for v1)

- No MIDI input/output.
- No audio output — v1 is visual/silent only (see Audio below); revisit in a later phase.
- No multi-track / polyrhythm support.
- No mouse support requirement (keyboard-first, as is typical for TUIs).

## Core concepts

### Tempo state
- `bpm`: current tempo (e.g. 40–300 range, standard metronome bounds).
- `beatsPerMeasure`: for visual/audio accent on beat 1 (optional v1 feature — see open questions).
- `running`: bool, whether the metronome is currently ticking.

### Tempo training (auto tempo increase)
- `stepBPM`: how much to increase each interval (default +10 BPM).
- `stepIntervalMeasures`: how often to increase, in elapsed measures (default 8 measures). Decided: v1 counts measures, not wall-clock time — this is how musicians naturally think about practice intervals ("run it 8 bars, then bump the tempo"), and phase 1 already tracks `beatsPerMeasure` (fixed 4/4) for the beat-1 accent, so measure counting is free.
- `targetBPM` (optional ceiling): stop auto-increasing once reached.

### Audio
Decided: v1 is visual-only, no sound. Bubble Tea only handles the terminal UI, and audio would require a separate library (e.g. `gopxl/beep`) plus platform output backend — deferred to a later phase. The beat pulse/flash in the UI is the only beat indicator for v1. This also sidesteps environments without audio output (SSH, CI, etc.) for free.

### Timing engine
A `time.Ticker` driven by BPM (interval = 60s/bpm) works for a basic beat pulse, but drift compounds over long sessions. Approach:
- Recompute an absolute "next beat time" each tick (`time.Sleep(next.Sub(now))`) to avoid cumulative drift — recommended.
- Run the beat clock on its own goroutine that emits Bubble Tea messages (`tea.Msg`) for each beat, decoupling timing accuracy from UI frame rate.

### Persistence
Not in phase 1 (in-memory only, starts from defaults every run). See `phases/` for what's in scope per phase — persistence is a later phase.

### UI layout (Bubble Tea model)
Rough sketch of a single-screen TUI:

```
┌─────────────────────────────────────────┐
│              ♩ = 120 BPM                 │
│                                           │
│         ●   ○   ○   ○      (beat dots)   │
│                                           │
│        [pulsing shape/color on beat]     │
│                                           │
│  Tempo Training: +10 bpm every 8 measures│
│  Target: 160 bpm      Measure: 5 of 8    │
│                                           │
│  ↑/↓ bpm   space: start/stop   t: train  │
└─────────────────────────────────────────┘
```

"Fun" elements for minimal functionality:
- A shape (circle/bar) that pulses or changes color exactly on the beat.
- Color shifts as BPM increases (e.g. cool → warm as it speeds up) to visually reinforce the training arc.
- Tap-tempo style flash animation, maybe ASCII pendulum swing left-right in time with the beat.

### Key bindings (draft)
- `space`: start/stop
- `↑` / `↓` or `+` / `-`: adjust BPM by 1
- `shift+↑` / `shift+↓`: adjust BPM by 10
- `t`: toggle tempo-training mode
- `[` / `]`: adjust tempo-training step size (`stepBPM`) down/up by 1, independent of the main BPM keys. Clamped to a sane range (e.g. 1–20 BPM), default 10. Works whether tempo training is on or off, so the step can be dialed in before starting.
- `{` / `}`: adjust tempo-training interval (`stepIntervalMeasures`) down/up by 1 measure. Clamped to a sane range (e.g. 1–32 measures), default 8. Works whether tempo training is on or off.
- `q` / `ctrl+c`: quit

## Decisions carried into phasing

- Tempo-training interval is elapsed measures (default 8), not wall-clock time.
- Fixed 4/4 time; beat 1 of each measure renders as a visual accent (distinct color/size). No configurable time signature yet.

See `phases/` for what's actually in scope for each implementation phase (e.g. audio and persistence are later phases, not phase 1).
