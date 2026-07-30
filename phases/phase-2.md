# Phase 2 — Tempo Training

Automatic tempo stepping toward a target BPM over time, layered on top of phase 1's core metronome.

## In scope

- Tempo training: `stepBPM` step every `stepIntervalMeasures` elapsed measures (default 8 measures), toward `targetBPM` (default 180). Steps up if the target is above the current BPM, down if below, clamped so it never overshoots the target. Holds once `bpm == targetBPM` ("target reached").
- Toggle tempo training on/off (key binding: `t`).
- Adjust `stepBPM` up/down with dedicated keys `[` / `]`, clamped to a sane range (e.g. 1–20 BPM), default 10. Works whether tempo training is on or off.
- Adjust `stepIntervalMeasures` up/down with dedicated keys `{` / `}`, clamped to a sane range (e.g. 1–32 measures), default 8. Works whether tempo training is on or off.
- Adjust `targetBPM` up/down with dedicated keys `n` / `m` (±1) and `shift+n` / `shift+m` (±10), clamped 20–300 (same bounds as `bpm`), default 180. Works whether tempo training is on or off. There is no "unset" state — the target always has a numeric value.
- `j` / `k` and `shift+j` / `shift+k` mirror `up`/`down`/`shift+up`/`shift+down` for adjusting `bpm` directly — an alternate, vim-style way to reach the same BPM controls from phase 1.
- Tempo training does not advance while the metronome is stopped (no measures elapse while stopped).
- Tempo training displayed as an always-visible header line ("Tempo Training: off / on / target reached (N bpm)") plus a key/value table (Start, Step, Interval, Target rows) that is only rendered while training is on, per `design/06-tempo-training-table.md` option E. Interval row reads the full word "measures" (e.g. "8 measures").
- `startBPM`: the BPM the current run began at. Mirrors the live BPM readout while stopped (so it moves along with manual BPM adjustments); captured and held fixed the moment the metronome starts playing. When the metronome stops, `bpm` reverts to `startBPM` (undoing any drift from tempo training that run) — the relationship is one-directional; `startBPM` never reverts to match a drifted `bpm`.
- Measure counter (`x/8`) in the status bar, showing `measuresSinceStep`/`stepIntervalMeasures`. Only shown while tempo training is on. Holds at the interval value between beat 4 and beat 1 (matching the deferred step-application timing), resetting to 0 when the pending step actually lands. See `design/07-status-bar.md`.

## Out of scope

- Audio (phase 3).
- Persistence (phase 4).
- `stepBPM`, `stepIntervalMeasures`, and `targetBPM` are all user-adjustable this phase; nothing further deferred.

## Reference

See `design/01-overview.md` Tempo training section.
