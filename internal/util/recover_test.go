package util

import (
	"strings"
	"sync"
	"testing"
)

func TestRecoverCapturesPanic(t *testing.T) {
	var got string
	func() {
		defer Recover(func(s string) { got = s }, "worker")
		panic("boom")
	}()

	if got == "" {
		t.Fatal("expected panic to be reported")
	}
	if !strings.Contains(got, "panic in worker") {
		t.Errorf("message missing goroutine name: %q", got)
	}
	if !strings.Contains(got, "boom") {
		t.Errorf("message missing panic value: %q", got)
	}
	if !strings.Contains(got, "runtime/debug.Stack") && !strings.Contains(got, "goroutine") {
		t.Errorf("message missing stack trace: %q", got)
	}
}

func TestRecoverSilentWithoutPanic(t *testing.T) {
	called := false
	func() {
		defer Recover(func(string) { called = true }, "worker")
	}()
	if called {
		t.Error("logged despite no panic")
	}
}

func TestRecoverNilLogfDoesNotPanic(t *testing.T) {
	// A nil sink must fall back to stdout rather than panicking inside the
	// recovery path itself — that would defeat the whole purpose.
	func() {
		defer Recover(nil, "worker")
		panic("boom")
	}()
}

func TestGoRecoversAndKeepsProcessAlive(t *testing.T) {
	var (
		mu   sync.Mutex
		got  string
		done = make(chan struct{})
	)

	Go(func(s string) {
		mu.Lock()
		got = s
		mu.Unlock()
		close(done)
	}, "bot", func() {
		panic("goroutine boom")
	})

	<-done

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(got, "goroutine boom") {
		t.Errorf("panic not reported: %q", got)
	}
}
