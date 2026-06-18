package types

import "encoding/json"

type Interaction struct {
	ID             Snowflake       `json:"id"`
	ApplicationID  Snowflake       `json:"application_id"`
	Type           InteractionType `json:"type"`
	Data           json.RawMessage `json:"data,omitempty"`
	GuildID        Snowflake       `json:"guild_id,omitempty"`
	ChannelID      Snowflake       `json:"channel_id,omitempty"`
	Member         *Member         `json:"member,omitempty"`
	User           *User           `json:"user,omitempty"`
	Token          string          `json:"token"`
	Version        int             `json:"version"`
	Message        interface{}     `json:"message,omitempty"`
	AppPermissions string          `json:"app_permissions,omitempty"`
	Locale         string          `json:"locale,omitempty"`
	GuildLocale    string          `json:"guild_locale,omitempty"`
}

type InteractionType int

const (
	InteractionTypePing                           InteractionType = 1
	InteractionTypeApplicationCommand             InteractionType = 2
	InteractionTypeMessageComponent               InteractionType = 3
	InteractionTypeApplicationCommandAutocomplete InteractionType = 4
	InteractionTypeModalSubmit                    InteractionType = 5
)

type ApplicationCommand struct {
	ID                       Snowflake                   `json:"id"`
	Type                     ApplicationCommandType      `json:"type,omitempty"`
	ApplicationID            Snowflake                   `json:"application_id"`
	GuildID                  Snowflake                   `json:"guild_id,omitempty"`
	Name                     string                      `json:"name"`
	NameLocalizations        map[string]string           `json:"name_localizations,omitempty"`
	Description              string                      `json:"description"`
	DescriptionLocalizations map[string]string           `json:"description_localizations,omitempty"`
	Options                  []*ApplicationCommandOption `json:"options,omitempty"`
	DefaultMemberPermissions *string                     `json:"default_member_permissions,omitempty"`
	DMPermission             *bool                       `json:"dm_permission,omitempty"`
	NSFW                     bool                        `json:"nsfw,omitempty"`
	Version                  Snowflake                   `json:"version"`
}

type ApplicationCommandType int

const (
	ApplicationCommandTypeChatInput ApplicationCommandType = 1
	ApplicationCommandTypeUser      ApplicationCommandType = 2
	ApplicationCommandTypeMessage   ApplicationCommandType = 3
)

type ApplicationCommandOption struct {
	Type                     ApplicationCommandOptionType `json:"type"`
	Name                     string                       `json:"name"`
	NameLocalizations        map[string]string            `json:"name_localizations,omitempty"`
	Description              string                       `json:"description"`
	DescriptionLocalizations map[string]string            `json:"description_localizations,omitempty"`
	Required                 bool                         `json:"required,omitempty"`
	Choices                  []*ApplicationCommandChoice  `json:"choices,omitempty"`
	Options                  []*ApplicationCommandOption  `json:"options,omitempty"`
	ChannelTypes             []ChannelType                `json:"channel_types,omitempty"`
	MinValue                 interface{}                  `json:"min_value,omitempty"`
	MaxValue                 interface{}                  `json:"max_value,omitempty"`
	MinLength                int                          `json:"min_length,omitempty"`
	MaxLength                int                          `json:"max_length,omitempty"`
	Autocomplete             bool                         `json:"autocomplete,omitempty"`
}

type ApplicationCommandOptionType int

const (
	ApplicationCommandOptionTypeSubCommand      ApplicationCommandOptionType = 1
	ApplicationCommandOptionTypeSubCommandGroup ApplicationCommandOptionType = 2
	ApplicationCommandOptionTypeString          ApplicationCommandOptionType = 3
	ApplicationCommandOptionTypeInteger         ApplicationCommandOptionType = 4
	ApplicationCommandOptionTypeBoolean         ApplicationCommandOptionType = 5
	ApplicationCommandOptionTypeUser            ApplicationCommandOptionType = 6
	ApplicationCommandOptionTypeChannel         ApplicationCommandOptionType = 7
	ApplicationCommandOptionTypeRole            ApplicationCommandOptionType = 8
	ApplicationCommandOptionTypeMentionable     ApplicationCommandOptionType = 9
	ApplicationCommandOptionTypeNumber          ApplicationCommandOptionType = 10
	ApplicationCommandOptionTypeAttachment      ApplicationCommandOptionType = 11
)

type ApplicationCommandChoice struct {
	Name              string            `json:"name"`
	NameLocalizations map[string]string `json:"name_localizations,omitempty"`
	Value             interface{}       `json:"value"`
}

type InteractionData struct {
	ID       Snowflake                `json:"id"`
	Name     string                   `json:"name"`
	Type     ApplicationCommandType   `json:"type"`
	Resolved *ResolvedData            `json:"resolved,omitempty"`
	Options  []*InteractionDataOption `json:"options,omitempty"`
	GuildID  Snowflake                `json:"guild_id,omitempty"`
	TargetID Snowflake                `json:"target_id,omitempty"`
}

type InteractionDataOption struct {
	Name    string                       `json:"name"`
	Type    ApplicationCommandOptionType `json:"type"`
	Value   interface{}                  `json:"value,omitempty"`
	Options []*InteractionDataOption     `json:"options,omitempty"`
	Focused bool                         `json:"focused,omitempty"`
}

type ResolvedData struct {
	Users       map[Snowflake]*User       `json:"users,omitempty"`
	Members     map[Snowflake]*Member     `json:"members,omitempty"`
	Roles       map[Snowflake]*Role       `json:"roles,omitempty"`
	Channels    map[Snowflake]*Channel    `json:"channels,omitempty"`
	Messages    map[Snowflake]*Message    `json:"messages,omitempty"`
	Attachments map[Snowflake]*Attachment `json:"attachments,omitempty"`
}

type ComponentInteractionData struct {
	CustomID      string        `json:"custom_id"`
	ComponentType ComponentType `json:"component_type"`
	Values        []string      `json:"values,omitempty"`
}

type ModalSubmitData struct {
	CustomID   string               `json:"custom_id"`
	Components []ActionRowComponent `json:"components"`
}

type ActionRowComponent struct {
	Type       ComponentType     `json:"type"`
	Components []json.RawMessage `json:"components"`
}

type Button struct {
	Type     ComponentType `json:"type"`
	Style    ButtonStyle   `json:"style"`
	Label    string        `json:"label,omitempty"`
	Emoji    *Emoji        `json:"emoji,omitempty"`
	CustomID string        `json:"custom_id,omitempty"`
	URL      string        `json:"url,omitempty"`
	Disabled bool          `json:"disabled,omitempty"`
}

type ButtonStyle int

const (
	ButtonStylePrimary   ButtonStyle = 1
	ButtonStyleSecondary ButtonStyle = 2
	ButtonStyleSuccess   ButtonStyle = 3
	ButtonStyleDanger    ButtonStyle = 4
	ButtonStyleLink      ButtonStyle = 5
)

type SelectMenu struct {
	Type        ComponentType   `json:"type"`
	CustomID    string          `json:"custom_id"`
	Options     []*SelectOption `json:"options,omitempty"`
	Placeholder string          `json:"placeholder,omitempty"`
	MinValues   int             `json:"min_values,omitempty"`
	MaxValues   int             `json:"max_values,omitempty"`
	Disabled    bool            `json:"disabled,omitempty"`
}

type SelectOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Emoji       *Emoji `json:"emoji,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

type TextInput struct {
	Type        ComponentType  `json:"type"`
	CustomID    string         `json:"custom_id"`
	Style       TextInputStyle `json:"style"`
	Label       string         `json:"label"`
	MinLength   int            `json:"min_length,omitempty"`
	MaxLength   int            `json:"max_length,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Value       string         `json:"value,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
}

type TextInputStyle int

const (
	TextInputStyleShort     TextInputStyle = 1
	TextInputStyleParagraph TextInputStyle = 2
)

type SlashCommandData struct {
	ApplicationID Snowflake               `json:"application_id"`
	GuildID       Snowflake               `json:"guild_id,omitempty"`
	ChannelID     Snowflake               `json:"channel_id"`
	SessionID     string                  `json:"session_id"`
	Data          SlashCommandPayloadData `json:"data"`
	Nonce         string                  `json:"nonce,omitempty"`
	Type          int                     `json:"type"`
}

type SlashCommandPayloadData struct {
	Version            Snowflake              `json:"version"`
	ID                 Snowflake              `json:"id"`
	Name               string                 `json:"name"`
	Type               ApplicationCommandType `json:"type"`
	Options            []*CommandOptionValue  `json:"options,omitempty"`
	ApplicationCommand *ApplicationCommand    `json:"application_command,omitempty"`
	Attachments        []interface{}          `json:"attachments,omitempty"`
}

type CommandOptionValue struct {
	Type    ApplicationCommandOptionType `json:"type"`
	Name    string                       `json:"name"`
	Value   interface{}                  `json:"value,omitempty"`
	Options []*CommandOptionValue        `json:"options,omitempty"`
}

type InteractionPayload struct {
	Type          InteractionType `json:"type"`
	ApplicationID Snowflake       `json:"application_id"`
	GuildID       Snowflake       `json:"guild_id,omitempty"`
	ChannelID     Snowflake       `json:"channel_id"`
	MessageID     Snowflake       `json:"message_id,omitempty"`
	SessionID     string          `json:"session_id"`
	Data          interface{}     `json:"data"`
	Nonce         string          `json:"nonce,omitempty"`
}

type ComponentData struct {
	ComponentType ComponentType `json:"component_type"`
	CustomID      string        `json:"custom_id"`
	Values        []string      `json:"values,omitempty"`
}
