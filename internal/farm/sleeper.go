package farm

import "sync"

// sleepHandle identifies one in-flight sleep so a sleeper only clears its own.
type sleepHandle struct {
	cancel func()
}

// sleeper tracks the single cancellable wait the huntbot upgrader uses between
// passes. Starting a new sleep cancels whichever one it displaces, and the
// displaced sleeper learns it was cancelled rather than treating it as elapsed.
//
// Its mutex is a leaf: cancel callbacks are always invoked after it is released.
type sleeper struct {
	mu     sync.Mutex
	active *sleepHandle
}

// Set makes h the active sleep, cancelling whichever one it replaces.
func (s *sleeper) Set(h *sleepHandle) {
	s.mu.Lock()
	prev := s.active
	s.active = h
	s.mu.Unlock()

	if prev != nil {
		prev.cancel()
	}
}

// Clear releases h if it is still the active sleep.
func (s *sleeper) Clear(h *sleepHandle) {
	s.mu.Lock()
	if s.active == h {
		s.active = nil
	}
	s.mu.Unlock()
}

// Cancel wakes the active sleep, if any.
func (s *sleeper) Cancel() {
	s.mu.Lock()
	h := s.active
	s.active = nil
	s.mu.Unlock()

	if h != nil {
		h.cancel()
	}
}
