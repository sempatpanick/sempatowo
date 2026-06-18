package types

// UserSettingsProtoType represents the type of user settings proto
type UserSettingsProtoType int

const (
	UserSettingsProtoRecency      UserSettingsProtoType = 2
	UserSettingsProtoPreloaded    UserSettingsProtoType = 1
	UserSettingsProtoTestSettings UserSettingsProtoType = 3
)

// UserSettingsProtoResponse represents the API response for remote settings
type UserSettingsProtoResponse struct {
	Settings  string `json:"settings"` // Base64 encoded protobuf
	OutOfDate bool   `json:"out_of_date,omitempty"`
}

// InboxTab Enum
const (
	InboxTabUnspecified = 0
	InboxTabMentions    = 1
	InboxTabUnreads     = 2
	InboxTabTodos       = 3 // Reminders
	InboxTabForYou      = 4
	InboxTabGameInvites = 5
	InboxTabBookmarks   = 6
	InboxTabScheduled   = 7
)

// Theme Enum
const (
	ThemeUnset    = 0
	ThemeDark     = 1
	ThemeLight    = 2
	ThemeDarker   = 3
	ThemeMidnight = 4
)

// UI Density Enum
const (
	UIDensityUnset      = 0
	UIDensityCompact    = 1
	UIDensityCozy       = 2
	UIDensityResponsive = 3
)

// Note: Since we don't have the .proto files or a Go protobuf generator setup here,
// we are defining the high-level API types. The actual manipulation of the 'Settings'
// string (which is binary protobuf data) would require a protobuf library and definition.
// For now, we provide the ability to Fetch and Update the raw blob.
