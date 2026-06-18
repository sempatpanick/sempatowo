package types

// ReportReason represents a reason for reporting content
type ReportReason struct {
	Reason      int    `json:"reason"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ReportOption represents a report option in V2/V3 API
type ReportOption struct {
	Value       string                `json:"value"`
	Label       string                `json:"label"`
	Description string                `json:"description"`
	SubQuestion string                `json:"sub_question,omitempty"`
	SubTypes    []ReportOptionSubType `json:"sub_types,omitempty"`
}

// ReportOptionSubType represents a sub-type for a report option
type ReportOptionSubType struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ReportMenu represents the structure of a V3 report menu
type ReportMenu struct {
	Name          string             `json:"name"`
	Version       string             `json:"version"`
	Variant       string             `json:"variant"`
	PostbackURL   string             `json:"postback_url"`
	Language      string             `json:"language,omitempty"`
	RootNodeID    int                `json:"root_node_id"`
	SuccessNodeID int                `json:"success_node_id"`
	FailNodeID    int                `json:"fail_node_id"`
	Nodes         map[int]ReportNode `json:"nodes"`
}

// ReportNode represents a single step/screen in the report flow
type ReportNode struct {
	ID                    int               `json:"id"`
	ReportType            string            `json:"report_type,omitempty"`
	Key                   string            `json:"key"`
	Header                string            `json:"header"`
	Subheader             string            `json:"subheader,omitempty"`
	Info                  string            `json:"info,omitempty"`
	Children              []ReportNodeChild `json:"children"`
	Elements              []ReportElement   `json:"elements"`
	Button                *ReportButton     `json:"button,omitempty"`
	IsMultiSelectRequired bool              `json:"is_multi_select_required"`
	IsAutoSubmit          bool              `json:"is_auto_submit"`
}

// ReportNodeChild represents a navigation option to another node
type ReportNodeChild struct {
	Name         string `json:"name"`
	TargetNodeID int    `json:"target_node_id"`
}

// ReportButton represents the action button on a report node
type ReportButton struct {
	Type   string `json:"type"` // next, submit, done, cancel
	Target *int   `json:"target,omitempty"`
}

// ReportElement represents a UI component in the report menu
type ReportElement struct {
	Name              string      `json:"name"`
	Type              string      `json:"type"` // checkbox, dropdown, free_text, etc.
	Data              interface{} `json:"data"` // Structure varies by Type
	ShouldSubmitData  bool        `json:"should_submit_data"`
	SkipIfUnlocalized bool        `json:"skip_if_unlocalized"`
	IsLocalized       bool        `json:"is_localized"`
}

// ReportDropdownElement data structure
type ReportDropdownElement struct {
	Title   string                 `json:"title"`
	Options []ReportDropdownOption `json:"options"`
}

// ReportDropdownOption structure
type ReportDropdownOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ReportFreeTextElement data structure
type ReportFreeTextElement struct {
	Title          string `json:"title,omitempty"`
	Subtitle       string `json:"subtitle,omitempty"`
	Placeholder    string `json:"placeholder,omitempty"`
	Rows           int    `json:"rows"`
	CharacterLimit int    `json:"character_limit"`
	Pattern        string `json:"pattern,omitempty"`
}

// ReportSubmitData represents the data payload for submitting a report
type ReportSubmitData struct {
	Version     string              `json:"version"`
	Variant     string              `json:"variant"`
	Name        string              `json:"name"`
	Language    string              `json:"language"`
	Breadcrumbs []int               `json:"breadcrumbs"`
	Elements    map[string][]string `json:"elements,omitempty"`

	// Optional context fields depending on report type
	ChannelID             Snowflake `json:"channel_id,omitempty"`
	MessageID             Snowflake `json:"message_id,omitempty"`
	GuildID               Snowflake `json:"guild_id,omitempty"`
	StageInstanceID       Snowflake `json:"stage_instance_id,omitempty"`
	GuildScheduledEventID Snowflake `json:"guild_scheduled_event_id,omitempty"`
	ReportedUserID        Snowflake `json:"reported_user_id,omitempty"`
	ApplicationID         Snowflake `json:"application_id,omitempty"`
	UserID                Snowflake `json:"user_id,omitempty"`
	WidgetID              Snowflake `json:"widget_id,omitempty"`

	// Unauthenticated flow
	EmailToken string `json:"email_token,omitempty"`
}
