package main

import "github.com/chriserin/mn/internal/bignum"

// renderBigNumber renders n's decimal digits as multi-line ASCII art at the
// given integer scale (1 = native size). See internal/bignum for the glyph
// data and scaling, and cmd/bignumpreview for a standalone way to preview a
// scale before wiring it into Model.bannerScale.
func renderBigNumber(n, scale int) string {
	return bignum.Render(n, scale)
}
