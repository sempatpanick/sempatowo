package quest

import (
	"math/rand"
	"strings"
)

var actionVerbs = []string{"stare", "kill", "greet", "punch", "wave", "slap"}

type repeatState struct {
	catchRank      string
	huntTill       int
	battleTill     int
	repeatStopFlag bool
}

func (m *Manager) handleSelfQuests(quests []LocalQuest) {
	for _, q := range quests {
		till := q.Total - q.Current
		if till <= 0 {
			continue
		}
		m.mu.Lock()
		m.repeat.catchRank = ""
		m.repeat.repeatStopFlag = false
		m.mu.Unlock()

		switch {
		case strings.HasPrefix(q.QuestID, "find_animal_"):
			if !m.canRunCmd("hunt") {
				continue
			}
			rank := strings.TrimPrefix(q.QuestID, "find_animal_")
			m.mu.Lock()
			m.repeat.catchRank = rank
			m.repeat.huntTill = till
			m.mu.Unlock()
			m.bot.Log("Quest: catch " + rank + " animal via hunt")
			m.runRepeat("hunt", 0, "")

		case q.QuestID == "hunt":
			if !m.canRunCmd("hunt") {
				continue
			}
			m.bot.Log("Quest: hunt x" + itoa(till))
			m.runRepeat("hunt", till, "")

		case q.QuestID == "battle_xp":
			if !m.canRunCmd("battle") {
				continue
			}
			m.mu.Lock()
			m.repeat.battleTill = till
			m.mu.Unlock()
			m.bot.Log("Quest: battle xp " + itoa(till))
			m.runRepeat("battle", 0, "")

		case q.QuestID == "battle":
			if !m.canRunCmd("battle") {
				continue
			}
			m.bot.Log("Quest: battle x" + itoa(till))
			m.runRepeat("battle", till, "")

		case q.QuestID == "owo":
			if !m.canRunCmd("owo") {
				continue
			}
			m.bot.Log("Quest: owo x" + itoa(till))
			m.runRepeat("owo", till, "")

		case q.QuestID == "gamble":
			if !m.canRunCmd("gamble") || !m.bot.IsGambleEnabled() {
				continue
			}
			m.bot.Log("Quest: gamble x" + itoa(till))
			game := "cf"
			if rand.Intn(2) == 0 {
				game = "slots"
			}
			m.runGambleRepeat(game, till)

		case q.QuestID == "action_send":
			m.bot.Log("Quest: actions x" + itoa(till))
			m.runActions(till, "", "")
		}
	}
}

func (m *Manager) handleHelpableQuests(quests []HelpableQuest) {
	if len(quests) == 0 {
		return
	}
	m.bot.Log("Attempting to help other quests")
	for _, q := range quests {
		till := q.Total - q.Current
		if till <= 0 {
			continue
		}
		claimed := true
		if q.QuestID != "cookie" && q.QuestID != "pray" && q.QuestID != "curse" {
			claimed = Global().ClaimQuest(q.UserID, q.QuestID, m.bot.UserID())
		}
		if !claimed {
			continue
		}
		target := "<@" + q.UserID + ">"
		ch := q.ChannelID
		if ch == "" {
			ch = m.bot.HuntChannelID()
		}

		switch q.QuestID {
		case "action_receive":
			m.runActions(till, q.UserID, ch)
		case "battle_friend":
			if m.canRunCmd("battle") {
				m.runRepeat("battle", till, target)
			}
		case "cookie", "pray", "curse":
			if m.canRunHelpCmd(q.QuestID) {
				m.runRepeat(q.QuestID, till, target)
			}
		}
	}
}

func (m *Manager) canRunCmd(name string) bool {
	if !m.bot.CanEnableQuestCmds() {
		return false
	}
	switch name {
	case "hunt":
		return !m.bot.IsHuntEnabled()
	case "battle":
		return !m.bot.IsBattleEnabled()
	case "owo":
		return true
	case "gamble":
		return true
	default:
		return true
	}
}

func (m *Manager) canRunHelpCmd(name string) bool {
	if !m.bot.CanEnableQuestCmds() {
		return false
	}
	switch name {
	case "cookie":
		return !m.bot.IsCookieEnabled()
	case "pray":
		return !m.bot.IsPrayEnabled()
	case "curse":
		return !m.bot.IsCurseEnabled()
	default:
		return true
	}
}

func (m *Manager) runRepeat(cmd string, till int, arg string) {
	ch := m.bot.HuntChannelID()
	count := till
	useCount := till > 0
	for {
		if !m.bot.CanSend() {
			return
		}
		m.mu.Lock()
		stop := m.repeat.repeatStopFlag
		m.mu.Unlock()
		if stop && !useCount {
			break
		}

		text := m.buildCmd(cmd, arg)
		m.bot.SendMessage(ch, text)
		if useCount {
			m.bot.SleepRange(2, 5)
			count--
			if count <= 0 {
				break
			}
		} else if stop {
			break
		}
		m.bot.SleepRange(15, 17)
	}
}

func (m *Manager) runGambleRepeat(game string, till int) {
	ch := m.bot.HuntChannelID()
	for i := 0; i < till; i++ {
		if !m.bot.CanSend() {
			return
		}
		amount := 1
		var text string
		switch game {
		case "slots":
			text = m.bot.RandomPrefix([]string{"s", "slots"}) + " 1"
		default:
			text = m.bot.RandomPrefix([]string{"cf", "coinflip"}) + " 1"
		}
		m.bot.SendMessage(ch, text)
		m.bot.SleepRange(2, 5)
		m.bot.GambleEnqueue(game, amount)
		m.bot.SleepRange(16, 18)
	}
}

func (m *Manager) runActions(till int, helpUserID, channelID string) {
	ch := channelID
	if ch == "" {
		ch = m.bot.HuntChannelID()
	}
	for i := 0; i < till; i++ {
		if !m.bot.CanSend() {
			return
		}
		verb := actionVerbs[rand.Intn(len(actionVerbs))]
		text := m.bot.RandomPrefix([]string{verb})
		if helpUserID != "" {
			text += " <@" + helpUserID + ">"
		}
		m.bot.SendMessage(ch, text)
		m.bot.SleepRange(2, 5)
		if helpUserID != "" {
			done, current, ok := Global().UpdateProgress(m.bot.UserID(), helpUserID, "action_receive")
			if ok {
				m.local.SyncProgress("action_receive", current, done)
				if done {
					break
				}
			}
		}
		m.bot.SleepRange(4, 6)
	}
}

func (m *Manager) buildCmd(cmd, arg string) string {
	switch cmd {
	case "owo":
		if arg != "" {
			return "owo " + arg
		}
		return "owo"
	case "battle":
		t := m.bot.RandomPrefix([]string{"battle", "b"})
		if arg != "" {
			return t + " " + arg
		}
		return t
	case "hunt":
		t := m.bot.RandomPrefix([]string{"hunt", "h"})
		if arg != "" {
			return t + " " + arg
		}
		return t
	case "cookie":
		return m.bot.RandomPrefix([]string{"cookie"}) + " " + arg
	case "pray", "curse":
		return m.bot.RandomPrefix([]string{cmd}) + " " + arg
	default:
		return cmd
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
