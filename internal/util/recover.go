package util

import (
	"fmt"
	"runtime/debug"
)

// Recover keeps a panic from taking down the process and reports it with a
// stack trace instead.
//
// This matters more here than in a typical program: main runs every account in
// one process (one goroutine per token), and the Discord library invokes
// message callbacks on its own goroutines. An unrecovered panic while parsing
// one OwO message would otherwise kill every other account with it.
//
// Use it as the first statement of a goroutine or library callback:
//
//	defer util.Recover(b.logDanger, "queue")
//
// logf may be nil, in which case the panic is printed to stdout.
func Recover(logf func(string), name string) {
	r := recover()
	if r == nil {
		return
	}
	msg := fmt.Sprintf("panic in %s: %v\n%s", name, r, debug.Stack())
	if logf != nil {
		logf(msg)
		return
	}
	fmt.Println(msg)
}

// Go runs fn in a new goroutine guarded by Recover. See Recover for why.
func Go(logf func(string), name string, fn func()) {
	go func() {
		defer Recover(logf, name)
		fn()
	}()
}
