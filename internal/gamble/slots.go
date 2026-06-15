package gamble

import (
	"regexp"
	"strings"
	"sync"
)

var (
	// Amount comes after <:emoji:id>, not the snowflake ID inside the emoji.
	slotsWonRe = regexp.MustCompile(`(?i)and won(?:[^>]*>\s*\*?\*?([\d,]+)|\*\*([\d,]+)\*\*)`)
	slotsBetRe = regexp.MustCompile(`(?i)bet(?:[^>]*>\s*\*?\*?([\d,]+)|\*\*([\d,]+)\*\*)`)
)

type slotsGame struct {
	m              *Manager
	mu             sync.Mutex
	turnsLost      int
	exceededMax    bool
	awaitingResult bool
}

func (g *slotsGame) run(stop <-chan struct{}, startup bool) {
	if startup {
		sleepRange(briefCooldownMin, briefCooldownMax)
	}
	for {
		if stopped(stop) || !g.m.bot.CanSend() {
			return
		}
		if !g.m.bot.Gamble().Slots.Enabled {
			return
		}
		g.mu.Lock()
		exceeded := g.exceededMax
		g.mu.Unlock()
		if exceeded {
			return
		}

		settings := g.m.bot.Gamble().Slots
		dec := g.m.decideBet(settings, g.loseStreak(), func(amount int) string {
			return g.m.bot.RandomPrefix([]string{"s", "slots", "slot"}) + " " + itoa(amount)
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
			g.m.bot.Log("Slots: " + dec.text)
			g.m.bot.SendGambleBet(g.m.bot.HuntChannelID(), QueueSlots, dec.text)
			return
		}
	}
}

func (g *slotsGame) loseStreak() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.turnsLost
}

func (g *slotsGame) scheduleNext(stop <-chan struct{}) {
	go func() {
		min, max := g.m.bot.Gamble().Slots.CooldownSec()
		sleepRange(min, max)
		if stopped(stop) {
			return
		}
		g.run(stop, false)
	}()
}

func (g *slotsGame) onResult(msg Message) {
	g.mu.Lock()
	exceeded := g.exceededMax
	awaiting := g.awaitingResult
	g.mu.Unlock()
	if exceeded || !awaiting {
		return
	}
	lower := strings.ToLower(msg.Content)
	if !strings.Contains(lower, "slots") {
		return
	}

	stop := g.m.stopChan()
	if strings.Contains(lower, "and won nothing") {
		bet, ok := parseRegexAmount(slotsBetRe, msg.Content)
		if !ok {
			return
		}
		g.mu.Lock()
		g.awaitingResult = false
		g.turnsLost++
		g.mu.Unlock()
		if g.m.bot.CashCheck() {
			g.m.state.updateCash(bet, false, true)
		}
		g.m.state.addGain(-bet)
		gain, _, _ := g.m.state.snapshot()
		g.m.bot.Log("lost " + itoa(bet) + " in slots, net profit " + itoa(gain))
		g.m.bot.SignalGambleResult(QueueSlots)
		g.scheduleNext(stop)
		return
	}

	if strings.Contains(lower, "<:eggplant:417475705719226369>") && strings.Contains(lower, "and won") {
		g.mu.Lock()
		g.awaitingResult = false
		g.mu.Unlock()
		g.m.bot.Log("slots: no win or loss")
		g.m.bot.SignalGambleResult(QueueSlots)
		g.scheduleNext(stop)
		return
	}

	if strings.Contains(lower, "and won") {
		won, ok1 := parseRegexAmount(slotsWonRe, msg.Content)
		bet, ok2 := parseRegexAmount(slotsBetRe, msg.Content)
		if !ok1 || !ok2 {
			return
		}
		if won == bet {
			g.mu.Lock()
			g.awaitingResult = false
			g.mu.Unlock()
			g.m.bot.Log("slots: draw (" + itoa(bet) + ")")
			g.m.bot.SignalGambleResult(QueueSlots)
			g.scheduleNext(stop)
			return
		}
		profit := won - bet
		g.mu.Lock()
		g.awaitingResult = false
		g.turnsLost = 0
		g.mu.Unlock()
		if g.m.bot.CashCheck() {
			g.m.state.updateCash(profit, false, false)
		}
		g.m.state.addGain(profit)
		gain, _, _ := g.m.state.snapshot()
		g.m.bot.Log("won " + itoa(won) + " in slots, net profit " + itoa(gain))
		g.m.bot.SignalGambleResult(QueueSlots)
		g.scheduleNext(stop)
	}
}
