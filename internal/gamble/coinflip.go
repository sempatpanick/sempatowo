package gamble

import (
	"regexp"
	"strings"
	"sync"
)

var (
	// Bet amount comes after <:emoji:id> or is wrapped in **...** — not the emoji snowflake ID.
	cfWonRe  = regexp.MustCompile(`(?i)you won(?:[^>]*>\s*\*?\*?([\d,]+)|\*\*([\d,]+)\*\*)`)
	cfLoseRe = regexp.MustCompile(`(?i)spent(?:[^>]*>\s*\*?\*?([\d,]+)|\*\*([\d,]+)\*\*)`)
)

type coinflipGame struct {
	m              *Manager
	mu             sync.Mutex
	turnsLost      int
	exceededMax    bool
	awaitingResult bool
}

func (g *coinflipGame) run(stop <-chan struct{}, startup bool) {
	if startup {
		sleepRange(briefCooldownMin, briefCooldownMax)
	}
	for {
		if stopped(stop) || !g.m.bot.CanSend() {
			return
		}
		if !g.m.bot.Gamble().Coinflip.Enabled {
			return
		}
		g.mu.Lock()
		exceeded := g.exceededMax
		g.mu.Unlock()
		if exceeded {
			return
		}

		settings := g.m.bot.Gamble().Coinflip
		dec := g.m.decideBet(settings.GambleGame, g.loseStreak(), func(amount int) string {
			side := coinflipSide(settings.Options)
			txt := g.m.bot.RandomPrefix([]string{"cf", "coinflip", "coin", "flip"}) + " " + itoa(amount)
			if side != "h" {
				txt += " " + side
			}
			return txt
		})

		if dec.stop {
			g.mu.Lock()
			g.exceededMax = true
			g.mu.Unlock()
			return
		}
		if dec.pause {
			sleepRange(moderateCooldownMin, moderateCooldownMax)
			continue
		}
		if dec.send {
			g.mu.Lock()
			g.awaitingResult = true
			g.mu.Unlock()
			g.m.bot.Log("Coinflip → " + dec.text)
			g.m.bot.SendGambleBet(g.m.bot.HuntChannelID(), QueueCoinflip, dec.text)
			return
		}
	}
}

func (g *coinflipGame) loseStreak() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.turnsLost
}

func (g *coinflipGame) scheduleNext(stop <-chan struct{}) {
	go func() {
		min, max := g.m.bot.Gamble().Coinflip.CooldownSec()
		sleepRange(min, max)
		if stopped(stop) {
			return
		}
		g.run(stop, false)
	}()
}

func (g *coinflipGame) onResult(msg Message) {
	g.mu.Lock()
	exceeded := g.exceededMax
	awaiting := g.awaitingResult
	g.mu.Unlock()
	if exceeded || !awaiting {
		return
	}
	lower := strings.ToLower(msg.Content)
	if !strings.Contains(lower, "chose") {
		return
	}

	stop := g.m.stopChan()
	if strings.Contains(lower, "and you lost it all") {
		lose, ok := parseRegexAmount(cfLoseRe, msg.Content)
		if !ok {
			return
		}
		g.mu.Lock()
		g.awaitingResult = false
		g.turnsLost++
		g.mu.Unlock()
		if g.m.bot.CashCheck() {
			g.m.state.updateCash(lose, false, true)
		}
		g.m.state.addGain(-lose)
		gain, _, _ := g.m.state.snapshot()
		g.m.bot.Log("Coinflip → lost " + itoa(lose) + " (net " + itoa(gain) + ")")
		g.m.bot.SignalGambleResult(QueueCoinflip)
		g.scheduleNext(stop)
		return
	}

	won, ok1 := parseRegexAmount(cfWonRe, msg.Content)
	lose, ok2 := parseRegexAmount(cfLoseRe, msg.Content)
	if !ok1 || !ok2 {
		return
	}
	profit := won - lose
	g.mu.Lock()
	g.awaitingResult = false
	g.turnsLost = 0
	g.mu.Unlock()
	if g.m.bot.CashCheck() {
		g.m.state.updateCash(profit, false, false)
	}
	g.m.state.addGain(profit)
	gain, _, _ := g.m.state.snapshot()
	g.m.bot.Log("Coinflip → won " + itoa(won) + " (net " + itoa(gain) + ")")
	g.m.bot.SignalGambleResult(QueueCoinflip)
	g.scheduleNext(stop)
}
