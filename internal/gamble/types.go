package gamble

import "github.com/semptpanick/sempatowo/internal/config"

const maxBetAmount = 250_000

const (
	QueueCoinflip = "coinflip"
	QueueSlots    = "slots"
)

const (
	briefCooldownMin    = 0.7
	briefCooldownMax    = 2.7
	moderateCooldownMin = 70.0
	moderateCooldownMax = 200.0
)

type EmbedField struct {
	Name  string
	Value string
}

type EmbedAuthor struct {
	Name string
}

type EmbedFooter struct {
	Text string
}

type MessageEmbed struct {
	Author *EmbedAuthor
	Fields []EmbedField
	Footer *EmbedFooter
}

type Message struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
	Embeds    []MessageEmbed
	Reactions []Reaction
}

type Reaction struct {
	Emoji string
	Me    bool
}

// BotContext connects gamble logic to Discord transport.
type BotContext interface {
	HuntChannelID() string
	OwoBotID() string
	Nickname() string
	Username() string
	UserID() string
	CashCheck() bool
	Gamble() config.Gamble
	RandomPrefix(commands []string) string
	SendMessage(channelID, text string) error
	SendGambleBet(channelID, game, text string) error
	SignalGambleResult(game string)
	CanSend() bool
	Log(msg string)
	Debug(msg string)
	Sleep(seconds float64)
	AddReaction(channelID, messageID, emoji string) error
	RemoveReaction(channelID, messageID, emoji string) error
}
