package config

// SchemaVersion is the current config file format. A file without this key is
// treated as the pre-1.0 shape and migrated by legacy.go on load.
const SchemaVersion = 1

// Settings holds all runtime options for one Discord account.
//
// The layout is feature-first: everything needed to run one feature lives in
// one block. The old schema split each feature across status/interval/channels/
// target plus a top-level gate, so enabling a feature meant editing up to five
// places and forgetting one silently did nothing.
type Settings struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label,omitempty"`

	Typing   bool   `json:"typing"`
	Prefix   string `json:"prefix"`
	OwoBotID string `json:"owoBotId"`

	// DefaultChannel is where commands go unless a feature overrides it.
	DefaultChannel string `json:"defaultChannel"`

	// SendMessageInterval is the minimum spacing between outgoing messages.
	SendMessageInterval Duration `json:"sendMessageInterval"`

	// StopWhenChecklistDone halts farming once the checklist is fully ticked.
	StopWhenChecklistDone bool `json:"stopWhenChecklistDone"`

	// TrackBalance keeps a running cash total, which the gamble safety limits
	// and the daily reward handler both depend on.
	TrackBalance bool `json:"trackBalance"`

	Features Features `json:"features"`
}

// Features groups one block per automated behaviour.
type Features struct {
	Hunt      ScheduledFeature `json:"hunt"`
	Battle    ScheduledFeature `json:"battle"`
	Pray      TargetedFeature  `json:"pray"`
	Curse     TargetedFeature  `json:"curse"`
	Zoo       ScheduledFeature `json:"zoo"`
	Inventory ScheduledFeature `json:"inventory"`
	Checklist ScheduledFeature `json:"checklist"`
	Cookie    CookieFeature    `json:"cookie"`
	Lootbox   LootboxFeature   `json:"lootbox"`
	Crate     Toggle           `json:"crate"`
	Gems      Toggle           `json:"gems"`
	Daily     Toggle           `json:"daily"`
	Quest     QuestFeature     `json:"quest"`
	Huntbot   Huntbot          `json:"huntbot"`
	Gamble    Gamble           `json:"gamble"`
}

// Toggle is a feature with nothing to configure but on/off.
type Toggle struct {
	Enabled bool `json:"enabled"`
}

// ScheduledFeature runs a command on a repeating randomised delay. A fixed
// interval is expressed as min == max; jitter is available everywhere rather
// than only on hunt and battle, since a perfectly periodic command is the
// easiest kind of automation to spot.
type ScheduledFeature struct {
	Enabled bool `json:"enabled"`
	// Channel overrides Settings.DefaultChannel when non-empty.
	Channel string `json:"channel,omitempty"`
	Delay   Range  `json:"delay"`
}

// TargetedFeature is a scheduled command aimed at another user ("" = self).
type TargetedFeature struct {
	ScheduledFeature
	Target string `json:"target"`
}

// CookieFeature is triggered by the checklist rather than on a timer.
type CookieFeature struct {
	Enabled bool   `json:"enabled"`
	Target  string `json:"target"`
}

type LootboxFeature struct {
	Enabled bool `json:"enabled"`
	Fabled  bool `json:"fabled"`
}

type QuestFeature struct {
	Enabled bool   `json:"enabled"`
	Channel string `json:"channel,omitempty"`
	// Delay is how often to re-check quest progress.
	Delay Range `json:"delay"`
	// OwoDelay spaces out the plain "owo" messages used to complete the
	// say-owo quest.
	OwoDelay Range     `json:"owoDelay"`
	Auto     AutoQuest `json:"auto"`
}

type AutoQuest struct {
	Enabled bool `json:"enabled"`
	// AcknowledgeExperimental is the safety gate: auto-quest clicks buttons and
	// posts in shared channels, so it stays off until explicitly acknowledged.
	AcknowledgeExperimental       bool              `json:"acknowledgeExperimental"`
	HelpChannel                   AutoQuestHelpChan `json:"helpChannel"`
	EnableCommandsToCompleteQuest bool              `json:"enableCommandsToCompleteQuest"`
	HelpOthers                    bool              `json:"helpOthers"`
	CheckCooldown                 Range             `json:"checkCooldown"`
}

type AutoQuestHelpChan struct {
	PostInHelpChannel bool   `json:"postInHelpChannel"`
	ChannelID         string `json:"channelId"`
}

type Huntbot struct {
	Enabled     bool            `json:"enabled"`
	CashToSpend int             `json:"cashToSpend"`
	Upgrader    HuntbotUpgrader `json:"upgrader"`
}

type HuntbotUpgrader struct {
	Enabled bool `json:"enabled"`
	// Cooldown was "sleeptime" in the old schema, where it could be either a
	// number or a [min,max] pair.
	Cooldown Range          `json:"cooldown"`
	Traits   HuntbotTraits  `json:"traits"`
	Weights  HuntbotWeights `json:"weights"`
}

type HuntbotTraits struct {
	Efficiency bool `json:"efficiency"`
	Duration   bool `json:"duration"`
	Cost       bool `json:"cost"`
	Gain       bool `json:"gain"`
	Exp        bool `json:"exp"`
	Radar      bool `json:"radar"`
}

type HuntbotWeights struct {
	Efficiency float64 `json:"efficiency"`
	Duration   float64 `json:"duration"`
	Cost       float64 `json:"cost"`
	Gain       float64 `json:"gain"`
	Exp        float64 `json:"exp"`
	Radar      float64 `json:"radar"`
}

type Gamble struct {
	AllottedAmount int              `json:"allottedAmount"`
	GoalSystem     GambleGoalSystem `json:"goalSystem"`
	Coinflip       CoinflipSettings `json:"coinflip"`
	Slots          GambleGame       `json:"slots"`
	Blackjack      GambleGame       `json:"blackjack"`
}

type GambleGoalSystem struct {
	Enabled bool `json:"enabled"`
	Amount  int  `json:"amount"`
}

type GambleGame struct {
	Enabled          bool    `json:"enabled"`
	StartValue       int     `json:"startValue"`
	MultiplierOnLose float64 `json:"multiplierOnLose"`
	Cooldown         Range   `json:"cooldown"`
}

type CoinflipSettings struct {
	GambleGame
	Options CoinflipOptions `json:"options"`
}

type CoinflipOptions struct {
	Heads bool `json:"heads"`
	Tails bool `json:"tails"`
}

// CooldownSec returns min/max cooldown in seconds.
func (g GambleGame) CooldownSec() (min, max float64) {
	if g.Cooldown.IsZero() {
		return 16, 18
	}
	return g.Cooldown.SecondsRange()
}

// --- channel resolution ---

func (s Settings) channelOr(override string) string {
	if override != "" {
		return override
	}
	return s.DefaultChannel
}

// FarmChannel is where the ordinary OwO commands go.
func (s Settings) FarmChannel() string { return s.channelOr(s.Features.Hunt.Channel) }

// QuestChannel is where quest checks go.
func (s Settings) QuestChannel() string { return s.channelOr(s.Features.Quest.Channel) }

// ChannelFor resolves a scheduled feature's channel against the default.
func (s Settings) ChannelFor(f ScheduledFeature) string { return s.channelOr(f.Channel) }

// AutoQuestActive reports whether auto-quest should run: both the feature and
// its experimental acknowledgement have to be set.
func (s Settings) AutoQuestActive() bool {
	return s.Features.Quest.Auto.Enabled && s.Features.Quest.Auto.AcknowledgeExperimental
}

// AnyGambleEnabled reports whether at least one gamble game is on.
func (s Settings) AnyGambleEnabled() bool {
	g := s.Features.Gamble
	return g.Coinflip.Enabled || g.Slots.Enabled || g.Blackjack.Enabled
}

// Defaults returns the built-in config template.
func Defaults() Settings {
	const defaultChannel = "1513744333579489310"

	return Settings{
		SchemaVersion:         SchemaVersion,
		Typing:                true,
		Prefix:                "w",
		OwoBotID:              "408785106942164992",
		DefaultChannel:        defaultChannel,
		SendMessageInterval:   secs(5),
		StopWhenChecklistDone: false,
		TrackBalance:          true,
		Features: Features{
			Hunt:      ScheduledFeature{Enabled: true, Delay: rangeMillis(50000, 200000)},
			Battle:    ScheduledFeature{Enabled: true, Delay: rangeMillis(50000, 200000)},
			Pray:      TargetedFeature{ScheduledFeature: ScheduledFeature{Enabled: true, Delay: rangeSecs(305, 305)}},
			Curse:     TargetedFeature{ScheduledFeature: ScheduledFeature{Enabled: false, Delay: rangeSecs(305, 305)}},
			Zoo:       ScheduledFeature{Enabled: false, Delay: rangeSecs(300, 300)},
			Inventory: ScheduledFeature{Enabled: true, Delay: rangeSecs(300, 300)},
			// Checklist defaults to false: the loop was disabled in code for a
			// while, so opting in explicitly keeps existing configs behaving
			// as they did.
			Checklist: ScheduledFeature{Enabled: false, Delay: rangeSecs(1000, 1000)},
			Cookie:    CookieFeature{Enabled: false, Target: "469369739131617291"},
			Lootbox:   LootboxFeature{Enabled: true, Fabled: true},
			Crate:     Toggle{Enabled: true},
			Gems:      Toggle{Enabled: true},
			Daily:     Toggle{Enabled: true},
			Quest: QuestFeature{
				Enabled:  false,
				Delay:    rangeSecs(60, 60),
				OwoDelay: rangeSecs(32, 32),
				Auto: AutoQuest{
					Enabled:                       false,
					AcknowledgeExperimental:       false,
					HelpChannel:                   AutoQuestHelpChan{PostInHelpChannel: false},
					EnableCommandsToCompleteQuest: true,
					HelpOthers:                    true,
					CheckCooldown:                 rangeSecs(10, 30),
				},
			},
			Huntbot: Huntbot{
				Enabled:     false,
				CashToSpend: 10000,
				Upgrader: HuntbotUpgrader{
					Enabled:  true,
					Cooldown: rangeSecs(10, 15),
					Traits: HuntbotTraits{
						Efficiency: true, Duration: true, Cost: true,
						Gain: true, Exp: true, Radar: true,
					},
					Weights: HuntbotWeights{
						Efficiency: 4, Duration: 2, Cost: 5,
						Gain: 4, Exp: 3, Radar: 1,
					},
				},
			},
			Gamble: Gamble{
				AllottedAmount: 10000,
				GoalSystem:     GambleGoalSystem{Enabled: true, Amount: 30000},
				Coinflip: CoinflipSettings{
					GambleGame: GambleGame{
						Enabled: false, StartValue: 200, MultiplierOnLose: 2,
						Cooldown: rangeSecs(16, 18),
					},
					Options: CoinflipOptions{Heads: true, Tails: false},
				},
				Slots: GambleGame{
					Enabled: false, StartValue: 200, MultiplierOnLose: 2,
					Cooldown: rangeSecs(16, 18),
				},
				Blackjack: GambleGame{
					Enabled: false, StartValue: 200, MultiplierOnLose: 2,
					Cooldown: rangeSecs(16, 18),
				},
			},
		},
	}
}
