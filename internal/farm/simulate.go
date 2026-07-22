package farm

import (
	"fmt"
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/util"
)

// RunSimulateCaptcha connects to Discord, injects a fake OwO captcha after ready,
// and blocks so you can verify pause, browser open, and notifications.
func (b *Bot) RunSimulateCaptcha() error {
	b.simulateCaptcha = true
	return b.run()
}

func (b *Bot) scheduleSimulateCaptcha() {
	b.log.Info("Simulate captcha mode — auto farm will not start")
	b.log.Info("Injecting fake OwO captcha in 2s...")
	time.AfterFunc(2*time.Second, func() {
		defer util.Recover(b.logDanger, "simulateCaptcha")
		b.injectSimulateCaptcha()
	})
}

func (b *Bot) injectSimulateCaptcha() {
	msg := b.buildSimulateCaptchaMessage()
	if msg == nil {
		b.logDanger("Simulate captcha failed — user or config not ready")
		return
	}

	b.log.Info("Injecting fake OwO captcha message...")
	b.onMessage(msg)

	b.mu.Lock()
	paused := b.captchaSolving && !b.active
	b.mu.Unlock()

	if paused {
		b.log.Info("Simulate OK — farm paused. Check browser/notification, then solve or Ctrl+C to exit.")
		b.log.Info("After solving, OwO verification success will resume the farm automatically.")
		return
	}
	b.logDanger("Simulate FAILED — captcha was not detected; check detection logs above")
}

func (b *Bot) buildSimulateCaptchaMessage() *discord.Message {
	uid := b.userID()
	if uid == "" || b.cfg.Load() == nil {
		return nil
	}
	s := b.settings()
	owoID, err := discord.ParseSnowflake(s.OwoID)
	if err != nil {
		return nil
	}
	userID, err := discord.ParseSnowflake(uid)
	if err != nil {
		return nil
	}

	name := b.username()
	content := fmt.Sprintf(
		"**⚠️ | %s**, please complete your captcha to verify that you are human! (simulate)",
		name,
	)

	return &discord.Message{
		GuildID: 1,
		Content: content,
		Author:  &discord.User{ID: owoID},
		Mentions: []*discord.User{
			{ID: userID},
		},
	}
}
