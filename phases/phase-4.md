# Phase 4 — Persistence

Persist settings across runs.

## In scope

- On exit (or on each change), write current BPM and tempo-training config (`bpm`, `stepBPM`, `stepIntervalSeconds`, `targetBPM`) to a small JSON config file in the user config dir (`os.UserConfigDir()`, e.g. `~/.config/metronome-mn/config.json` on Linux).
- On startup, load the config file if present; fall back to defaults otherwise.

## Out of scope

- Multiple named presets/profiles.
- Cloud sync or cross-machine settings.

## Reference

See `design/01-overview.md` Persistence section.
