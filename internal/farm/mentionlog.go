package farm

import (
	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/mentionlog"
)

// logMention persists a raw OwO message addressed to this account, when the
// mentionLog feature is enabled. Content is msg.Content verbatim (not
// zero-width-normalized) so the file mirrors exactly what OwO sent. A write
// failure is logged and swallowed — logging must never disrupt the farm.
func (b *Bot) logMention(msg *discord.Message, event string) {
	if b == nil || b.mentionLog == nil || msg == nil {
		return
	}
	if !b.settings().Features.MentionLog.Enabled {
		return
	}

	rec := mentionlog.Record{
		Event:     event,
		ChannelID: msg.ChannelID.String(),
		MessageID: msg.ID.String(),
		Content:   msg.Content,
	}
	if msg.GuildID != 0 {
		rec.GuildID = msg.GuildID.String()
	}
	for _, e := range msg.Embeds {
		if e == nil {
			continue
		}
		em := mentionlog.Embed{Title: e.Title, Description: e.Description}
		if e.Author != nil {
			em.Author = e.Author.Name
		}
		if e.Footer != nil {
			em.Footer = e.Footer.Text
		}
		rec.Embeds = append(rec.Embeds, em)
	}

	if err := b.mentionLog.Append(rec); err != nil {
		b.logDebug("mention log write failed: " + err.Error())
	}
}
