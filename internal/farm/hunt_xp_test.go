package farm

import "testing"

// handleHuntGems accumulates totalXP from every "gained **N xp**" in the
// message. OwO comma-groups from 1,000 up, and the old \d+ pattern skipped
// those matches entirely — the total stayed silently low with nothing in the
// log to indicate it.
func TestHandleHuntGemsAccumulatesXP(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "small ungrouped value",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **2xp**!",
			want:    2,
		},
		{
			name:    "just under the grouping boundary",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **999xp**!",
			want:    999,
		},
		{
			name:    "comma grouped",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **2,800xp**!",
			want:    2800,
		},
		{
			name:    "large comma grouped",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **1,234,567xp**!",
			want:    1234567,
		},
		{
			name:    "ungrouped four digits still counts",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **1500xp**!",
			want:    1500,
		},
		{
			name: "multiple gains in one message",
			content: "**user**, hunt is empowered! <a:gfox:1> gained **2,800xp**! " +
				"<a:gcat:2> gained **150xp**!",
			want: 2950,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := New("token", nil)
			b.handleHuntGems(tc.content, "user")

			if _, xp := b.stats.Totals(); xp != tc.want {
				t.Errorf("totalXP = %d, want %d", xp, tc.want)
			}
		})
	}
}
