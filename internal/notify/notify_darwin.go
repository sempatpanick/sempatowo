//go:build darwin

package notify

import "github.com/gen2brain/beeep"

func notifyPlatform(title, body, url string) {
	_ = beeep.Alert(title, body, "")
	_ = url // click-to-open not supported on macOS without terminal-notifier extras
}
