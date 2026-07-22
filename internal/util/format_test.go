package util

import "testing"

func TestParseAmount(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"280", 280, true},
		{"+280", 280, true},
		{"-280", -280, true},
		{"999", 999, true},
		// The grouping boundary — where a plain Atoi starts failing.
		{"1,000", 1000, true},
		{"2,800", 2800, true},
		{"+2,800", 2800, true},
		{"-1,200", -1200, true},
		{"12,345", 12345, true},
		{"1,234,567", 1234567, true},
		// Ungrouped large values must keep working.
		{"2800", 2800, true},
		{" +2,800 ", 2800, true},
		{"", 0, false},
		{"abc", 0, false},
		{"2.800", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := ParseAmount(tc.in)
			if ok != tc.ok {
				t.Fatalf("ParseAmount(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("ParseAmount(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseAmountRoundTripsFormatInt(t *testing.T) {
	for _, n := range []int{0, 7, 999, 1000, 2800, 12345, 1234567, -1200} {
		if got, ok := ParseAmount(FormatInt(n)); !ok || got != n {
			t.Errorf("round trip of %d: ParseAmount(%q) = %d, %v", n, FormatInt(n), got, ok)
		}
	}
}
