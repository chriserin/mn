package main

import (
	"strconv"
	"strings"
)

// bigDigitGlyphs holds the ASCII art for each digit 0-9 in figlet's "big"
// font, six rows tall. Captured via `figlet -f big -w 200 <digit>` for each
// digit (figlet pads every glyph to a fixed box width and emits two
// trailing blank rows per character, which are dropped here).
var bigDigitGlyphs = [10][6]string{
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

// renderBigNumber renders n's decimal digits as multi-line ASCII art,
// composited from bigDigitGlyphs with a single-space gap between digits.
func renderBigNumber(n int) string {
	digits := strconv.Itoa(n)
	rows := make([]string, len(bigDigitGlyphs[0]))
	for i, d := range digits {
		glyph := bigDigitGlyphs[d-'0']
		for row := range rows {
			if i > 0 {
				rows[row] += " "
			}
			rows[row] += glyph[row]
		}
	}
	return strings.Join(rows, "\n")
}
