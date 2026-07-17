package farm

import (
	"testing"
	"time"
)

func TestSleepUntilReportsElapsed(t *testing.T) {
	c := &huntbotCtx{bot: &Bot{}}

	start := time.Now()
	if !c.SleepUntil(0.2, 0) {
		t.Fatal("SleepUntil = false, want true when the timer runs out")
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("returned after %v, want at least 200ms", elapsed)
	}
	if c.bot.sleep != nil {
		t.Fatal("sleep handle should be cleared once the wait finishes")
	}
}

func TestSleepUntilReportsCancel(t *testing.T) {
	c := &huntbotCtx{bot: &Bot{}}

	go func() {
		time.Sleep(50 * time.Millisecond)
		c.CancelSleep()
	}()

	start := time.Now()
	if c.SleepUntil(10, 0) {
		t.Fatal("SleepUntil = true, want false when cancelled mid-wait")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancel took %v to take effect, want prompt return", elapsed)
	}
}

// A second sleeper displaces the first, and the displaced one must learn it was
// cancelled rather than treating it as an elapsed timer.
func TestSecondSleeperCancelsFirst(t *testing.T) {
	b := &Bot{}
	first := &huntbotCtx{bot: b}
	second := &huntbotCtx{bot: b}

	result := make(chan bool, 1)
	go func() { result <- first.SleepUntil(10, 0) }()
	time.Sleep(50 * time.Millisecond)

	go func() { second.SleepUntil(10, 0) }()

	select {
	case elapsed := <-result:
		if elapsed {
			t.Fatal("displaced sleeper reported an elapsed timer, want cancelled")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("displaced sleeper never returned")
	}

	b.CancelSleep()
}
