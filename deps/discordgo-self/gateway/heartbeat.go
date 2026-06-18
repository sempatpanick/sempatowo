package gateway

import (
	"sync"
	"sync/atomic"
	"time"
)

// Heartbeat manages gateway heartbeating.
type Heartbeat struct {
	interval time.Duration
	sequence *int64

	lastSend    time.Time
	lastAck     time.Time
	latency     time.Duration
	ackReceived atomic.Bool

	sendFunc func(payload *PayloadWrapper) error

	stopCh  chan struct{}
	stopped atomic.Bool
	mu      sync.RWMutex
}

// NewHeartbeat creates a new heartbeat manager.
func NewHeartbeat(interval time.Duration, sequence *int64, sendFunc func(*PayloadWrapper) error) *Heartbeat {
	return &Heartbeat{
		interval: interval,
		sequence: sequence,
		sendFunc: sendFunc,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the heartbeat loop.
func (h *Heartbeat) Start() {
	go h.loop()
}

// Stop stops the heartbeat loop.
func (h *Heartbeat) Stop() {
	if h.stopped.CompareAndSwap(false, true) {
		close(h.stopCh)
	}
}

// Ack acknowledges a heartbeat.
func (h *Heartbeat) Ack() {
	h.mu.Lock()
	h.lastAck = time.Now()
	h.latency = h.lastAck.Sub(h.lastSend)
	h.ackReceived.Store(true)
	h.mu.Unlock()
}

// Latency returns the current latency.
func (h *Heartbeat) Latency() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.latency
}

// LastAck returns the time of the last ack.
func (h *Heartbeat) LastAck() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastAck
}

// IsAcked returns whether the last heartbeat was acknowledged.
func (h *Heartbeat) IsAcked() bool {
	return h.ackReceived.Load()
}

// loop runs the heartbeat loop.
func (h *Heartbeat) loop() {
	h.send()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			if !h.ackReceived.Load() {
				return
			}

			h.send()
		}
	}
}

// send sends a heartbeat.
func (h *Heartbeat) send() {
	h.mu.Lock()
	h.lastSend = time.Now()
	h.ackReceived.Store(false)
	h.mu.Unlock()

	var seq *int64
	if h.sequence != nil && *h.sequence > 0 {
		seq = h.sequence
	}

	payload := BuildHeartbeatPayload(seq)
	h.sendFunc(payload)
}

// UpdateInterval updates the heartbeat interval.
func (h *Heartbeat) UpdateInterval(interval time.Duration) {
	h.mu.Lock()
	h.interval = interval
	h.mu.Unlock()
}
