package quest

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Manager runs the auto-quest loop (owo-dusk quest cog).
type Manager struct {
	bot   BotContext
	local *LocalHandler

	mu      sync.Mutex
	running bool
	stop    chan struct{}

	repeat repeatState

	questUpdated chan struct{}
}

func NewManager(bot BotContext, local *LocalHandler) *Manager {
	return &Manager{
		bot:          bot,
		local:        local,
		questUpdated: make(chan struct{}, 1),
	}
}

func (m *Manager) Start() {
	if m == nil || m.bot == nil || !m.bot.AutoQuestActive() {
		return
	}
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stop = make(chan struct{})
	stop := m.stop
	m.mu.Unlock()

	go m.loop(stop)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	close(m.stop)
	m.stop = nil
	m.running = false
}

func (m *Manager) stopChan() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stop
}

func (m *Manager) loop(stop <-chan struct{}) {
	for {
		if stopped(stop) || !m.bot.CanSend() || !m.bot.AutoQuestActive() {
			return
		}

		m.signalQuestUpdated() // clear stale
		ch := m.bot.HuntChannelID()
		m.bot.Log("Checking quests")
		m.bot.SendMessage(ch, m.bot.RandomPrefix([]string{"quest", "q"}))

		select {
		case <-stop:
			return
		case <-m.questUpdated:
		case <-time.After(20 * time.Second):
		}

		s := m.bot.AutoQuest()
		if s.HelpChannel.PostInHelpChannel && m.local.HelpRequired() && !m.local.HelpAlreadyPosted() {
			helpCh := m.bot.QuestHelpChannelID()
			if helpCh != "" {
				m.bot.SendMessage(helpCh, m.bot.RandomPrefix([]string{"quest", "q"}))
				m.local.MarkHelpPosted()
			}
		}

		self := m.local.SelfDoable()
		m.handleSelfQuests(self)

		if s.HelpOthers {
			helpable := m.local.HelpableFromGlobal()
			m.handleHelpableQuests(helpable)
		}

		if m.local.IsAllDone() {
			m.local.WaitTillNextQuest(stop)
			continue
		}

		min, max := s.CheckCooldown.SecondsRange()
		m.bot.SleepRange(min, max)
	}
}

func (m *Manager) signalQuestUpdated() {
	select {
	case m.questUpdated <- struct{}{}:
	default:
	}
}

func (m *Manager) HandleRawMessage(event string, data json.RawMessage) {
	if m == nil || !m.bot.AutoQuestActive() {
		return
	}
	var msg RawMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Author.ID != m.bot.OwoBotID() || msg.ChannelID != m.bot.HuntChannelID() {
		return
	}
	ui := ParseQuestUI(msg.Components, m.bot.UserID())
	if !ui.ValidQuestLog && event == "MESSAGE_CREATE" {
		return
	}
	if !ui.ValidQuestLog && ui.QuestImageURL == "" && !ui.AllDone {
		return
	}

	if ui.ClaimCustomID != "" {
		_ = m.bot.ClickButton(msg.ChannelID, msg.ID, ui.ClaimCustomID, m.bot.OwoBotID())
		m.bot.Log("Claimed a finished quest")
	}

	if ui.AllDone {
		m.bot.Log("All quests finished for today")
		m.local.SetAllDone(true, ui.NextQuestTimestamp)
		m.signalQuestUpdated()
		return
	}

	if ui.QuestImageURL != "" {
		m.bot.Log("Updating quest list from image")
		guildID := msg.GuildID
		if guildID == "" {
			guildID = m.bot.GuildID()
		}
		if err := m.local.UpdateFromOCR(ui.QuestImageURL, msg.ChannelID, guildID, ui.NextQuestTimestamp); err != nil {
			m.bot.Log("Quest OCR failed: " + err.Error())
			m.signalQuestUpdated()
		} else {
			m.signalQuestUpdated()
		}
	}
}

func (m *Manager) HandleHuntMessage(content, nick string) {
	if m == nil || nick == "" || !strings.Contains(content, nick) {
		return
	}
	lower := strings.ToLower(content)

	m.mu.Lock()
	catchRank := m.repeat.catchRank
	battleTill := m.repeat.battleTill
	m.mu.Unlock()

	if catchRank != "" && (strings.Contains(lower, "you found:") || strings.Contains(lower, "caught")) {
		if HasRankedEmoji(content, catchRank) {
			m.mu.Lock()
			m.repeat.huntTill--
			if m.repeat.huntTill <= 0 {
				m.repeat.repeatStopFlag = true
			}
			m.mu.Unlock()
		}
	}

	if battleTill > 0 && strings.Contains(lower, "xp") {
		// battle XP tracked via embed footer in HandleBattleEmbed
	}
}

func (m *Manager) HandleBattleEmbed(authorName, footerText string) {
	m.mu.Lock()
	till := m.repeat.battleTill
	m.mu.Unlock()
	if till <= 0 {
		return
	}
	if !strings.Contains(authorName, m.bot.DisplayName()) && !strings.Contains(authorName, m.bot.Username()) {
		return
	}
	xp := parseBattleXP(footerText)
	if xp <= 0 {
		return
	}
	m.mu.Lock()
	m.repeat.battleTill -= xp
	if m.repeat.battleTill <= 0 {
		m.repeat.repeatStopFlag = true
	}
	m.mu.Unlock()
}

var battleXPRe = regexp.MustCompile(`\+([\d,]+)\s*xp`)

func parseBattleXP(footer string) int {
	m := battleXPRe.FindStringSubmatch(strings.ToLower(footer))
	if len(m) < 2 {
		return 0
	}
	n := 0
	for _, c := range strings.ReplaceAll(m[1], ",", "") {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func stopped(stop <-chan struct{}) bool {
	if stop == nil {
		return true
	}
	select {
	case <-stop:
		return true
	default:
		return false
	}
}
