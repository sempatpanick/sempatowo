package quest

import (
	"sync"
	"time"
)

// LocalHandler stores quests for one account.
type LocalHandler struct {
	mu                  sync.Mutex
	userID              string
	global              *GlobalHandler
	apiKey              string
	quests              []LocalQuest
	nextQuestTimestamp  int64
	allDone             bool
	helpPosted          bool
}

func NewLocalHandler(userID, apiKey string) *LocalHandler {
	return &LocalHandler{
		userID: userID,
		global: Global(),
		apiKey: apiKey,
	}
}

func (l *LocalHandler) UpdateFromOCR(url, channelID, guildID string, nextTS int64) error {
	parsed, err := FetchQuestDetails(url, l.apiKey)
	if err != nil {
		return err
	}

	l.mu.Lock()
	existing := make(map[string]bool)
	for _, q := range l.quests {
		existing[q.QuestID] = true
	}
	l.quests = nil
	recorded := map[string]bool{}

	for _, pq := range parsed {
		meta, ok := LookupQuest(pq.Text)
		if !ok || pq.Complete {
			continue
		}
		entry := LocalQuest{
			Text:     pq.Text,
			QuestID:  meta.ID,
			Current:  pq.Current,
			Total:    pq.Total,
			Complete: pq.Complete,
			Helpable: meta.Helpable,
		}
		if !recorded[meta.ID] {
			recorded[meta.ID] = true
			l.quests = append(l.quests, entry)
			if meta.Helpable && !existing[meta.ID] {
				l.global.RegisterHelpable(HelpableQuest{
					UserID:    l.userID,
					QuestID:   meta.ID,
					Current:   pq.Current,
					Total:     pq.Total,
					ChannelID: channelID,
					GuildID:   guildID,
				})
			}
		} else {
			for i := range l.quests {
				if l.quests[i].QuestID == meta.ID {
					l.quests[i].Current += pq.Current
					l.quests[i].Total += pq.Total
					if meta.Helpable {
						l.global.UpdateTotals(l.userID, meta.ID, l.quests[i].Current, l.quests[i].Total)
					}
					break
				}
			}
		}
	}
	if nextTS > 0 {
		l.nextQuestTimestamp = nextTS
	}
	l.mu.Unlock()
	return nil
}

func (l *LocalHandler) SetAllDone(done bool, nextTS int64) {
	l.mu.Lock()
	l.allDone = done
	if nextTS > 0 {
		l.nextQuestTimestamp = nextTS
	}
	l.mu.Unlock()
}

func (l *LocalHandler) SelfDoable() []LocalQuest {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []LocalQuest
	for _, q := range l.quests {
		if !q.Complete && !q.Helpable {
			out = append(out, q)
		}
	}
	return out
}

func (l *LocalHandler) HelpRequired() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, q := range l.quests {
		if !q.Complete && q.Helpable {
			return true
		}
	}
	return false
}

func (l *LocalHandler) HelpableFromGlobal() []HelpableQuest {
	return l.global.AvailableQuests(l.userID)
}

func (l *LocalHandler) SyncProgress(questID string, current int, completed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.quests {
		if l.quests[i].QuestID == questID {
			l.quests[i].Current = current
			if completed {
				l.quests[i].Complete = true
				l.global.RemoveQuest(l.userID, questID)
			}
			return
		}
	}
}

func (l *LocalHandler) WaitTillNextQuest(stop <-chan struct{}) {
	l.mu.Lock()
	ts := l.nextQuestTimestamp
	l.allDone = false
	l.helpPosted = false
	l.mu.Unlock()
	if ts <= 0 {
		return
	}
	wait := time.Until(time.Unix(ts, 0))
	if wait <= 0 {
		return
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-stop:
	case <-t.C:
	}
}

func (l *LocalHandler) MarkHelpPosted() {
	l.mu.Lock()
	l.helpPosted = true
	l.mu.Unlock()
}

func (l *LocalHandler) HelpAlreadyPosted() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.helpPosted
}

func (l *LocalHandler) IsAllDone() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.allDone
}
