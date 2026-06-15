package gamble

import (
	"encoding/json"
	"strings"
)

// HandleRawMessageUpdate processes MESSAGE_UPDATE gateway payloads. Discord often
// omits author on edits; the typed OnMessageUpdate handler may drop those events.
func (m *Manager) HandleRawMessageUpdate(data json.RawMessage) {
	if m == nil {
		return
	}
	var raw struct {
		ID        string `json:"id"`
		ChannelID string `json:"channel_id"`
		Author    *struct {
			ID string `json:"id"`
		} `json:"author"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	if raw.ChannelID != m.bot.HuntChannelID() {
		return
	}
	if raw.Author != nil && raw.Author.ID != "" && raw.Author.ID != m.bot.OwoBotID() {
		return
	}
	content := normalizeZW(raw.Content)
	if content == "" {
		return
	}
	authorID := m.bot.OwoBotID()
	if raw.Author != nil && raw.Author.ID != "" {
		authorID = raw.Author.ID
	}
	msg := Message{
		ID:        raw.ID,
		ChannelID: raw.ChannelID,
		AuthorID:  authorID,
		Content:   content,
	}
	m.HandleGambleResult(msg)
	if m.bot.Gamble().Blackjack.Enabled {
		m.blackjack.onUpdate(msg)
	}
}

func normalizeZW(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x200B && r <= 0x200D || r == 0xFEFF {
			return -1
		}
		return r
	}, s)
}

func contentForUser(content, nick, username, uid string) bool {
	if uid != "" && strings.Contains(content, uid) {
		return true
	}
	for _, name := range []string{nick, username} {
		if name != "" && strings.Contains(content, name) {
			return true
		}
	}
	return false
}
