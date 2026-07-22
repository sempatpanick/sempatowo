package farm

import (
	"sync"
	"time"
)

// captchaTimers holds the countdown warnings armed while a captcha is
// outstanding, so they can all be cancelled the moment it is solved.
//
// Its mutex is a leaf; the timer callbacks themselves run on their own
// goroutines and never hold it.
type captchaTimers struct {
	mu     sync.Mutex
	timers []*time.Timer
}

// Add arms a timer and takes ownership of cancelling it.
func (c *captchaTimers) Add(t *time.Timer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timers = append(c.timers, t)
}

// Clear stops every armed timer.
func (c *captchaTimers) Clear() {
	c.mu.Lock()
	timers := c.timers
	c.timers = nil
	c.mu.Unlock()

	for _, t := range timers {
		t.Stop()
	}
}
