// Package bignum renders decimal numbers as multi-line ASCII-art digits, at
// one of a few discrete integer scales. It's factored out of the main mn
// package so both the metronome app and the standalone cmd/bignumpreview
// tool (for eyeballing a scale before wiring it into the app) can render
// identically without duplicating the glyph data.
package bignum

import (
	"strconv"
	"strings"
)

// Glyphs holds the ASCII art for each digit 0-9 in figlet's "big" font, six
// rows tall. Captured via `figlet -f big -w 200 <digit>` for each digit
// (figlet pads every glyph to a fixed box width and emits two trailing
// blank rows per character, which are dropped here).
var Glyphs = [10][]string{
	{`  ___  `, ` / _ \ `, `| | | |`, `| | | |`, `| |_| |`, ` \___/ `},
	{` __ `, `/_ |`, ` | |`, ` | |`, ` | |`, ` |_|`},
	{` ___  `, `|__ \ `, `   ) |`, `  / / `, ` / /_ `, `|____|`},
	{` ____  `, `|___ \ `, `  __) |`, ` |__ < `, ` ___) |`, `|____/ `},
	{` _  _   `, `| || |  `, `| || |_ `, `|__   _|`, `   | |  `, `   |_|  `},
	{` _____ `, `| ____|`, `| |__  `, `|___ \ `, ` ___) |`, `|____/ `},
	{`   __  `, `  / /  `, ` / /_  `, `| '_ \ `, `| (_) |`, ` \___/ `},
	{` ______ `, `|____  |`, `    / / `, `   / /  `, `  / /   `, ` /_/    `},
	{`  ___  `, ` / _ \ `, `| (_) |`, ` > _ < `, `| (_) |`, ` \___/ `},
	{`  ___  `, ` / _ \ `, `| (_) |`, ` \__, |`, `   / / `, `  /_/  `},
}

// Glyphs2x and Glyphs3x are discrete, hand-adjustable glyph sets for the
// larger banner tiers (see Model.bannerScale) — not computed from Glyphs at
// render time. Each digit was seeded from Glyphs by scaling every character
// into a scale x scale cell, then baked into source here the same way
// Glyphs itself was captured from figlet once rather than generated at
// runtime. '/' and '\\' cells are staircased rather than filled solid:
// each of the scale output rows for a given source row places the slash at
// a different column (stepping toward bottom-right for '\\', bottom-left
// for '/'), so an enlarged diagonal still reads as a diagonal instead of a
// blocky square. Baking the result in like this means each digit at each
// tier can still be hand touched up afterward without fighting a shared
// scaling formula that has to look right at every size at once.
var Glyphs2x = [10][]string{
	{`    ______    `, `    ______    `, `   /  __  \   `, `  /   __   \  `, `||  ||  ||  ||`, `||  ||  ||  ||`, `||  ||  ||  ||`, `||  ||  ||  ||`, `||  ||__||  ||`, `||  ||__||  ||`, `  \ ______ /  `, `   \______/   `},
	{`   __   `, `  /___  `, ` //   ||`, `//_   ||`, `  ||  ||`, `  ||  ||`, `  ||  ||`, `  ||  ||`, `  ||  ||`, `  ||  ||`, `  || _||`, `  ||__||`},                                                                                                 // digit 1
	{`   ____     `, `  ______    `, ` |____  \   `, `||_____  \\ `, `       )  ||`, `      )   | `, `     /   /  `, `    /   /   `, `   /   /    `, `  /   / __  `, ` ||_______||`, `||_______|| `},                                                 // digit 2
	{`   ______     `, `  ________    `, ` |______  \   `, `||______   \  `, `      __))  ||`, `      __))  | `, `  ||____  <<  `, `  ||____  <<  `, `      __))  | `, `      __))  ||`, ` |________ /  `, `||________/   `},                         // digit 3
	{`   _     _      `, `  __    __      `, `||  |  |  ||    `, `||  |  |  ||    `, `||  |  |  ||_   `, `||  ||||  ||__  `, `||__         _||`, `||____      __||`, `      ||  ||    `, `      ||  ||    `, `      || _||    `, `      ||__||    `}, // digit 4
	{`    ______    `, `  __________  `, `||  ________||`, `||  ________||`, `||  ||__      `, `||  ||____    `, `||______  \   `, `||______   \  `, `      __))  ||`, `      __))  ||`, `||________ /  `, `||________/   `},                         // digit 5
	{`      ____    `, `      ____    `, `     /   /    `, `    /   /     `, `   /   /_     `, `  /   / __    `, ` |   '__  \   `, `||  ''__   \  `, `||  ((  ))  ||`, ` |  ((__))  | `, `  \  ____  /  `, `   \______/   `},                         // digit 6
	{`    ________    `, `  ____________  `, `||________    ||`, `||________    ||`, `         /   /  `, `        /   /   `, `       /   /    `, `      /   /     `, `     /   /      `, `    /   /       `, `   /__ /        `, `  / __/         `}, // digit 7
	{`     ____     `, `    ______    `, `   /  _   \   `, `  /   __   \  `, `||  ((  ))  ||`, `||  ((__))  ||`, `  >>  _   <<  `, `  >>  __  <<  `, `||  ((  ))  ||`, `||  ((__))  ||`, `  \  ____  /  `, `   \______/   `},                         // digit 8
	{`     ____     `, `    ______    `, `  //  _   \   `, ` //   __   \  `, `||  ((  ))  | `, `||  ((__))  ||`, ` \\ ____,,  ||`, `  \\____,   | `, `       /   /  `, `      /   /   `, `     /__ /    `, `    / __/     `},                         // digit 9

}

var Glyphs3x = [10][]string{
	{`      _________      `, `      _________      `, `      _________      `, `     /   ___   \     `, `    /    ___    \    `, `   /     ___     \   `, `|||   |||   |||   |||`, `|||   |||   |||   |||`, `|||   |||   |||   |||`, `|||   |||   |||   |||`, `|||   |||   |||   |||`, `|||   |||   |||   |||`, `|||   |||___|||   |||`, `|||   |||___|||   |||`, `|||   |||___|||   |||`, `   \  _________  /   `, `    \ _________ /    `, `     \_________/     `},
	{`     _______`, `    _______ `, `   _______  `, `  /___   |||`, ` //___   |||`, `// ___   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||   |||`, `   |||  _|||`, `   ||| __|||`, `   |||___|||`},                                                                                                                                                                                                                         // digit 1
	{`     _______      `, `    ________      `, `   _________      `, ` ||______   \     `, `|||______    \    `, `|||______     \\  `, `         ))    |\ `, `         )))   |||`, `         ))    |/ `, `        /     /   `, `       /     /    `, `      /     /     `, `     /     /_     `, `   //     //__    `, `  //     // ___   `, ` /|____________|| `, `/||____________|||`, `|||____________|||`},                                                                                                             // digit 2
	{`    _________        `, `   ___________       `, `   ____________      `, ` ||________    \     `, `|||______       \    `, ` ||_________     \\  `, `      ______))    |\ `, `           _)))   |||`, `      ______))    |/ `, `    ||______    <<   `, `   ||______    <<    `, `    ||______    <<   `, `     _______))    |\ `, `          __)))   |||`, `     _______))    |/ `, ` ||_________     /   `, `|||__________   /    `, ` ||____________/     `},                                                       // digit 3
	{`    _        _          `, `   ___      ___         `, `   ___      ___         `, `|||   ||  ||   |||      `, `|||   ||  ||   |||      `, `|||   ||  ||   |||      `, `|||   ||  ||   |||_     `, `|||   ||  ||   |||__    `, `|||   ||||||   |||___   `, `|||____           _|||  `, `|||_____          __||| `, `|||______         ___|||`, `         |||   |||      `, `         |||   |||      `, `         |||   |||      `, `         |||  _|||      `, `         |||_ _|||      `, `         |||___|||      `}, // digit 4
	{`     ___________      `, `    _____________     `, `   _______________    `, ` ||   __________|||   `, ` ||   ___________|||  `, ` ||   ____________||| `, `|||   |_____          `, `|||   ||_____         `, `|||   |||______       `, ` ||_________   \      `, ` ||_________    \     `, ` ||_________     \    `, `         ___)))   ||| `, `           __)))   |||`, `   _________)))   ||| `, `|||____________  /    `, `|||____________ /     `, `|||____________/      `},                                     // digit 5
	{`          ____      `, `          _____     `, `         ______     `, `        /     /     `, `       /     /      `, `      /     /       `, `     /     /_       `, `    /     / __      `, `   /     /  ___     `, `  |     '_     \    `, ` ||    ''__     \   `, `|||   '''___     \  `, `|||    ((   ))    | `, ` ||   ((     ))   ||`, `  |    ((___))    | `, `   \\    ___    //  `, `    \\  _____  //   `, `     \_________/    `},                                                                         // digit 6
	{`       ____________     `, `     _______________    `, `   __________________   `, ` ||____________      || `, `|||____________      |||`, ` ||____________      || `, `              //    /   `, `             //    /    `, `            //    /     `, `           /     /      `, `          /     /       `, `         /     /        `, `        /     /         `, `       /    //          `, `      /    //           `, `     /___ //            `, `    / ___ /             `, `   /  ___/              `}, // digit 7
	{`        _____        `, `       _______       `, `      _________      `, `     /    _    \     `, `    //   ___   \\    `, `   /     ___     \   `, ` ||    ((   ))    || `, `|||   ((     ))   |||`, ` ||    ((___))    || `, `   >>    ___    <<   `, `    >>    _    <<    `, `   >>    ___    <<   `, ` ||    ((   ))    || `, `|||   ((     ))   |||`, ` ||    ((___))    || `, `   \     ___     /   `, `    \\  _____  //    `, `     \_________/     `},                                                       // digit 8
	{`        _____        `, `       _______       `, `      _________      `, `     /    _    \     `, `    //   ___   \\    `, `   /     ___     \   `, ` ||    ((   ))    \\ `, `|||   ((     ))   |||`, ` ||    ((___))    |||`, `   \\   ____,     |/ `, `    \\ _____,,    /  `, `     \_____,,,   //  `, `           /     /   `, `          /     /    `, `         /     /     `, `        /___ //      `, `       //___//       `, `      // ___/        `},                                                       // digit 9

}

// MaxScale is the largest scale Render supports (Glyphs3x). Scales above it
// clamp down to MaxScale; scales below 1 clamp up to 1.
const MaxScale = 3

// tiers indexes the discrete glyph sets by scale (1-based; index 0 unused).
var tiers = [MaxScale + 1][10][]string{
	1: Glyphs,
	2: Glyphs2x,
	3: Glyphs3x,
}

// Render renders n's decimal digits as multi-line ASCII art, composited
// from the glyph tier matching scale (clamped to [1, MaxScale]) with a gap
// between digits that grows with scale (see gapFor) so the spacing keeps
// pace with the enlarged glyphs instead of looking cramped next to them.
// Every digit is center-padded to the tier's widest digit (see tierWidth)
// before compositing, so each occupies the same column width — otherwise a
// narrow digit like 1 sits closer to its neighbor than a wide one like 0,
// and the whole banner visibly reflows width as the BPM changes digit by
// digit.
func Render(n, scale int) string {
	if scale < 1 {
		scale = 1
	}
	if scale > MaxScale {
		scale = MaxScale
	}
	glyphs := tiers[scale]
	gap := strings.Repeat(" ", gapFor(scale))
	digitWidth := tierWidth(glyphs)

	digits := strconv.Itoa(n)
	rows := make([]string, len(glyphs[0]))
	for i, d := range digits {
		glyph := glyphs[d-'0']
		for row := range rows {
			if i > 0 {
				rows[row] += gap
			}
			rows[row] += centerPad(glyph[row], digitWidth)
		}
	}
	out := rows[0]
	for _, r := range rows[1:] {
		out += "\n" + r
	}
	return out
}

// gapFor returns the number of spaces left between digits at the given
// scale: 1 at native size, 2 at 2x, 3 at 3x.
func gapFor(scale int) int {
	return scale
}

// tierWidth returns the widest row across every digit in glyphs, i.e. the
// fixed column width every digit in that tier gets center-padded to.
func tierWidth(glyphs [10][]string) int {
	width := 0
	for _, glyph := range glyphs {
		for _, row := range glyph {
			if len(row) > width {
				width = len(row)
			}
		}
	}
	return width
}

// centerPad pads s with spaces on both sides to width, leaving s unchanged
// if it's already at or beyond width. Any odd leftover space goes on the
// right.
func centerPad(s string, width int) string {
	pad := width - len(s)
	if pad <= 0 {
		return s
	}
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
