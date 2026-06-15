package notify

import (
	"fmt"
	"os"
	"strings"
)

const captchaTitle = "OwO Captcha — ~10 min to solve"

// CaptchaUrgent shows a desktop notification with sound (Windows/macOS/Linux).
func CaptchaUrgent(account, message, url string) {
	if !enabled() {
		return
	}
	if account == "" {
		account = "account"
	}
	body := fmt.Sprintf("[%s] %s", account, message)
	go notifyPlatform(captchaTitle, body, url)
}

func enabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("NOTIFICATIONS")))
	return !(raw == "false" || raw == "0" || raw == "no" || raw == "off")
}
