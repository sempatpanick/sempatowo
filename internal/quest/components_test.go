package quest

import (
	"encoding/json"
	"testing"
)

// A realistic quest log: a section naming the user, the quest image, and an
// enabled claim button.
const questLogComponents = `[
  {"type": 17, "components": [
    {"type": 9,
     "components": [{"type": 10, "content": "<@229948970904846336>'s Quest Log"}],
     "accessory": {"type": 2, "custom_id": "quests:claim", "disabled": false}},
    {"type": 12, "items": [
      {"media": {"url": "https://cdn.discordapp.com/quest-rows-123.png"}}
    ]},
    {"type": 10, "content": "Next quest <t:1753142400:f>"}
  ]}
]`

func TestParseQuestUI(t *testing.T) {
	ui := ParseQuestUI(json.RawMessage(questLogComponents), "229948970904846336")

	if !ui.ValidQuestLog {
		t.Error("ValidQuestLog = false for the user's own quest log")
	}
	if ui.ClaimCustomID != "quests:claim" {
		t.Errorf("ClaimCustomID = %q, want quests:claim", ui.ClaimCustomID)
	}
	if ui.QuestImageURL == "" {
		t.Error("QuestImageURL not found in the media gallery")
	}
	if ui.NextQuestTimestamp != 1753142400 {
		t.Errorf("NextQuestTimestamp = %d, want 1753142400", ui.NextQuestTimestamp)
	}
	if ui.AllDone {
		t.Error("AllDone = true without the completion message")
	}
}

// A disabled claim button means the reward is not available; clicking it would
// be a wasted interaction at best.
func TestParseQuestUISkipsDisabledClaim(t *testing.T) {
	raw := `[{"type": 2, "custom_id": "quests:claim", "disabled": true}]`

	if got := ParseQuestUI(json.RawMessage(raw), "").ClaimCustomID; got != "" {
		t.Errorf("ClaimCustomID = %q, want empty for a disabled button", got)
	}
}

func TestParseQuestUIDetectsAllDone(t *testing.T) {
	raw := `[{"type": 10, "content": "UwU You finished all of your quests! Next <t:1753142400:f>"}]`

	ui := ParseQuestUI(json.RawMessage(raw), "")
	if !ui.AllDone {
		t.Error("AllDone = false for the completion message")
	}
	if ui.NextQuestTimestamp != 1753142400 {
		t.Errorf("NextQuestTimestamp = %d, want the parsed timestamp", ui.NextQuestTimestamp)
	}
}

// Malformed component payloads arrive from a bot we do not control, so they
// must yield an empty result rather than a panic.
func TestParseQuestUIToleratesGarbage(t *testing.T) {
	for _, raw := range []string{``, `{}`, `[`, `[null]`, `["not an object"]`, `[{"type": 10}]`} {
		ui := ParseQuestUI(json.RawMessage(raw), "1")
		if ui.ValidQuestLog || ui.ClaimCustomID != "" || ui.QuestImageURL != "" {
			t.Errorf("%q produced a non-empty result: %+v", raw, ui)
		}
	}
}

func TestIsUserQuestLog(t *testing.T) {
	const uid = "229948970904846336"

	if !isUserQuestLog("<@"+uid+"> Quest Log", uid) {
		t.Error("mention form not recognised")
	}
	if !isUserQuestLog("someone's Quest Log", "") {
		t.Error("possessive form not recognised")
	}
	// A mention of somebody else must not be claimed as ours.
	if isUserQuestLog("<@111111111111111111> Quest Log", uid) {
		t.Error("another user's quest log was accepted")
	}
	if isUserQuestLog("Quest Log", uid) {
		t.Error("a bare heading with no owner was accepted")
	}
}

func TestContainsQuestRows(t *testing.T) {
	if !containsQuestRows("https://cdn.discordapp.com/x/quest-rows-1.png") {
		t.Error("quest image URL not recognised")
	}
	if containsQuestRows("https://cdn.discordapp.com/x/checklist.png") {
		t.Error("an unrelated image was treated as the quest image")
	}
}
