package util

import (
	"fmt"
	"strings"
)

var superscriptMap = map[rune]rune{
	'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4',
	'⁵': '5', '⁶': '6', '⁷': '7', '⁸': '8', '⁹': '9',
}

// SuperscriptToNumber converts superscript digits (e.g. "⁴") to an integer.
func SuperscriptToNumber(s string) int {
	var b strings.Builder
	for _, ch := range s {
		if mapped, ok := superscriptMap[ch]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(ch)
		}
	}
	var n int
	fmt.Sscanf(b.String(), "%d", &n)
	return n
}
