package farm

import (
	"strings"

	discord "github.com/hytams/discordgo-self"
)

// --- nil-safe accessors ---

func (b *Bot) discordClient() *discord.Client {
	if b == nil {
		return nil
	}
	return b.client
}

func (b *Bot) discordUser() *discord.User {
	client := b.discordClient()
	if client == nil || client.User == nil {
		return nil
	}
	return &client.User.User
}

func (b *Bot) userID() string {
	u := b.discordUser()
	if u == nil {
		return ""
	}
	return u.ID.String()
}

func (b *Bot) username() string {
	u := b.discordUser()
	if u == nil {
		return ""
	}
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

func (b *Bot) logInfo(msg string) {
	if b != nil && b.log != nil {
		b.log.Info(msg)
	}
}

func (b *Bot) logDanger(msg string) {
	if b != nil && b.log != nil {
		b.log.Danger(msg)
	}
}

func (b *Bot) logDebug(msg string) {
	if b != nil && b.log != nil {
		b.log.Debug(msg)
	}
}

func embedContainsSafe(embed *discord.Embed, name string) bool {
	if embed == nil || name == "" {
		return false
	}
	if embed.Author != nil && strings.Contains(embed.Author.Name, name) {
		return true
	}
	if strings.Contains(embed.Description, name) {
		return true
	}
	if strings.Contains(embed.Title, name) {
		return true
	}
	if embed.Footer != nil && strings.Contains(embed.Footer.Text, name) {
		return true
	}
	return false
}

func firstEmbed(msg *discord.Message) *discord.Embed {
	if msg == nil || len(msg.Embeds) == 0 {
		return nil
	}
	return msg.Embeds[0]
}

// owoMessageText returns a short summary for logging (content, or embed footer text).
func owoMessageText(msg *discord.Message) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Content); text != "" {
		return text
	}
	var parts []string
	for _, embed := range msg.Embeds {
		if embed == nil || embed.Footer == nil {
			continue
		}
		if t := strings.TrimSpace(embed.Footer.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " | ")
}
