# Status Bar (replaces the play/stop banner)

Supersedes the shape+lettering banner explored in
`03-status-banner-variations.md` / `04-full-mockups-v2.md` / `05-full-mockups-l1.md`.
Those documents are kept for history but no longer reflect the design.

## Layout

A single-line status bar at the top of the screen, replacing the banner
entirely:

```
mn  ·  PLAYING  ·  3/8
```

Three fields, separated by `·`:

1. **App name** — `mn`, matching the Go module/binary name.
2. **Playing status** — `PLAYING` / `STOPPED`, plain text (no shape/lettering
   graphic this time — the banner's job of being "prominent" is instead
   handled by the status bar's fixed position at the top of the screen).
3. **Measure counter** — `x/8` (current measures elapsed / `stepIntervalMeasures`)
   for tempo training. Only shown while tempo training is on; omitted
   entirely (falls back to just `mn  ·  PLAYING`) while training is off.

## Measure counter semantics

The counter is 1-based: it shows the current measure number within the
interval, out of `stepIntervalMeasures`. It starts at `1/stepIntervalMeasures`
before any measure completes.

Internally, `measuresSinceStep` still increments the moment a measure
completes (beat 4 struck), so the interval threshold can be checked and a
tempo step marked pending in time. But the *displayed* number doesn't
advance on beat 4 — it holds at the current measure's number through beats
2, 3, and 4, and only advances once beat 1 of the next measure actually
lands. This keeps the on-screen counter matching the measure the player is
currently hearing, rather than jumping ahead a beat early. At the interval
boundary this means the display holds at `stepIntervalMeasures/stepIntervalMeasures`
through the gap between beat 4 and beat 1 (matching the deferred tempo-step
timing already established — see `design/01-overview.md`'s "Tempo-change
timing" note), and resets to `1/stepIntervalMeasures` exactly when beat 1
lands and the pending step is applied. Stopping playback also resets the
counter to `1/stepIntervalMeasures`, along with any pending tempo step.

Example sequence with `stepIntervalMeasures = 8`, tempo training on:

```
1/8  (before any measure completes)
1/8  (measure 1 complete — beat 4 struck, display still holds)
2/8  (beat 1 of measure 2 struck — display advances)
2/8  (measure 2 complete, display holds)
3/8  (beat 1 of measure 3 struck)
...
8/8  (measure 8 complete; step is pending, display holds here)
1/8  (beat 1 of measure 9 struck; step applied, BPM updated, counter reset)
```

## States

Stopped, training off:
```
mn  ·  STOPPED
```

Playing, training off:
```
mn  ·  PLAYING
```

Playing, training on, mid-interval:
```
mn  ·  PLAYING  ·  3/8
```

Stopped, training on (counter resets when playback stops):
```
mn  ·  STOPPED  ·  1/8
```

## What's removed

- `shapeStopped` / `shapePlaying` ASCII glyphs.
- `letterStopped` / `letterPlaying` figlet lettering.
- The 3-row banner and its fixed-height layout-stability concerns — no
  longer relevant since the status bar is always exactly one line
  regardless of state.

The separate "Tempo Training: on/off/target reached (N bpm)" header and the
Start/Step/Interval/Target table (`design/06-tempo-training-table.md`
option E) are unchanged and still rendered below the beat display.
