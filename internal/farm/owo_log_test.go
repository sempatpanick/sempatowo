package farm

import (
	"testing"

	"github.com/semptpanick/sempatowo/internal/util"
)

func TestSummarizeOwOMessage(t *testing.T) {
	nick := "sempatpanick"
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "hunt",
			content: "**🌱 | sempatpanick**, hunt is empowered by <:egem3:510366792800272394>`[190/450]`  !\n" +
				"**<:blank:427371936482328596> |** You found:  :butterfly: :snail:\n" +
				"<:blank:427371936482328596> **|** <a:gfox:418291892376305664> gained **2xp**!",
			want: "Hunt → butterfly, snail (+2 pet xp)",
		},
		{
			name:    "inventory skipped",
			content: "**====== sempatpanick's Inventory ======**\n`065`<:cgem3:510366792024195072>⁸",
			want:    "",
		},
		{
			name:    "slots skipped",
			content: "**  `___SLOTS___`**\nsempatpanick bet <:cowoncy:416043450337853441> 3,000",
			want:    "",
		},
		{
			name:    "coinflip skipped",
			content: "**sempatpanick** spent **<:cowoncy:416043450337853441> 2,000** and chose **heads**\nThe coin spins...",
			want:    "",
		},
		{
			name:    "rank",
			content: "Rank #4,943,098 • @sempatpanick",
			want:    "Rank #4,943,098",
		},
		{
			name: "pray",
			content: "**🙏 | <@229948970904846336>** prays... Luck is on your side!\n" +
				"**<:blank:427371936482328596> |** You have **283** luck point(s)!",
			want: "Pray → 283 luck",
		},
		{
			name:    "battle",
			content: "You won in 6 turns!  |  +200 xp  |  Streak: 13",
			want:    "Battle → won in 6 turns (+200 xp, streak 13)",
		},
		{
			name:    "cash skipped",
			content: "<:cowoncy:416043450337853441> **| sempatpanick**, you currently have **__114,308__ cowoncy!**",
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := summarizeOwOMessage(tc.content, nick)
			if got != tc.want {
				t.Fatalf("summarizeOwOMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStripDiscordText(t *testing.T) {
	in := "**sempatpanick** spent <:cowoncy:416043450337853441> `2,000`"
	want := "sempatpanick spent 2,000"
	if got := stripDiscordText(in); got != want {
		t.Fatalf("stripDiscordText() = %q, want %q", got, want)
	}
}

func TestFormatInt(t *testing.T) {
	if got := util.FormatInt(114308); got != "114,308" {
		t.Fatalf("FormatInt() = %q", got)
	}
}
