package farm

import (
	"container/heap"
	"time"
)

type scheduledCmd struct {
	name    string
	nextRun time.Time
	seq     uint64
}

type cmdHeap []*scheduledCmd

func (h cmdHeap) Len() int { return len(h) }
func (h cmdHeap) Less(i, j int) bool {
	if h[i].nextRun.Equal(h[j].nextRun) {
		return h[i].seq < h[j].seq
	}
	return h[i].nextRun.Before(h[j].nextRun)
}
func (h cmdHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *cmdHeap) Push(x any)   { *h = append(*h, x.(*scheduledCmd)) }
func (h *cmdHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type farmCmdDef struct {
	name           string
	enabled        func(*Bot) bool
	channel        func(*Bot) string
	text           func(*Bot) string
	log            string
	delayMs        func(*Bot) int
	startupDelayMs func(*Bot) int
}

var farmCommands = []farmCmdDef{
		{
			name: "hunt",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Hunt
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text:    func(b *Bot) string { return b.randomPrefix([]string{"hunt", "h"}) },
			log:     "Hunting",
			delayMs: func(b *Bot) int { return b.actionDelay("hunt") },
		},
		{
			name: "battle",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Battle
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text:    func(b *Bot) string { return b.randomPrefix([]string{"battle", "b"}) },
			log:     "Battling",
			delayMs: func(b *Bot) int { return b.actionDelay("battle") },
			startupDelayMs: func(b *Bot) int {
				return b.actionDelay("battle") / 2
			},
		},
		{
			name: "pray",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Pray
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text: func(b *Bot) string {
				txt := b.randomPrefix([]string{"pray"})
				if target := b.settings().Target.Pray; target != "" {
					txt += " <@" + target + ">"
				}
				return txt
			},
			log: "Praying",
			delayMs: func(b *Bot) int {
				return b.settings().Interval.Pray
			},
			startupDelayMs: func(b *Bot) int {
				return b.actionDelay("hunt") / 3
			},
		},
		{
			name: "curse",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Curse
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text: func(b *Bot) string {
				txt := b.randomPrefix([]string{"curse"})
				if target := b.settings().Target.Curse; target != "" {
					txt += " <@" + target + ">"
				}
				return txt
			},
			log: "Cursing",
			delayMs: func(b *Bot) int {
				return b.settings().Interval.Curse
			},
		},
		{
			name: "zoo",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Zoo
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text:    func(b *Bot) string { return b.randomPrefix([]string{"zoo", "z", "Z", "Zoo"}) },
			log:     "Zoo",
			delayMs: func(b *Bot) int {
				return b.settings().Interval.Zoo
			},
		},
		{
			name: "inventory",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Inventory
			},
			channel: func(b *Bot) string { return b.settings().Channels.Hunt },
			text:    func(b *Bot) string { return b.randomPrefix([]string{"inv", "inventory"}) },
			log:     "Checking inventory",
			delayMs: func(b *Bot) int {
				return b.settings().Interval.Inventory
			},
		},
		{
			name: "quest",
			enabled: func(b *Bot) bool {
				return b.settings().Status.Quest
			},
			channel: func(b *Bot) string { return b.settings().Channels.Quest },
			text:    func(b *Bot) string { return b.randomPrefix([]string{"quest", "q"}) },
			log:     "Checking quest",
			delayMs: func(b *Bot) int {
				return b.settings().Interval.Quest.Check
			},
		},
}

func (b *Bot) farmCmdByName(name string) *farmCmdDef {
	for i := range farmCommands {
		if farmCommands[i].name == name {
			return &farmCommands[i]
		}
	}
	return nil
}

func (b *Bot) pushScheduledCmd(name string, nextRun time.Time) {
	b.cmdSeq++
	heap.Push(&b.cmdHeap, &scheduledCmd{name: name, nextRun: nextRun, seq: b.cmdSeq})
}

func (b *Bot) startFarmScheduler() {
	b.mu.Lock()
	b.stopFarmSchedulerLocked()

	stop := make(chan struct{})
	b.cmdSchedStop = stop
	b.cmdSeq = 0
	b.cmdHeap = nil

	now := time.Now()
	for _, def := range farmCommands {
		if !def.enabled(b) {
			continue
		}
		delay := 0
		if def.startupDelayMs != nil {
			delay = def.startupDelayMs(b)
		}
		b.pushScheduledCmd(def.name, now.Add(time.Duration(delay)*time.Millisecond))
	}
	heap.Init(&b.cmdHeap)
	if len(b.cmdHeap) == 0 {
		b.cmdSchedStop = nil
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	go b.runFarmScheduler(stop)
}

func (b *Bot) stopFarmSchedulerLocked() {
	if b.cmdSchedStop != nil {
		close(b.cmdSchedStop)
		b.cmdSchedStop = nil
	}
	b.cmdHeap = nil
}

func (b *Bot) runFarmScheduler(stop <-chan struct{}) {
	for {
		b.mu.Lock()
		if len(b.cmdHeap) == 0 {
			b.cmdSchedStop = nil
			b.mu.Unlock()
			return
		}
		next := b.cmdHeap[0]
		wait := time.Until(next.nextRun)
		b.mu.Unlock()

		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-stop:
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		select {
		case <-stop:
			return
		default:
		}

		b.mu.Lock()
		if len(b.cmdHeap) == 0 || b.cmdSchedStop == nil {
			b.mu.Unlock()
			return
		}
		item := heap.Pop(&b.cmdHeap).(*scheduledCmd)
		b.mu.Unlock()

		if !b.canSend() {
			return
		}

		def := b.farmCmdByName(item.name)
		if def == nil || !def.enabled(b) {
			continue
		}

		b.log.Info(def.log)
		b.enqueue(def.channel(b), def.text(b))

		delay := def.delayMs(b)
		if delay <= 0 {
			continue
		}

		b.mu.Lock()
		if b.canSendLocked() && b.cmdSchedStop != nil && def.enabled(b) {
			b.pushScheduledCmd(def.name, time.Now().Add(time.Duration(delay)*time.Millisecond))
		}
		b.mu.Unlock()
	}
}
