# mn

A metronome TUI built with Bubble Tea, with tempo control and a beat-pulse
visualization.

## Install and run

```
go build -o mn .
./mn
```

or run directly:

```
go run .
```

## Key bindings

| Key                       | Action                   |
| ------------------------- | ------------------------ |
| `space`                   | Start/stop the metronome |
| `up` / `down`             | Adjust BPM by 1          |
| `shift+up` / `shift+down` | Adjust BPM by 10         |
| `q` / `ctrl+c`            | Quit                     |

BPM is clamped between 20 and 300.

## Development

This project follows a phased workflow:

1. Design documents in `design/`, discussed before implementation.
2. Gherkin scenarios in `.ft` files under `fts/`, tracked with the
   [`ft`](https://github.com/chriserin/ft) CLI.
3. Scenarios marked `ready` in `ft list` are implemented test-first.

Run the test suite with:

```
go test ./...
```

Check scenario status with:

```
ft status
```
