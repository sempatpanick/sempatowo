package types

type Thread struct {
	Channel
	TotalMessageSent int         `json:"total_message_sent,omitempty"`
	AppliedTags      []Snowflake `json:"applied_tags,omitempty"`
}

type ThreadCreateData struct {
	Name                string      `json:"name"`
	AutoArchiveDuration int         `json:"auto_archive_duration,omitempty"`
	Type                ChannelType `json:"type,omitempty"`
	Invitable           bool        `json:"invitable,omitempty"`
	RateLimitPerUser    int         `json:"rate_limit_per_user,omitempty"`
}

type ForumThreadCreateData struct {
	Name                string              `json:"name"`
	AutoArchiveDuration int                 `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int                 `json:"rate_limit_per_user,omitempty"`
	Message             *ForumThreadMessage `json:"message"`
	AppliedTags         []Snowflake         `json:"applied_tags,omitempty"`
}

type ForumThreadMessage struct {
	Content         string        `json:"content,omitempty"`
	Embeds          []*Embed      `json:"embeds,omitempty"`
	AllowedMentions interface{}   `json:"allowed_mentions,omitempty"`
	Components      []interface{} `json:"components,omitempty"`
	StickerIDs      []Snowflake   `json:"sticker_ids,omitempty"`
	Flags           int           `json:"flags,omitempty"`
}
