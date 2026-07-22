package farm

import (
	"time"

	"github.com/semptpanick/sempatowo/internal/util"
)

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
			return b.settings().Features.Hunt.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text:    func(b *Bot) string { return b.randomPrefix([]string{"hunt", "h"}) },
		delayMs: func(b *Bot) int { return b.actionDelay("hunt") },
	},
	{
		name: "battle",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Battle.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text:    func(b *Bot) string { return b.randomPrefix([]string{"battle", "b"}) },
		delayMs: func(b *Bot) int { return b.actionDelay("battle") },
		startupDelayMs: func(b *Bot) int {
			return b.actionDelay("battle") / 2
		},
	},
	{
		name: "pray",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Pray.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text: func(b *Bot) string {
			txt := b.randomPrefix([]string{"pray"})
			if target := b.settings().Features.Pray.Target; target != "" {
				txt += " <@" + target + ">"
			}
			return txt
		},
		delayMs: func(b *Bot) int {
			return int(b.settings().Features.Pray.Delay.Pick() / time.Millisecond)
		},
		startupDelayMs: func(b *Bot) int {
			return b.actionDelay("hunt") / 3
		},
	},
	{
		name: "curse",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Curse.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text: func(b *Bot) string {
			txt := b.randomPrefix([]string{"curse"})
			if target := b.settings().Features.Curse.Target; target != "" {
				txt += " <@" + target + ">"
			}
			return txt
		},
		log: "Cursing",
		delayMs: func(b *Bot) int {
			return int(b.settings().Features.Curse.Delay.Pick() / time.Millisecond)
		},
	},
	{
		name: "zoo",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Zoo.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text:    func(b *Bot) string { return b.randomPrefix([]string{"zoo", "z", "Z", "Zoo"}) },
		log:     "Zoo",
		delayMs: func(b *Bot) int {
			return int(b.settings().Features.Zoo.Delay.Pick() / time.Millisecond)
		},
	},
	{
		name: "inventory",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Inventory.Enabled
		},
		channel: func(b *Bot) string { return b.settings().FarmChannel() },
		text:    func(b *Bot) string { return b.randomPrefix([]string{"inv", "inventory"}) },
		delayMs: func(b *Bot) int {
			return int(b.settings().Features.Inventory.Delay.Pick() / time.Millisecond)
		},
	},
	{
		name: "quest",
		enabled: func(b *Bot) bool {
			return b.settings().Features.Quest.Enabled
		},
		channel: func(b *Bot) string { return b.settings().QuestChannel() },
		text:    func(b *Bot) string { return b.randomPrefix([]string{"quest", "q"}) },
		log:     "Checking quest",
		delayMs: func(b *Bot) int {
			return int(b.settings().Features.Quest.Delay.Pick() / time.Millisecond)
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

func (b *Bot) startFarmScheduler() {
	stop, wake := b.sched.Begin()

	now := time.Now()
	for _, def := range farmCommands {
		if !def.enabled(b) {
			continue
		}
		delay := 0
		if def.startupDelayMs != nil {
			delay = def.startupDelayMs(b)
		}
		b.sched.Push(def.name, now.Add(time.Duration(delay)*time.Millisecond))
	}
	if b.sched.Init() == 0 {
		b.sched.Abandon(stop)
		return
	}

	util.Go(b.logDanger, "farmScheduler", func() { b.runFarmScheduler(stop, wake) })
}

func (b *Bot) runFarmScheduler(stop <-chan struct{}, wake <-chan struct{}) {
	for {
		if !b.sched.IsCurrent(stop) {
			return
		}
		wait, empty := b.sched.NextDue()

		// An empty heap means every command is in flight awaiting its OwO
		// response, not that the work is done — block until one reschedules.
		if empty {
			select {
			case <-stop:
				return
			case <-wake:
			}
			continue
		}

		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-stop:
				timer.Stop()
				return
			case <-wake:
				// Something due sooner may have been pushed; recompute.
				timer.Stop()
				continue
			case <-timer.C:
			}
		}

		select {
		case <-stop:
			return
		default:
		}

		item, ok := b.sched.PopDue(stop)
		if !ok {
			if !b.sched.IsCurrent(stop) {
				return
			}
			continue
		}

		if !b.canSend() {
			// Paused (e.g. captcha). Mark this run dead — but only if it is
			// still the current one — so reschedules don't push into a heap
			// nobody is draining.
			b.sched.StopRun(stop)
			return
		}

		def := b.farmCmdByName(item.name)
		if def == nil || !def.enabled(b) {
			continue
		}

		if def.log != "" {
			b.log.Info(def.log)
		}
		b.enqueue(def.channel(b), def.text(b))

		if def.delayMs(b) > 0 {
			b.markFarmAwaiting(def.name)
		}
	}
}
