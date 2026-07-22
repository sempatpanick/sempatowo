package farm

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/gamble"
	"github.com/semptpanick/sempatowo/internal/quest"
)

type autoQuestCtx struct {
	bot *Bot
}

func (b *Bot) newAutoQuestContext() *autoQuestCtx {
	return &autoQuestCtx{bot: b}
}

func (c *autoQuestCtx) UserID() string        { return c.bot.userID() }
func (c *autoQuestCtx) Username() string      { return c.bot.username() }
func (c *autoQuestCtx) DisplayName() string   { return c.bot.username() }
func (c *autoQuestCtx) HuntChannelID() string { return c.bot.settings().Channels.Hunt }
func (c *autoQuestCtx) QuestHelpChannelID() string {
	hc := c.bot.settings().AutoQuest.HelpChannel
	if hc.ChannelID != "" {
		return hc.ChannelID
	}
	return c.bot.settings().Channels.Quest
}
func (c *autoQuestCtx) OwoBotID() string { return c.bot.settings().OwoID }
func (c *autoQuestCtx) AutoQuest() config.AutoQuest {
	return c.bot.settings().AutoQuest
}
func (c *autoQuestCtx) AllowAutoQuest() bool { return c.bot.settings().AllowAutoQuest }
func (c *autoQuestCtx) OCRApiKey() string {
	if k := strings.TrimSpace(c.bot.settings().OCRApi); k != "" {
		return k
	}
	return "helloworld"
}
func (c *autoQuestCtx) CanEnableQuestCmds() bool {
	return c.bot.settings().AutoQuest.EnableCommandsToCompleteQuest
}
func (c *autoQuestCtx) IsHuntEnabled() bool   { return c.bot.settings().Status.Hunt }
func (c *autoQuestCtx) IsBattleEnabled() bool { return c.bot.settings().Status.Battle }
func (c *autoQuestCtx) IsCookieEnabled() bool { return c.bot.settings().Status.Cookie }
func (c *autoQuestCtx) IsPrayEnabled() bool   { return c.bot.settings().Status.Pray }
func (c *autoQuestCtx) IsCurseEnabled() bool  { return c.bot.settings().Status.Curse }
func (c *autoQuestCtx) IsGambleEnabled() bool {
	g := c.bot.settings().Gamble
	return g.Coinflip.Enabled || g.Slots.Enabled || g.Blackjack.Enabled
}
func (c *autoQuestCtx) Nickname() string {
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
func (c *autoQuestCtx) GuildID() string {
	client := c.bot.discordClient()
	chID := c.bot.settings().Channels.Hunt
	if client == nil || client.State == nil || chID == "" {
		return ""
	}
	sf, err := discord.ParseSnowflake(chID)
	if err != nil {
		return ""
	}
	if ch, ok := client.State.GetChannel(sf); ok && ch != nil {
		return ch.GuildID.String()
	}
	for _, g := range client.State.Guilds {
		if g != nil {
			return g.ID.String()
		}
	}
	return ""
}
func (c *autoQuestCtx) RandomPrefix(cmds []string) string { return c.bot.randomPrefix(cmds) }
func (c *autoQuestCtx) SendMessage(channelID, text string) error {
	c.bot.enqueue(channelID, text)
	return nil
}
func (c *autoQuestCtx) CanSend() bool  { return c.bot.canSend() }
func (c *autoQuestCtx) Log(msg string) { c.bot.logInfo(msg) }
func (c *autoQuestCtx) SleepRange(min, max float64) {
	if max <= min {
		time.Sleep(time.Duration(min * float64(time.Second)))
		return
	}
	d := min + rand.Float64()*(max-min)
	time.Sleep(time.Duration(d * float64(time.Second)))
}
func (c *autoQuestCtx) ClickButton(channelID, messageID, customID, applicationID string) error {
	client := c.bot.discordClient()
	if client == nil {
		return nil
	}
	ch, err := discord.ParseSnowflake(channelID)
	if err != nil {
		return err
	}
	msg, err := discord.ParseSnowflake(messageID)
	if err != nil {
		return err
	}
	app, err := discord.ParseSnowflake(applicationID)
	if err != nil {
		return err
	}
	return client.ClickButton(ch, msg, customID, app)
}
func (c *autoQuestCtx) GambleEnqueue(game string, amount int) {
	ch := c.bot.settings().Channels.Hunt
	var text, qGame string
	switch game {
	case "slots":
		qGame = gamble.QueueSlots
		text = c.bot.randomPrefix([]string{"s", "slots"}) + " " + strconv.Itoa(amount)
	default:
		qGame = gamble.QueueCoinflip
		text = c.bot.randomPrefix([]string{"cf", "coinflip"}) + " " + strconv.Itoa(amount)
	}
	c.bot.enqueueGambleBet(ch, text, qGame)
}

func (b *Bot) startAutoQuestIfNeeded() {
	s := b.settings()
	if !s.AutoQuest.Enabled {
		b.stopAutoQuest()
		return
	}
	if !s.AllowAutoQuest {
		b.logInfo("autoQuest disabled — set allowAutoQuest: true in config to enable (experimental)")
		b.stopAutoQuest()
		return
	}
	uid := b.userID()
	if uid == "" {
		return
	}
	if b.autoQuest == nil {
		local := quest.NewLocalHandler(uid, b.newAutoQuestContext().OCRApiKey())
		b.autoQuest = quest.NewManager(b.newAutoQuestContext(), local)
	}
	b.autoQuest.Start()
}

func (b *Bot) stopAutoQuest() {
	if b.autoQuest != nil {
		b.autoQuest.Stop()
	}
}

func (b *Bot) restartAutoQuest() {
	b.stopAutoQuest()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if !ready {
		return
	}
	b.startAutoQuestIfNeeded()
}

func (b *Bot) onRawGateway(event string, data json.RawMessage) {
	if event == "MESSAGE_UPDATE" && b.gamble != nil && !b.captchaSolving {
		b.gamble.HandleRawMessageUpdate(data)
	}
	if b.autoQuest == nil {
		return
	}
	if event != "MESSAGE_CREATE" && event != "MESSAGE_UPDATE" {
		return
	}
	b.autoQuest.HandleRawMessage(event, data)
}

func (b *Bot) handleAutoQuestMessage(msg *discord.Message, nick string) {
	if b.autoQuest == nil || msg == nil {
		return
	}
	b.autoQuest.HandleHuntMessage(msg.Content, nick)
	for _, e := range msg.Embeds {
		if e == nil {
			continue
		}
		author := ""
		if e.Author != nil {
			author = e.Author.Name
		}
		footer := ""
		if e.Footer != nil {
			footer = e.Footer.Text
		}
		if author != "" || footer != "" {
			b.autoQuest.HandleBattleEmbed(author, footer)
		}
	}
}
