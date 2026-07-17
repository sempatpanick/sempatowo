package farm

import (
	"testing"
	"time"
)

func startTestScheduler(b *Bot, cmds map[string]time.Time) (stop chan struct{}) {
	b.mu.Lock()
	stop = make(chan struct{})
	b.cmdSchedStop = stop
	b.cmdWake = make(chan struct{}, 1)
	wake := b.cmdWake
	for name, at := range cmds {
		b.pushScheduledCmd(name, at)
	}
	b.mu.Unlock()
	go b.runFarmScheduler(stop, wake)
	return stop
}

// waitDispatched waits for the scheduler to run name, observed via the
// farmAwaiting marker it sets on every dispatch.
func waitDispatched(t *testing.T, b *Bot, name string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b.mu.Lock()
		_, ok := b.farmAwaiting[name]
		b.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s was never dispatched within %v", name, timeout)
}

// Regression for the 5-minute farm pauses: the scheduler used to sleep on the
// current heap head's timer, so a command pushed for an earlier time (a hunt
// rescheduled for +15s while inventory was the head at +5min) sat overdue
// until the head's timer fired.
func TestSchedulerWakesForEarlierPush(t *testing.T) {
	b := New("token")
	b.ready = true

	stop := startTestScheduler(b, map[string]time.Time{
		"inventory": time.Now().Add(5 * time.Minute),
	})
	defer close(stop)

	time.Sleep(50 * time.Millisecond) // let the scheduler block on the 5min timer

	b.mu.Lock()
	b.pushScheduledCmd("hunt", time.Now().Add(50*time.Millisecond))
	b.mu.Unlock()

	waitDispatched(t, b, "hunt", 2*time.Second)
}

// Regression for the permanent stop: with every command in flight the heap is
// momentarily empty, and the scheduler used to treat that as "done" and exit
// for good, silently discarding all later reschedules.
func TestSchedulerSurvivesDrainedHeap(t *testing.T) {
	b := New("token")
	b.ready = true

	stop := startTestScheduler(b, map[string]time.Time{
		"hunt": time.Now(),
	})
	defer func() {
		b.mu.Lock()
		if b.cmdSchedStop == stop {
			b.stopFarmSchedulerLocked()
		}
		b.mu.Unlock()
	}()

	waitDispatched(t, b, "hunt", 2*time.Second)
	time.Sleep(100 * time.Millisecond) // heap now sits empty, hunt in flight

	b.mu.Lock()
	alive := b.cmdSchedStop == stop
	delete(b.farmAwaiting, "hunt") // reset the marker for the second dispatch
	b.mu.Unlock()
	if !alive {
		t.Fatal("scheduler declared itself done while hunt was still awaiting its response")
	}

	// The hunt response arrives and reschedules; the scheduler must pick it up.
	b.mu.Lock()
	b.pushScheduledCmd("hunt", time.Now().Add(50*time.Millisecond))
	b.mu.Unlock()

	waitDispatched(t, b, "hunt", 2*time.Second)
}
