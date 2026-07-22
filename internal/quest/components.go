package quest

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

const (
	typeButton       = 2
	typeSection      = 9
	typeTextDisplay  = 10
	typeMediaGallery = 12
)

// RawMessage is a Discord gateway message payload (components v2 aware).
type RawMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
	Author    struct {
		ID string `json:"id"`
	} `json:"author"`
	Content    string          `json:"content"`
	Components json.RawMessage `json:"components"`
}

type componentNode struct {
	Type       int               `json:"type"`
	CustomID   string            `json:"custom_id"`
	Disabled   bool              `json:"disabled"`
	Content    string            `json:"content"`
	Components []json.RawMessage `json:"components"`
	Accessory  json.RawMessage   `json:"accessory"`
	Items      []struct {
		Media struct {
			URL string `json:"url"`
		} `json:"media"`
	} `json:"items"`
}

type QuestUI struct {
	ClaimCustomID      string
	QuestImageURL      string
	AllDone            bool
	NextQuestTimestamp int64
	ValidQuestLog      bool
}

var tsRe = regexp.MustCompile(`<t:(\d+):f>`)

func ParseQuestUI(raw json.RawMessage, userID string) QuestUI {
	var ui QuestUI
	var comps []json.RawMessage
	if err := json.Unmarshal(raw, &comps); err != nil {
		return ui
	}
	walkComponents(comps, userID, &ui)
	return ui
}

func walkComponents(comps []json.RawMessage, userID string, ui *QuestUI) {
	for _, raw := range comps {
		var n componentNode
		if err := json.Unmarshal(raw, &n); err != nil {
			continue
		}
		switch n.Type {
		case typeSection:
			for _, sub := range n.Components {
				var inner componentNode
				if err := json.Unmarshal(sub, &inner); err != nil {
					continue
				}
				if inner.Type == typeTextDisplay && inner.Content != "" {
					if isUserQuestLog(inner.Content, userID) {
						ui.ValidQuestLog = true
					}
				}
			}
			if len(n.Accessory) > 0 {
				walkComponents([]json.RawMessage{n.Accessory}, userID, ui)
			}
		case typeTextDisplay:
			if n.Content != "" {
				if isUserQuestLog(n.Content, userID) {
					ui.ValidQuestLog = true
				}
				lower := n.Content
				if containsAllDone(lower) {
					ui.AllDone = true
				}
				if m := tsRe.FindStringSubmatch(lower); len(m) == 2 {
					ui.NextQuestTimestamp, _ = strconv.ParseInt(m[1], 10, 64)
				}
			}
		case typeButton:
			if n.CustomID == "quests:claim" && !n.Disabled {
				ui.ClaimCustomID = n.CustomID
			}
		case typeMediaGallery:
			for _, item := range n.Items {
				if item.Media.URL != "" && containsQuestRows(item.Media.URL) {
					ui.QuestImageURL = item.Media.URL
				}
			}
		}
		if len(n.Components) > 0 {
			walkComponents(n.Components, userID, ui)
		}
		if len(n.Accessory) > 0 {
			walkComponents([]json.RawMessage{n.Accessory}, userID, ui)
		}
	}
}

func isUserQuestLog(s, userID string) bool {
	if userID != "" && strings.Contains(s, "<@"+userID+">") && strings.Contains(s, "Quest Log") {
		return true
	}
	return strings.Contains(s, "'s Quest Log") || strings.Contains(s, "'s quest log")
}

func containsAllDone(s string) bool {
	return strings.Contains(s, "UwU You finished all of your quests!")
}

func containsQuestRows(url string) bool {
	return strings.Contains(url, "quest-rows")
}
