package quest

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
)

//go:embed emojis.json
var emojisJSON []byte

var emojiRank map[string]string

var emojiPattern = regexp.MustCompile(`<a:[a-zA-Z0-9_]+:[0-9]+>|:[a-zA-Z0-9_]+:`)

func init() {
	emojiRank = make(map[string]string)
	var raw map[string]struct {
		Rank string `json:"rank"`
	}
	if err := json.Unmarshal(emojisJSON, &raw); err != nil {
		return
	}
	for k, v := range raw {
		emojiRank[k] = v.Rank
	}
}

func HasRankedEmoji(content, rank string) bool {
	if emojiRank == nil {
		return false
	}
	for _, em := range emojiPattern.FindAllString(content, -1) {
		if emojiRank[em] == rank {
			return true
		}
	}
	// Also match :name: style keys in map
	lower := strings.ToLower(content)
	if strings.Contains(lower, rank) {
		return true
	}
	return false
}
