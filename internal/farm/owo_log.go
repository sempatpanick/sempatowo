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
	// :bug:! → bug — strip the shortcode colons and the trailing "!" OwO adds
	// after the caught animal, so the log reads "caught a common bug".
	huntEmojiRe = regexp.MustCompile(`:([a-z0-9_]+):!?`)
	// HuntBot's completion push: "I AM BACK WITH 598 ANIMALS, 4725 ESSENCE, AND
	// 5512 EXPERIENCE". [\d,]+ because OwO comma-groups large essence/xp totals;
	// [\s|]* spans the line break plus the "|" left by each row's stripped
	// separator, since OwO wraps the header across two lines.
	huntbotReturnRe = regexp.MustCompile(`I AM BACK WITH ([\d,]+) ANIMALS,[\s|]*([\d,]+) ESSENCE, AND ([\d,]+) EXPERIENCE`)
	// :bee:⁶⁵ → animal shortcode + its superscript count, which sits flush
	// against the closing colon with no space.
	huntbotCatchRe = regexp.MustCompile(`:([a-z0-9_]+):([⁰¹²³⁴⁵⁶⁷⁸⁹]+)`)
	battleResultRe = regexp.MustCompile(`(?i)you (won|lost) in (\d+) turns!`)
	// [\d,]+ because OwO comma-groups anything from 1,000 up; \d+ silently
	// failed to match the whole pattern and dropped the xp from the summary.
	battleXpRe          = regexp.MustCompile(`([+-][\d,]+)\s*xp`)
	battleStreakRe      = regexp.MustCompile(`(?i)streak:\s*(\d+)`)
	battleStreakEndedRe = regexp.MustCompile(`(?i)streak ended at (\d+)`)
	prayLuckRe          = regexp.MustCompile(`(?i)(?:you have\s+)?(\d+)\s+luck point`)
	prayFlavorRe        = regexp.MustCompile(`(?i)prays\.\.\.\s*(.+?)(?:\s*\||$)`)
	rankRe              = regexp.MustCompile(`Rank (#[\d,]+)`)
	inventoryItemRe     = regexp.MustCompile("`(\\d+)`<a?:(\\w+):\\d+>([⁰¹²³⁴⁵⁶⁷⁸⁹]+)")
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

// isHuntCatchMessage matches OwO's modern essence-catch reply, e.g.
// "…spent 5 and caught a common :bug:! | gained 1xp!". It deliberately does not
// fire on the older "You found:" reply, which keeps its own animal-list summary.
func isHuntCatchMessage(content string) bool {
	if strings.Contains(content, "You found:") {
		return false
	}
	return strings.Contains(strings.ToLower(content), "caught")
}

// summarizeHuntCatch turns a modern catch reply into
// "Hunt → spent 5 and caught a common bug | gained 1xp!". stripDiscordText
// removes the "🌱 | name" prefix decorations, the <:custom:id> emojis and the
// bold/backtick markup; we keep only the clause from "spent"/"caught" onward so
// the log drops the leading name and any empowerment banner, then flatten the
// ":bug:!" shortcode down to "bug".
func summarizeHuntCatch(content string) string {
	clean := stripDiscordText(content)
	body := extractHuntBody(clean)
	if body == "" {
		return ""
	}
	// ":bug:!" → "bug": drop the colons and the "!" OwO puts right after the
	// caught animal, while leaving the "!" on "gained 1xp!" alone.
	body = huntEmojiRe.ReplaceAllString(body, "$1")
	body = strings.Join(strings.Fields(body), " ")
	return "Hunt → " + body
}

// extractHuntBody returns the cleaned message from the first "spent " or
// "caught " keyword onward, so the prefix ("🌱 | name") is dropped without
// having to know the account nick.
func extractHuntBody(clean string) string {
	lower := strings.ToLower(clean)
	idx := -1
	for _, kw := range []string{"spent ", "caught "} {
		if i := strings.Index(lower, kw); i >= 0 && (idx == -1 || i < idx) {
			idx = i
		}
	}
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(clean[idx:])
}

// summarizeHuntbotReturn turns OwO's HuntBot completion push into
// "Huntbot → received 598 ANIMALS, 4725 ESSENCE, AND 5512 EXPERIENCE:
// bee (x65), snail (x75), …". stripDiscordText drops the <:common:id> rarity
// markers and the <:rbot:id>/<:blank:id> decorations and the backticks while
// leaving OwO's :bee: shortcodes intact; SuperscriptToNumber decodes the ⁶⁵
// counts. Returns "" when neither the header nor any catch is recognised.
func summarizeHuntbotReturn(content string) string {
	clean := stripDiscordText(content)
	summary := "Huntbot → received"
	if m := huntbotReturnRe.FindStringSubmatch(clean); len(m) > 3 {
		summary += " " + m[1] + " ANIMALS, " + m[2] + " ESSENCE, AND " + m[3] + " EXPERIENCE"
	}
	var parts []string
	for _, c := range huntbotCatchRe.FindAllStringSubmatch(clean, -1) {
		parts = append(parts, c[1]+" (x"+strconv.Itoa(util.SuperscriptToNumber(c[2]))+")")
	}
	if len(parts) > 0 {
		summary += ": " + strings.Join(parts, ", ")
	}
	if summary == "Huntbot → received" {
		return ""
	}
	return summary
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

	// HuntBot's "I AM BACK WITH …" completion push carries no nick, no "caught"
	// and no "You found:", so it fell through to the generic fallback (dropped
	// for lack of a nick). Summarize it before those paths — it can't collide.
	if strings.Contains(content, "I AM BACK WITH") {
		if s := summarizeHuntbotReturn(content); s != "" {
			return s
		}
	}

	// OwO's essence catches read "…spent 5 and caught a common :bug:! | gained
	// 1xp!" and carry no "You found:" line, so they fell through to the generic
	// fallback that only strips <:custom:id> emojis — leaving the bare ":bug:!"
	// in the log. Handle them before the "You found:" path and keep OwO's own
	// :shortcode: names intact.
	if isHuntCatchMessage(content) {
		if s := summarizeHuntCatch(content); s != "" {
			return s
		}
	}

	if strings.Contains(content, "You found:") || strings.Contains(content, ", hunt") {
		summary := "Hunt"
		if m := huntFoundRe.FindStringSubmatch(content); len(m) > 1 {
			if animals := formatAnimalList(m[1]); animals != "" {
				summary += " → " + animals
			}
		}
		if m := xpRe.FindStringSubmatch(content); len(m) > 1 {
			summary += " (" + formatSignedXP(m[1]) + " pet xp)"
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
		summary += " (" + formatSignedXP(xp[1]) + " xp" + streak + ")"
	} else if streak != "" {
		summary += " (" + strings.TrimPrefix(streak, ", ") + ")"
	}
	return summary
}

// formatSignedXP normalizes a captured xp amount — "+2,800", "+2800", "-15" —
// to one comma-grouped form with an explicit sign, so the log reads the same
// whether or not OwO grouped the number.
func formatSignedXP(raw string) string {
	n, ok := util.ParseAmount(raw)
	if !ok {
		return raw
	}
	if n >= 0 {
		return "+" + util.FormatInt(n)
	}
	return util.FormatInt(n)
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
		if b.stats.SeenBattleLog(msg.ID.String() + "|" + summary) {
			return
		}
	}
	b.logInfo(summary)
}
