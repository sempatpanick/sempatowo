package farm

import (
	"testing"

	discord "github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/types"
)

func TestDetectCaptchaExplicitPrompt(t *testing.T) {
	// Full verification phrase in DM — always for this account.
	msg := &discord.Message{Content: "Please complete your captcha to verify that you are human!"}
	if !detectCaptcha(msg, msg.Content, "nick", "123", "user") {
		t.Fatal("expected explicit captcha prompt to match in DM")
	}
}

func TestDetectCaptchaEmbedMention(t *testing.T) {
	id, _ := discord.ParseSnowflake("229948970904846336")
	msg := &discord.Message{
		GuildID: 1,
		Content: "",
		Mentions: []*discord.User{{ID: id}},
		Embeds: []*discord.Embed{{
			Description: "**⚠️ | sempatpanick**, please check your DMs and complete captcha!",
		}},
	}
	uid := "229948970904846336"
	if !detectCaptcha(msg, "", "sempatpanick", uid, "sempatpanick") {
		t.Fatal("expected captcha in embed with mention")
	}
}

func TestDetectCaptchaVerifyButton(t *testing.T) {
	msg := &discord.Message{
		GuildID: 1,
		Content: "**⚠️ | sempatpanick** complete captcha",
		Components: []types.Component{
			map[string]interface{}{
				"type": 1,
				"components": []interface{}{
					map[string]interface{}{"type": 2, "label": "Verify"},
				},
			},
		},
	}
	if !detectCaptcha(msg, msg.Content, "sempatpanick", "123", "sempatpanick") {
		t.Fatal("expected verify button captcha")
	}
}

func TestDetectCaptchaNotForOtherUser(t *testing.T) {
	msg := &discord.Message{
		GuildID: 1,
		Content: "**⚠️ | someoneelse** please complete captcha",
	}
	if detectCaptcha(msg, msg.Content, "sempatpanick", "123", "sempatpanick") {
		t.Fatal("should not match captcha for another user")
	}
}

func TestDetectCaptchaRealHumanLink(t *testing.T) {
	// OwO inserts zero-width chars between letters — must strip before matching.
	zw := "\u200b"
	content := "**⚠️ |** <@229948970904846336>, a" + zw + "re yo" + zw + "u a" + zw + " re" + zw + "al huma" + zw + "n? Plea" + zw + "se u" + zw + "se th" + zw + "e l" + zw + "ink b" + zw + "elow s" + zw + "o I" + zw + " c" + zw + "an chec" + zw + "k!\n" +
		"<:blank:427371936482328596> **|** P" + zw + "lease complet" + zw + "e th" + zw + "is with" + zw + "in 1" + zw + "0 m" + zw + "inutes o" + zw + "r i" + zw + "t m" + zw + "ay re" + zw + "sult i" + zw + "n a" + zw + " ban" + zw + "!"
	id, _ := discord.ParseSnowflake("229948970904846336")
	msg := &discord.Message{
		GuildID: 1,
		Content: content,
		Mentions: []*discord.User{{ID: id}},
	}
	uid := "229948970904846336"
	if !detectCaptcha(msg, normalizeZW(content), "sempatpanick", uid, "sempatpanick") {
		t.Fatal("expected real-human link captcha sample to match")
	}
}

func TestDetectCaptchaVerifyCount(t *testing.T) {
	zw := "\u200b"
	content := "⚠️ **|** <@229948970904846336>! Pl" + zw + "ease compl" + zw + "ete yo" + zw + "ur capt" + zw + "cha t" + zw + "o ve" + zw + "rify th" + zw + "at yo" + zw + "u ar" + zw + "e hu" + zw + "man! (5/5)"
	id, _ := discord.ParseSnowflake("229948970904846336")
	msg := &discord.Message{
		GuildID: 1,
		Content: content,
		Mentions: []*discord.User{{ID: id}},
	}
	if !detectCaptcha(msg, normalizeZW(content), "sempatpanick", "229948970904846336", "sempatpanick") {
		t.Fatal("expected (5/5) captcha verify sample to match")
	}
}
