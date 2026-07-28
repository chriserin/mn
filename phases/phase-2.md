# Phase 2 — Tempo Training

Automatic tempo increase over time, layered on top of phase 1's core metronome.

## In scope

- Tempo training: `stepBPM` increase every `stepIntervalMeasures` elapsed measures (default 8 measures), up to an optional `targetBPM` ceiling (stops auto-increasing once reached).
- Toggle tempo training on/off (key binding: `t`).
- Adjust `stepBPM` up/down with dedicated keys `[` / `]`, clamped to a sane range (e.g. 1–20 BPM), default 10. Works whether tempo training is on or off.
- Adjust `stepIntervalMeasures` up/down with dedicated keys `{` / `}`, clamped to a sane range (e.g. 1–32 measures), default 8. Works whether tempo training is on or off.
- Tempo training does not advance while the metronome is stopped (no measures elapse while stopped).
- Status line reflecting tempo training state, current step size, and current interval: off / on / target reached, matching the mockups in `design/02-mockups.md` and `design/04-full-mockups-v2.md`.

## Out of scope

- Audio (phase 3).
- Persistence (phase 4).
- Configuring `targetBPM` interactively — it stays unset (no ceiling) by default in phase 2. `stepBPM` and `stepIntervalMeasures` are both user-adjustable this phase.

## Reference

See `design/01-overview.md` Tempo training section.
