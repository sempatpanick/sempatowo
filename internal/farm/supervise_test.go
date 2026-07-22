package farm

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSuperviseLoopReturnsAfterGrace(t *testing.T) {
	var connected atomic.Bool
	connected.Store(false)

	done := make(chan struct{})
	start := time.Now()
	go func() {
		superviseLoop(connected.Load, 5*time.Millisecond, 40*time.Millisecond, nil, nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("superviseLoop did not return after the grace window")
	}

	// It must wait out the grace window rather than bailing on the first
	// missed poll — the library needs that time to resume on its own.
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Errorf("returned after %s, want at least the 40ms grace window", elapsed)
	}
}

func TestSuperviseLoopStaysWhileConnected(t *testing.T) {
	var connected atomic.Bool
	connected.Store(true)

	done := make(chan struct{})
	go func() {
		superviseLoop(connected.Load, 5*time.Millisecond, 20*time.Millisecond, nil, nil)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("superviseLoop returned while the gateway was still connected")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestSuperviseLoopResetsAfterRecovery(t *testing.T) {
	// A blip shorter than the grace window must not trigger a full reconnect.
	var connected atomic.Bool
	connected.Store(true)

	var ups, downs atomic.Int32
	done := make(chan struct{})
	go func() {
		superviseLoop(connected.Load, 5*time.Millisecond, 60*time.Millisecond,
			func() { downs.Add(1) },
			func() { ups.Add(1) })
		close(done)
	}()

	connected.Store(false)
	time.Sleep(25 * time.Millisecond) // shorter than grace
	connected.Store(true)

	select {
	case <-done:
		t.Fatal("returned despite the gateway recovering inside the grace window")
	case <-time.After(150 * time.Millisecond):
	}

	if downs.Load() == 0 {
		t.Error("disconnect was never reported")
	}
	if ups.Load() == 0 {
		t.Error("recovery was never reported")
	}
}
