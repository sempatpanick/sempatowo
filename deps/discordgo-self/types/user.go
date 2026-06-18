package types

type User struct {
	ID                   Snowflake   `json:"id"`
	Username             string      `json:"username"`
	Discriminator        string      `json:"discriminator"`
	GlobalName           string      `json:"global_name,omitempty"`
	Avatar               string      `json:"avatar,omitempty"`
	AvatarDecoration     string      `json:"avatar_decoration,omitempty"`
	AvatarDecorationData interface{} `json:"avatar_decoration_data,omitempty"`
	Banner               string      `json:"banner,omitempty"`
	BannerColor          string      `json:"banner_color,omitempty"`
	AccentColor          int         `json:"accent_color,omitempty"`
	Bot                  bool        `json:"bot,omitempty"`
	System               bool        `json:"system,omitempty"`
	MFAEnabled           bool        `json:"mfa_enabled,omitempty"`
	Locale               string      `json:"locale,omitempty"`
	Verified             bool        `json:"verified,omitempty"`
	Email                string      `json:"email,omitempty"`
	Flags                int         `json:"flags,omitempty"`
	PublicFlags          int         `json:"public_flags,omitempty"`
	PremiumType          int         `json:"premium_type,omitempty"`
	Phone                string      `json:"phone,omitempty"`
	NSFWAllowed          bool        `json:"nsfw_allowed,omitempty"`
	Bio                  string      `json:"bio,omitempty"`
	Pronouns             string      `json:"pronouns,omitempty"`
}

type CurrentUser struct {
	User
	Email              string `json:"email,omitempty"`
	Verified           bool   `json:"verified,omitempty"`
	MFAEnabled         bool   `json:"mfa_enabled,omitempty"`
	Locale             string `json:"locale,omitempty"`
	PurchasedFlags     int    `json:"purchased_flags,omitempty"`
	PremiumUsageFlags  int    `json:"premium_usage_flags,omitempty"`
	HasBouncedEmail    bool   `json:"has_bounced_email,omitempty"`
	LinkedUsers        []any  `json:"linked_users,omitempty"`
	AuthenticatorTypes []int  `json:"authenticator_types,omitempty"`
}

type Member struct {
	User                       *User       `json:"user,omitempty"`
	Nick                       string      `json:"nick,omitempty"`
	Avatar                     string      `json:"avatar,omitempty"`
	Roles                      []Snowflake `json:"roles"`
	JoinedAt                   string      `json:"joined_at"`
	PremiumSince               string      `json:"premium_since,omitempty"`
	Deaf                       bool        `json:"deaf"`
	Mute                       bool        `json:"mute"`
	Pending                    bool        `json:"pending,omitempty"`
	Permissions                string      `json:"permissions,omitempty"`
	CommunicationDisabledUntil string      `json:"communication_disabled_until,omitempty"`
	Flags                      int         `json:"flags,omitempty"`
	GuildID                    Snowflake   `json:"guild_id,omitempty"`
}

type Relationship struct {
	ID       Snowflake        `json:"id"`
	Type     RelationshipType `json:"type"`
	Nickname string           `json:"nickname,omitempty"`
	User     *User            `json:"user,omitempty"`
	Since    string           `json:"since,omitempty"`
}

type RelationshipType int

const (
	RelationshipTypeNone            RelationshipType = 0
	RelationshipTypeFriend          RelationshipType = 1
	RelationshipTypeBlocked         RelationshipType = 2
	RelationshipTypePendingIncoming RelationshipType = 3
	RelationshipTypePendingOutgoing RelationshipType = 4
	RelationshipTypeImplicit        RelationshipType = 5
)

type UserSettings struct {
	Locale                            string        `json:"locale,omitempty"`
	ShowCurrentGame                   bool          `json:"show_current_game,omitempty"`
	InlineAttachmentMedia             bool          `json:"inline_attachment_media,omitempty"`
	InlineEmbedMedia                  bool          `json:"inline_embed_media,omitempty"`
	GifAutoPlay                       bool          `json:"gif_auto_play,omitempty"`
	RenderEmbeds                      bool          `json:"render_embeds,omitempty"`
	RenderReactions                   bool          `json:"render_reactions,omitempty"`
	AnimateEmoji                      bool          `json:"animate_emoji,omitempty"`
	EnableTTSCommand                  bool          `json:"enable_tts_command,omitempty"`
	MessageDisplayCompact             bool          `json:"message_display_compact,omitempty"`
	ConvertEmoticons                  bool          `json:"convert_emoticons,omitempty"`
	ExplicitContentFilter             int           `json:"explicit_content_filter,omitempty"`
	DisableGamesTab                   bool          `json:"disable_games_tab,omitempty"`
	Theme                             string        `json:"theme,omitempty"`
	DeveloperMode                     bool          `json:"developer_mode,omitempty"`
	GuildPositions                    []Snowflake   `json:"guild_positions,omitempty"`
	DetectPlatformAccounts            bool          `json:"detect_platform_accounts,omitempty"`
	Status                            string        `json:"status,omitempty"`
	CustomStatus                      *CustomStatus `json:"custom_status,omitempty"`
	TimezoneOffset                    int           `json:"timezone_offset,omitempty"`
	StreamNotificationsEnabled        bool          `json:"stream_notifications_enabled,omitempty"`
	AllowAccessibilityDetection       bool          `json:"allow_accessibility_detection,omitempty"`
	ContactSyncEnabled                bool          `json:"contact_sync_enabled,omitempty"`
	NativePhoneIntegrationEnabled     bool          `json:"native_phone_integration_enabled,omitempty"`
	AnimateStickers                   int           `json:"animate_stickers,omitempty"`
	FriendDiscoveryFlags              int           `json:"friend_discovery_flags,omitempty"`
	ViewNSFWGuilds                    bool          `json:"view_nsfw_guilds,omitempty"`
	ViewNSFWCommands                  bool          `json:"view_nsfw_commands,omitempty"`
	PasswordLess                      bool          `json:"passwordless,omitempty"`
	FriendSourceFlags                 interface{}   `json:"friend_source_flags,omitempty"`
	GuildFolders                      []interface{} `json:"guild_folders,omitempty"`
	RestrictedGuilds                  []Snowflake   `json:"restricted_guilds,omitempty"`
	DefaultGuildsRestricted           bool          `json:"default_guilds_restricted,omitempty"`
	ActivityRestrictedGuildIDs        []Snowflake   `json:"activity_restricted_guild_ids,omitempty"`
	ActivityJoiningRestrictedGuildIDs []Snowflake   `json:"activity_joining_restricted_guild_ids,omitempty"`
}

type CustomStatus struct {
	Text      string `json:"text,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

type Presence struct {
	User         *User        `json:"user,omitempty"`
	Status       Status       `json:"status"`
	Activities   []*Activity  `json:"activities,omitempty"`
	ClientStatus ClientStatus `json:"client_status,omitempty"`
	GuildID      Snowflake    `json:"guild_id,omitempty"`
}

type ClientStatus struct {
	Desktop string `json:"desktop,omitempty"`
	Mobile  string `json:"mobile,omitempty"`
	Web     string `json:"web,omitempty"`
}

type Status string

const (
	StatusOnline    Status = "online"
	StatusIdle      Status = "idle"
	StatusDND       Status = "dnd"
	StatusInvisible Status = "invisible"
	StatusOffline   Status = "offline"
)

type Activity struct {
	Name          string        `json:"name"`
	Type          ActivityType  `json:"type"`
	URL           string        `json:"url,omitempty"`
	CreatedAt     int64         `json:"created_at,omitempty"`
	Timestamps    *Timestamps   `json:"timestamps,omitempty"`
	ApplicationID Snowflake     `json:"application_id,omitempty"`
	Details       string        `json:"details,omitempty"`
	State         string        `json:"state,omitempty"`
	Emoji         *Emoji        `json:"emoji,omitempty"`
	Party         *Party        `json:"party,omitempty"`
	Assets        *Assets       `json:"assets,omitempty"`
	Secrets       *Secrets      `json:"secrets,omitempty"`
	Instance      bool          `json:"instance,omitempty"`
	Flags         int           `json:"flags,omitempty"`
	Buttons       []interface{} `json:"buttons,omitempty"`
	SyncID        string        `json:"sync_id,omitempty"`
	SessionID     string        `json:"session_id,omitempty"`
}

type ActivityType int

const (
	ActivityTypePlaying   ActivityType = 0
	ActivityTypeStreaming ActivityType = 1
	ActivityTypeListening ActivityType = 2
	ActivityTypeWatching  ActivityType = 3
	ActivityTypeCustom    ActivityType = 4
	ActivityTypeCompeting ActivityType = 5
)

type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type Party struct {
	ID   string `json:"id,omitempty"`
	Size []int  `json:"size,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type Secrets struct {
	Join     string `json:"join,omitempty"`
	Spectate string `json:"spectate,omitempty"`
	Match    string `json:"match,omitempty"`
}

type Session struct {
	SessionID  string         `json:"session_id"`
	Status     Status         `json:"status"`
	Activities []*Activity    `json:"activities,omitempty"`
	ClientInfo *SessionClient `json:"client_info,omitempty"`
	Active     bool           `json:"active,omitempty"`
}

type SessionClient struct {
	Client  string `json:"client"`
	OS      string `json:"os"`
	Version int    `json:"version"`
}

type VoiceState struct {
	GuildID                 Snowflake `json:"guild_id,omitempty"`
	ChannelID               Snowflake `json:"channel_id,omitempty"`
	UserID                  Snowflake `json:"user_id"`
	Member                  *Member   `json:"member,omitempty"`
	SessionID               string    `json:"session_id"`
	Deaf                    bool      `json:"deaf"`
	Mute                    bool      `json:"mute"`
	SelfDeaf                bool      `json:"self_deaf"`
	SelfMute                bool      `json:"self_mute"`
	SelfStream              bool      `json:"self_stream,omitempty"`
	SelfVideo               bool      `json:"self_video"`
	Suppress                bool      `json:"suppress"`
	RequestToSpeakTimestamp string    `json:"request_to_speak_timestamp,omitempty"`
}

type VoiceServerUpdate struct {
	Token    string    `json:"token"`
	GuildID  Snowflake `json:"guild_id"`
	Endpoint string    `json:"endpoint"`
}

type ReadState struct {
	ID               Snowflake `json:"id"`
	LastMessageID    Snowflake `json:"last_message_id,omitempty"`
	LastPinTimestamp string    `json:"last_pin_timestamp,omitempty"`
	MentionCount     int       `json:"mention_count,omitempty"`
	Badge            int       `json:"badge_count,omitempty"`
	Flags            int       `json:"flags,omitempty"`
}
