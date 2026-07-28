# UI Mockups (Phase 1)

ASCII mockups of the terminal screen at various states. No audio, no persistence — just BPM, start/stop, beat pulse with 4/4 accent, and tempo training.

## 1. Startup (stopped, default 120 BPM)

A small ASCII glyph in the top corner replaces the plain "▸ stopped" caption — a filled square block for stopped, a filled triangle for running. It's the first thing the eye lands on.

```
┌────────────────────────────────────────────────┐
│ ■■■                                              │
│ ■■■   STOPPED                                    │
│ ■■■                                              │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ○     ○     ○                │
│              1     2     3     4                │
│                                                  │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 2. Running, beat 1 (accented)

Beat 1 flashes larger/filled and in a different color (shown here as `●` bold vs `○`). A `^` caret above beat 1 is a color-agnostic accent marker, so the accent still reads correctly in no-color / monochrome / colorblind-unfriendly terminals.

```
┌────────────────────────────────────────────────┐
│                                                  │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ●     ○     ○     ○                │
│             (1)     2     3     4                │
│                                                  │
│                  ▸ running                       │
│                                                  │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

Color intent (256-color terminal): beat 1 accent in a warm color (e.g. orange/red), beats 2-4 pulse in a cooler accent (e.g. cyan) when struck, dim gray when idle. The caret is always shown above beat 1 (whether or not it's the currently struck beat) so the measure's downbeat position is visible at a glance; color alone marks the moment of the strike.

## 3. Running, beat 3 (unaccented pulse)

The caret over beat 1 stays visible even when a different beat is currently struck — it marks the downbeat position, not "this beat is active right now".

```
┌────────────────────────────────────────────────┐
│                                                  │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ○     ●     ○                │
│               1     2    (3)    4                │
│                                                  │
│                  ▸ running                       │
│                                                  │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 4. Tempo training enabled, mid-session

Shows step size, interval, target, and elapsed/remaining time until next bump — this is the "fun" progress element even without audio.

```
┌────────────────────────────────────────────────┐
│                                                  │
│                  ♩ = 138 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ●     ○     ○                │
│               1    (2)    3     4                │
│                                                  │
│                  ▸ running                       │
│                                                  │
│                                                  │
│   Tempo Training: ON  +2 bpm / 30s → target 160  │
│   ████████████░░░░░░░░░░  next bump in 00:14     │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 5. Tempo training reached target

```
┌────────────────────────────────────────────────┐
│                                                  │
│                  ♩ = 160 BPM  🎯                 │
│                                                  │
│               ^                                  │
│              ●     ○     ○     ○                │
│             (1)     2     3     4                │
│                                                  │
│                  ▸ running                       │
│                                                  │
│                                                  │
│   Tempo Training: target reached (160 bpm)        │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 6. BPM at floor/ceiling (clamped)

Bumping past the bound is a no-op; a brief flash/color change on the BPM number could indicate "can't go further" (nice-to-have, not required for phase 1 functionality).

```
┌────────────────────────────────────────────────┐
│                                                  │
│                  ♩ = 300 BPM  (max)               │
│               ^                                  │
│              ○     ○     ○     ○                │
│               1     2     3     4                │
│                  ▸ stopped                       │
│   Tempo Training: off                            │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## Notes on motion (not capturable in static ASCII)

- The beat dot for the current beat should visibly grow/shrink or change brightness over the pulse window (e.g. bright immediately on the beat, fading over ~150ms), not just snap on/off — this is the main "feels alive" element for phase 1.
- The active dot position moves left→right across the 4 beat slots each measure, giving a simple visual rhythm even for someone glancing at the terminal.
- Consider a color gradient on the `♩ = N BPM` number that shifts hue as BPM increases (cool at 60, warm by 200+), reinforcing the tempo-training arc described in the design overview.
- The `^` caret above beat 1 is a color-agnostic accent marker: it's always rendered above beat 1's slot regardless of which beat is currently pulsing, so the accent survives in no-color terminals or for colorblind users. Color is layered on top for sighted users in color terminals, but the caret alone should fully convey "this is beat 1 / the downbeat."
