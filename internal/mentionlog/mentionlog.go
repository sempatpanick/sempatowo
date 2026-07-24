// Package mentionlog persists raw OwO messages addressed to an account to a
// per-account file for later inspection. The file is a directly-valid JSON
// array: one object per element, so it parses with `jq .` as-is.
package mentionlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Embed is a minimal projection of a Discord embed — the text fields OwO
// actually fills, so an addressed embed's content is not lost.
type Embed struct {
	Author      string `json:"author,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Footer      string `json:"footer,omitempty"`
}

// Record is one logged message. Content is the raw msg.Content, not
// zero-width-normalized, so the file mirrors exactly what OwO sent.
type Record struct {
	Time      time.Time `json:"time"`
	Event     string    `json:"event"` // "create" or "update"
	GuildID   string    `json:"guildId,omitempty"`
	ChannelID string    `json:"channelId,omitempty"`
	MessageID string    `json:"messageId,omitempty"`
	Content   string    `json:"content"`
	Embeds    []Embed   `json:"embeds,omitempty"`
}

// Sink appends records to var/data/{userID}_mentions.log.
type Sink struct {
	mu   sync.Mutex
	path string
}

func New(dataDir, userID string) *Sink {
	_ = os.MkdirAll(dataDir, 0o755)
	return &Sink{path: filepath.Join(dataDir, userID+"_mentions.log")}
}

// Append splices rec into the file's JSON array, right before the closing
// bracket, so the file stays a valid JSON array after every write. It works on
// brackets alone and never parses the existing elements, so a hand-edited or
// partially-written record can't corrupt later appends. Addressed messages are
// low volume, so rewriting the whole (small) file per call is fine.
func (s *Sink) Append(rec Record) error {
	if rec.Time.IsZero() {
		rec.Time = time.Now().UTC()
	}
	elem, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var out []byte
	trimmed := bytes.TrimRight(existing, " \t\r\n")
	if i := bytes.LastIndexByte(trimmed, ']'); i >= 0 {
		// Keep every existing element byte-for-byte; insert before the ']'.
		head := bytes.TrimRight(trimmed[:i], " \t\r\n")
		if bytes.HasSuffix(head, []byte("[")) {
			out = append(head, "\n  "...) // first element
		} else {
			out = append(head, ",\n  "...) // subsequent element
		}
	} else {
		// New or empty file: open a fresh array.
		out = []byte("[\n  ")
	}
	out = append(out, elem...)
	out = append(out, "\n]\n"...)

	return os.WriteFile(s.path, out, 0o644)
}
