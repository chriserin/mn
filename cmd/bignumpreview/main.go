// Command bignumpreview renders a number at one or more scales with
// internal/bignum, so a scale can be eyeballed in a real terminal before
// it's wired into Model.bannerScale.
//
// Usage:
//
//	go run ./cmd/bignumpreview [-n 120] [-scale 1,2,3]
package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/chriserin/mn/internal/bignum"
)

func main() {
	n := flag.Int("n", 120, "number to render")
	scales := flag.String("scale", "1,2,3", "comma-separated integer scales to render side by side")
	flag.Parse()

	var factors []int
	for _, s := range strings.Split(*scales, ",") {
		f, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			fmt.Printf("invalid -scale value %q: %v\n", s, err)
			return
		}
		factors = append(factors, f)
	}

	for _, f := range factors {
		out := bignum.Render(*n, f)
		fmt.Printf("scale %dx (%d cols x %d rows):\n%s\n\n",
			f, lineWidth(out), strings.Count(out, "\n")+1, out)
	}
}

func lineWidth(s string) int {
	width := 0
	for _, line := range strings.Split(s, "\n") {
		if len(line) > width {
			width = len(line)
		}
	}
	return width
}
