package gamble

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

var bjCardRe = regexp.MustCompile(`\d+`)

type blackjackGame struct {
	m           *Manager
	mu          sync.Mutex
	turnsLost   int
	exceededMax bool
	gameEvent   chan struct{}
}

func findOptimalMove(dealer, player int, soft bool) string {
	if dealer <= 6 {
		if soft {
			if player < 18 {
				return "hit"
			}
			return "stand"
		}
		if player <= 11 {
			return "hit"
		}
		if player == 12 {
			if dealer == 2 {
				return "hit"
			}
			return "stand"
		}
		return "stand"
	}
	if soft {
		if player == 18 {
			if dealer >= 9 {
				return "hit"
			}
			return "stand"
		}
		if player < 19 {
			return "hit"
		}
		return "stand"
	}
	if player < 17 {
		return "hit"
	}
	return "stand"
}

func fetchBJHands(embed MessageEmbed) (dealer, player int, soft bool, ok bool) {
	if len(embed.Fields) < 2 {
		return 0, 0, false, false
	}
	dealerName := embed.Fields[0].Name
	playerName := embed.Fields[1].Name
	if dealerName == "" || playerName == "" {
		return 0, 0, false, false
	}
	soft = strings.Contains(playerName, "*")
	dm := bjCardRe.FindString(dealerName)
	pm := bjCardRe.FindString(playerName)
	if dm == "" || pm == "" {
		return 0, 0, false, false
	}
	dealer, _ = parseCommaAmount(dm)
	player, _ = parseCommaAmount(pm)
	return dealer, player, soft, true
}

func (g *blackjackGame) run(stop <-chan struct{}, startup bool) {
	if startup {
		sleepRange(briefCooldownMin, briefCooldownMax)
	}
	for {
		if stopped(stop) || !g.m.bot.CanSend() {
			return
		}
		if !g.m.bot.Gamble().Blackjack.Enabled {
			return
		}
		g.mu.Lock()
		exceeded := g.exceededMax
		g.mu.Unlock()
		if exceeded {
			return
		}

		settings := g.m.bot.Gamble().Blackjack
		dec := g.m.decideBet(settings, g.loseStreak(), func(amount int) string {
			return g.m.bot.RandomPrefix([]string{"bj", "blackjack"}) + " " + itoa(amount)
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
			g.m.bot.Log("Blackjack: " + dec.text)
			g.m.bot.SendMessage(g.m.bot.HuntChannelID(), dec.text)
			return
		}
	}
}

func (g *blackjackGame) loseStreak() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.turnsLost
}

func (g *blackjackGame) scheduleNext(stop <-chan struct{}) {
	go func() {
		min, max := g.m.bot.Gamble().Blackjack.CooldownSec()
		sleepRange(min, max)
		if stopped(stop) {
			return
		}
		g.run(stop, false)
	}()
}

func (g *blackjackGame) embedForUser(embed MessageEmbed) bool {
	if embed.Author == nil {
		return false
	}
	name := g.m.bot.Username()
	return name != "" && strings.Contains(embed.Author.Name, name)
}

func (g *blackjackGame) onMessage(msg Message) {
	g.mu.Lock()
	exceeded := g.exceededMax
	g.mu.Unlock()
	if exceeded {
		return
	}
	for _, embed := range msg.Embeds {
		if !g.embedForUser(embed) || embed.Footer == nil {
			continue
		}
		ft := embed.Footer.Text
		if strings.Contains(ft, "game in progress") || strings.Contains(ft, "resuming previous game") {
			g.handleInProgress(msg, embed)
		}
	}
}

func (g *blackjackGame) onUpdate(msg Message) {
	g.mu.Lock()
	exceeded := g.exceededMax
	g.mu.Unlock()
	if exceeded {
		return
	}
	for _, embed := range msg.Embeds {
		if !g.embedForUser(embed) || embed.Footer == nil {
			continue
		}
		ft := embed.Footer.Text
		stop := g.m.stopChan()

		switch {
		case strings.Contains(ft, "game in progress"):
			g.signalGameEvent()
			g.handleInProgress(msg, embed)

		case strings.Contains(ft, "You lost"):
			g.signalGameEvent()
			amt, ok := parseCommaAmount(ft)
			if !ok {
				return
			}
			g.mu.Lock()
			g.turnsLost++
			g.mu.Unlock()
			if g.m.bot.CashCheck() {
				g.m.state.updateCash(amt, false, true)
			}
			g.m.state.addGain(-amt)
			gain, _, _ := g.m.state.snapshot()
			g.m.bot.Log("lost " + itoa(amt) + " in bj, net profit " + itoa(gain))
			g.scheduleNext(stop)

		case strings.Contains(ft, "You won"):
			g.signalGameEvent()
			amt, ok := parseCommaAmount(ft)
			if !ok {
				return
			}
			g.mu.Lock()
			g.turnsLost = 0
			g.mu.Unlock()
			if g.m.bot.CashCheck() {
				g.m.state.updateCash(amt, false, false)
			}
			g.m.state.addGain(amt)
			gain, _, _ := g.m.state.snapshot()
			g.m.bot.Log("won " + itoa(amt) + " in bj, net profit " + itoa(gain))
			g.scheduleNext(stop)

		case strings.Contains(ft, "You tied!") || strings.Contains(ft, "You both bust!"):
			g.signalGameEvent()
			g.m.bot.Log("blackjack: no win or loss")
			g.scheduleNext(stop)
		}
	}
}

func (g *blackjackGame) signalGameEvent() {
	g.mu.Lock()
	ev := g.gameEvent
	g.mu.Unlock()
	if ev == nil {
		return
	}
	select {
	case ev <- struct{}{}:
	default:
	}
}

func (g *blackjackGame) handleInProgress(msg Message, embed MessageEmbed) {
	dealer, player, soft, ok := fetchBJHands(embed)
	if !ok {
		return
	}
	move := findOptimalMove(dealer, player, soft)
	go g.clickReaction(msg, move)
}

func (g *blackjackGame) clickReaction(msg Message, move string) {
	emoji := "🛑"
	if move == "hit" {
		emoji = "👊"
	}
	ch := g.m.bot.HuntChannelID()
	stop := g.m.stopChan()

	clicked := false
	for tries := 0; tries < 3; tries++ {
		if stopped(stop) {
			return
		}
		time.Sleep(1500 * time.Millisecond)

		for _, r := range msg.Reactions {
			if r.Emoji != emoji {
				continue
			}
			if r.Me {
				_ = g.m.bot.RemoveReaction(ch, msg.ID, emoji)
			} else {
				_ = g.m.bot.AddReaction(ch, msg.ID, emoji)
			}
			clicked = true
			break
		}
		if clicked {
			break
		}
		_ = g.m.bot.AddReaction(ch, msg.ID, emoji)
		clicked = true
		break
	}
	if !clicked {
		g.scheduleNext(stop)
		return
	}

	g.mu.Lock()
	g.gameEvent = make(chan struct{}, 1)
	ev := g.gameEvent
	g.mu.Unlock()

	select {
	case <-ev:
		// OwO edited the message — onUpdate handles the next move or result.
	case <-time.After(4 * time.Second):
		g.scheduleNext(stop)
	}
}
