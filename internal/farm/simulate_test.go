package farm

import (
	"testing"

	discord "github.com/hytams/discordgo-self"
)

func TestBuildSimulateCaptchaMessageRequiresUser(t *testing.T) {
	b := New("token")
	if msg := b.buildSimulateCaptchaMessage(); msg != nil {
		t.Fatal("expected nil without connected user")
	}
}

func TestSimulateCaptchaMessageDetects(t *testing.T) {
	uid := "229948970904846336"
	id, err := discord.ParseSnowflake(uid)
	if err != nil {
		t.Fatal(err)
	}
	msg := &discord.Message{
		GuildID:  1,
		Content:  "**⚠️ | testuser**, please complete your captcha to verify that you are human! (simulate)",
		Mentions: []*discord.User{{ID: id}},
	}
	if !detectCaptcha(msg, normalizeZW(msg.Content), "testuser", uid, "testuser") {
		t.Fatal("simulate sample should match detectCaptcha")
	}
}
