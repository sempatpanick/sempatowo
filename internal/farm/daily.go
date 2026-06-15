package farm

import (
	discord "github.com/hytams/discordgo-self"
	"github.com/sempatowo/sempatowo/internal/daily"
)

type dailyCtx struct {
	bot *Bot
}

func (b *Bot) newDailyContext() *dailyCtx {
	return &dailyCtx{bot: b}
}

func (c *dailyCtx) UserID() string             { return c.bot.userID() }
func (c *dailyCtx) HuntChannelID() string    { return c.bot.settings().Channels.Hunt }
func (c *dailyCtx) OwoBotID() string         { return c.bot.settings().OwoID }
func (c *dailyCtx) AutoDaily() bool          { return c.bot.settings().AutoDaily }
func (c *dailyCtx) CashCheck() bool          { return c.bot.settings().CashCheck }
func (c *dailyCtx) RandomPrefix(cmds []string) string { return c.bot.randomPrefix(cmds) }
func (c *dailyCtx) SendMessage(channelID, text string) error {
	c.bot.enqueue(channelID, text)
	return nil
}
func (c *dailyCtx) CanSend() bool { return c.bot.canSend() }
func (c *dailyCtx) Log(msg string) { c.bot.logInfo(msg) }
func (c *dailyCtx) Nickname() string {
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
func (c *dailyCtx) OnDailyReward(amount int) {
	if c.bot.gamble != nil && c.bot.settings().CashCheck {
		c.bot.gamble.UpdateBalance(amount, true)
	}
}

func (b *Bot) startDailyIfNeeded() {
	if !b.settings().AutoDaily {
		b.stopDaily()
		return
	}
	uid := b.userID()
	if uid == "" {
		return
	}
	if b.daily == nil {
		store := daily.NewStore("data", uid)
		b.daily = daily.NewManager(b.newDailyContext(), store)
	}
	b.daily.Start()
}

func (b *Bot) stopDaily() {
	if b.daily != nil {
		b.daily.Stop()
	}
}

func (b *Bot) restartDaily() {
	b.stopDaily()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if !ready {
		return
	}
	b.startDailyIfNeeded()
}

func (b *Bot) handleDailyMessage(msg *discord.Message, nick string) {
	if b.daily == nil || msg == nil {
		return
	}
	b.daily.HandleMessage(msg.Content, nick)
}
