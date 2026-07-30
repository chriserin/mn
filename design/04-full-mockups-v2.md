# Full-Screen Mockups v2 — Shape-Based Status Indicator

> **Superseded:** the play/stop banner shown here was removed in favor of a
> one-line status bar. See `07-status-bar.md`.

Same six states as `02-mockups.md`, updated to use the aspect-ratio-corrected
block/triangle shape (variant H, medium size, corner placement J1 from
`03-status-banner-variations.md`) in place of the original plain
"▸ stopped"/"▸ running" caption. Beat dots, the beat-1 caret, and tempo
training layout are unchanged from `02-mockups.md`.

## 1. Startup (stopped, default 120 BPM)

```
┌────────────────────────────────────────────────┐
│ ██████████                                       │
│ ██████████  STOPPED                              │
│ ██████████                                       │
│                                                  │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ○     ○     ○                │
│              1     2     3     4                │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 2. Running, beat 1 (accented)

```
┌────────────────────────────────────────────────┐
│ ████◥                                            │
│ ████████◥   RUNNING                              │
│ ████████████                                     │
│ ████████◢                                        │
│ ████◢                                            │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ●     ○     ○     ○                │
│             (1)     2     3     4                │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 3. Running, beat 3 (unaccented pulse)

```
┌────────────────────────────────────────────────┐
│ ████◥                                            │
│ ████████◥   RUNNING                              │
│ ████████████                                     │
│ ████████◢                                        │
│ ████◢                                            │
│                  ♩ = 120 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ○     ●     ○                │
│               1     2    (3)    4                │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 4. Tempo training enabled, mid-session

```
┌────────────────────────────────────────────────┐
│ ████◥                                            │
│ ████████◥   RUNNING                              │
│ ████████████                                     │
│ ████████◢                                        │
│ ████◢                                            │
│                  ♩ = 138 BPM                     │
│                                                  │
│               ^                                  │
│              ○     ●     ○     ○                │
│               1    (2)    3     4                │
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
│ ████◥                                            │
│ ████████◥   RUNNING                              │
│ ████████████                                     │
│ ████████◢                                        │
│ ████◢                                            │
│                  ♩ = 160 BPM  🎯                 │
│                                                  │
│               ^                                  │
│              ●     ○     ○     ○                │
│             (1)     2     3     4                │
│                                                  │
│   Tempo Training: target reached (160 bpm)        │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## 6. BPM at floor/ceiling (clamped, stopped)

```
┌────────────────────────────────────────────────┐
│ ██████████                                       │
│ ██████████  STOPPED                              │
│ ██████████                                       │
│                                                  │
│                  ♩ = 300 BPM  (max)               │
│               ^                                  │
│              ○     ○     ○     ○                │
│               1     2     3     4                │
│                                                  │
│   Tempo Training: off                            │
│                                                  │
│  space start/stop   ↑↓ ±1   ⇧↑⇧↓ ±10   t train   q quit │
└────────────────────────────────────────────────┘
```

## Open questions on this direction

1. The RUNNING shape is 5 rows tall vs. STOPPED's 3 rows tall (from variant H being asymmetric in the source comparison) — should both states use the same row count so the box doesn't visually resize/jump when toggling? Recommend locking both to a fixed 5-row slot (pad STOPPED's square or trim RUNNING's triangle) so start/stop doesn't cause layout shift.
2. Color intent (layered on top of the shape, still readable without it): STOPPED shape in dim gray/red-tinted, RUNNING shape in bright green — reinforcing the same "state" signal the shape already conveys.
3. Should the shape itself pulse/brighten in sync with the beat (like the beat-dot row does), doubling up the "alive" feeling, or stay static so it doesn't compete visually with the beat-dot animation?
