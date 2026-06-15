package quest

import "github.com/sempatowo/sempatowo/internal/config"

// ParsedQuest is one row from the quest OCR image.
type ParsedQuest struct {
	Text     string
	Current  int
	Total    int
	Complete bool
}

// LocalQuest is a quest assigned to this account.
type LocalQuest struct {
	Text     string
	QuestID  string
	Current  int
	Total    int
	Complete bool
	Helpable bool
}

// HelpableQuest is a quest another user needs help with.
type HelpableQuest struct {
	UserID      string
	QuestID     string
	Current     int
	Total       int
	Complete    bool
	ClaimUserID string
	ChannelID   string
	GuildID     string
}

// BotContext connects auto-quest logic to Discord transport.
type BotContext interface {
	UserID() string
	Username() string
	DisplayName() string
	GuildID() string
	HuntChannelID() string
	QuestHelpChannelID() string
	OwoBotID() string
	Nickname() string
	AutoQuest() config.AutoQuest
	AllowAutoQuest() bool
	OCRApiKey() string
	CanEnableQuestCmds() bool

	IsHuntEnabled() bool
	IsBattleEnabled() bool
	IsGambleEnabled() bool
	IsCookieEnabled() bool
	IsPrayEnabled() bool
	IsCurseEnabled() bool

	RandomPrefix(commands []string) string
	SendMessage(channelID, text string) error
	CanSend() bool
	Log(msg string)

	ClickButton(channelID, messageID, customID, applicationID string) error
	SleepRange(min, max float64)

	GambleEnqueue(game string, amount int)
}
