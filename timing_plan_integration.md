# Integration plan: audio-clock-driven timing from mndriverpoc

## Purpose

`../mndriverpoc` validated an approach to metronome click/UI timing that's
meaningfully tighter than what `mn` does today. This document lays out what
would need to change in `mn` to adopt it, and — per `CLAUDE.md`'s phased
workflow — is meant as input to a `design/NN-*.md` doc and `fts/*.ft`
scenarios, not a substitute for either. Nothing here should be implemented
without that discussion happening first.

## What mndriverpoc validated

- **Buffer size must align to the hardware's actual I/O quantum** (queried
  via `AudioObjectGetPropertyData`, not assumed), or CoreAudio hands buffers
  back in uneven bursts instead of one evenly-spaced callback per period.
- **Latency from "click scheduled" to "audible" is `(bufferCount-1) ×
  bufferInterval`** — a direct, quantified tradeoff between buffer count
  (underrun safety margin) and latency. `mndriverpoc` runs at `bufferCount=2`
  for minimal latency (~11.6ms/buffer at 44.1kHz).
- **`AudioQueueGetCurrentTime` (sample position ↔ host time, rate-corrected
  via `mRateScalar`) predicts a future click's wall-clock time more
  accurately than timestamping our own buffer-rotation callback**, and does
  so per-click rather than needing a long-lived extrapolation — which matters
  because it's what makes tempo changes safe to schedule against (see below).
- **Scheduling the next beat from the previous beat's actual placed sample
  position (`lastClickFrame + samplesPerBeat`), not from a beat count times a
  fixed origin, is what allows tempo to change mid-stream** without
  discontinuity — at the cost of reintroducing sub-sample rounding noise
  (~11us/beat, not accumulating) that a fixed-origin scheme avoids. `mn`
  already changes tempo live (manual up/down, tempo training auto-step), so
  this is directly applicable, not a hypothetical.
- **Mic-loopback measurement** (`internal/oto/recorder_darwin.go`) is the
  only thing in the whole POC that checks the prediction against physical
  reality rather than trusting CoreAudio's self-reported timing end to end.
  It found the UI firing **~40ms ahead of the audible click**, stable to
  <1ms over a minute of measurement and independent of `bufferCount` — a
  large but consistent, correctable bias (`PredictionOffset`), not jitter.
- **Driving the UI from the audio engine, not a wall-clock timer**: the UI
  event is computed inside the audio fill path and delivered as a predicted
  future timestamp; the UI schedules its own update for that instant rather
  than reacting immediately on receipt.

## What mn does today

(`audio.go`, `model.go`, `design/08-audio.md`)

- `github.com/gopxl/beep/v2`, `speaker.Play(streamer)` per beat — a
  fire-and-forget software mixer with a 100ms buffer (chosen specifically to
  avoid CoreAudio underrun/silence bugs at smaller sizes). No latency
  awareness at all: nothing in this path knows or predicts when a queued
  click actually becomes audible.
- Beat timing is `tea.Tick(interval, ...)` (`tickCmd`), recomputed from the
  current BPM each beat — avoids cumulative drift at the *wall-clock*
  timer level, but is still a software timer, not sample-accurate, and
  carries no relationship to audio buffer state at all.
- **The same `beatMsg` handler triggers both** the visual beat advance
  (`Model.advanceBeat`) **and** the click (`clickCmd(m.clickAccented())`),
  synchronously, in the same `Update()` call — i.e., today's design doesn't
  even try to align UI-to-audio timing; both are just fired together off the
  wall-clock tick and whatever happens after that (beep's mixing, CoreAudio's
  own buffering) is unaccounted for.
- Click synthesis: two-tone accent (1600Hz beat 1 / 1000Hz beats 2-4), 25ms,
  exponential-decay envelope, no attack ramp — this logic (the *waveform*,
  as opposed to the *scheduling*) should carry over essentially unchanged.
- `speaker.Init` failure (no audio device: SSH, CI, headless) is caught once
  and silently degrades to visual-only for the rest of the run — this
  behavior needs an equivalent in whatever replaces it.
- Tests never touch real audio: `playClick` is a package-level func var
  swapped for a spy, and beat-accent logic is unit tested as pure
  computation. This pattern needs to survive the swap.

## Architectural shift required

This is the central change, and it's an inversion of who's authoritative:

- **Today**: `tea.Tick` (wall-clock) decides when a beat happens; audio and
  UI both react to that decision independently in the same `Update()` call.
- **Proposed**: the audio fill callback (running on its own goroutine,
  counting samples) decides when a beat happens and what its predicted
  audible wall-clock time is; `Update()` becomes a *consumer* of that
  decision via a `tea.Cmd` that resolves at the predicted instant, not the
  source of it.

Concretely, `beatIndex`/accent decisions (currently `Model.advanceBeat` /
`clickAccented`) need to move into the audio-side scheduler (an
`mn`-flavored equivalent of `mndriverpoc/internal/metronome`, extended to
choose between the two accent waveforms), because that's the only place
that knows a click's absolute sample position *before* it's committed to a
buffer. The `Model` should treat beat number/accent as reported by the
resulting event, not maintain an independently-advancing counter that could
drift out of sync with the audio engine's (e.g. if an event is ever
dropped — see Open questions).

## Staged plan

1. **Resolve the platform-support question first** (see Open questions) —
   it determines whether stage 2 is a straight swap or needs a fallback
   path.
2. **Vendor the audio engine** into `mn`'s own `internal/` tree: `context.go`,
   `driver_darwin.go`, `api_darwin.go`, `driver_macos.go`, `hwinfo_darwin.go`,
   `machtime_darwin.go` from `mndriverpoc/internal/oto`, unchanged in
   substance. Leave `recorder_darwin.go` (mic-loopback) out of the vendored
   set entirely — it's a dev-only measurement tool, not a runtime dependency
   (see Non-goals).
3. **Port click synthesis and scheduling** into an `mn`-side package
   mirroring `mndriverpoc/internal/{click,metronome}`, but: generate *two*
   waveforms (accent/plain, reusing `audio.go`'s existing frequency/envelope
   constants) instead of one, and have `Fill` choose between them using
   `beatIndex % beatsPerMeasure` — the accent decision has to happen at fill
   time now, not in `Model.advanceBeat` after the fact.
4. **Bridge `oto.Events()` into Bubble Tea's `Msg` loop.** Standard pattern:
   a `tea.Cmd` that blocks on a channel receive and returns a message,
   re-armed after each message — but the message shouldn't fire on receipt,
   it should fire at `ClickEvent.PredictedAudible` (mirroring
   `mndriverpoc/main.go`'s `scheduleFlash`, expressed as a `tea.Cmd`/timer
   instead of a bare `time.AfterFunc`). This message replaces `beatMsg` as
   what drives `advanceBeat`.
5. **Wire tempo changes through `SetTempo`.** Every place `model.go`
   currently mutates `m.bpm` during playback (up/down keys, tempo-training
   auto-step) needs a corresponding `SetTempo` call into the audio engine,
   replacing `tickCmd`'s role of recomputing a wall-clock interval.
6. **Match existing no-audio-device behavior.** `oto.NewContext` returning an
   error needs to fall back to silent/visual-only exactly like
   `ensureAudioInit`/`audioAvailable` does today — same user-facing contract,
   different plumbing underneath.
7. **Preserve the testing seams.** `Fill`'s accent/scheduling logic is pure
   and unit-testable without hardware, same spirit as today's `playClick`
   spy — write those tests the same way `clickAccented` is tested now.
   Real-hardware validation (does it actually still sound and sync right)
   stays manual, same as it effectively is today.
8. **Remove the `beep` dependency** from `go.mod` once the cutover is
   confirmed working end-to-end.

## Open questions (resolve before/during the design doc, not here)

- **Cross-platform support.** The vendored engine is macOS-only (AudioQueue
  via `purego`). Nothing in `VISION.md`/`README.md` states a platform target
  one way or the other. If `mn` needs to run on Linux/Windows, this needs
  either a build-tag fallback to the current `beep` path on non-darwin, or
  an explicit decision that `mn` is macOS-only going forward. This blocks
  stage 2 and should be settled first.
- **`PredictionOffset` portability.** The ~40ms measured offset is specific
  to the machine it was measured on (and, per `mndriverpoc`'s own findings,
  didn't come from any property CoreAudio exposes, so it can't be
  auto-derived). Ship a measured default and accept the residual error on
  other hardware? Add a hidden/debug tuning path? This wants a real answer,
  not a silent default.
- **`currentBeat` single-source-of-truth.** If a `ClickEvent` is ever
  dropped (the events channel is non-blocking and can drop under load, and
  there's a known gap where the very first click of a run isn't reported at
  all — see below), does the UI's beat counter get out of sync with the
  audio engine's? Recommend the UI never counts beats independently, only
  ever reflects what the last event reported — but this needs to be an
  explicit decision, not an accident.
- **The "first click of the stream" gap.** `mndriverpoc` found
  `AudioQueueGetCurrentTime` errors on the very first buffer (device not
  running yet), so beat 1 of a fresh run never gets a `ClickEvent`. This
  might be a non-issue in practice: `model.go`'s Start handler *already*
  strikes beat 1 immediately without waiting for a tick
  (`m.currentBeat = 1` before the first `tickCmd`/`clickCmd` fires) — so the
  visual side may not need beat 1's event at all. Worth confirming rather
  than assuming.

## Non-goals

- Shipping mic-loopback measurement (`recorder_darwin.go`) as a runtime
  feature. It's a development/calibration tool — if `PredictionOffset` ever
  needs re-measuring (new hardware, changed buffer config), that's a manual
  dev-time step using a copy of that tool, not something `mn` does at
  runtime.
- Auto-calibrating `PredictionOffset` from within the app.
- Achieving audio-clock-driven timing on non-macOS platforms in this pass
  (see Open questions — if cross-platform is required, that's follow-on
  work, not part of this integration).
