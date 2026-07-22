package notify

import (
	"fmt"
)

const captchaTitle = "OwO Captcha — ~10 min to solve"

// CaptchaUrgent shows a desktop notification with sound (Windows/macOS/Linux).
// Whether notifications are on is decided at startup by config.Env, not read
// from the environment here.
func CaptchaUrgent(account, message, url string, enabled bool) {
	if !enabled {
		return
	}
	if account == "" {
		account = "account"
	}
	body := fmt.Sprintf("[%s] %s", account, message)
	go notifyPlatform(captchaTitle, body, url)
}
