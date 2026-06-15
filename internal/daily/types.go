package daily

// BotContext connects daily logic to Discord transport.
type BotContext interface {
	UserID() string
	HuntChannelID() string
	OwoBotID() string
	Nickname() string
	AutoDaily() bool
	CashCheck() bool
	RandomPrefix(commands []string) string
	SendMessage(channelID, text string) error
	CanSend() bool
	Log(msg string)
	OnDailyReward(amount int)
}
