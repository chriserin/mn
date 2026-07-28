# Status Banner Variations

Exploring options for a prominent play/stopped indicator, since the earlier
plain "▸ stopped" caption was too quiet. Each variation shows both states so
they can be compared side by side. Pick one (or mix elements) to fold back
into `02-mockups.md`.

## A. Corner glyph block (small ASCII icon + label, top-left)

```
STOPPED                          RUNNING
┌──────────────────┐             ┌──────────────────┐
│ ■■■                │             │  ▶▶                │
│ ■■■  STOPPED       │             │ ▶▶▶  RUNNING       │
│ ■■■                │             │  ▶▶                │
└──────────────────┘             └──────────────────┘
```

## B. Full-width banner bar (top of box, inverse/solid background)

Rendered as a solid block bar so it reads as a distinct "state strip" regardless of color support (bold/reverse-video text).

```
STOPPED                                    RUNNING
┌────────────────────────────┐             ┌────────────────────────────┐
│██████████ STOPPED ██████████│             │██████████ RUNNING ██████████│
│                              │             │                              │
│         ♩ = 120 BPM          │             │         ♩ = 120 BPM          │
└────────────────────────────┘             └────────────────────────────┘
```

## C. Large ASCII-art word banner (biggest visual weight)

Uses block-letter ASCII art so state is readable even at a glance from across a room — most "fun/arcade" in tone, but takes real vertical space.

```
 ____ _____ ___  ____
/ ___|_   _/ _ \|  _ \
\___ \ | || | | | |_) |
 ___) || || |_| |  __/
|____/ |_| \___/|_|

  ____  _   _ _   _
 |  _ \| | | | \ | |
 | |_) | | | |  \| |
 |  _ <| |_| | |\  |
 |_| \_\\___/|_| \_|
```

Likely too tall for a compact TUI (would dominate a ~12-line box) — better suited for a full-screen splash moment (e.g. flash briefly on transition) than a persistent status line.

## D. Symbol-only, oversized single glyph (minimal text, icon does the work)

```
STOPPED         RUNNING
┌───────┐       ┌───────┐
│       │       │       │
│  ■■■  │       │  ▶▶▶  │
│  ■■■  │       │ ▶▶▶▶▶ │
│       │       │  ▶▶▶  │
└───────┘       └───────┘
```

Pairs well with a text label elsewhere (e.g. next to BPM) rather than repeating "STOPPED"/"RUNNING" as a word here.

## E. Transport-style bar (music software convention: ▶ / ■ / ⏸)

Familiar from DAWs/media players — likely the most immediately legible to musicians, who are the target users.

```
STOPPED                              RUNNING
┌──────────────────────────┐         ┌──────────────────────────┐
│  [ ■ ]  STOPPED            │         │  [ ▶ ]  RUNNING            │
└──────────────────────────┘         └──────────────────────────┘
```

## F. Pulsing border/frame color as the primary signal + small glyph

Border itself changes character style to indicate state (double-line when running, single-line when stopped), with a small glyph as the color-agnostic backup — ties the "is it alive" feeling to the whole frame, not just one corner.

```
STOPPED (single-line border)          RUNNING (double-line border)
┌──────────────────────────┐         ╔══════════════════════════╗
│  ■ STOPPED                 │         ║  ▶ RUNNING                 ║
│      ♩ = 120 BPM            │         ║      ♩ = 120 BPM            ║
└──────────────────────────┘         ╚══════════════════════════╝
```

## H. Big, bold block shapes, corrected for character aspect ratio

Terminal character cells are roughly twice as tall as they are wide, so a
shape built with equal row-count and column-count reads as a tall rectangle,
not a square. Fix: use roughly 2x as many columns as rows for the "stopped"
square, and widen the triangle's apex proportionally too.

**STOPPED** (visually square, 5 rows × ~10 columns):

```
██████████
██████████
██████████
██████████
██████████
```

**RUNNING** (right-pointing triangle, apex widened to match, smooth diagonal via `◥`/`◢`):

```
████◥
████████◥
████████████
████████◢
████◢
```

In the box:

```
┌──────────────────────────────────────────┐
│  ██████████                                │
│  ██████████                                │
│  ██████████        ♩ = 120 BPM             │
│  ██████████                                │
│  ██████████                                │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│  ████◥                                     │
│  ████████◥                                 │
│  ████████████      ♩ = 120 BPM             │
│  ████████◢                                 │
│  ████◢                                     │
└──────────────────────────────────────────┘
```

## I. Size variants of the H shapes

Same aspect-ratio-corrected shapes as H, at three sizes, to help pick how
much vertical space the indicator should claim inside a ~12-row box.

**Compact (3 rows × ~6 columns)** — sits comfortably alongside other status text without dominating:

```
STOPPED       RUNNING
██████        ██◥
██████        ██████
██████        ██◢
```

**Medium (5 rows × ~10 columns)** — this is size H above, a good balance of presence vs. space:

```
STOPPED           RUNNING
██████████        ████◥
██████████        ████████◥
██████████        ████████████
██████████        ████████◢
██████████        ████◢
```

**Large (7 rows × ~14 columns)** — most prominent, closer in weight to variant C's word-art but as a pure shape rather than letters; likely too tall to also fit a beat-dot row and controls legend in a compact box:

```
STOPPED                 RUNNING
██████████████          ██████◥
██████████████          ██████████◥
██████████████          ██████████████◥
██████████████          ██████████████████
██████████████          ██████████████◢
██████████████          ██████████◢
██████████████          ██████◢
```

Recommendation: **medium**, matching the beat-dot row and BPM readout in visual weight without pushing the controls legend off-screen.

## J. Placement variants (where the shape sits in the box)

**J1 — top-left corner** (shape + label stacked, text runs alongside):

```
┌──────────────────────────────────────────┐
│ ██████████                                 │
│ ██████████  STOPPED                        │
│ ██████████                                 │
│                    ♩ = 120 BPM              │
└──────────────────────────────────────────┘
```

**J2 — centered above the BPM readout** (shape becomes the visual anchor of the whole screen, BPM/beats arrange underneath it):

```
┌──────────────────────────────────────────┐
│              ██████████                    │
│              ██████████                    │
│              ██████████                    │
│                                            │
│                ♩ = 120 BPM                 │
└──────────────────────────────────────────┘
```

**J3 — inline to the right of the BPM readout** (shape reads as a "status pill" next to the number, keeps the shape small and secondary to BPM):

```
┌──────────────────────────────────────────┐
│                                            │
│      ♩ = 120 BPM     ██████████            │
│                      ██████████            │
│                      ██████████            │
└──────────────────────────────────────────┘
```

Recommendation: **J1**, since it keeps a consistent left anchor across states (matches how the caret/beat-dot row already anchors center), and reads immediately without requiring the eye to track between two centered elements.

## K. ASCII art lettering (figlet-style block letters spelling the state)

Rather than an icon/shape, the word itself is rendered in block letters —
generated with `figlet` (`mini` font) so the glyphs are guaranteed correctly
aligned rather than hand-drawn. This reads unambiguously as text at a
glance, no icon-literacy required.

**STOPPED** (figlet `mini`, 3 rows × 20 columns):

```
 ______  _  _  _ _
(_  |/ \|_)|_)|_| \
__) |\_/|  |  |_|_/
```

**PLAYING** (figlet `mini`, 3 rows × 21 columns):

```
 _        ___     __
|_)|  /\\_/| |\ |/__
|  |_/--\|_|_| \|\_|
```

In the box, as a top banner line:

```
┌────────────────────────────────────────────────┐
│  ______  _  _  _ _                               │
│ (_  |/ \|_)|_)|_| \                              │
│ __) |\_/|  |  |_|_/                              │
│                                                  │
│                  ♩ = 120 BPM                     │
└────────────────────────────────────────────────┘

┌────────────────────────────────────────────────┐
│  _        ___     __                             │
│ |_)|  /\\_/| |\ |/__                             │
│ |  |_/--\|_|_| \|\_|                             │
│                                                  │
│                  ♩ = 120 BPM                     │
└────────────────────────────────────────────────┘
```

Shorter alternative — `figlet -f small STOP`/`PLAY` (4 rows × ~20 columns,
slightly bolder strokes, 4-letter word avoids needing "-ED"/"-ING" suffixes):

```
 ___ _____ ___  ___         ___ _      ___   __
/ __|_   _/ _ \| _ \       | _ \ |    /_\ \ / /
\__ \ | || (_) |  _/       |  _/ |__ / _ \ V /
|___/ |_| \___/|_|         |_| |____/_/ \_\_|
```

Recommendation: **mini STOPPED/PLAYING** — fits in 3 compact rows (vs. 4 for
`small`), and the full words read more clearly than the abbreviated
STOP/PLAY at a glance since there's no ambiguity about tense (is "PLAY" a
button to press, or the current state?).

## L. Combining shape (H) + lettering (K)

Pairs the compact aspect-ratio-corrected shape from **I** (3-row compact
size) with the figlet `mini` lettering from **K**, so the state is signaled
redundantly by icon *and* word — most robust for a persistent banner: shape
gives instant peripheral-vision recognition, lettering removes any ambiguity.

**L1 — shape beside lettering** (shape acts like a bullet/icon to the left of the word):

```
STOPPED:

██████  ______  _  _  _ _
██████ (_  |/ \|_)|_)|_| \
██████ __) |\_/|  |  |_|_/

RUNNING/PLAYING:

██◥    _        ___     __
██████|_)|  /\\_/| |\ |/__
██◢   |  |_/--\|_|_| \|\_|
```

**L2 — shape stacked above lettering** (centered, more like a badge/logo lockup):

```
STOPPED:

  ██████
  ██████
  ██████
 ______  _  _  _ _
(_  |/ \|_)|_)|_| \
__) |\_/|  |  |_|_/

RUNNING/PLAYING:

  ██◥
  ██████
  ██◢
 _        ___     __
|_)|  /\\_/| |\ |/__
|  |_/--\|_|_| \|\_|
```

**L1 in the box** (recommended — keeps the combined banner to 3 rows, same height as the lettering alone, so toggling state doesn't resize the layout):

```
┌────────────────────────────────────────────────┐
│ ██████  ______  _  _  _ _                        │
│ ██████ (_  |/ \|_)|_)|_| \                       │
│ ██████ __) |\_/|  |  |_|_/                       │
│                                                  │
│                  ♩ = 120 BPM                     │
│               ^                                  │
│              ○     ○     ○     ○                │
│              1     2     3     4                │
│   Tempo Training: off                            │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘

┌────────────────────────────────────────────────┐
│ ██◥    _        ___     __                        │
│ ██████|_)|  /\\_/| |\ |/__                       │
│ ██◢   |  |_/--\|_|_| \|\_|                       │
│                                                  │
│                  ♩ = 120 BPM                     │
│               ^                                  │
│              ●     ○     ○     ○                │
│             (1)     2     3     4                │
│   Tempo Training: off                            │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## G. Combined recommendation

Corner glyph (A) for compactness + transport bracket styling (E) for
familiarity + border-weight change (F) as a passive ambient cue. All three
are color-agnostic on their own, so the design still works with color turned
off entirely.

```
┌──────────────────────────────────────────┐
│ [■] STOPPED                                │
│                                            │
│                ♩ = 120 BPM                 │
└──────────────────────────────────────────┘

╔══════════════════════════════════════════╗
║ [▶] RUNNING                                ║
║                                            ║
║                ♩ = 120 BPM                 ║
╚══════════════════════════════════════════╝
```
