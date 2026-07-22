package farm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/util"
)

var (
	discordCustomEmojiRe = regexp.MustCompile(`<a?:\w+:\d+>`)
	discordMentionRe     = regexp.MustCompile(`<@!?\d+>`)
	markdownBoldRe       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	markdownCodeRe       = regexp.MustCompile("`([^`]+)`")
	unicodeEmojiNameRe   = regexp.MustCompile(`:([a-z0-9_]+):`)
	huntFoundRe          = regexp.MustCompile(`(?i)you found:\s*(.+)`)
	battleResultRe       = regexp.MustCompile(`(?i)you (won|lost) in (\d+) turns!`)
	battleXpRe           = regexp.MustCompile(`([+-]\d+)\s*xp`)
	battleStreakRe       = regexp.MustCompile(`(?i)streak:\s*(\d+)`)
	battleStreakEndedRe  = regexp.MustCompile(`(?i)streak ended at (\d+)`)
	prayLuckRe           = regexp.MustCompile(`(?i)(?:you have\s+)?(\d+)\s+luck point`)
	prayFlavorRe         = regexp.MustCompile(`(?i)prays\.\.\.\s*(.+?)(?:\s*\||$)`)
	rankRe               = regexp.MustCompile(`Rank (#[\d,]+)`)
	inventoryItemRe      = regexp.MustCompile("`(\\d+)`<a?:(\\w+):\\d+>([⁰¹²³⁴⁵⁶⁷⁸⁹]+)")
)

// shouldSkipOwOLog returns true for gamble/cash messages handled by other loggers.
func shouldSkipOwOLog(content string) bool {
	if content == "" {
		return false
	}
	lower := strings.ToLower(content)
	if strings.Contains(content, "___SLOTS___") {
		return true
	}
	if strings.Contains(lower, "slots") && strings.Contains(lower, "bet") {
		return true
	}
	if strings.Contains(content, "coin spins") || strings.Contains(content, "and chose **") {
		return true
	}
	if strings.Contains(content, "cowoncy!") || strings.Contains(content, "you currently have") {
		return true
	}
	if strings.Contains(lower, "here is your daily") && strings.Contains(lower, "cowoncy") {
		return true
	}
	if strings.Contains(content, "⏱") && strings.Contains(content, "Nu!") && strings.Contains(content, "You need to wait") {
		return true
	}
	return false
}

// stripDiscordText removes markdown and custom Discord emojis for readable one-line logs.
func stripDiscordText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = discordCustomEmojiRe.ReplaceAllString(s, "")
	s = discordMentionRe.ReplaceAllString(s, "@user")
	s = markdownBoldRe.ReplaceAllString(s, "$1")
	s = markdownCodeRe.ReplaceAllString(s, "$1")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func formatAnimalList(raw string) string {
	raw = unicodeEmojiNameRe.ReplaceAllStringFunc(raw, func(m string) string {
		name := strings.Trim(m, ":")
		return name + " "
	})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Fields(raw)
	return strings.Join(parts, ", ")
}

// summarizeOwOMessage returns a short one-line summary for OwO bot replies.
// Returns "" when the message is handled elsewhere or not worth logging.
func summarizeOwOMessage(content, nick string) string {
	if content == "" {
		return ""
	}

	// Skip messages already summarized by gamble / daily handlers.
	if shouldSkipOwOLog(content) {
		return ""
	}

	if strings.Contains(content, "'s Inventory") {
		return summarizeInventory(content, nick)
	}

	if m := rankRe.FindStringSubmatch(content); len(m) > 1 {
		return "Rank " + m[1]
	}

	if battleResultRe.MatchString(content) {
		return summarizeBattle(content)
	}

	if strings.Contains(content, "You found:") || strings.Contains(content, ", hunt") {
		summary := "Hunt"
		if m := huntFoundRe.FindStringSubmatch(content); len(m) > 1 {
			if animals := formatAnimalList(m[1]); animals != "" {
				summary += " → " + animals
			}
		}
		if m := xpRe.FindStringSubmatch(content); len(m) > 1 {
			summary += " (+" + m[1] + " pet xp)"
		}
		return summary
	}

	if strings.Contains(content, "Verification") || strings.Contains(content, "verify") {
		return "Verification required"
	}

	return ""
}

type inventoryItem struct {
	id    string
	name  string
	count int
}

func parseInventoryItems(content string) []inventoryItem {
	var items []inventoryItem
	for _, m := range inventoryItemRe.FindAllStringSubmatch(content, -1) {
		items = append(items, inventoryItem{
			id:    m[1],
			name:  m[2],
			count: util.SuperscriptToNumber(m[3]),
		})
	}
	return items
}

func summarizeInventory(content, nick string) string {
	if nick != "" && !strings.Contains(content, nick) {
		return ""
	}
	items := parseInventoryItems(content)
	if len(items) == 0 {
		return "Inventory → empty"
	}
	const maxShown = 10
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.name+"×"+strconv.Itoa(it.count))
	}
	list := strings.Join(parts, ", ")
	if len(parts) > maxShown {
		list = strings.Join(parts[:maxShown], ", ") + fmt.Sprintf(" +%d more", len(parts)-maxShown)
	}
	label := "items"
	if len(items) == 1 {
		label = "item"
	}
	return fmt.Sprintf("Inventory → %d %s · %s", len(items), label, list)
}

func truncateLogText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) > 200 {
		return s[:197] + "..."
	}
	return s
}

// owoMessageText returns a readable one-line fallback from raw OwO text.
func owoMessageText(content string) string {
	if shouldSkipOwOLog(content) {
		return ""
	}
	return truncateLogText(stripDiscordText(content))
}

// owoMessageTextFromMsg returns content or embed footer/description when content is empty.
func owoMessageTextFromMsg(msg *discord.Message) string {
	if msg == nil {
		return ""
	}
	if shouldSkipOwOLog(msg.Content) {
		return ""
	}
	if t := owoMessageText(msg.Content); t != "" {
		return t
	}
	var parts []string
	for _, embed := range msg.Embeds {
		if embed == nil {
			continue
		}
		if embed.Footer != nil {
			if t := owoMessageText(embed.Footer.Text); t != "" {
				parts = append(parts, t)
			}
		}
		if embed.Description != "" {
			if t := owoMessageText(embed.Description); t != "" {
				parts = append(parts, t)
			}
		}
	}
	return truncateLogText(strings.Join(parts, " | "))
}

func battleStreakInfo(text string) string {
	if m := battleStreakRe.FindStringSubmatch(text); len(m) > 1 {
		return ", streak " + m[1]
	}
	if m := battleStreakEndedRe.FindStringSubmatch(text); len(m) > 1 {
		return ", streak ended at " + m[1]
	}
	return ""
}

func summarizeBattle(text string) string {
	text = strings.TrimSpace(text)
	if !battleResultRe.MatchString(text) {
		return ""
	}
	m := battleResultRe.FindStringSubmatch(text)
	outcome := strings.ToLower(m[1])
	turns := m[2]
	summary := "Battle → " + outcome + " in " + turns + " turns"
	streak := battleStreakInfo(text)
	if xp := battleXpRe.FindStringSubmatch(text); len(xp) > 1 {
		summary += " (" + xp[1] + " xp" + streak + ")"
	} else if streak != "" {
		summary += " (" + strings.TrimPrefix(streak, ", ") + ")"
	}
	return summary
}

func isPrayMessage(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "prays") || strings.Contains(lower, "luck point")
}

func summarizePray(text string) string {
	if !isPrayMessage(text) {
		return ""
	}
	plain := stripDiscordText(text)
	summary := "Pray"
	if m := prayLuckRe.FindStringSubmatch(plain); len(m) > 1 {
		summary += " → " + m[1] + " luck"
	}
	if m := prayFlavorRe.FindStringSubmatch(plain); len(m) > 1 {
		if flavor := strings.TrimSpace(m[1]); flavor != "" {
			if strings.HasPrefix(summary, "Pray →") {
				summary += " — " + flavor
			} else {
				summary += " → " + flavor
			}
		}
	}
	if summary == "Pray" {
		return ""
	}
	return summary
}

func prayTextFromMessage(msg *discord.Message, content string) string {
	if isPrayMessage(content) {
		return content
	}
	if msg == nil {
		return ""
	}
	for _, embed := range msg.Embeds {
		if embed == nil {
			continue
		}
		if isPrayMessage(embed.Description) {
			return embed.Description
		}
	}
	return ""
}

func battleFooterFromMessage(msg *discord.Message, nick string) string {
	if msg == nil {
		return ""
	}
	for _, embed := range msg.Embeds {
		if embed == nil || embed.Footer == nil {
			continue
		}
		footer := strings.TrimSpace(embed.Footer.Text)
		if !battleResultRe.MatchString(footer) {
			continue
		}
		if embed.Author != nil && embed.Author.Name != "" && nick != "" {
			if !strings.Contains(embed.Author.Name, nick) {
				continue
			}
		}
		return footer
	}
	return ""
}

func summarizeOwOFromMessage(msg *discord.Message, content, nick string) string {
	if pray := prayTextFromMessage(msg, content); pray != "" {
		if summary := summarizePray(pray); summary != "" {
			return summary
		}
	}
	if summary := summarizeOwOMessage(content, nick); summary != "" {
		return summary
	}
	if footer := battleFooterFromMessage(msg, nick); footer != "" {
		return summarizeBattle(footer)
	}
	if shouldSkipOwOLog(content) {
		return ""
	}
	if t := owoMessageTextFromMsg(msg); t != "" {
		if nick == "" || strings.Contains(strings.ToLower(t), strings.ToLower(nick)) {
			return t
		}
	}
	return ""
}

func (b *Bot) logOwOResponse(msg *discord.Message, content, nick string) {
	summary := summarizeOwOFromMessage(msg, content, nick)
	if summary == "" {
		return
	}
	if strings.HasPrefix(summary, "Battle →") && msg != nil && msg.ID != 0 {
		key := msg.ID.String() + "|" + summary
		b.mu.Lock()
		if b.lastBattleLog == key {
			b.mu.Unlock()
			return
		}
		b.lastBattleLog = key
		b.mu.Unlock()
	}
	b.logInfo(summary)
}
