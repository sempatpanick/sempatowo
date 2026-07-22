package farm

import "sync"

// farmStats holds what the bot has learned by reading OwO's replies: the
// inventory, the checklist ticks, and the running counters.
//
// These used to be plain Bot fields written straight from the message handlers.
// The gateway library dispatches MESSAGE_CREATE and MESSAGE_UPDATE from
// separate goroutines, so an inventory reply arriving next to a hunt reply had
// one handler writing b.inventory while the other ranged over it — an
// unsynchronised map, which is a crash rather than a stale read.
//
// Its mutex is a leaf: no method here calls back into Bot, so it can never take
// part in a lock cycle.
type farmStats struct {
	mu        sync.Mutex
	inventory map[string]int
	checklist checklistState

	totalXP    int
	totalHunts int

	// lastBattleLog dedupes the battle summary, which OwO sends once on create
	// and again on each edit of the same message.
	lastBattleLog string
}

type checklistState struct {
	daily, vote, cookie, quest, lootbox, crate bool
}

// allDone reports whether every checklist item is ticked.
func (c checklistState) allDone() bool {
	return c.daily && c.cookie && c.quest && c.lootbox && c.crate
}

func newFarmStats() *farmStats {
	return &farmStats{inventory: make(map[string]int)}
}

// SetInventory replaces the known inventory from an inventory reply.
func (s *farmStats) SetInventory(inv map[string]int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventory = inv
}

// Inventory returns a copy, so callers can range over it without holding the
// lock and without racing the next inventory reply.
func (s *farmStats) Inventory() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.inventory))
	for k, v := range s.inventory {
		out[k] = v
	}
	return out
}

// ConsumeItems decrements the local count for items just spent, so the next
// hunt does not try to use a gem that is already gone.
func (s *farmStats) ConsumeItems(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.inventory[id]--
		if s.inventory[id] <= 0 {
			delete(s.inventory, id)
		}
	}
}

func (s *farmStats) AddHunt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalHunts++
}

func (s *farmStats) AddXP(xp int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalXP += xp
}

func (s *farmStats) Totals() (hunts, xp int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totalHunts, s.totalXP
}

func (s *farmStats) SetChecklist(c checklistState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checklist = c
}

func (s *farmStats) Checklist() checklistState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checklist
}

// SeenBattleLog records a battle summary and reports whether it is new. OwO
// edits the battle message several times; only the first should be logged.
func (s *farmStats) SeenBattleLog(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastBattleLog == key {
		return true
	}
	s.lastBattleLog = key
	return false
}
