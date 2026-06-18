package gateway

import (
	"encoding/json"
)

// Gateway opcodes.
const (
	OpcodeDispatch            = 0
	OpcodeHeartbeat           = 1
	OpcodeIdentify            = 2
	OpcodePresenceUpdate      = 3
	OpcodeVoiceStateUpdate    = 4
	OpcodeVoicePing           = 5
	OpcodeResume              = 6
	OpcodeReconnect           = 7
	OpcodeRequestGuildMembers = 8
	OpcodeInvalidSession      = 9
	OpcodeHello               = 10
	OpcodeHeartbeatAck        = 11
	OpcodeCallConnect         = 13
	OpcodeGuildSubscribe      = 14
	OpcodeSearchRecentMembers = 35
	OpcodeBulkGuildSubscribe  = 37
)

// Gateway close codes.
const (
	CloseNormal               = 1000
	CloseGoingAway            = 1001
	CloseAbnormal             = 1006
	CloseUnknownError         = 4000
	CloseUnknownOpcode        = 4001
	CloseDecodeError          = 4002
	CloseNotAuthenticated     = 4003
	CloseAuthenticationFailed = 4004
	CloseAlreadyAuthenticated = 4005
	CloseInvalidSequence      = 4007
	CloseRateLimited          = 4008
	CloseSessionTimeout       = 4009
	CloseInvalidShard         = 4010
	CloseShardingRequired     = 4011
	CloseInvalidAPIVersion    = 4012
	CloseInvalidIntents       = 4013
	CloseDisallowedIntents    = 4014
)

// Gateway events.
const (
	EventReady   = "READY"
	EventResumed = "RESUMED"

	EventMessageCreate              = "MESSAGE_CREATE"
	EventMessageUpdate              = "MESSAGE_UPDATE"
	EventMessageDelete              = "MESSAGE_DELETE"
	EventMessageDeleteBulk          = "MESSAGE_DELETE_BULK"
	EventMessageReactionAdd         = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove      = "MESSAGE_REACTION_REMOVE"
	EventMessageReactionRemoveAll   = "MESSAGE_REACTION_REMOVE_ALL"
	EventMessageReactionRemoveEmoji = "MESSAGE_REACTION_REMOVE_EMOJI"
	EventMessageAck                 = "MESSAGE_ACK"

	EventChannelCreate          = "CHANNEL_CREATE"
	EventChannelUpdate          = "CHANNEL_UPDATE"
	EventChannelDelete          = "CHANNEL_DELETE"
	EventChannelPinsUpdate      = "CHANNEL_PINS_UPDATE"
	EventChannelRecipientAdd    = "CHANNEL_RECIPIENT_ADD"
	EventChannelRecipientRemove = "CHANNEL_RECIPIENT_REMOVE"

	EventTypingStart = "TYPING_START"

	EventUserUpdate              = "USER_UPDATE"
	EventUserSettingsUpdate      = "USER_SETTINGS_UPDATE"
	EventUserGuildSettingsUpdate = "USER_GUILD_SETTINGS_UPDATE"
	EventUserNoteUpdate          = "USER_NOTE_UPDATE"

	EventPresenceUpdate   = "PRESENCE_UPDATE"
	EventPresencesReplace = "PRESENCES_REPLACE"
	EventSessionsReplace  = "SESSIONS_REPLACE"

	EventRelationshipAdd    = "RELATIONSHIP_ADD"
	EventRelationshipRemove = "RELATIONSHIP_REMOVE"

	EventGuildCreate             = "GUILD_CREATE"
	EventGuildUpdate             = "GUILD_UPDATE"
	EventGuildDelete             = "GUILD_DELETE"
	EventGuildMemberAdd          = "GUILD_MEMBER_ADD"
	EventGuildMemberUpdate       = "GUILD_MEMBER_UPDATE"
	EventGuildMemberRemove       = "GUILD_MEMBER_REMOVE"
	EventGuildMembersChunk       = "GUILD_MEMBERS_CHUNK"
	EventGuildRoleCreate         = "GUILD_ROLE_CREATE"
	EventGuildRoleUpdate         = "GUILD_ROLE_UPDATE"
	EventGuildRoleDelete         = "GUILD_ROLE_DELETE"
	EventGuildBanAdd             = "GUILD_BAN_ADD"
	EventGuildBanRemove          = "GUILD_BAN_REMOVE"
	EventGuildEmojisUpdate       = "GUILD_EMOJIS_UPDATE"
	EventGuildIntegrationsUpdate = "GUILD_INTEGRATIONS_UPDATE"

	EventInteractionCreate  = "INTERACTION_CREATE"
	EventInteractionSuccess = "INTERACTION_SUCCESS"
	EventInteractionFailure = "INTERACTION_FAILURE"

	EventVoiceStateUpdate  = "VOICE_STATE_UPDATE"
	EventVoiceServerUpdate = "VOICE_SERVER_UPDATE"

	EventCallCreate = "CALL_CREATE"
	EventCallUpdate = "CALL_UPDATE"
	EventCallDelete = "CALL_DELETE"

	EventReadStateUpdate = "READ_STATE_UPDATE"

	EventApplicationCommandPermissionsUpdate = "APPLICATION_COMMAND_PERMISSIONS_UPDATE"
)

// PayloadWrapper wraps gateway payloads.
type PayloadWrapper struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  *int64          `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

// HelloPayload represents the HELLO event data.
type HelloPayload struct {
	HeartbeatInterval int      `json:"heartbeat_interval"`
	Trace             []string `json:"_trace,omitempty"`
}

// HeartbeatPayload represents a heartbeat payload.
type HeartbeatPayload struct {
	Op int    `json:"op"`
	D  *int64 `json:"d"`
}

// PresenceUpdatePayload represents presence update send data.
type PresenceUpdatePayload struct {
	Status     string        `json:"status"`
	Since      int64         `json:"since"`
	Activities []interface{} `json:"activities"`
	AFK        bool          `json:"afk"`
}
