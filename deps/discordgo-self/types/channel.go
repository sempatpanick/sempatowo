package types

type Channel struct {
	ID                            Snowflake              `json:"id"`
	Type                          ChannelType            `json:"type"`
	GuildID                       Snowflake              `json:"guild_id,omitempty"`
	Position                      int                    `json:"position,omitempty"`
	PermissionOverwrites          []*PermissionOverwrite `json:"permission_overwrites,omitempty"`
	Name                          string                 `json:"name,omitempty"`
	Topic                         string                 `json:"topic,omitempty"`
	NSFW                          bool                   `json:"nsfw,omitempty"`
	LastMessageID                 Snowflake              `json:"last_message_id,omitempty"`
	Bitrate                       int                    `json:"bitrate,omitempty"`
	UserLimit                     int                    `json:"user_limit,omitempty"`
	RateLimitPerUser              int                    `json:"rate_limit_per_user,omitempty"`
	Recipients                    []*User                `json:"recipients,omitempty"`
	Icon                          string                 `json:"icon,omitempty"`
	OwnerID                       Snowflake              `json:"owner_id,omitempty"`
	ApplicationID                 Snowflake              `json:"application_id,omitempty"`
	Managed                       bool                   `json:"managed,omitempty"`
	ParentID                      Snowflake              `json:"parent_id,omitempty"`
	LastPinTimestamp              string                 `json:"last_pin_timestamp,omitempty"`
	RTCRegion                     string                 `json:"rtc_region,omitempty"`
	VideoQualityMode              int                    `json:"video_quality_mode,omitempty"`
	MessageCount                  int                    `json:"message_count,omitempty"`
	MemberCount                   int                    `json:"member_count,omitempty"`
	ThreadMetadata                *ThreadMetadata        `json:"thread_metadata,omitempty"`
	Member                        *ThreadMember          `json:"member,omitempty"`
	DefaultAutoArchiveDuration    int                    `json:"default_auto_archive_duration,omitempty"`
	Permissions                   string                 `json:"permissions,omitempty"`
	Flags                         int                    `json:"flags,omitempty"`
	TotalMessageSent              int                    `json:"total_message_sent,omitempty"`
	AvailableTags                 []*ForumTag            `json:"available_tags,omitempty"`
	AppliedTags                   []Snowflake            `json:"applied_tags,omitempty"`
	DefaultReactionEmoji          *DefaultReaction       `json:"default_reaction_emoji,omitempty"`
	DefaultThreadRateLimitPerUser int                    `json:"default_thread_rate_limit_per_user,omitempty"`
	DefaultSortOrder              *int                   `json:"default_sort_order,omitempty"`
	DefaultForumLayout            int                    `json:"default_forum_layout,omitempty"`
}

type ChannelType int

const (
	ChannelTypeGuildText          ChannelType = 0
	ChannelTypeDM                 ChannelType = 1
	ChannelTypeGuildVoice         ChannelType = 2
	ChannelTypeGroupDM            ChannelType = 3
	ChannelTypeGuildCategory      ChannelType = 4
	ChannelTypeGuildAnnouncement  ChannelType = 5
	ChannelTypeAnnouncementThread ChannelType = 10
	ChannelTypePublicThread       ChannelType = 11
	ChannelTypePrivateThread      ChannelType = 12
	ChannelTypeGuildStageVoice    ChannelType = 13
	ChannelTypeGuildDirectory     ChannelType = 14
	ChannelTypeGuildForum         ChannelType = 15
	ChannelTypeGuildMedia         ChannelType = 16
)

type PermissionOverwrite struct {
	ID    Snowflake `json:"id"`
	Type  int       `json:"type"`
	Allow string    `json:"allow"`
	Deny  string    `json:"deny"`
}

type ThreadMetadata struct {
	Archived            bool   `json:"archived"`
	AutoArchiveDuration int    `json:"auto_archive_duration"`
	ArchiveTimestamp    string `json:"archive_timestamp"`
	Locked              bool   `json:"locked"`
	Invitable           bool   `json:"invitable,omitempty"`
	CreateTimestamp     string `json:"create_timestamp,omitempty"`
}

type ThreadMember struct {
	ID            Snowflake `json:"id,omitempty"`
	UserID        Snowflake `json:"user_id,omitempty"`
	JoinTimestamp string    `json:"join_timestamp"`
	Flags         int       `json:"flags"`
	Member        *Member   `json:"member,omitempty"`
}

type ForumTag struct {
	ID        Snowflake `json:"id"`
	Name      string    `json:"name"`
	Moderated bool      `json:"moderated"`
	EmojiID   Snowflake `json:"emoji_id,omitempty"`
	EmojiName string    `json:"emoji_name,omitempty"`
}

type DefaultReaction struct {
	EmojiID   Snowflake `json:"emoji_id,omitempty"`
	EmojiName string    `json:"emoji_name,omitempty"`
}

type ChannelCreateData struct {
	Name                          string                 `json:"name"`
	Type                          ChannelType            `json:"type,omitempty"`
	Topic                         string                 `json:"topic,omitempty"`
	Bitrate                       int                    `json:"bitrate,omitempty"`
	UserLimit                     int                    `json:"user_limit,omitempty"`
	RateLimitPerUser              int                    `json:"rate_limit_per_user,omitempty"`
	Position                      int                    `json:"position,omitempty"`
	PermissionOverwrites          []*PermissionOverwrite `json:"permission_overwrites,omitempty"`
	ParentID                      Snowflake              `json:"parent_id,omitempty"`
	NSFW                          bool                   `json:"nsfw,omitempty"`
	RTCRegion                     string                 `json:"rtc_region,omitempty"`
	VideoQualityMode              int                    `json:"video_quality_mode,omitempty"`
	DefaultAutoArchiveDuration    int                    `json:"default_auto_archive_duration,omitempty"`
	DefaultReactionEmoji          *DefaultReaction       `json:"default_reaction_emoji,omitempty"`
	AvailableTags                 []*ForumTag            `json:"available_tags,omitempty"`
	DefaultSortOrder              *int                   `json:"default_sort_order,omitempty"`
	DefaultForumLayout            int                    `json:"default_forum_layout,omitempty"`
	DefaultThreadRateLimitPerUser int                    `json:"default_thread_rate_limit_per_user,omitempty"`
}

type ChannelModifyData struct {
	Name                          string                 `json:"name,omitempty"`
	Type                          ChannelType            `json:"type,omitempty"`
	Position                      *int                   `json:"position,omitempty"`
	Topic                         *string                `json:"topic,omitempty"`
	NSFW                          *bool                  `json:"nsfw,omitempty"`
	RateLimitPerUser              *int                   `json:"rate_limit_per_user,omitempty"`
	Bitrate                       *int                   `json:"bitrate,omitempty"`
	UserLimit                     *int                   `json:"user_limit,omitempty"`
	PermissionOverwrites          []*PermissionOverwrite `json:"permission_overwrites,omitempty"`
	ParentID                      *Snowflake             `json:"parent_id,omitempty"`
	RTCRegion                     *string                `json:"rtc_region,omitempty"`
	VideoQualityMode              *int                   `json:"video_quality_mode,omitempty"`
	DefaultAutoArchiveDuration    *int                   `json:"default_auto_archive_duration,omitempty"`
	Flags                         *int                   `json:"flags,omitempty"`
	AvailableTags                 []*ForumTag            `json:"available_tags,omitempty"`
	DefaultReactionEmoji          *DefaultReaction       `json:"default_reaction_emoji,omitempty"`
	DefaultThreadRateLimitPerUser *int                   `json:"default_thread_rate_limit_per_user,omitempty"`
	DefaultSortOrder              *int                   `json:"default_sort_order,omitempty"`
	DefaultForumLayout            *int                   `json:"default_forum_layout,omitempty"`
	Archived                      *bool                  `json:"archived,omitempty"`
	AutoArchiveDuration           *int                   `json:"auto_archive_duration,omitempty"`
	Locked                        *bool                  `json:"locked,omitempty"`
	Invitable                     *bool                  `json:"invitable,omitempty"`
	AppliedTags                   []Snowflake            `json:"applied_tags,omitempty"`
}

type ChannelPosition struct {
	ID              Snowflake  `json:"id"`
	Position        *int       `json:"position,omitempty"`
	LockPermissions *bool      `json:"lock_permissions,omitempty"`
	ParentID        *Snowflake `json:"parent_id,omitempty"`
}
