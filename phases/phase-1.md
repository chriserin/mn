# Phase 1 — Core Metronome

Baseline playable metronome. No tempo training, no audio, no persistence — those are later phases.

## In scope

- BPM state: adjustable tempo, default e.g. 120, clamped to a sane range (e.g. 20–300).
- Start/stop control.
- Visual beat pulse driven by an accurate timing engine (recompute absolute next-beat time each tick to avoid drift), fixed at 4/4 time, with beat 1 rendered as a visual accent (distinct color/size, plus a color-agnostic caret) vs. beats 2-4.
- Key bindings: start/stop, BPM up/down (small and large increments), quit.
- All settings are in-memory only; every run starts from defaults.

## Out of scope (deferred to later phases)

- Tempo training / auto BPM increase (see phase-2.md).
- Audio output (see phase-3.md).
- Persisting settings across runs (see phase-4.md).
- Configurable time signature / beats-per-measure other than 4/4.
- MIDI, multi-track, mouse support.

## Reference

See `design/01-overview.md` for the full rationale and UI sketch.
