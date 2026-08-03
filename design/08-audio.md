# Audio — Synthesized Click

Implements the phase 3 scope (`phases/phase-3.md`): an audible click on each
beat, layered on top of phase 1's visual pulse and phase 2's tempo training,
with no shipped audio asset files and no configurable timbre.

## Library

`github.com/gopxl/beep/v2` (maintained fork of `faiface/beep`), as suggested
in `phases/phase-3.md`. It provides `speaker.Init`/`speaker.Play` for output
and a `beep.Streamer` interface simple enough to hand-write a synthesized
click from, so no bundled audio asset or decoder is needed.

## Click synthesis

Decided: a short sine-wave burst shaped by a fast-decay envelope, not a
sustained tone — a raw sine held for the streamer's full duration sounds
like a beep, not a metronome click.

- Sample rate: 44100 Hz, mono.
- Duration: ~25ms per click.
- Envelope: no attack ramp (click onset should be immediate), exponential
  decay across the 25ms so the tail fades out rather than cutting off
  abruptly.
- Accent: beat 1 uses a higher pitch (~1600 Hz) than beats 2-4 (~1000 Hz),
  matching a traditional two-tone metronome click (high/low), rather than
  differentiating by volume or duration. This mirrors the phase 1 visual
  accent (beat 1 gets distinct styling from beats 2-4) with the equivalent
  audio treatment called for in `phases/phase-3.md`.

Implementation is a small hand-written `beep.Streamer` (a struct tracking
sample position, generating `sin(2π·freq·t) * decayEnvelope(t)` per sample)
rather than composing existing `beep/generators` effects, since the decay
envelope needs to be baked into each sample.

## Playback plumbing

- `speaker.Init(sampleRate, bufferSize)` is called lazily, once, the first
  time a click needs to play (not eagerly at startup) — see "No audio
  output" below for why.
- `bufferSize` is 100ms (`sampleRate.N(time.Second/10)`), matching beep's
  own documented example. A much smaller buffer underruns almost
  immediately under normal goroutine scheduling/GC jitter, and some
  CoreAudio backends stop invoking their data callback after an underrun
  and never resume — audible as "the first click plays, then permanent
  silence for the rest of the run."
- Each beat, `speaker.Play(streamer)` queues that beat's click. `speaker`
  mixes concurrently-playing streamers internally, so a click that hasn't
  finished its 25ms decay when the next beat's click starts (very fast
  tempos) layers rather than cutting off — acceptable since 25ms is short
  relative to the fastest supported tempo's beat interval (60s/300bpm =
  200ms).
- Triggering: `Model.Update`'s existing `beatMsg` handler (`model.go`) calls
  `advanceBeat`, which already knows whether the struck beat is 1
  (accented) or 2-4. That same handler returns a `tea.Cmd` that performs
  the click playback as a side effect, batched alongside `tickCmd`. This
  reuses the *exact* same beat-clock event (`beatMsg`) that drives the
  visual pulse for audio triggering too — satisfying the phase-3 "driven by
  the same beat-clock" requirement — while keeping `Update` itself free of
  I/O (side effects live in the `tea.Cmd`, consistent with Bubble Tea's
  model).
- The actual `speaker.Play` call is reached through a package-level
  function variable (e.g. `var playClick = func(accented bool) {...}`) so
  tests can substitute a spy and assert which beats triggered playback
  without touching real audio hardware — no test in this repo should
  depend on an audio device being present (see Testing below).

## No audio output (SSH, CI, headless environments)

Decided: no CLI flag and no in-app toggle for phase 3 — narrower than
`phases/phase-3.md`'s suggested scope, which offered a flag/toggle as *one
way* to satisfy "environments without audio output shouldn't break."
Instead: audio is auto-detected and fails silently.

- `speaker.Init`'s error return is checked at the lazy-init call site. If it
  errors (no audio device found, `/dev/dsp`-less container, etc.), a
  package-level `audioAvailable` flag is set to `false` and every
  subsequent `playClick` call becomes a no-op — checked once per process,
  not retried per beat.
- No warning is printed and no status bar indicator is shown: the metronome
  simply runs visual-only, identical to today's phase 1/2 behavior, with no
  observable difference to the user beyond the absence of sound. This keeps
  SSH/CI runs working with zero configuration.
- This intentionally drops the manual disable mechanism phase-3.md
  mentioned as an example; automatic fallback covers the actual
  requirement ("a way to disable audio... for environments without audio
  output") without adding a flag or keybinding surface.

## Testing

Playback can't be asserted against real sound in `go test`. Two testable
seams:

1. **Which beats accent** — pure logic, already exposed by `advanceBeat`'s
   return value / `currentBeat`. A new small helper (e.g.
   `Model.clickAccented() bool`, `true` when `currentBeat == 1`) is unit
   tested directly, same as existing beat-pulse tests in `model_test.go`.
2. **That a click is triggered per beat, with the right accent flag** — the
   `playClick` function variable is swapped for a spy in tests (recording
   calls and their `accented` argument), and the `tea.Cmd` returned from the
   `beatMsg` case in `Update` is invoked directly (calling the returned
   `func() tea.Msg`) to trigger playback without running the full Bubble
   Tea program, consistent with how `model_test.go` already drives behavior
   through `Update()` directly rather than a running program.

No test initializes a real `speaker` or requires an audio device — keeps
`go test ./...` hardware-independent, matching how `preview_test.go`'s
visual-only harnesses are already excluded from the default run.

## Out of scope (per phases/phase-3.md)

- Persistence of any audio-related setting (phase 4 — moot here anyway
  since there's no setting to persist).
- Configurable click timbre/pitch/instrument choice.
- A manual mute flag or keybinding (see "No audio output" above).
