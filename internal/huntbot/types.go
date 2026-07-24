package huntbot

import "github.com/semptpanick/sempatowo/internal/config"

type Trait string

const (
	TraitEfficiency Trait = "efficiency"
	TraitDuration   Trait = "duration"
	TraitCost       Trait = "cost"
	TraitGain       Trait = "gain"
	TraitExp        Trait = "exp"
	TraitRadar      Trait = "radar"
)

var AllTraits = []Trait{
	TraitEfficiency, TraitDuration, TraitCost, TraitGain, TraitExp, TraitRadar,
}

type TraitState struct {
	Enabled      bool
	CurrentLevel int
	Invested     int
}

type UpgradeDetails struct {
	Essence    int
	Efficiency TraitState
	Duration   TraitState
	Cost       TraitState
	Gain       TraitState
	Exp        TraitState
	Radar      TraitState
}

type EmbedField struct {
	Name  string
	Value string
}

type EmbedAuthor struct {
	Name string
}

type MessageEmbed struct {
	Author *EmbedAuthor
	Fields []EmbedField
}

type Attachment struct {
	URL string
}

type Message struct {
	ChannelID   string
	AuthorID    string
	Content     string
	Embeds      []MessageEmbed
	Attachments []Attachment
}

type Command struct {
	ID        string // "huntbot" or "upgrade"
	Arguments string
}

// BotContext is the interface between HuntBot logic and Discord transport.
type BotContext interface {
	HuntChannelID() string
	OwoBotID() string
	// OwnUserID is this account's Discord user ID, used to tell a HuntBot return
	// push addressed to this account from one meant for another account sharing
	// the channel. "" when the client isn't ready yet.
	OwnUserID() string
	Nickname() string
	Settings() config.Huntbot
	RandomPrefix(commands []string) string
	SendMessage(channelID, text string) error
	CanSend() bool
	Log(msg string)
	Sleep(seconds float64)
	// SleepUntil blocks for seconds plus up to noiseSec of jitter. It reports
	// false when the wait was cut short by CancelSleep.
	SleepUntil(seconds float64, noiseSec float64) bool
	CancelSleep()
}
