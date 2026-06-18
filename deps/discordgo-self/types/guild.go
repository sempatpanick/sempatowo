package types

type Guild struct {
	ID                          Snowflake      `json:"id"`
	Name                        string         `json:"name"`
	Icon                        string         `json:"icon,omitempty"`
	IconHash                    string         `json:"icon_hash,omitempty"`
	Splash                      string         `json:"splash,omitempty"`
	DiscoverySplash             string         `json:"discovery_splash,omitempty"`
	Owner                       bool           `json:"owner,omitempty"`
	OwnerID                     Snowflake      `json:"owner_id"`
	Permissions                 string         `json:"permissions,omitempty"`
	Region                      string         `json:"region,omitempty"`
	AFKChannelID                Snowflake      `json:"afk_channel_id,omitempty"`
	AFKTimeout                  int            `json:"afk_timeout"`
	WidgetEnabled               bool           `json:"widget_enabled,omitempty"`
	WidgetChannelID             Snowflake      `json:"widget_channel_id,omitempty"`
	VerificationLevel           int            `json:"verification_level"`
	DefaultMessageNotifications int            `json:"default_message_notifications"`
	ExplicitContentFilter       int            `json:"explicit_content_filter"`
	Roles                       []*Role        `json:"roles"`
	Emojis                      []*Emoji       `json:"emojis"`
	Features                    []string       `json:"features"`
	MFALevel                    int            `json:"mfa_level"`
	ApplicationID               Snowflake      `json:"application_id,omitempty"`
	SystemChannelID             Snowflake      `json:"system_channel_id,omitempty"`
	SystemChannelFlags          int            `json:"system_channel_flags"`
	RulesChannelID              Snowflake      `json:"rules_channel_id,omitempty"`
	MaxPresences                int            `json:"max_presences,omitempty"`
	MaxMembers                  int            `json:"max_members,omitempty"`
	VanityURLCode               string         `json:"vanity_url_code,omitempty"`
	Description                 string         `json:"description,omitempty"`
	Banner                      string         `json:"banner,omitempty"`
	PremiumTier                 int            `json:"premium_tier"`
	PremiumSubscriptionCount    int            `json:"premium_subscription_count,omitempty"`
	PreferredLocale             string         `json:"preferred_locale"`
	PublicUpdatesChannelID      Snowflake      `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers        int            `json:"max_video_channel_users,omitempty"`
	MaxStageVideoChannelUsers   int            `json:"max_stage_video_channel_users,omitempty"`
	ApproximateMemberCount      int            `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount    int            `json:"approximate_presence_count,omitempty"`
	WelcomeScreen               *WelcomeScreen `json:"welcome_screen,omitempty"`
	NSFWLevel                   int            `json:"nsfw_level"`
	Stickers                    []*Sticker     `json:"stickers,omitempty"`
	PremiumProgressBarEnabled   bool           `json:"premium_progress_bar_enabled"`
	SafetyAlertsChannelID       Snowflake      `json:"safety_alerts_channel_id,omitempty"`

	JoinedAt    string        `json:"joined_at,omitempty"`
	Large       bool          `json:"large,omitempty"`
	Unavailable bool          `json:"unavailable,omitempty"`
	MemberCount int           `json:"member_count,omitempty"`
	VoiceStates []*VoiceState `json:"voice_states,omitempty"`
	Members     []*Member     `json:"members,omitempty"`
	Channels    []*Channel    `json:"channels,omitempty"`
	Threads     []*Channel    `json:"threads,omitempty"`
	Presences   []*Presence   `json:"presences,omitempty"`
}

type Role struct {
	ID           Snowflake `json:"id"`
	Name         string    `json:"name"`
	Color        int       `json:"color"`
	Hoist        bool      `json:"hoist"`
	Icon         string    `json:"icon,omitempty"`
	UnicodeEmoji string    `json:"unicode_emoji,omitempty"`
	Position     int       `json:"position"`
	Permissions  string    `json:"permissions"`
	Managed      bool      `json:"managed"`
	Mentionable  bool      `json:"mentionable"`
	Tags         *RoleTags `json:"tags,omitempty"`
	Flags        int       `json:"flags,omitempty"`
}

type RoleTags struct {
	BotID                 Snowflake `json:"bot_id,omitempty"`
	IntegrationID         Snowflake `json:"integration_id,omitempty"`
	PremiumSubscriber     bool      `json:"premium_subscriber,omitempty"`
	SubscriptionListingID Snowflake `json:"subscription_listing_id,omitempty"`
	AvailableForPurchase  bool      `json:"available_for_purchase,omitempty"`
	GuildConnections      bool      `json:"guild_connections,omitempty"`
}

type WelcomeScreen struct {
	Description     string                  `json:"description,omitempty"`
	WelcomeChannels []*WelcomeScreenChannel `json:"welcome_channels"`
}

type WelcomeScreenChannel struct {
	ChannelID   Snowflake `json:"channel_id"`
	Description string    `json:"description"`
	EmojiID     Snowflake `json:"emoji_id,omitempty"`
	EmojiName   string    `json:"emoji_name,omitempty"`
}

type Emoji struct {
	ID            Snowflake   `json:"id,omitempty"`
	Name          string      `json:"name"`
	Roles         []Snowflake `json:"roles,omitempty"`
	User          *User       `json:"user,omitempty"`
	RequireColons bool        `json:"require_colons,omitempty"`
	Managed       bool        `json:"managed,omitempty"`
	Animated      bool        `json:"animated,omitempty"`
	Available     bool        `json:"available,omitempty"`
}

type Sticker struct {
	ID          Snowflake   `json:"id"`
	PackID      Snowflake   `json:"pack_id,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Tags        string      `json:"tags"`
	Asset       string      `json:"asset,omitempty"`
	Type        StickerType `json:"type"`
	FormatType  int         `json:"format_type"`
	Available   bool        `json:"available,omitempty"`
	GuildID     Snowflake   `json:"guild_id,omitempty"`
	User        *User       `json:"user,omitempty"`
	SortValue   int         `json:"sort_value,omitempty"`
}

type StickerType int

const (
	StickerTypeStandard StickerType = 1
	StickerTypeGuild    StickerType = 2
)

type StickerItem struct {
	ID         Snowflake `json:"id"`
	Name       string    `json:"name"`
	FormatType int       `json:"format_type"`
}

type StickerPack struct {
	ID             Snowflake  `json:"id"`
	Stickers       []*Sticker `json:"stickers"`
	Name           string     `json:"name"`
	SKUID          Snowflake  `json:"sku_id"`
	CoverStickerID Snowflake  `json:"cover_sticker_id,omitempty"`
	Description    string     `json:"description"`
	BannerAssetID  Snowflake  `json:"banner_asset_id,omitempty"`
}

type RoleCreateData struct {
	Name         string `json:"name,omitempty"`
	Permissions  string `json:"permissions,omitempty"`
	Color        int    `json:"color,omitempty"`
	Hoist        bool   `json:"hoist,omitempty"`
	Icon         string `json:"icon,omitempty"`
	UnicodeEmoji string `json:"unicode_emoji,omitempty"`
	Mentionable  bool   `json:"mentionable,omitempty"`
}

type RoleModifyData struct {
	Name         *string `json:"name,omitempty"`
	Permissions  *string `json:"permissions,omitempty"`
	Color        *int    `json:"color,omitempty"`
	Hoist        *bool   `json:"hoist,omitempty"`
	Icon         *string `json:"icon,omitempty"`
	UnicodeEmoji *string `json:"unicode_emoji,omitempty"`
	Mentionable  *bool   `json:"mentionable,omitempty"`
}

type RolePosition struct {
	ID       Snowflake `json:"id"`
	Position *int      `json:"position,omitempty"`
}

type MemberModifyData struct {
	Nick                       *string     `json:"nick,omitempty"`
	Roles                      []Snowflake `json:"roles,omitempty"`
	Mute                       *bool       `json:"mute,omitempty"`
	Deaf                       *bool       `json:"deaf,omitempty"`
	ChannelID                  *Snowflake  `json:"channel_id,omitempty"`
	CommunicationDisabledUntil *string     `json:"communication_disabled_until,omitempty"`
	Flags                      *int        `json:"flags,omitempty"`
}

type WelcomeScreenData struct {
	Enabled         *bool                   `json:"enabled,omitempty"`
	WelcomeChannels []*WelcomeScreenChannel `json:"welcome_channels,omitempty"`
	Description     *string                 `json:"description,omitempty"`
}

type AuditLog struct {
	ApplicationCommands  []interface{}    `json:"application_commands,omitempty"`
	AuditLogEntries      []*AuditLogEntry `json:"audit_log_entries"`
	AutoModerationRules  []interface{}    `json:"auto_moderation_rules,omitempty"`
	GuildScheduledEvents []interface{}    `json:"guild_scheduled_events,omitempty"`
	Integrations         []interface{}    `json:"integrations,omitempty"`
	Threads              []*Channel       `json:"threads,omitempty"`
	Users                []*User          `json:"users,omitempty"`
	Webhooks             []interface{}    `json:"webhooks,omitempty"`
}

type AuditLogEntry struct {
	TargetID   string                 `json:"target_id,omitempty"`
	Changes    []*AuditLogChange      `json:"changes,omitempty"`
	UserID     Snowflake              `json:"user_id,omitempty"`
	ID         Snowflake              `json:"id"`
	ActionType int                    `json:"action_type"`
	Options    map[string]interface{} `json:"options,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
}

type AuditLogChange struct {
	NewValue interface{} `json:"new_value,omitempty"`
	OldValue interface{} `json:"old_value,omitempty"`
	Key      string      `json:"key"`
}
