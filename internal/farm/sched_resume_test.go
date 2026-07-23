package farm

import (
	"testing"
	"time"
)

// schedule reads the current heap as name -> due time.
func schedule(s *farmSchedState) map[string]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]time.Time, len(s.heap))
	for _, item := range s.heap {
		out[item.name] = item.nextRun
	}
	return out
}

func TestSuspendCapturesHeapAndAwaiting(t *testing.T) {
	var s farmSchedState
	s.Begin()
	due := time.Now().Add(20 * time.Minute)
	s.Push("inventory", due)
	s.Init()
	// Awaiting commands are absent from the heap, so a snapshot that ignored
	// them would drop those commands from the resumed cycle entirely.
	s.MarkAwaiting("hunt")

	if n := s.Suspend(); n != 2 {
		t.Fatalf("Suspend saved %d commands, want 2 (heap + awaiting)", n)
	}

	saved := s.TakeSuspended()
	if len(saved) != 2 {
		t.Fatalf("TakeSuspended returned %d commands, want 2", len(saved))
	}
	if saved[0].name != "hunt" {
		t.Fatalf("first resumed command is %q, want the awaiting hunt", saved[0].name)
	}
	if saved[1].name != "inventory" || !saved[1].nextRun.Equal(due) {
		t.Fatalf("inventory came back as %+v, want its original due time %v", saved[1], due)
	}
	if rest := s.TakeSuspended(); rest != nil {
		t.Fatalf("TakeSuspended handed out the same snapshot twice: %+v", rest)
	}
}

func TestStopDiscardsSuspended(t *testing.T) {
	var s farmSchedState
	s.Begin()
	s.Push("hunt", time.Now())
	s.Init()
	s.Suspend()

	// Only a pause is resumable; a teardown starts the next session clean.
	s.Stop()
	if saved := s.TakeSuspended(); saved != nil {
		t.Fatalf("Stop kept a resumable schedule: %+v", saved)
	}
}

func TestResumeKeepsUnexpiredDelays(t *testing.T) {
	b := New("token", nil)
	b.active = true
	b.ready = true

	now := time.Now()
	inventoryDue := now.Add(4 * time.Minute)
	b.sched.Begin()
	b.pushResumedCommands([]scheduledCmd{
		{name: "hunt", nextRun: now.Add(-time.Second)},
		{name: "inventory", nextRun: inventoryDue},
	}, now)
	b.sched.Init()

	got := schedule(&b.sched)
	if due, ok := got["inventory"]; !ok || !due.Equal(inventoryDue) {
		t.Fatalf("inventory scheduled for %v (present=%v), want its pre-pause %v", due, ok, inventoryDue)
	}
	if due, ok := got["hunt"]; !ok || !due.Equal(now) {
		t.Fatalf("overdue hunt scheduled for %v (present=%v), want %v", due, ok, now)
	}
}

// Commands can go missing from the snapshot: the run loop may have popped one
// just as the pause landed, or config may have enabled it while paused. Either
// way the resumed cycle has to be complete.
func TestResumeFillsInMissingCommands(t *testing.T) {
	b := New("token", nil)
	b.active = true
	b.ready = true

	now := time.Now()
	b.sched.Begin()
	b.pushResumedCommands([]scheduledCmd{{name: "hunt", nextRun: now}}, now)
	b.sched.Init()

	got := schedule(&b.sched)
	// Enabled by default alongside hunt.
	for _, name := range []string{"battle", "pray", "inventory"} {
		due, ok := got[name]
		if !ok {
			t.Fatalf("%s was dropped from the resumed cycle", name)
		}
		if due.Before(now) {
			t.Fatalf("%s scheduled in the past (%v)", name, due)
		}
	}

	// Everything due at once is the burst OwO's captcha watches for, so the
	// filled-in commands are spread instead.
	if got["battle"].Equal(got["pray"]) {
		t.Fatalf("battle and pray both scheduled for %v, want them staggered", got["battle"])
	}
}

func TestResumeDropsCommandsDisabledWhilePaused(t *testing.T) {
	b := New("token", nil)
	b.active = true
	b.ready = true

	now := time.Now()
	b.sched.Begin()
	// zoo is off in the defaults this bot falls back to.
	b.pushResumedCommands([]scheduledCmd{{name: "zoo", nextRun: now}}, now)
	b.sched.Init()

	if due, ok := schedule(&b.sched)["zoo"]; ok {
		t.Fatalf("resumed a disabled zoo for %v", due)
	}
}
