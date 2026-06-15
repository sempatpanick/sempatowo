//go:build windows

package notify

import toast "git.sr.ht/~jackmordaunt/go-toast"

func notifyPlatform(title, body, url string) {
	n := toast.Notification{
		AppID: "sempatowo",
		Title: title,
		Body:  body,
		Audio: toast.Reminder,
	}
	if url != "" {
		n.ActivationArguments = url
	}
	_ = n.Push()
}
