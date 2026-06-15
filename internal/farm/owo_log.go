package farm

import (
	"regexp"
	"strings"
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
	prayLuckRe           = regexp.MustCompile(`(\d+)\s+luck point`)
	rankRe               = regexp.MustCompile(`Rank (#[\d,]+)`)
)

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

	// Skip messages already summarized by gamble / inventory / daily handlers.
	if strings.Contains(content, "'s Inventory") ||
		strings.Contains(content, "___SLOTS___") ||
		strings.Contains(content, "cowoncy!") ||
		strings.Contains(content, "coin spins") ||
		strings.Contains(content, "and chose **") {
		return ""
	}

	if m := rankRe.FindStringSubmatch(content); len(m) > 1 {
		return "Rank " + m[1]
	}

	if strings.Contains(content, "prays") || strings.Contains(content, "luck point") {
		if m := prayLuckRe.FindStringSubmatch(stripDiscordText(content)); len(m) > 1 {
			return "Pray → " + m[1] + " luck"
		}
		return "Pray"
	}

	if battleResultRe.MatchString(content) {
		m := battleResultRe.FindStringSubmatch(content)
		outcome := strings.ToLower(m[1])
		turns := m[2]
		summary := "Battle → " + outcome + " in " + turns + " turns"
		if xp := battleXpRe.FindStringSubmatch(content); len(xp) > 1 {
			summary += " (" + xp[1] + " xp"
			if streak := battleStreakRe.FindStringSubmatch(content); len(streak) > 1 {
				summary += ", streak " + streak[1]
			}
			summary += ")"
		}
		return summary
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

	plain := stripDiscordText(content)
	if len(plain) > 120 {
		return plain[:117] + "..."
	}
	if plain != "" && !strings.Contains(plain, nick) {
		return plain
	}

	return ""
}

func (b *Bot) logOwOResponse(content, nick string) {
	if summary := summarizeOwOMessage(content, nick); summary != "" {
		b.logInfo(summary)
	}
}
