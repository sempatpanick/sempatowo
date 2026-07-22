package util

import (
	"fmt"
	"strconv"
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

// ParseAmount parses an integer that may carry thousands separators and a
// leading sign, e.g. "+2,800" → 2800.
//
// OwO writes any value of 1,000 or more with commas, so a plain strconv.Atoi
// silently fails on exactly the large numbers that matter most.
func ParseAmount(s string) (int, bool) {
	n, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(s), ",", ""))
	return n, err == nil
}

// FormatInt formats an integer with thousands separators (e.g. 114308 → "114,308").
func FormatInt(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return sign + s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return sign + strings.Join(parts, ",")
}
