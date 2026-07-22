package captcha

import (
	"fmt"
	"sync"
)

type queueEntry struct {
	accountID string
	done      chan struct{}
}

var (
	queueMu       sync.Mutex
	activeAccount string
	waitingQueue  []queueEntry
)

// BrowserQueueEnabled is true when browsers are not isolated per account, so
// every account has to take turns at the one shared system browser.
func BrowserQueueEnabled(isolated bool) bool {
	return !isolated
}

// AcquireBrowserSlot waits until this account may open the captcha browser.
func AcquireBrowserSlot(accountID string, isolated bool) {
	if !BrowserQueueEnabled(isolated) {
		return
	}

	queueMu.Lock()
	if activeAccount == "" || activeAccount == accountID {
		activeAccount = accountID
		queueMu.Unlock()
		return
	}

	entry := queueEntry{accountID: accountID, done: make(chan struct{})}
	waitingQueue = append(waitingQueue, entry)
	queueMu.Unlock()

	fmt.Printf("[browser-queue] %q waiting (active: %q, %d queued)\n", accountID, activeAccount, len(waitingQueue))
	<-entry.done
}

// ReleaseBrowserSlot frees the browser for the next waiting account.
func ReleaseBrowserSlot(accountID string, isolated bool) {
	if !BrowserQueueEnabled(isolated) {
		return
	}

	queueMu.Lock()
	defer queueMu.Unlock()

	for i, entry := range waitingQueue {
		if entry.accountID == accountID {
			close(entry.done)
			waitingQueue = append(waitingQueue[:i], waitingQueue[i+1:]...)
			fmt.Printf("[browser-queue] %q removed from wait queue\n", accountID)
			return
		}
	}

	if activeAccount != accountID {
		return
	}

	activeAccount = ""
	if len(waitingQueue) == 0 {
		fmt.Println("[browser-queue] slot released — queue empty")
		return
	}

	next := waitingQueue[0]
	waitingQueue = waitingQueue[1:]
	activeAccount = next.accountID
	fmt.Printf("[browser-queue] slot granted to %q\n", next.accountID)
	close(next.done)
}
