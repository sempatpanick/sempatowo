package gateway

import (
	"encoding/json"

	"github.com/hytams/discordgo-self/props"
)

// IdentifyPayload represents the IDENTIFY payload.
type IdentifyPayload struct {
	Token        string                 `json:"token"`
	Capabilities int                    `json:"capabilities"`
	Properties   map[string]interface{} `json:"properties"`
	Presence     PresencePayload        `json:"presence"`
	Compress     bool                   `json:"compress"`
	ClientState  ClientState            `json:"client_state"`
}

// PresencePayload represents the presence in identify.
type PresencePayload struct {
	Status     string        `json:"status"`
	Since      int64         `json:"since"`
	Activities []interface{} `json:"activities"`
	AFK        bool          `json:"afk"`
}

// ClientState represents the client state.
type ClientState struct {
	GuildVersions            map[string]int `json:"guild_versions"`
	HighestLastMessageID     string         `json:"highest_last_message_id,omitempty"`
	ReadStateVersion         int            `json:"read_state_version,omitempty"`
	UserGuildSettingsVersion int            `json:"user_guild_settings_version,omitempty"`
	UserSettingsVersion      int            `json:"user_settings_version,omitempty"`
	PrivateChannelsVersion   string         `json:"private_channels_version,omitempty"`
	APICodeVersion           int            `json:"api_code_version,omitempty"`
}

// ResumePayload represents the RESUME payload.
type ResumePayload struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// BuildIdentifyPayload builds an identify payload.
func BuildIdentifyPayload(token string, superProps *props.SuperProperties, capabilities int) *PayloadWrapper {
	data, _ := json.Marshal(IdentifyPayload{
		Token:        token,
		Capabilities: capabilities,
		Properties:   superProps.GatewayProperties(),
		Presence: PresencePayload{
			Status:     "online",
			Since:      0,
			Activities: []interface{}{},
			AFK:        false,
		},
		Compress: false,
		ClientState: ClientState{
			GuildVersions: map[string]int{},
		},
	})

	return &PayloadWrapper{
		Op: OpcodeIdentify,
		D:  data,
	}
}

// BuildResumePayload builds a resume payload.
func BuildResumePayload(token, sessionID string, sequence int64) *PayloadWrapper {
	data, _ := json.Marshal(ResumePayload{
		Token:     token,
		SessionID: sessionID,
		Sequence:  sequence,
	})

	return &PayloadWrapper{
		Op: OpcodeResume,
		D:  data,
	}
}

// BuildHeartbeatPayload builds a heartbeat payload.
func BuildHeartbeatPayload(sequence *int64) *PayloadWrapper {
	data, _ := json.Marshal(sequence)

	return &PayloadWrapper{
		Op: OpcodeHeartbeat,
		D:  data,
	}
}

// BuildPresenceUpdatePayload builds a presence update payload.
func BuildPresenceUpdatePayload(status string, activities []interface{}, afk bool, since int64) *PayloadWrapper {
	data, _ := json.Marshal(PresencePayload{
		Status:     status,
		Since:      since,
		Activities: activities,
		AFK:        afk,
	})

	return &PayloadWrapper{
		Op: OpcodePresenceUpdate,
		D:  data,
	}
}
