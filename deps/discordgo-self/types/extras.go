package types

type Invite struct {
	Code                     string          `json:"code"`
	Guild                    *Guild          `json:"guild,omitempty"`
	Channel                  *Channel        `json:"channel,omitempty"`
	Inviter                  *User           `json:"inviter,omitempty"`
	TargetType               int             `json:"target_type,omitempty"`
	TargetUser               *User           `json:"target_user,omitempty"`
	TargetApplication        *Application    `json:"target_application,omitempty"`
	ApproximatePresenceCount int             `json:"approximate_presence_count,omitempty"`
	ApproximateMemberCount   int             `json:"approximate_member_count,omitempty"`
	ExpiresAt                string          `json:"expires_at,omitempty"`
	StageInstance            *StageInstance  `json:"stage_instance,omitempty"`
	GuildScheduledEvent      *ScheduledEvent `json:"guild_scheduled_event,omitempty"`
	Uses                     int             `json:"uses,omitempty"`
	MaxUses                  int             `json:"max_uses,omitempty"`
	MaxAge                   int             `json:"max_age,omitempty"`
	Temporary                bool            `json:"temporary,omitempty"`
	CreatedAt                string          `json:"created_at,omitempty"`
}

type StageInstance struct {
	ID                    Snowflake `json:"id"`
	GuildID               Snowflake `json:"guild_id"`
	ChannelID             Snowflake `json:"channel_id"`
	Topic                 string    `json:"topic"`
	PrivacyLevel          int       `json:"privacy_level"`
	DiscoverableDisabled  bool      `json:"discoverable_disabled"`
	GuildScheduledEventID Snowflake `json:"guild_scheduled_event_id,omitempty"`
}

type ScheduledEvent struct {
	ID                 Snowflake               `json:"id"`
	GuildID            Snowflake               `json:"guild_id"`
	ChannelID          Snowflake               `json:"channel_id,omitempty"`
	CreatorID          Snowflake               `json:"creator_id,omitempty"`
	Name               string                  `json:"name"`
	Description        string                  `json:"description,omitempty"`
	ScheduledStartTime string                  `json:"scheduled_start_time"`
	ScheduledEndTime   string                  `json:"scheduled_end_time,omitempty"`
	PrivacyLevel       int                     `json:"privacy_level"`
	Status             int                     `json:"status"`
	EntityType         int                     `json:"entity_type"`
	EntityID           Snowflake               `json:"entity_id,omitempty"`
	EntityMetadata     *ScheduledEventMetadata `json:"entity_metadata,omitempty"`
	Creator            *User                   `json:"creator,omitempty"`
	UserCount          int                     `json:"user_count,omitempty"`
	Image              string                  `json:"image,omitempty"`
}

type ScheduledEventMetadata struct {
	Location string `json:"location,omitempty"`
}

type ScheduledEventParams struct {
	ChannelID          *Snowflake              `json:"channel_id,omitempty"`
	EntityMetadata     *ScheduledEventMetadata `json:"entity_metadata,omitempty"`
	Name               string                  `json:"name,omitempty"`
	PrivacyLevel       int                     `json:"privacy_level,omitempty"`
	ScheduledStartTime string                  `json:"scheduled_start_time,omitempty"`
	ScheduledEndTime   string                  `json:"scheduled_end_time,omitempty"`
	Description        *string                 `json:"description,omitempty"`
	EntityType         int                     `json:"entity_type,omitempty"`
	Status             int                     `json:"status,omitempty"`
	Image              *string                 `json:"image,omitempty"`
}

type Webhook struct {
	ID            Snowflake `json:"id"`
	Type          int       `json:"type"`
	GuildID       Snowflake `json:"guild_id,omitempty"`
	ChannelID     Snowflake `json:"channel_id,omitempty"`
	User          *User     `json:"user,omitempty"`
	Name          string    `json:"name,omitempty"`
	Avatar        string    `json:"avatar,omitempty"`
	Token         string    `json:"token,omitempty"`
	ApplicationID Snowflake `json:"application_id,omitempty"`
	SourceGuild   *Guild    `json:"source_guild,omitempty"`
	SourceChannel *Channel  `json:"source_channel,omitempty"`
	URL           string    `json:"url,omitempty"`
}

type WebhookMessageData struct {
	Content         string           `json:"content,omitempty"`
	Username        string           `json:"username,omitempty"`
	AvatarURL       string           `json:"avatar_url,omitempty"`
	TTS             bool             `json:"tts,omitempty"`
	Embeds          []*Embed         `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	Components      []Component      `json:"components,omitempty"`
	Flags           MessageFlags     `json:"flags,omitempty"`
	ThreadName      string           `json:"thread_name,omitempty"`
}

type AutoModerationRule struct {
	ID              Snowflake               `json:"id"`
	GuildID         Snowflake               `json:"guild_id"`
	Name            string                  `json:"name"`
	CreatorID       Snowflake               `json:"creator_id"`
	EventType       int                     `json:"event_type"`
	TriggerType     int                     `json:"trigger_type"`
	TriggerMetadata *AutoModerationTrigger  `json:"trigger_metadata"`
	Actions         []*AutoModerationAction `json:"actions"`
	Enabled         bool                    `json:"enabled"`
	ExemptRoles     []Snowflake             `json:"exempt_roles"`
	ExemptChannels  []Snowflake             `json:"exempt_channels"`
}

type AutoModerationTrigger struct {
	KeywordFilter                []string `json:"keyword_filter,omitempty"`
	RegexPatterns                []string `json:"regex_patterns,omitempty"`
	Presets                      []int    `json:"presets,omitempty"`
	AllowList                    []string `json:"allow_list,omitempty"`
	MentionTotalLimit            int      `json:"mention_total_limit,omitempty"`
	MentionRaidProtectionEnabled bool     `json:"mention_raid_protection_enabled,omitempty"`
}

type AutoModerationAction struct {
	Type     int                           `json:"type"`
	Metadata *AutoModerationActionMetadata `json:"metadata,omitempty"`
}

type AutoModerationActionMetadata struct {
	ChannelID       Snowflake `json:"channel_id,omitempty"`
	DurationSeconds int       `json:"duration_seconds,omitempty"`
	CustomMessage   string    `json:"custom_message,omitempty"`
}

type AutoModerationRuleModifyData struct {
	Name            string                  `json:"name,omitempty"`
	EventType       int                     `json:"event_type,omitempty"`
	TriggerMetadata *AutoModerationTrigger  `json:"trigger_metadata,omitempty"`
	Actions         []*AutoModerationAction `json:"actions,omitempty"`
	Enabled         *bool                   `json:"enabled,omitempty"`
	ExemptRoles     []Snowflake             `json:"exempt_roles,omitempty"`
	ExemptChannels  []Snowflake             `json:"exempt_channels,omitempty"`
}

type Poll struct {
	Question         PollMedia     `json:"question"`
	Answers          []*PollAnswer `json:"answers"`
	Expiry           string        `json:"expiry,omitempty"`
	AllowMultiselect bool          `json:"allow_multiselect"`
	LayoutType       int           `json:"layout_type"`
	Results          *PollResults  `json:"results,omitempty"`
}

type PollMedia struct {
	Text  string `json:"text,omitempty"`
	Emoji *Emoji `json:"emoji,omitempty"`
}

type PollAnswer struct {
	AnswerID  int       `json:"answer_id"`
	PollMedia PollMedia `json:"poll_media"`
}

type PollResults struct {
	IsFinalized  bool               `json:"is_finalized"`
	AnswerCounts []*PollAnswerCount `json:"answer_counts"`
}

type PollAnswerCount struct {
	ID      int  `json:"id"`
	Count   int  `json:"count"`
	MeVoted bool `json:"me_voted"`
}

type PollCreateData struct {
	Question         PollMedia           `json:"question"`
	Answers          []*PollCreateAnswer `json:"answers"`
	Duration         int                 `json:"duration,omitempty"`
	AllowMultiselect bool                `json:"allow_multiselect,omitempty"`
	LayoutType       int                 `json:"layout_type,omitempty"`
}

type PollCreateAnswer struct {
	PollMedia PollMedia `json:"poll_media"`
}

type ProfileEditData struct {
	Bio              *string `json:"bio,omitempty"`
	AccentColor      *int    `json:"accent_color,omitempty"`
	Banner           *string `json:"banner,omitempty"`
	Avatar           *string `json:"avatar,omitempty"`
	AvatarDecoration *string `json:"avatar_decoration,omitempty"`
}

type DetectableApplication struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Icon           string      `json:"icon,omitempty"`
	Description    string      `json:"description,omitempty"`
	Summary        string      `json:"summary,omitempty"`
	Type           interface{} `json:"type,omitempty"`
	Hook           bool        `json:"hook,omitempty"`
	GuildID        string      `json:"guild_id,omitempty"`
	Executables    interface{} `json:"executables,omitempty"`
	Publishers     interface{} `json:"publishers,omitempty"`
	Developers     interface{} `json:"developers,omitempty"`
	Aliases        []string    `json:"aliases,omitempty"`
	ThirdPartySkus interface{} `json:"third_party_skus,omitempty"`
	Flags          int         `json:"flags,omitempty"`
}

type Ban struct {
	Reason string `json:"reason,omitempty"`
	User   *User  `json:"user"`
}

type GuildPreview struct {
	ID                       Snowflake  `json:"id"`
	Name                     string     `json:"name"`
	Icon                     string     `json:"icon,omitempty"`
	Splash                   string     `json:"splash,omitempty"`
	DiscoverySplash          string     `json:"discovery_splash,omitempty"`
	Emojis                   []*Emoji   `json:"emojis"`
	Features                 []string   `json:"features"`
	ApproximateMemberCount   int        `json:"approximate_member_count"`
	ApproximatePresenceCount int        `json:"approximate_presence_count"`
	Description              string     `json:"description,omitempty"`
	Stickers                 []*Sticker `json:"stickers,omitempty"`
}
