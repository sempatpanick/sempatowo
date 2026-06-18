package types

type Message struct {
	ID                Snowflake           `json:"id"`
	ChannelID         Snowflake           `json:"channel_id"`
	GuildID           Snowflake           `json:"guild_id,omitempty"`
	Author            *User               `json:"author"`
	Member            *Member             `json:"member,omitempty"`
	Content           string              `json:"content"`
	Timestamp         string              `json:"timestamp"`
	EditedTimestamp   string              `json:"edited_timestamp,omitempty"`
	TTS               bool                `json:"tts"`
	MentionEveryone   bool                `json:"mention_everyone"`
	Mentions          []*User             `json:"mentions,omitempty"`
	MentionRoles      []Snowflake         `json:"mention_roles,omitempty"`
	MentionChannels   []*ChannelMention   `json:"mention_channels,omitempty"`
	Attachments       []*Attachment       `json:"attachments,omitempty"`
	Embeds            []*Embed            `json:"embeds,omitempty"`
	Reactions         []*Reaction         `json:"reactions,omitempty"`
	Nonce             interface{}         `json:"nonce,omitempty"`
	Pinned            bool                `json:"pinned"`
	WebhookID         Snowflake           `json:"webhook_id,omitempty"`
	Type              MessageType         `json:"type"`
	Activity          *MessageActivity    `json:"activity,omitempty"`
	Application       *Application        `json:"application,omitempty"`
	ApplicationID     Snowflake           `json:"application_id,omitempty"`
	MessageReference  *MessageReference   `json:"message_reference,omitempty"`
	Flags             MessageFlags        `json:"flags,omitempty"`
	ReferencedMessage *Message            `json:"referenced_message,omitempty"`
	Interaction       *MessageInteraction `json:"interaction,omitempty"`
	Thread            *Channel            `json:"thread,omitempty"`
	Components        []Component         `json:"components,omitempty"`
	StickerItems      []*StickerItem      `json:"sticker_items,omitempty"`
	Stickers          []*Sticker          `json:"stickers,omitempty"`
	Position          int                 `json:"position,omitempty"`
	Poll              *Poll               `json:"poll,omitempty"`
}

type MessageType int

const (
	MessageTypeDefault                                 MessageType = 0
	MessageTypeRecipientAdd                            MessageType = 1
	MessageTypeRecipientRemove                         MessageType = 2
	MessageTypeCall                                    MessageType = 3
	MessageTypeChannelNameChange                       MessageType = 4
	MessageTypeChannelIconChange                       MessageType = 5
	MessageTypeChannelPinnedMessage                    MessageType = 6
	MessageTypeUserJoin                                MessageType = 7
	MessageTypeGuildBoost                              MessageType = 8
	MessageTypeGuildBoostTier1                         MessageType = 9
	MessageTypeGuildBoostTier2                         MessageType = 10
	MessageTypeGuildBoostTier3                         MessageType = 11
	MessageTypeChannelFollowAdd                        MessageType = 12
	MessageTypeGuildDiscoveryDisqualified              MessageType = 14
	MessageTypeGuildDiscoveryRequalified               MessageType = 15
	MessageTypeGuildDiscoveryGracePeriodInitialWarning MessageType = 16
	MessageTypeGuildDiscoveryGracePeriodFinalWarning   MessageType = 17
	MessageTypeThreadCreated                           MessageType = 18
	MessageTypeReply                                   MessageType = 19
	MessageTypeChatInputCommand                        MessageType = 20
	MessageTypeThreadStarterMessage                    MessageType = 21
	MessageTypeGuildInviteReminder                     MessageType = 22
	MessageTypeContextMenuCommand                      MessageType = 23
	MessageTypeAutoModerationAction                    MessageType = 24
	MessageTypeRoleSubscriptionPurchase                MessageType = 25
	MessageTypeInteractionPremiumUpsell                MessageType = 26
	MessageTypeStageStart                              MessageType = 27
	MessageTypeStageEnd                                MessageType = 28
	MessageTypeStageSpeaker                            MessageType = 29
	MessageTypeStageTopic                              MessageType = 31
	MessageTypeGuildApplicationPremiumSubscription     MessageType = 32
)

type MessageFlags int

const (
	MessageFlagCrossposted                      MessageFlags = 1 << 0
	MessageFlagIsCrosspost                      MessageFlags = 1 << 1
	MessageFlagSuppressEmbeds                   MessageFlags = 1 << 2
	MessageFlagSourceMessageDeleted             MessageFlags = 1 << 3
	MessageFlagUrgent                           MessageFlags = 1 << 4
	MessageFlagHasThread                        MessageFlags = 1 << 5
	MessageFlagEphemeral                        MessageFlags = 1 << 6
	MessageFlagLoading                          MessageFlags = 1 << 7
	MessageFlagFailedToMentionSomeRolesInThread MessageFlags = 1 << 8
	MessageFlagSuppressNotifications            MessageFlags = 1 << 12
	MessageFlagIsVoiceMessage                   MessageFlags = 1 << 13
)

type MessageReference struct {
	MessageID       Snowflake `json:"message_id,omitempty"`
	ChannelID       Snowflake `json:"channel_id,omitempty"`
	GuildID         Snowflake `json:"guild_id,omitempty"`
	FailIfNotExists bool      `json:"fail_if_not_exists,omitempty"`
}

type MessageActivity struct {
	Type    int    `json:"type"`
	PartyID string `json:"party_id,omitempty"`
}

type MessageInteraction struct {
	ID     Snowflake `json:"id"`
	Type   int       `json:"type"`
	Name   string    `json:"name"`
	User   *User     `json:"user"`
	Member *Member   `json:"member,omitempty"`
}

type ChannelMention struct {
	ID      Snowflake   `json:"id"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
	Name    string      `json:"name"`
}

type Attachment struct {
	ID           Snowflake `json:"id"`
	Filename     string    `json:"filename"`
	Description  string    `json:"description,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	Size         int       `json:"size"`
	URL          string    `json:"url"`
	ProxyURL     string    `json:"proxy_url"`
	Height       int       `json:"height,omitempty"`
	Width        int       `json:"width,omitempty"`
	Ephemeral    bool      `json:"ephemeral,omitempty"`
	DurationSecs float64   `json:"duration_secs,omitempty"`
	Waveform     string    `json:"waveform,omitempty"`
	Flags        int       `json:"flags,omitempty"`
}

type Embed struct {
	Title       string          `json:"title,omitempty"`
	Type        string          `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Color       int             `json:"color,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty"`
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedVideo struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type Reaction struct {
	Count        int         `json:"count"`
	CountDetails interface{} `json:"count_details,omitempty"`
	Me           bool        `json:"me"`
	MeBurst      bool        `json:"me_burst,omitempty"`
	Emoji        *Emoji      `json:"emoji"`
	BurstColors  []string    `json:"burst_colors,omitempty"`
}

type AllowedMentions struct {
	Parse       []string    `json:"parse,omitempty"`
	Roles       []Snowflake `json:"roles,omitempty"`
	Users       []Snowflake `json:"users,omitempty"`
	RepliedUser bool        `json:"replied_user,omitempty"`
}

type MessageSendData struct {
	Content          string            `json:"content,omitempty"`
	TTS              bool              `json:"tts,omitempty"`
	Embeds           []*Embed          `json:"embeds,omitempty"`
	AllowedMentions  *AllowedMentions  `json:"allowed_mentions,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	Components       []Component       `json:"components,omitempty"`
	StickerIDs       []Snowflake       `json:"sticker_ids,omitempty"`
	Flags            MessageFlags      `json:"flags,omitempty"`
	Nonce            string            `json:"nonce,omitempty"`
	Poll             *PollCreateData   `json:"poll,omitempty"`
}

type MessageEditData struct {
	Content         *string          `json:"content,omitempty"`
	Embeds          []*Embed         `json:"embeds,omitempty"`
	Flags           *MessageFlags    `json:"flags,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	Components      []Component      `json:"components,omitempty"`
	Attachments     []*Attachment    `json:"attachments,omitempty"`
}

type Component interface{}

type ComponentType int

const (
	ComponentTypeActionRow         ComponentType = 1
	ComponentTypeButton            ComponentType = 2
	ComponentTypeStringSelect      ComponentType = 3
	ComponentTypeTextInput         ComponentType = 4
	ComponentTypeUserSelect        ComponentType = 5
	ComponentTypeRoleSelect        ComponentType = 6
	ComponentTypeMentionableSelect ComponentType = 7
	ComponentTypeChannelSelect     ComponentType = 8
)

type Application struct {
	ID                  Snowflake   `json:"id"`
	Name                string      `json:"name"`
	Icon                string      `json:"icon,omitempty"`
	Description         string      `json:"description"`
	RPCOrigins          []string    `json:"rpc_origins,omitempty"`
	BotPublic           bool        `json:"bot_public"`
	BotRequireCodeGrant bool        `json:"bot_require_code_grant"`
	TermsOfServiceURL   string      `json:"terms_of_service_url,omitempty"`
	PrivacyPolicyURL    string      `json:"privacy_policy_url,omitempty"`
	Owner               *User       `json:"owner,omitempty"`
	VerifyKey           string      `json:"verify_key"`
	Team                interface{} `json:"team,omitempty"`
	GuildID             Snowflake   `json:"guild_id,omitempty"`
	PrimarySKUID        Snowflake   `json:"primary_sku_id,omitempty"`
	Slug                string      `json:"slug,omitempty"`
	CoverImage          string      `json:"cover_image,omitempty"`
	Flags               int         `json:"flags,omitempty"`
	Tags                []string    `json:"tags,omitempty"`
	CustomInstallURL    string      `json:"custom_install_url,omitempty"`
}
