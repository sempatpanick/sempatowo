package gamble

import (
	"regexp"
	"strings"
	"sync"
)

var cashRe = regexp.MustCompile(`(\d{1,3}(?:,\d{3})*)\s+cowoncy`)

// Manager coordinates coinflip, slots, and blackjack.
type Manager struct {
	bot   BotContext
	state sharedState

	coinflip  *coinflipGame
	slots     *slotsGame
	blackjack *blackjackGame

	mu      sync.Mutex
	running bool
	stop    chan struct{}
}

func NewManager(bot BotContext) *Manager {
	m := &Manager{bot: bot}
	m.coinflip = &coinflipGame{m: m}
	m.slots = &slotsGame{m: m}
	m.blackjack = &blackjackGame{m: m}
	return m
}

func (m *Manager) Start() {
	if m == nil || m.bot == nil {
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

	g := m.bot.Gamble()
	if g.Coinflip.Enabled {
		go m.coinflip.run(stop, true)
	}
	if g.Slots.Enabled {
		go m.slots.run(stop, true)
	}
	if g.Blackjack.Enabled {
		go m.blackjack.run(stop, true)
	}
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

func (m *Manager) Restart() {
	m.Stop()
	if m.anyEnabled() {
		m.Start()
	}
}

func (m *Manager) anyEnabled() bool {
	g := m.bot.Gamble()
	return g.Coinflip.Enabled || g.Slots.Enabled || g.Blackjack.Enabled
}

func (m *Manager) stopChan() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stop
}

func (m *Manager) RequestCash() {
	if m == nil || !m.bot.CashCheck() || !m.bot.CanSend() {
		return
	}
	go func() {
		sleepRange(4.5, 34.4)
		if !m.bot.CanSend() {
			return
		}
		ch := m.bot.HuntChannelID()
		m.bot.SendMessage(ch, m.bot.RandomPrefix([]string{"cash", "money", "cowoncy"}))
	}()
}

func (m *Manager) HandleCash(content, nick string) {
	if m == nil || !m.bot.CashCheck() {
		return
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "you currently have") || !strings.Contains(content, nick) {
		return
	}
	clean := strings.NewReplacer("*", "", "_", "").Replace(content)
	parts := cashRe.FindStringSubmatch(clean)
	if len(parts) < 2 {
		return
	}
	amt, ok := parseCommaAmount(parts[1])
	if !ok {
		return
	}
	m.state.updateCash(amt, true, false)
	m.state.setCheckedCash()
	m.bot.Log("Balance → " + logAmt(amt))
}

// UpdateBalance sets or adjusts the tracked cowoncy balance (e.g. after daily reward).
func (m *Manager) UpdateBalance(amount int, override bool) {
	if m == nil {
		return
	}
	m.state.updateCash(amount, override, false)
	if override {
		m.state.setCheckedCash()
	}
}

func (m *Manager) HandleMessage(msg Message) {
	if m == nil || msg.AuthorID != m.bot.OwoBotID() || msg.ChannelID != m.bot.HuntChannelID() {
		return
	}
	if m.bot.Gamble().Blackjack.Enabled {
		m.blackjack.onMessage(msg)
	}
}

func (m *Manager) HandleGambleResult(msg Message) {
	if m == nil || msg.ChannelID != m.bot.HuntChannelID() {
		return
	}
	if msg.AuthorID != "" && msg.AuthorID != m.bot.OwoBotID() {
		return
	}
	if !contentForUser(msg.Content, m.bot.Nickname(), m.bot.Username(), m.bot.UserID()) {
		return
	}
	g := m.bot.Gamble()
	if g.Coinflip.Enabled {
		m.coinflip.onResult(msg)
	}
	if g.Slots.Enabled {
		m.slots.onResult(msg)
	}
}

func (m *Manager) HandleMessageUpdate(msg Message) {
	if m == nil || msg.ChannelID != m.bot.HuntChannelID() {
		return
	}
	if msg.AuthorID != "" && msg.AuthorID != m.bot.OwoBotID() {
		return
	}
	m.HandleGambleResult(msg)
	if m.bot.Gamble().Blackjack.Enabled {
		m.blackjack.onUpdate(msg)
	}
}

func stopped(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}
