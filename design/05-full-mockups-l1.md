# Full-Screen Mockups — L1 (Shape + Lettering Combo)

Final direction: the L1 banner from `03-status-banner-variations.md` —
compact shape icon beside figlet `mini` lettering, 3 rows tall for both
states so toggling start/stop never resizes the box. All six states from
`02-mockups.md` / `04-full-mockups-v2.md`, updated to this banner.

## 1. Startup (stopped, default 120 BPM)

```
┌────────────────────────────────────────────────┐
│ ██████  ______  _  _  _ _                        │
│ ██████ (_  |/ \|_)|_)|_| \                       │
│ ██████ __) |\_/|  |  |_|_/                       │
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
│ ██◥    _        ___     __                        │
│ ██████|_)|  /\\_/| |\ |/__                       │
│ ██◢   |  |_/--\|_|_| \|\_|                       │
│                                                  │
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
│ ██◥    _        ___     __                        │
│ ██████|_)|  /\\_/| |\ |/__                       │
│ ██◢   |  |_/--\|_|_| \|\_|                       │
│                                                  │
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
│ ██◥    _        ___     __                        │
│ ██████|_)|  /\\_/| |\ |/__                       │
│ ██◢   |  |_/--\|_|_| \|\_|                       │
│                                                  │
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
│ ██◥    _        ___     __                        │
│ ██████|_)|  /\\_/| |\ |/__                       │
│ ██◢   |  |_/--\|_|_| \|\_|                       │
│                                                  │
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
│ ██████  ______  _  _  _ _                        │
│ ██████ (_  |/ \|_)|_)|_| \                       │
│ ██████ __) |\_/|  |  |_|_/                       │
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

## Implementation notes

- Both banner states are exactly 3 rows tall (shape column padded to match
  lettering height), so start/stop never causes the box to grow/shrink.
- Shape column is a fixed 6-character-wide gutter; lettering starts at the
  same column (col 9) in every state.
- Lettering generated via `figlet -f mini STOPPED` / `figlet -f mini
  PLAYING` — regenerate directly from figlet at build/render time rather
  than hardcoding the strings, so it stays correctly kerned if the font or
  word ever changes.
- Color layered on top (not required for legibility, since shape + word are
  already redundant signals): dim/red tint for stopped, bright green for
  running.
