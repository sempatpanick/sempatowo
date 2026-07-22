package farm

import (
	"testing"

	discord "github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/types"
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
			name:    "inventory",
			content: "**====== sempatpanick's Inventory ======**\n`065`<:cgem3:510366792024195072>⁸    `066`<:ugem3:510366792095367189>³    `067`<:rgem3:510366792653340674>³",
			want:    "Inventory → 3 items · cgem3×8, ugem3×3, rgem3×3",
		},
		{
			name:    "slots skipped",
			content: "**  `___SLOTS___`**\n` ` <a:slot_gif:417473893368987649> <a:slot_gif:417473893368987649> <a:slot_gif:417473893368987649> ` ` sempatpanick bet <:cowoncy:416043450337853441> 3,000\n  `|         |`\n  `|         |`",
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
			content: "**🙏 | <@229948970904846336>** prays... May luck be in your favor.\n" +
				"**<:blank:427371936482328596> |** You have **286** luck point(s)!",
			want: "Pray → 286 luck — May luck be in your favor.",
		},
		{
			name:    "battle footer",
			content: "",
			want:    "Battle → won in 2 turns (+200 xp, streak 14)",
		},
		{
			name:    "cash skipped",
			content: "<:cowoncy:416043450337853441> **| sempatpanick**, you currently have **__114,308__ cowoncy!**",
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var msg *discord.Message
			if tc.name == "battle footer" {
				msg = &discord.Message{
					Embeds: []*discord.Embed{{
						Author: &types.EmbedAuthor{Name: nick + "'s Battle"},
						Footer: &types.EmbedFooter{Text: "You won in 2 turns!  |  +200 xp  |  Streak: 14"},
					}},
				}
			}
			got := summarizeOwOFromMessage(msg, tc.content, nick)
			if got != tc.want {
				t.Fatalf("summarizeOwOFromMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSummarizePray(t *testing.T) {
	msg := "**🙏 | <@229948970904846336>** prays... May luck be in your favor.\n" +
		"**<:blank:427371936482328596> |** You have **286** luck point(s)!"
	want := "Pray → 286 luck — May luck be in your favor."
	if got := summarizePray(msg); got != want {
		t.Fatalf("summarizePray() = %q, want %q", got, want)
	}
}

func TestSummarizeBattle(t *testing.T) {
	cases := []struct {
		footer string
		want   string
	}{
		{
			footer: "You won in 2 turns!  |  +200 xp  |  Streak: 14",
			want:   "Battle → won in 2 turns (+200 xp, streak 14)",
		},
		{
			footer: "You lost in 16 turns!  |  +63 xp  |  Streak ended at 4",
			want:   "Battle → lost in 16 turns (+63 xp, streak ended at 4)",
		},
		// OwO comma-groups from 1,000 up. These used to drop the xp entirely.
		{
			footer: "You won in 3 turns!  |  +2,800 xp  |  Streak: 21",
			want:   "Battle → won in 3 turns (+2,800 xp, streak 21)",
		},
		{
			footer: "You won in 4 turns!  |  +12,345 xp  |  Streak: 30",
			want:   "Battle → won in 4 turns (+12,345 xp, streak 30)",
		},
		// Just under the grouping boundary.
		{
			footer: "You won in 1 turns!  |  +999 xp  |  Streak: 2",
			want:   "Battle → won in 1 turns (+999 xp, streak 2)",
		},
		// Ungrouped four-digit input still renders grouped, so the log is
		// consistent regardless of how OwO formatted it.
		{
			footer: "You won in 2 turns!  |  +1500 xp  |  Streak: 5",
			want:   "Battle → won in 2 turns (+1,500 xp, streak 5)",
		},
		{
			footer: "You lost in 9 turns!  |  -1,200 xp  |  Streak ended at 7",
			want:   "Battle → lost in 9 turns (-1,200 xp, streak ended at 7)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := summarizeBattle(tc.footer); got != tc.want {
				t.Fatalf("summarizeBattle() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSummarizeOwOMessageLegacy(t *testing.T) {
	nick := "sempatpanick"
	got := summarizeOwOMessage("You won in 6 turns!  |  +200 xp  |  Streak: 13", nick)
	want := "Battle → won in 6 turns (+200 xp, streak 13)"
	if got != want {
		t.Fatalf("summarizeOwOMessage() = %q, want %q", got, want)
	}
}

func TestOwoMessageText(t *testing.T) {
	raw := "**sempatpanick** spent <:cowoncy:416043450337853441> `2,000`"
	want := "sempatpanick spent 2,000"
	if got := owoMessageText(raw); got != want {
		t.Fatalf("owoMessageText() = %q, want %q", got, want)
	}
}

func TestOwoMessageTextFromMsg(t *testing.T) {
	msg := &discord.Message{
		Embeds: []*discord.Embed{{
			Footer: &types.EmbedFooter{Text: "You won in 2 turns!  |  +200 xp  |  Streak: 14"},
		}},
	}
	got := owoMessageTextFromMsg(msg)
	want := "You won in 2 turns! | +200 xp | Streak: 14"
	if got != want {
		t.Fatalf("owoMessageTextFromMsg() = %q, want %q", got, want)
	}
}

func TestShouldSkipOwOLogSlotsFallback(t *testing.T) {
	raw := "**  `___SLOTS___`**\nsempatpanick bet <:cowoncy:416043450337853441> 3,000\n  `|         |`"
	if got := owoMessageText(raw); got != "" {
		t.Fatalf("owoMessageText() = %q, want empty for slots", got)
	}
	msg := &discord.Message{Content: raw}
	if got := summarizeOwOFromMessage(msg, raw, "sempatpanick"); got != "" {
		t.Fatalf("summarizeOwOFromMessage() = %q, want empty for slots", got)
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
