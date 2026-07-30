# Status Bar (replaces the play/stop banner)

Supersedes the shape+lettering banner explored in
`03-status-banner-variations.md` / `04-full-mockups-v2.md` / `05-full-mockups-l1.md`.
Those documents are kept for history but no longer reflect the design.

## Layout

A single-line status bar at the top of the screen, styled like a vim
statusline plugin (airline/powerline): solid-color segments joined by
right-angle triangle wedge glyphs (`◥`/`◤`, Unicode Geometric Shapes block —
not the Nerd Font powerline arrows, so no patched font is required),
spanning the full terminal width. The left-hand segments are pinned to the
left edge, the measure-counter segment (when present) is pinned to the
right edge, and the gap between them is filled with the bar's background
color.

```
 PLAYING ◥ mn ◥                                                 ◤ 3/8 
```

Segments, left to right:

1. **Mode segment** — `PLAYING` / `STOPPED`, colored background (green while
   playing, red while stopped) so play state is readable at a glance without
   parsing text. Replaces the old plain-text playing-status field.
2. **App segment** — `mn`, matching the Go module/binary name, neutral gray
   background.
3. **Measure counter segment** (right edge) — `x/8` (current measure number
   out of `stepIntervalMeasures`) for tempo training, blue background. Only
   shown while tempo training is on; omitted entirely while training is off,
   in which case the bar's background fill runs all the way to the right
   edge.

Each segment is `Padding(0, 1)` (one space of breathing room on either
side) followed by a wedge rendered in that segment's own background color,
so the wedge reads as a seamless color transition into whatever comes next
(standard powerline technique). The left-hand group trails each segment
with `◥`; the right-edge segment leads with the mirror-image `◤`, since
it's approached from the opposite direction.

The bar requires the terminal width (via `tea.WindowSizeMsg`) to compute the
fill; `Model.width` is 0 until the first resize message arrives, in which
case the bar renders tight (no fill) rather than guessing a width.

## Colors

Segment backgrounds use the 4-bit ANSI palette (`lipgloss.Green`, `.Red`,
`.Blue`, `.BrightBlack`, `.Black`, values 0-15) rather than fixed 256-color
or hex values. These render as the basic SGR color codes (30-37/40-47,
90-97/100-107), which terminal emulators map to whatever colors the user's
own theme assigns to "green", "red", etc. — so the bar picks up the user's
terminal color scheme instead of imposing a fixed palette that might clash
with it.

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

Stopped, training off (red mode segment, no counter, fill runs to the edge):
```
 STOPPED ◥ mn ◥                                                       
```

Playing, training off (green mode segment):
```
 PLAYING ◥ mn ◥                                                       
```

Playing, training on, mid-interval (blue counter segment pinned right):
```
 PLAYING ◥ mn ◥                                                 ◤ 3/8 
```

Stopped, training on (counter resets when playback stops):
```
 STOPPED ◥ mn ◥                                                 ◤ 1/8 
```

## What's removed

- `shapeStopped` / `shapePlaying` ASCII glyphs.
- `letterStopped` / `letterPlaying` figlet lettering.
- The 3-row banner and its fixed-height layout-stability concerns — no
  longer relevant since the status bar is always exactly one line
  regardless of state.
- The plain-text `mn  ·  PLAYING  ·  3/8` layout from the first status-bar
  pass, replaced by the colored powerline segments above.

The separate "Tempo Training: on/off/target reached (N bpm)" header and the
Start/Step/Interval/Target table (`design/06-tempo-training-table.md`
option E) are unchanged and still rendered below the beat display.
