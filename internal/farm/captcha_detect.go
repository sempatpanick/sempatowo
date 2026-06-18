package farm

import (
	"encoding/json"
	"regexp"
	"strings"

	discord "github.com/hytams/discordgo-self"
)

var (
	captchaKeywordRe   = regexp.MustCompile(`(?i)(human|captcha|https://owobot\.com/captcha|letterword|complete your captcha|complete this within)`)
	realHumanRe        = regexp.MustCompile(`(?i)are you a real human`)
	captchaBanWarnRe   = regexp.MustCompile(`(?i)within \d+ minutes.*\bban`)
	captchaVerifyCount = regexp.MustCompile(`(?i)verify that you are human!\s*\(\d+/\d+\)`)
)

// detectCaptcha reports OwO captcha / ban verification messages for this account.
func detectCaptcha(msg *discord.Message, content, nick, uid, username string) bool {
	if msg == nil || uid == "" {
		return false
	}

	full := normalizeZW(owoMessageFullText(msg, content))
	lower := strings.ToLower(full)

	if isCaptchaVerificationPrompt(lower) {
		return isCaptchaTarget(msg, full, nick, uid, username)
	}

	if strings.Contains(lower, "verified that you are human") {
		return false
	}

	isTarget := isCaptchaTarget(msg, full, nick, uid, username)

	if strings.Contains(lower, "ban") && isTarget && strings.Contains(lower, "minute") {
		return true
	}

	hasSignal := captchaKeywordRe.MatchString(full) ||
		realHumanRe.MatchString(full) ||
		captchaBanWarnRe.MatchString(full) ||
		captchaVerifyCount.MatchString(full) ||
		hasVerifyButton(msg) ||
		(strings.Contains(full, "⚠️") && (len(msg.Attachments) > 0 || strings.Contains(lower, "human") || strings.Contains(lower, "captcha")))

	if !hasSignal {
		return false
	}

	return isTarget
}

func isCaptchaVerificationPrompt(lower string) bool {
	if strings.Contains(lower, "please complete your captcha to verify that you are human") {
		return true
	}
	if strings.Contains(lower, "please complete this within") && strings.Contains(lower, "ban") {
		return true
	}
	if realHumanRe.MatchString(lower) && strings.Contains(lower, "please") {
		return true
	}
	return captchaVerifyCount.MatchString(lower)
}

func isCaptchaTarget(msg *discord.Message, fullText, nick, uid, username string) bool {
	if msg.GuildID == 0 {
		return true
	}
	if strings.Contains(msg.Content, uid) {
		return true
	}
	for _, u := range msg.Mentions {
		if u != nil && u.ID.String() == uid {
			return true
		}
	}
	for _, name := range []string{nick, username} {
		if name != "" && strings.Contains(fullText, name) {
			return true
		}
	}
	return false
}

func owoMessageFullText(msg *discord.Message, content string) string {
	if msg == nil {
		return content
	}
	var b strings.Builder
	if content != "" {
		b.WriteString(content)
	} else if msg.Content != "" {
		b.WriteString(msg.Content)
	}
	for _, embed := range msg.Embeds {
		if embed == nil {
			continue
		}
		if embed.Author != nil && embed.Author.Name != "" {
			b.WriteString(" ")
			b.WriteString(embed.Author.Name)
		}
		if embed.Title != "" {
			b.WriteString(" ")
			b.WriteString(embed.Title)
		}
		if embed.Description != "" {
			b.WriteString(" ")
			b.WriteString(embed.Description)
		}
		if embed.Footer != nil && embed.Footer.Text != "" {
			b.WriteString(" ")
			b.WriteString(embed.Footer.Text)
		}
		for _, f := range embed.Fields {
			if f == nil {
				continue
			}
			b.WriteString(" ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(f.Value)
		}
	}
	return b.String()
}

func hasVerifyButton(msg *discord.Message) bool {
	if msg == nil || len(msg.Components) == 0 {
		return false
	}
	raw, err := json.Marshal(msg.Components)
	if err != nil {
		return false
	}
	s := strings.ToLower(string(raw))
	return strings.Contains(s, `"label":"verify"`) || strings.Contains(s, `"label": "verify"`)
}

func (b *Bot) tryHandleCaptcha(msg *discord.Message, content string) bool {
	uid := b.userID()
	if uid == "" {
		return false
	}
	nick := b.nickname(msg)
	if !detectCaptcha(msg, content, nick, uid, b.username()) {
		return false
	}
	if text := owoMessageTextFromMsg(msg); text != "" {
		b.logDanger("OwO captcha: " + text)
	} else if full := strings.TrimSpace(owoMessageFullText(msg, content)); full != "" {
		b.logDanger("OwO captcha: " + truncateLogText(stripDiscordText(full)))
	}
	b.logDanger("Captcha detected — stopping farm (~10 min to solve)")
	b.handleCaptcha()
	return true
}
