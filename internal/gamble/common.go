package gamble

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/util"
)

var commaAmountRe = regexp.MustCompile(`([\d,]+)`)

type sharedState struct {
	mu          sync.Mutex
	gainOrLose  int
	balance     int
	checkedCash bool

	goalReached    bool
	amountExceeded bool
	noBalance      bool
}

func (s *sharedState) updateCash(amount int, override, reduce bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if override {
		s.balance = amount
		return
	}
	if reduce {
		s.balance -= amount
	} else {
		s.balance += amount
	}
}

func (s *sharedState) addGain(delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gainOrLose += delta
}

func (s *sharedState) snapshot() (gainOrLose, balance int, checkedCash bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gainOrLose, s.balance, s.checkedCash
}

func (s *sharedState) setCheckedCash() {
	s.mu.Lock()
	s.checkedCash = true
	s.mu.Unlock()
}

func parseCommaAmount(s string) (int, bool) {
	m := commaAmountRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
	return n, err == nil
}

// parseRegexAmount returns the first non-empty capture group from re in content.
func parseRegexAmount(re *regexp.Regexp, content string) (int, bool) {
	m := re.FindStringSubmatch(content)
	if m == nil {
		return 0, false
	}
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			return parseCommaAmount(m[i])
		}
	}
	return 0, false
}

func randomBetween(min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + rand.Float64()*(max-min)
}

func sleepRange(min, max float64) {
	d := randomBetween(min, max)
	time.Sleep(time.Duration(d * float64(time.Second)))
}

type betDecision struct {
	amount int
	send   bool
	pause  bool // wait moderate cooldown and retry
	stop   bool // exceeded 250k — stop game
	text   string
}

func (m *Manager) decideBet(game config.GambleGame, turnsLost int, buildText func(amount int) string) betDecision {
	gain, balance, checkedCash := m.state.snapshot()
	settings := m.bot.Gamble()

	amount := int(float64(game.StartValue) * math.Pow(game.MultiplierOnLose, float64(turnsLost)))
	if amount <= 0 {
		amount = game.StartValue
	}

	gs := settings.GoalSystem
	if gs.Enabled && gain > gs.Amount {
		if !m.flagGoalReached(true) {
			m.bot.Log("goal reached - " + logAmt(gain) + "/" + logAmt(gs.Amount) + ", pausing gamble")
		}
		return betDecision{pause: true}
	}
	if gs.Enabled {
		m.flagGoalReached(false)
	}

	if m.bot.CashCheck() && checkedCash && amount > balance {
		if !m.flagNoBalance(true) {
			m.bot.Log("bet " + logAmt(amount) + " exceeds balance " + logAmt(balance) + ", pausing gamble")
		}
		return betDecision{pause: true}
	}
	if m.bot.CashCheck() && checkedCash {
		if m.flagNoBalance(false) {
			m.bot.Log("balance regained (" + logAmt(balance) + ") — resuming gamble")
		}
	}

	allotted := settings.AllottedAmount
	if gain+(allotted-amount) <= 0 {
		if !m.flagAmountExceeded(true) {
			m.bot.Log("allotted value " + logAmt(allotted) + " exceeded, pausing gamble")
		}
		return betDecision{pause: true}
	}
	m.flagAmountExceeded(false)

	if amount > maxBetAmount {
		m.bot.Log("bet " + logAmt(amount) + " exceeded 250k limit, stopping gamble")
		return betDecision{stop: true}
	}

	return betDecision{amount: amount, send: true, text: buildText(amount)}
}

func (m *Manager) flagGoalReached(v bool) bool {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	prev := m.state.goalReached
	m.state.goalReached = v
	return prev
}

func (m *Manager) flagNoBalance(v bool) bool {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	prev := m.state.noBalance
	m.state.noBalance = v
	return prev
}

func (m *Manager) flagAmountExceeded(v bool) bool {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	prev := m.state.amountExceeded
	m.state.amountExceeded = v
	return prev
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func logAmt(n int) string {
	return util.FormatInt(n)
}

func coinflipSide(opts config.CoinflipOptions) string {
	var choices []string
	if opts.Heads {
		choices = append(choices, "h")
	}
	if opts.Tails {
		choices = append(choices, "t")
	}
	if len(choices) == 0 {
		return "h"
	}
	return choices[rand.Intn(len(choices))]
}
