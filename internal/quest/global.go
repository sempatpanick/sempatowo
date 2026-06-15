package quest

import (
	"sync"
)

// GlobalHandler tracks helpable quests shared across accounts.
type GlobalHandler struct {
	mu     sync.Mutex
	quests []HelpableQuest
}

var global = &GlobalHandler{}

func Global() *GlobalHandler { return global }

func (g *GlobalHandler) RegisterHelpable(q HelpableQuest) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.quests = append(g.quests, q)
}

func (g *GlobalHandler) ClaimQuest(userID, questID, claimUserID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.quests {
		if g.quests[i].UserID == userID && g.quests[i].QuestID == questID && g.quests[i].ClaimUserID == "" {
			g.quests[i].ClaimUserID = claimUserID
			return true
		}
	}
	return false
}

func (g *GlobalHandler) UpdateProgress(claimUserID, questUserID, questID string) (completed bool, current int, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.quests {
		q := &g.quests[i]
		if q.UserID != questUserID || q.QuestID != questID {
			continue
		}
		if questID != "cookie" && questID != "curse" && questID != "pray" && q.ClaimUserID != claimUserID {
			return false, 0, false
		}
		q.Current++
		if q.Current >= q.Total {
			q.Complete = true
			completed = true
		}
		return completed, q.Current, true
	}
	return false, 0, false
}

func (g *GlobalHandler) RemoveQuest(questUserID, questID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	filtered := g.quests[:0]
	for _, q := range g.quests {
		if q.UserID == questUserID && q.QuestID == questID {
			continue
		}
		filtered = append(filtered, q)
	}
	g.quests = filtered
}

func (g *GlobalHandler) AvailableQuests(excludeUserID string) []HelpableQuest {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []HelpableQuest
	for _, q := range g.quests {
		if q.ClaimUserID != "" || q.Complete {
			continue
		}
		if excludeUserID != "" && q.UserID == excludeUserID {
			continue
		}
		out = append(out, q)
	}
	return out
}

func (g *GlobalHandler) UpdateTotals(questUserID, questID string, current, total int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.quests {
		if g.quests[i].UserID == questUserID && g.quests[i].QuestID == questID {
			g.quests[i].Current = current
			g.quests[i].Total = total
			return
		}
	}
}
