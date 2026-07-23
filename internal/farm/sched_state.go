package farm

import (
	"container/heap"
	"sort"
	"sync"
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

// farmSchedState is the scheduler's own state: which commands are due when,
// which ones are in flight waiting for OwO to answer, and the channels that
// identify the current run.
//
// Lock discipline: schedMu is a leaf, held only for the short bookkeeping below
// and never across a call into Bot. The bot's lock is taken around these
// methods in places (teardown, the captcha pause), so the reverse direction
// must stay impossible.
type farmSchedState struct {
	mu   sync.Mutex
	heap cmdHeap
	seq  uint64

	// stop identifies the current run. A goroutine that finds a different stop
	// than the one it was started with knows it has been superseded.
	stop chan struct{}
	// wake nudges the run loop when something is pushed ahead of the head.
	wake chan struct{}

	// awaiting holds commands that were sent and whose reply has not arrived.
	// While a command sits here it is absent from the heap, which is why an
	// empty heap does not mean the scheduler is finished.
	awaiting map[string]struct{}

	// suspended is the schedule saved by Suspend, waiting for the run that
	// resumes it. Stop clears it: only a pause is meant to be resumable.
	suspended []scheduledCmd
}

// Begin starts a new run, superseding any existing one, and returns its
// channels. The caller pushes the initial commands and then calls Launch.
func (s *farmSchedState) Begin() (stop chan struct{}, wake chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopLocked()
	s.stop = make(chan struct{})
	s.wake = make(chan struct{}, 1)
	s.seq = 0
	s.heap = nil
	return s.stop, s.wake
}

// Abandon rolls back a Begin that turned out to have nothing to schedule.
func (s *farmSchedState) Abandon(stop chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != stop {
		return
	}
	s.stop = nil
	s.wake = nil
	s.heap = nil
}

// Stop ends the current run and clears everything.
func (s *farmSchedState) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *farmSchedState) stopLocked() {
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
	s.wake = nil
	s.heap = nil
	s.awaiting = nil
	s.suspended = nil
}

// Suspend ends the current run but keeps its schedule, so a pause — a captcha —
// can resume the cycle where it left off instead of restarting every command
// from its startup delay. It reports how many commands were saved.
//
// Commands still awaiting a reply are saved too, dated now: while paused the
// handlers ignore OwO, so their reply can never arrive and the resumed run must
// send them again rather than wait forever. Without them they would be lost
// entirely — an awaiting command is absent from the heap.
func (s *farmSchedState) Suspend() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop == nil {
		return 0
	}

	now := time.Now()
	pending := make([]string, 0, len(s.awaiting))
	for name := range s.awaiting {
		pending = append(pending, name)
	}
	// Map iteration is random; sort so a resume is reproducible.
	sort.Strings(pending)

	saved := make([]scheduledCmd, 0, len(s.heap)+len(pending))
	for _, name := range pending {
		saved = append(saved, scheduledCmd{name: name, nextRun: now})
	}
	for _, item := range s.heap {
		saved = append(saved, *item)
	}
	sort.SliceStable(saved, func(i, j int) bool {
		return saved[i].nextRun.Before(saved[j].nextRun)
	})

	s.stopLocked()
	s.suspended = saved
	return len(saved)
}

// TakeSuspended hands over the schedule saved by Suspend, clearing it so a
// second run cannot resume the same state twice.
func (s *farmSchedState) TakeSuspended() []scheduledCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved := s.suspended
	s.suspended = nil
	return saved
}

// StopRun ends the run identified by stop, and reports whether it was still the
// current one. Used by the run loop itself, which must not tear down a
// successor that has already replaced it.
func (s *farmSchedState) StopRun(stop <-chan struct{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != stop {
		return false
	}
	s.stopLocked()
	return true
}

// Running reports whether any run is active.
func (s *farmSchedState) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stop != nil
}

// IsCurrent reports whether stop identifies the active run.
func (s *farmSchedState) IsCurrent(stop <-chan struct{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stop == stop
}

// Push schedules a command, waking the run loop so something due sooner than
// the current head is not stuck behind the head's timer.
func (s *farmSchedState) Push(name string, nextRun time.Time) {
	s.mu.Lock()
	s.seq++
	heap.Push(&s.heap, &scheduledCmd{name: name, nextRun: nextRun, seq: s.seq})
	wake := s.wake
	s.mu.Unlock()

	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

// Init restores the heap invariant after a batch of pushes, and reports how
// many commands are queued.
func (s *farmSchedState) Init() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	heap.Init(&s.heap)
	return len(s.heap)
}

// NextDue reports how long until the head is due. empty is true when nothing is
// scheduled, which means every command is in flight rather than that the work
// is done.
func (s *farmSchedState) NextDue() (wait time.Duration, empty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.heap) == 0 {
		return 0, true
	}
	return time.Until(s.heap[0].nextRun), false
}

// PopDue takes the head off the heap, if the run is still current and the heap
// is not empty.
func (s *farmSchedState) PopDue(stop <-chan struct{}) (*scheduledCmd, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != stop || len(s.heap) == 0 {
		return nil, false
	}
	return heap.Pop(&s.heap).(*scheduledCmd), true
}

// MarkAwaiting records that name was sent and its reply is outstanding.
func (s *farmSchedState) MarkAwaiting(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.awaiting == nil {
		s.awaiting = make(map[string]struct{})
	}
	s.awaiting[name] = struct{}{}
}

// ClaimAwaiting removes name from the outstanding set and reports whether it
// was there. Only the claimant reschedules, so a reply and its timeout racing
// cannot schedule the same command twice.
func (s *farmSchedState) ClaimAwaiting(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.awaiting[name]; !ok {
		return false
	}
	delete(s.awaiting, name)
	return true
}

// IsAwaiting reports whether name is outstanding.
func (s *farmSchedState) IsAwaiting(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.awaiting[name]
	return ok
}
