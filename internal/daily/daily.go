package daily

import (
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/semptpanick/sempatowo/internal/util"
)

const (
	briefCooldownMin    = 0.7
	briefCooldownMax    = 2.7
	moderateCooldownMin = 70.0
	moderateCooldownMax = 200.0
)

var (
	dailyClaimRe    = regexp.MustCompile(`(?i)Here is your daily\s+([\d,]+)\s*Cowoncy`)
	dailyCooldownRe = regexp.MustCompile(`(?i)⏱\s*\|.*?Nu!.*?You need to wait`)
)

// Manager runs standalone auto-daily (owo-dusk daily cog).
type Manager struct {
	bot   BotContext
	store *Store

	mu      sync.Mutex
	running bool
	stop    chan struct{}
}

func NewManager(bot BotContext, store *Store) *Manager {
	return &Manager{bot: bot, store: store}
}

func (m *Manager) Start() {
	if m == nil || m.bot == nil || !m.bot.AutoDaily() {
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

	go m.runInitial(stop)
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

func (m *Manager) runInitial(stop <-chan struct{}) {
	if !m.waitForWindow(stop) {
		return
	}
	sleepRange(briefCooldownMin, briefCooldownMax)
	if stopped(stop) || !m.bot.CanSend() || !m.bot.AutoDaily() {
		return
	}
	m.sendDaily()
}

func (m *Manager) waitForWindow(stop <-chan struct{}) bool {
	last := m.store.LastDaily()
	if util.ShouldRunSinceLastPSTDay(last) {
		return !stopped(stop)
	}
	sec := util.SecondsUntilNextPSTMidnight()
	m.bot.Log("Daily → already claimed (sleep until PST midnight)")
	sleepUntil(stop, sec)
	return !stopped(stop) && m.bot.AutoDaily()
}

func (m *Manager) sendDaily() {
	ch := m.bot.HuntChannelID()
	text := m.bot.RandomPrefix([]string{"daily"})
	m.bot.SendMessage(ch, text)
	m.store.SetLastDaily(util.NowPSTUnix())
}

func (m *Manager) scheduleAfterResponse(stop <-chan struct{}) {
	go func() {
		defer util.Recover(m.bot.Log, "dailyLoop")
		sec := util.SecondsUntilNextPSTMidnight()
		sleepUntil(stop, sec)
		if stopped(stop) || !m.bot.CanSend() || !m.bot.AutoDaily() {
			return
		}
		sleepRange(moderateCooldownMin, moderateCooldownMax)
		if stopped(stop) || !m.bot.CanSend() || !m.bot.AutoDaily() {
			return
		}
		m.sendDaily()
	}()
}

func (m *Manager) HandleMessage(content, nick string) {
	if m == nil || !m.bot.AutoDaily() || nick == "" || !strings.Contains(content, nick) {
		return
	}

	claimed := dailyClaimRe.MatchString(content)
	onCooldown := dailyCooldownRe.MatchString(content)
	if !claimed && !onCooldown {
		return
	}

	stop := m.stopChan()
	if stopped(stop) {
		return
	}

	if claimed {
		if m.bot.CashCheck() {
			if reward, ok := parseReward(content); ok {
				m.bot.OnDailyReward(reward)
				m.bot.Log("Daily → claimed (+" + util.FormatInt(reward) + ")")
				m.scheduleAfterResponse(stop)
				return
			}
		}
		m.bot.Log("Daily → claimed")
		m.scheduleAfterResponse(stop)
		return
	}

	m.bot.Log("Daily → on cooldown (next PST midnight)")
	m.scheduleAfterResponse(stop)
}

func parseReward(content string) (int, bool) {
	m := dailyClaimRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return 0, false
	}
	n := strings.ReplaceAll(m[1], ",", "")
	var v int
	for _, c := range n {
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + int(c-'0')
	}
	return v, true
}

func sleepRange(min, max float64) {
	d := min
	if max > min {
		d = min + rand.Float64()*(max-min)
	}
	time.Sleep(time.Duration(d * float64(time.Second)))
}

func sleepUntil(stop <-chan struct{}, seconds float64) {
	if seconds <= 0 {
		return
	}
	t := time.NewTimer(time.Duration(seconds * float64(time.Second)))
	defer t.Stop()
	select {
	case <-stop:
	case <-t.C:
	}
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
