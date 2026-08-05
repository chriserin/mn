# clickqueue

A macOS-only (CoreAudio `AudioQueue`) audio engine purpose-built for one
job: schedule a metronome click at an exact sample position and report,
with as little latency as achievable, the wall-clock instant it actually
becomes audible — so a UI can synchronize to it instead of guessing.

It is **not** a general-purpose audio player. There's no `Player`
abstraction, no mixer, no support for arbitrary `io.Reader` sources, no
volume control, no concurrent sounds. A single `FillFunc` callback decides
what every buffer contains.

## Relationship to oto, and license

This package began as a fork of
[`github.com/hajimehoshi/oto`](https://github.com/hajimehoshi/oto)'s macOS
`AudioQueue` driver (Apache 2.0), via two intermediate POCs: `../../timingexp`
(buffer-rotation/alignment exploration) and `../../mndriverpoc` (the
click-prediction/mic-loopback validation this was ported from — see
`../../timing_plan_integration.md`). It kept the package name `oto` through
both of those. Once vendored here, it became clear the name was actively
misleading: nothing in it can do what oto does, and its API surface
(`Fill`, `Events`, `PredictionOffset`) has no oto equivalent at all. Hence
`clickqueue`.

Three files (`api_darwin.go`, `context.go`, `driver_darwin.go`) are
Apache-2.0-licensed Derivative Works of oto and carry its original
copyright header plus an explicit modification notice, per the license's
§4(b)/(c); `LICENSE` in this directory is the Apache 2.0 text those files
are covered by, kept separate from `mn`'s own top-level (MIT) `LICENSE`.
`driver_macos.go` is verbatim oto, unmodified, no notice needed.
`hwinfo_darwin.go` and `machtime_darwin.go` have no oto original at all —
they don't carry an Apache header, since falsely attributing wholly
original code to oto would be its own inaccuracy; they're covered by `mn`'s
own license like the rest of the repository.

What's actually left of oto, file by file:

| File | Status |
|---|---|
| `driver_macos.go` | Verbatim (sleep/wake notification handling) |
| `api_darwin.go` | Mostly oto's C struct/binding definitions, plus `AudioQueueGetCurrentTime` (not in oto) — modified |
| `driver_darwin.go` | oto's buffer-rotation loop (`render`/`appendBuffer`/enqueue), but the mux/Player layer is gone — replaced by `FillFunc`, `ClickEvent`, and `predictAudibleTime` — modified |
| `context.go` | Keeps oto's public shape (`NewContext`, `Suspend`, `Resume`, `Err`); `Fill`/`Events`/`PredictionOffset` are new — modified |
| `hwinfo_darwin.go` | New, no oto original — queries the default output device's actual buffer size/sample rate/latency via `AudioObjectGetPropertyData`, used to align buffer size to the hardware's real I/O quantum (see `context.go`'s `alignUp`) |
| `machtime_darwin.go` | New, no oto original — converts `AudioQueueGetCurrentTime`'s `mach_absolute_time` host-time readings to `time.Time` |

## Why buffers roll their own timing instead of a wall-clock timer

The short version: `AudioQueueGetCurrentTime` gives a live (sample
position, host time, rate scalar) snapshot that maps any future sample
index to a predicted wall-clock time, computed fresh each click rather
than extrapolated over a long session — see `driver_darwin.go`'s
`predictAudibleTime` and its callers for the reasoning, and
`../../timing_plan_integration.md` for the fuller narrative (including the
mic-loopback measurement that validated it before this was ported into
`mn`).

## Known gaps

- The very first click of an app session can silently go unreported (not
  unheard — just not predicted): `AudioQueueGetCurrentTime` errors
  (`-66678`) until the device has actually started running. See
  `appendBuffer`'s comment in `driver_darwin.go`.
- `PredictionOffset` (an `mn`-side calibration constant, see `audio.go`) is
  empirical, not derived from any CoreAudio property — it was measured on
  one specific machine and isn't guaranteed to transfer to different
  hardware.
- macOS-only. There's no fallback engine in this package for other
  platforms; `mn` handles that at a higher level (see `audio.go`'s
  wall-clock tick fallback when `NewContext` fails).
