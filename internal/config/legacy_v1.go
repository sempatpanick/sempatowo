package config

import (
	"encoding/json"
	"fmt"
)

// v1 → v2 moved six keys and changed nothing else, so this migration does not
// need a frozen copy of the v1 struct the way legacy.go needs one for the
// pre-1.0 shape. Everything that stayed put — label, trackBalance, and the whole
// features tree — is read by the current decoder; only the six movers are picked
// out of the raw bytes a second time and placed in their new homes.
type movedInV1 struct {
	Typing                *bool     `json:"typing"`
	Prefix                *string   `json:"prefix"`
	OwoBotID              *string   `json:"owoBotId"`
	DefaultChannel        *string   `json:"defaultChannel"`
	SendMessageInterval   *Duration `json:"sendMessageInterval"`
	StopWhenChecklistDone *bool     `json:"stopWhenChecklistDone"`
}

// migrateV1 converts a schemaVersion 1 file into current Settings.
//
// The fields are pointers so that "absent" and "present but false" stay
// distinguishable: a plain bool would make `"typing": false` indistinguishable
// from a file that never mentioned typing, and the user would silently get the
// default back.
func migrateV1(data []byte) (Settings, []string, error) {
	// The v1 top-level keys are not v2 keys, so the ordinary decode leaves them
	// at their defaults and reports nothing — which is exactly what we want here,
	// since the loop below is what fills them in.
	s := Defaults()
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, nil, fmt.Errorf("reading v1 config: %w", err)
	}

	var old movedInV1
	if err := json.Unmarshal(data, &old); err != nil {
		return Settings{}, nil, fmt.Errorf("reading v1 config: %w", err)
	}

	if old.Typing != nil {
		s.Typing = *old.Typing
	}
	if old.Prefix != nil {
		s.Prefix = *old.Prefix
	}
	if old.OwoBotID != nil {
		s.OwoBotID = *old.OwoBotID
	}
	if old.DefaultChannel != nil {
		s.DefaultChannel = *old.DefaultChannel
	}
	if old.SendMessageInterval != nil {
		s.SendMessageInterval = *old.SendMessageInterval
	}
	if old.StopWhenChecklistDone != nil {
		s.Features.Checklist.StopFarmingWhenDone = *old.StopWhenChecklistDone
	}

	s.Schema = SchemaFileName
	s.SchemaVersion = SchemaVersion

	notes := []string{
		"prefix, defaultChannel and owoBotId moved under \"discord\"; " +
			"typing and sendMessageInterval under \"humanize\"",
	}
	if old.StopWhenChecklistDone != nil && *old.StopWhenChecklistDone {
		notes = append(notes,
			"stopWhenChecklistDone became features.checklist.stopFarmingWhenDone")
	}
	return s, notes, nil
}
