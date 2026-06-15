package farm

import (
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/gamble"
)

func (b *Bot) toGambleMessage(msg *discord.Message) gamble.Message {
	if msg == nil {
		return gamble.Message{}
	}
	out := gamble.Message{
		Content: msg.Content,
	}
	if msg.ID != 0 {
		out.ID = msg.ID.String()
	}
	if msg.ChannelID != 0 {
		out.ChannelID = msg.ChannelID.String()
	}
	if msg.Author != nil {
		out.AuthorID = msg.Author.ID.String()
	}
	for _, e := range msg.Embeds {
		if e == nil {
			continue
		}
		embed := gamble.MessageEmbed{}
		if e.Author != nil {
			embed.Author = &gamble.EmbedAuthor{Name: e.Author.Name}
		}
		if e.Footer != nil {
			embed.Footer = &gamble.EmbedFooter{Text: e.Footer.Text}
		}
		for _, f := range e.Fields {
			if f == nil {
				continue
			}
			embed.Fields = append(embed.Fields, gamble.EmbedField{Name: f.Name, Value: f.Value})
		}
		out.Embeds = append(out.Embeds, embed)
	}
	for _, r := range msg.Reactions {
		if r == nil || r.Emoji == nil {
			continue
		}
		emoji := r.Emoji.Name
		out.Reactions = append(out.Reactions, gamble.Reaction{Emoji: emoji, Me: r.Me})
	}
	return out
}

type gambleCtx struct {
	bot *Bot
}

func (b *Bot) newGambleContext() *gambleCtx {
	return &gambleCtx{bot: b}
}

func (c *gambleCtx) HuntChannelID() string { return c.bot.settings().Channels.Hunt }
func (c *gambleCtx) OwoBotID() string      { return c.bot.settings().OwoID }
func (c *gambleCtx) Nickname() string {
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
func (c *gambleCtx) Username() string { return c.bot.username() }
func (c *gambleCtx) UserID() string   { return c.bot.userID() }
func (c *gambleCtx) CashCheck() bool        { return c.bot.settings().CashCheck }
func (c *gambleCtx) Gamble() config.Gamble  { return c.bot.settings().Gamble }
func (c *gambleCtx) RandomPrefix(cmds []string) string { return c.bot.randomPrefix(cmds) }
func (c *gambleCtx) SendMessage(channelID, text string) error {
	c.bot.enqueue(channelID, text)
	return nil
}
func (c *gambleCtx) SendGambleBet(channelID, game, text string) error {
	c.bot.enqueueGambleBet(channelID, text, game)
	return nil
}
func (c *gambleCtx) SignalGambleResult(game string) {
	c.bot.signalGambleResult(game)
}
func (c *gambleCtx) CanSend() bool { return c.bot.canSend() }
func (c *gambleCtx) Log(msg string) { c.bot.logInfo(msg) }
func (c *gambleCtx) Debug(msg string) { c.bot.logDebug(msg) }
func (c *gambleCtx) Sleep(seconds float64) {
	if seconds > 0 {
		time.Sleep(time.Duration(seconds * float64(time.Second)))
	}
}
func (c *gambleCtx) AddReaction(channelID, messageID, emoji string) error {
	client := c.bot.discordClient()
	if client == nil {
		return nil
	}
	chID, err := discord.ParseSnowflake(channelID)
	if err != nil {
		return err
	}
	msgID, err := discord.ParseSnowflake(messageID)
	if err != nil {
		return err
	}
	return client.AddReaction(chID, msgID, emoji)
}
func (c *gambleCtx) RemoveReaction(channelID, messageID, emoji string) error {
	client := c.bot.discordClient()
	if client == nil {
		return nil
	}
	chID, err := discord.ParseSnowflake(channelID)
	if err != nil {
		return err
	}
	msgID, err := discord.ParseSnowflake(messageID)
	if err != nil {
		return err
	}
	return client.RemoveReaction(chID, msgID, emoji)
}
