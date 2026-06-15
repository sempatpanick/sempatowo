package huntbot

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sempatowo/sempatowo/internal/config"
)

var (
	passwordResetRe   = regexp.MustCompile(`Password will reset in (\d+)`)
	huntbotTimeRe     = regexp.MustCompile(`(\d+)([DHM])`)
	levelProgressRe   = regexp.MustCompile(`Lvl (\d+) \[(\d+)/\d+\]`)
	essenceRe         = regexp.MustCompile(`Animal Essence - ` + "`" + `(\d{1,3}(?:,\d{3})*)` + "`")
)

const briefCooldownMin = 5.0
const briefCooldownMax = 8.0

// Handler manages HuntBot autohunt commands and upgrades.
type Handler struct {
	bot            BotContext
	token          string
	upgradeDetails UpgradeDetails
	upgradeWaiters []chan struct{}
	mu             sync.Mutex
}

func NewHandler(bot BotContext, token string) *Handler {
	h := &Handler{bot: bot, token: token}
	h.upgradeDetails = newUpgradeDetails(bot.Settings())
	return h
}

func newUpgradeDetails(settings config.Huntbot) UpgradeDetails {
	t := settings.Upgrader.Traits
	return UpgradeDetails{
		Efficiency: TraitState{Enabled: t.Efficiency},
		Duration:   TraitState{Enabled: t.Duration},
		Cost:       TraitState{Enabled: t.Cost},
		Gain:       TraitState{Enabled: t.Gain},
		Exp:        TraitState{Enabled: t.Exp},
		Radar:      TraitState{Enabled: t.Radar},
	}
}

func (h *Handler) Start() {
	if h == nil || h.bot == nil || !h.bot.Settings().Enabled {
		return
	}
	h.bot.SleepUntil(briefCooldownMin, briefCooldownMax-briefCooldownMin)
	h.sendAutohunt("", false)
}

func (h *Handler) Stop() {
	if h == nil || h.bot == nil {
		return
	}
	h.bot.CancelSleep()
}

func (h *Handler) HandleMessage(msg Message) {
	if h == nil || h.bot == nil {
		return
	}
	if msg.ChannelID != h.bot.HuntChannelID() || msg.AuthorID != h.bot.OwoBotID() {
		return
	}

	nick := h.bot.Nickname()
	if nick == "" {
		return
	}
	content := msg.Content

	if strings.Contains(content, nick) {
		switch {
		case strings.Contains(content, "You successfully upgraded"):
			h.notifyUpgradeDone()
		case strings.Contains(content, "Here is your password!"):
			h.handlePassword(msg)
		case strings.Contains(content, "Please include your password!"):
			h.handlePasswordRetry(content)
		case strings.Contains(content, "I WILL BE BACK IN"):
			secs := parseDuration(content)
			h.bot.Log("huntbot will be back in " + strconv.Itoa(secs) + "s")
			h.resendAfter(float64(secs))
		}
	}

	for _, embed := range msg.Embeds {
		if embed.Author == nil || !strings.Contains(embed.Author.Name, nick+"'s HuntBot") {
			continue
		}
		h.bot.CancelSleep()

		if len(embed.Fields) == 0 {
			continue
		}

		h.parseEmbed(embed)
		settings := h.bot.Settings()

		if settings.Upgrader.Enabled {
			h.bot.Sleep(upgraderCooldown(settings.Upgrader))
			allocation := AllocateEssence(h.upgradeDetails, settings.Upgrader.Weights)
			for trait, amount := range allocation {
				if amount <= 0 {
					continue
				}
				text := h.bot.RandomPrefix([]string{"upgrade"}) + " " + trait + " " + strconv.Itoa(amount)
				if !h.bot.CanSend() {
					return
				}
				_ = h.bot.SendMessage(h.bot.HuntChannelID(), text)
				h.waitUpgrade(240 * time.Second)
				h.bot.SleepUntil(briefCooldownMin, briefCooldownMax-briefCooldownMin)
			}
		}

		if len(embed.Fields) > 8 {
			field := embed.Fields[8]
			if strings.Contains(field.Name, "HUNTBOT is currently hunting!") {
				secs := parseDuration(field.Value)
				h.bot.Log("huntbot will be back in " + strconv.Itoa(secs) + "s")
				h.resendAfter(float64(secs))
				continue
			}
		}

		h.resendAfter(briefCooldownMin)
		h.bot.Log("huntbot back! sending next huntbot command.")
	}
}

func (h *Handler) handlePassword(msg Message) {
	if len(msg.Attachments) == 0 {
		return
	}
	ans, err := SolvePasswordCaptcha(msg.Attachments[0].URL, h.token)
	if err != nil || ans == "" {
		h.bot.Log("huntbot password solve failed")
		return
	}
	h.bot.Log("huntbot received password, attempting to solve!")
	h.bot.SleepUntil(briefCooldownMin, briefCooldownMax-briefCooldownMin)
	h.sendAutohunt(ans, true)
}

func (h *Handler) handlePasswordRetry(content string) {
	secs := briefCooldownMin
	if m := passwordResetRe.FindStringSubmatch(content); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			secs = float64(v * 60)
		}
	}
	h.bot.Log("huntbot stuck in password, retrying")
	h.resendAfter(secs)
}

func (h *Handler) resendAfter(seconds float64) {
	h.bot.CancelSleep()
	h.bot.SleepUntil(seconds, 30)
	h.sendAutohunt("", true)
}

func (h *Handler) sendAutohunt(password string, withCash bool) {
	if h == nil || h.bot == nil || !h.bot.CanSend() {
		return
	}
	args := ""
	if withCash || password != "" {
		args = strconv.Itoa(h.bot.Settings().CashToSpend)
	}
	if password != "" {
		args += " " + password
	}
	text := h.bot.RandomPrefix([]string{"huntbot", "ah", "hb", "autohunt"})
	if args != "" {
		text += " " + args
	}
	_ = h.bot.SendMessage(h.bot.HuntChannelID(), text)
}

func (h *Handler) parseEmbed(embed MessageEmbed) {
	for _, field := range embed.Fields {
		nameLower := strings.ToLower(field.Name)
		for _, trait := range AllTraits {
			if strings.Contains(nameLower, string(trait)) {
				level, invested := parseLevel(field.Value)
				h.setTrait(trait, level, invested)
				break
			}
		}
		if strings.Contains(nameLower, "animal essence") {
			h.upgradeDetails.Essence = parseEssence(field.Name)
		}
	}
}

func (h *Handler) setTrait(t Trait, level, invested int) {
	state := TraitState{Enabled: true, CurrentLevel: level, Invested: invested}
	switch t {
	case TraitEfficiency:
		h.upgradeDetails.Efficiency = state
	case TraitDuration:
		h.upgradeDetails.Duration = state
	case TraitCost:
		h.upgradeDetails.Cost = state
	case TraitGain:
		h.upgradeDetails.Gain = state
	case TraitExp:
		h.upgradeDetails.Exp = state
	case TraitRadar:
		h.upgradeDetails.Radar = state
	}
}

func parseLevel(value string) (level, invested int) {
	if strings.Contains(value, "[MAX]") {
		return 1000, 0
	}
	m := levelProgressRe.FindStringSubmatch(value)
	if len(m) < 3 {
		return 0, 0
	}
	level, _ = strconv.Atoi(m[1])
	invested, _ = strconv.Atoi(m[2])
	return level, invested
}

func parseEssence(name string) int {
	m := essenceRe.FindStringSubmatch(name)
	if len(m) < 2 {
		return 0
	}
	clean := strings.ReplaceAll(m[1], ",", "")
	v, _ := strconv.Atoi(clean)
	return v
}

func parseDuration(text string) int {
	total := 0
	for _, m := range huntbotTimeRe.FindAllStringSubmatch(text, -1) {
		amount, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "M":
			total += amount * 60
		case "H":
			total += amount * 3600
		case "D":
			total += amount * 86400
		}
	}
	return total
}

func (h *Handler) waitUpgrade(timeout time.Duration) {
	ch := make(chan struct{})
	h.mu.Lock()
	h.upgradeWaiters = append(h.upgradeWaiters, ch)
	h.mu.Unlock()

	select {
	case <-ch:
	case <-time.After(timeout):
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for i, w := range h.upgradeWaiters {
		if w == ch {
			h.upgradeWaiters = append(h.upgradeWaiters[:i], h.upgradeWaiters[i+1:]...)
			break
		}
	}
}

func (h *Handler) notifyUpgradeDone() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.upgradeWaiters {
		close(ch)
	}
	h.upgradeWaiters = nil
}
