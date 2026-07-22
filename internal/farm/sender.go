package farm

import (
	"sync"
	"time"
)

const defaultSendInterval = 5 * time.Second

type queuedMsg struct {
	channel    string
	text       string
	waitGamble string // "coinflip", "slots", … — pause the queue until the result
}

// sender serialises every outgoing message onto one goroutine, spacing them by
// the configured interval and pausing after a bet until OwO answers.
//
// Lock discipline: sender.mu is a leaf. None of the callbacks below is ever
// invoked while it is held, and none of them reaches back into the sender. The
// bot's own lock may be taken around sender methods (teardown does exactly
// that), so the reverse direction has to stay impossible.
type sender struct {
	mu      sync.Mutex
	queue   []queuedMsg
	stop    chan struct{}
	running bool

	gambleMu   sync.Mutex
	gambleCh   chan struct{}
	gambleGame string

	// resultWait is how long to hold the queue open for a gamble result.
	resultWait time.Duration

	canSend  func() bool
	interval func() time.Duration
	deliver  func(channel, text string)
	spawn    func(name string, fn func())
}

func newSender(canSend func() bool, interval func() time.Duration, deliver func(channel, text string), spawn func(string, func())) *sender {
	return &sender{
		resultWait: gambleResultWait,
		canSend:    canSend,
		interval:   interval,
		deliver:    deliver,
		spawn:      spawn,
	}
}

// Enqueue adds a message to the outgoing queue, starting the runner if it is
// not already up.
func (s *sender) Enqueue(m queuedMsg) {
	if m.channel == "" || m.text == "" {
		return
	}
	if !s.canSend() {
		return
	}

	s.mu.Lock()
	s.queue = append(s.queue, m)
	// Only a queue that has been fully torn down restarts here. A runner that
	// exited because sending was paused leaves stop set, so messages accumulate
	// until whatever paused it explicitly stops the queue.
	start := s.stop == nil && !s.running
	if start {
		s.stop = make(chan struct{})
		s.running = true
	}
	stop := s.stop
	s.mu.Unlock()

	if start {
		s.spawn("queue", func() { s.run(stop) })
	}
}

// Send queues a plain message.
func (s *sender) Send(channel, text string) {
	s.Enqueue(queuedMsg{channel: channel, text: text})
}

// SendBet queues a gamble bet and holds the queue until its result arrives.
func (s *sender) SendBet(channel, text, game string) {
	if game == "" {
		return
	}
	s.Enqueue(queuedMsg{channel: channel, text: text, waitGamble: game})
}

// Pending reports how many messages are waiting.
func (s *sender) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

func (s *sender) run(stop <-chan struct{}) {
	interval := s.interval()
	if interval <= 0 {
		interval = defaultSendInterval
	}

	for {
		// Checked before taking the lock, because canSend reaches into the bot.
		paused := !s.canSend()

		s.mu.Lock()
		if paused || len(s.queue) == 0 {
			s.running = false
			if len(s.queue) == 0 {
				s.stop = nil
			}
			s.mu.Unlock()
			return
		}
		msg := s.queue[0]
		s.queue = s.queue[1:]
		s.mu.Unlock()

		s.deliver(msg.channel, msg.text)

		if msg.waitGamble != "" {
			s.waitGambleResult(msg.waitGamble, s.resultWait, stop)
		}

		timer := time.NewTimer(interval)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// waitGambleResult blocks until SignalGambleResult reports the named game, the
// timeout expires, or the queue is stopped.
func (s *sender) waitGambleResult(game string, timeout time.Duration, stop <-chan struct{}) {
	ch := make(chan struct{})
	s.gambleMu.Lock()
	s.gambleCh = ch
	s.gambleGame = game
	s.gambleMu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
	case <-stop:
	}

	s.gambleMu.Lock()
	if s.gambleCh == ch {
		s.gambleCh = nil
		s.gambleGame = ""
	}
	s.gambleMu.Unlock()
}

// SignalGambleResult releases a queue waiting on this game's result.
func (s *sender) SignalGambleResult(game string) {
	s.gambleMu.Lock()
	if s.gambleGame != game {
		s.gambleMu.Unlock()
		return
	}
	ch := s.gambleCh
	s.gambleCh = nil
	s.gambleGame = ""
	s.gambleMu.Unlock()

	if ch != nil {
		close(ch)
	}
}

// Stop tears the queue down. The next Enqueue starts a fresh runner.
func (s *sender) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
	s.running = false
}

// Clear discards anything still queued, for a session teardown where the
// pending commands are about to be meaningless.
func (s *sender) Clear() {
	s.mu.Lock()
	s.queue = nil
	s.mu.Unlock()
}
