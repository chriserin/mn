# Phase 3 — Audio

Add an audible click on each beat, layered on top of phase 1's visual metronome and phase 2's tempo training.

## In scope

- Synthesized click sound (sine or noise burst) generated in-memory, no shipped audio asset files.
- Likely library: `gopxl/beep` (maintained fork of `faiface/beep`) for playback.
- Accented click on beat 1 (distinct tone/volume from beats 2-4), matching the visual accent from phase 1.
- Audio timing driven by the same beat-clock goroutine as the visual pulse, to keep sound and animation in sync.
- A way to disable audio (e.g. `--visual-only` flag or in-app toggle) for environments without audio output (SSH, CI, etc.).

## Out of scope

- Persistence (phase 4).
- Configurable click timbre/instrument choices beyond a single default sound.

## Reference

See `design/01-overview.md` Audio section.
