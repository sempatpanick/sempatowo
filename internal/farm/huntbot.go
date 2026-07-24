package farm

import (
	"math/rand"
	"sync"
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/gamble"
	"github.com/semptpanick/sempatowo/internal/huntbot"
	"github.com/semptpanick/sempatowo/internal/util"
)

// --- Huntbot (separate from manual hunt; shares send queue via enqueue) ---

func (b *Bot) startGambleIfNeeded() {
	g := b.settings().Features.Gamble
	if !g.Coinflip.Enabled && !g.Slots.Enabled && !g.Blackjack.Enabled {
		b.stopGamble()
		return
	}
	if b.gamble == nil {
		b.gamble = gamble.NewManager(b.newGambleContext())
	}
	b.gamble.Start()
	if b.settings().TrackBalance {
		b.gamble.RequestCash()
	}
}

func (b *Bot) stopGamble() {
	if b.gamble != nil {
		b.gamble.Stop()
	}
}

func (b *Bot) restartGamble() {
	b.stopGamble()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if !ready {
		return
	}
	b.startGambleIfNeeded()
}

func (b *Bot) startHuntbotIfNeeded() {
	if !b.settings().Features.Huntbot.Enabled {
		b.stopHuntbot()
		return
	}
	b.mu.Lock()
	if b.huntbotStarted {
		b.mu.Unlock()
		return
	}
	b.huntbotStarted = true
	b.mu.Unlock()

	ctx := b.newHuntbotContext()
	b.huntbot = huntbot.NewHandler(ctx, b.token)
	util.Go(b.logDanger, "huntbot", b.huntbot.Start)
}

func (b *Bot) stopHuntbot() {
	if b.huntbot != nil {
		b.huntbot.Stop()
		b.huntbot = nil
	}
	b.huntbotStarted = false
}

func (b *Bot) restartHuntbot() {
	b.stopHuntbot()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if ready {
		b.startHuntbotIfNeeded()
	}
}

func (b *Bot) handleHuntbotMessage(msg *discord.Message) {
	if msg == nil || msg.Author == nil || b.huntbot == nil || !b.settings().Features.Huntbot.Enabled {
		return
	}
	hbMsg := huntbot.Message{
		ChannelID: msg.ChannelID.String(),
		AuthorID:  msg.Author.ID.String(),
		Content:   msg.Content,
	}
	for _, e := range msg.Embeds {
		if e == nil {
			continue
		}
		embed := huntbot.MessageEmbed{}
		if e.Author != nil {
			embed.Author = &huntbot.EmbedAuthor{Name: e.Author.Name}
		}
		for _, f := range e.Fields {
			if f == nil {
				continue
			}
			embed.Fields = append(embed.Fields, huntbot.EmbedField{Name: f.Name, Value: f.Value})
		}
		hbMsg.Embeds = append(hbMsg.Embeds, embed)
	}
	for _, a := range msg.Attachments {
		if a == nil {
			continue
		}
		hbMsg.Attachments = append(hbMsg.Attachments, huntbot.Attachment{URL: a.URL})
	}
	b.huntbot.HandleMessage(hbMsg)
}

// huntbotCtx adapts Bot to huntbot.BotContext. Each subsystem talks to the bot
// through one of these narrow interfaces rather than to *Bot directly, which is
// what lets them be tested against a fake.
type huntbotCtx struct {
	bot *Bot
}

func (b *Bot) newHuntbotContext() *huntbotCtx {
	return &huntbotCtx{bot: b}
}

func (c *huntbotCtx) HuntChannelID() string {
	if c == nil || c.bot == nil {
		return ""
	}
	return c.bot.settings().FarmChannel()
}
func (c *huntbotCtx) OwoBotID() string {
	if c == nil || c.bot == nil {
		return ""
	}
	return c.bot.settings().OwoBotID
}
func (c *huntbotCtx) OwnUserID() string {
	if c == nil || c.bot == nil {
		return ""
	}
	return c.bot.userID()
}
func (c *huntbotCtx) Nickname() string {
	if c == nil || c.bot == nil {
		return ""
	}
	client := c.bot.discordClient()
	user := c.bot.discordUser()
	if client != nil && client.State != nil && user != nil {
		for _, guild := range client.State.Guilds {
			if guild == nil {
				continue
			}
			if member, ok := client.State.GetMember(guild.ID, user.ID); ok && member != nil && member.Nick != "" {
				return member.Nick
			}
		}
	}
	return c.bot.username()
}
func (c *huntbotCtx) Settings() config.Huntbot          { return c.bot.settings().Features.Huntbot }
func (c *huntbotCtx) RandomPrefix(cmds []string) string { return c.bot.randomPrefix(cmds) }
func (c *huntbotCtx) CanSend() bool {
	if c == nil || c.bot == nil {
		return false
	}
	return c.bot.canSend()
}
func (c *huntbotCtx) SendMessage(channelID, text string) error {
	if c == nil || c.bot == nil {
		return nil
	}
	c.bot.enqueue(channelID, text)
	return nil
}
func (c *huntbotCtx) Log(msg string) { c.bot.logInfo(msg) }

func (c *huntbotCtx) Sleep(seconds float64) {
	if c == nil || c.bot == nil || seconds <= 0 {
		return
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

// SleepUntil reports false if CancelSleep cut the wait short, so callers can
// tell an elapsed timer from an aborted one.
func (c *huntbotCtx) SleepUntil(seconds, noise float64) bool {
	if c == nil || c.bot == nil {
		return false
	}
	d := seconds
	if noise > 0 {
		d += rand.Float64() * noise
	}
	if d <= 0 {
		return true
	}

	res := make(chan bool, 1)
	var once sync.Once
	timer := time.AfterFunc(time.Duration(d*float64(time.Second)), func() {
		once.Do(func() { res <- true })
	})
	h := &sleepHandle{cancel: func() {
		once.Do(func() {
			timer.Stop()
			res <- false
		})
	}}

	c.bot.sleeper.Set(h)
	elapsed := <-res
	c.bot.sleeper.Clear(h)
	return elapsed
}

func (c *huntbotCtx) CancelSleep() {
	if c == nil || c.bot == nil {
		return
	}
	c.bot.CancelSleep()
}

func (b *Bot) CancelSleep() { b.sleeper.Cancel() }
