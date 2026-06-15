package config

// Settings holds all runtime options for one Discord account.
type Settings struct {
	Typing             bool           `json:"typing"`
	Prefix             string         `json:"prefix"`
	Status             StatusSettings `json:"status"`
	Interval           Interval       `json:"interval"`
	Channels           Channels       `json:"channels"`
	Target             Targets        `json:"target"`
	OwoID              string         `json:"owoId"`
	ChecklistCompleted bool           `json:"checklist_completed"`
	CashCheck          bool           `json:"cashCheck"`
	AutoDaily          bool           `json:"autoDaily"`
	AllowAutoQuest     bool           `json:"allowAutoQuest"`
	OCRApi             string         `json:"ocrApi"`
	Huntbot            Huntbot        `json:"huntbot"`
	Gamble             Gamble         `json:"gamble"`
	AutoQuest          AutoQuest      `json:"autoQuest"`
}

type StatusSettings struct {
	Hunt          bool `json:"hunt"`
	Battle        bool `json:"battle"`
	Zoo           bool `json:"zoo"`
	Pray          bool `json:"pray"`
	Curse         bool `json:"curse"`
	Lootbox       bool `json:"lootbox"`
	LootboxFabled bool `json:"lootbox_fabled"`
	Crate         bool `json:"crate"`
	Cookie        bool `json:"cookie"`
	Gems          bool `json:"gems"`
	Inventory     bool `json:"inventory"`
	Quest         bool `json:"quest"`
}

type ActionDelay struct {
	MinDelay     int `json:"minDelay"`
	MaxDelay     int `json:"maxDelay"`
	SlowestTime  int `json:"slowestTime,omitempty"`
	FastestTime  int `json:"fastestTime,omitempty"`
}

type Interval struct {
	SendMessage int         `json:"send_message"`
	Zoo         int         `json:"zoo"`
	Pray        int         `json:"pray"`
	Curse       int         `json:"curse"`
	Hunt        ActionDelay `json:"hunt"`
	Battle      ActionDelay `json:"battle"`
	Inventory   int         `json:"inventory"`
	Checklist   int         `json:"checklist"`
	Quest       QuestDelay  `json:"quest"`
}

type QuestDelay struct {
	Owo   int `json:"owo"`
	Check int `json:"check"`
}

type Channels struct {
	Hunt  string `json:"hunt"`
	Quest string `json:"quest"`
}

type Targets struct {
	Pray   string `json:"pray"`
	Curse  string `json:"curse"`
	Cookie string `json:"cookie"`
}

type Huntbot struct {
	Enabled     bool            `json:"enabled"`
	CashToSpend int             `json:"cashToSpend"`
	Upgrader    HuntbotUpgrader `json:"upgrader"`
}

type HuntbotUpgrader struct {
	Enabled   bool              `json:"enabled"`
	Sleeptime jsonSleeptime     `json:"sleeptime"`
	Traits    HuntbotTraits     `json:"traits"`
	Weights   HuntbotWeights    `json:"weights"`
}

// jsonSleeptime accepts either a number or [min, max] in JSON.
type jsonSleeptime struct {
	Single *float64
	Range  *[2]float64
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
	Enabled          bool         `json:"enabled"`
	StartValue       int          `json:"startValue"`
	MultiplierOnLose float64      `json:"multiplierOnLose"`
	Cooldown         jsonSecRange `json:"cooldown"`
}

type CoinflipSettings struct {
	GambleGame
	Options CoinflipOptions `json:"options"`
}

type CoinflipOptions struct {
	Heads bool `json:"heads"`
	Tails bool `json:"tails"`
}

type AutoQuest struct {
	Enabled                       bool              `json:"enabled"`
	HelpChannel                   AutoQuestHelpChan `json:"helpChannel"`
	EnableCommandsToCompleteQuest bool              `json:"enableCommandsToCompleteQuest"`
	HelpOthers                    bool              `json:"helpOthers"`
	CheckCooldown                 jsonSecRange      `json:"checkCooldown"`
}

type AutoQuestHelpChan struct {
	PostInHelpChannel bool   `json:"postInHelpChannel"`
	ChannelID         string `json:"channelId"`
}

// jsonSecRange accepts a [min, max] cooldown in seconds (owo-dusk style).
type jsonSecRange struct {
	Range *[2]float64
}

// CooldownSec returns min/max cooldown in seconds.
func (g GambleGame) CooldownSec() (min, max float64) {
	if g.Cooldown.Range == nil {
		return 16, 18
	}
	return (*g.Cooldown.Range)[0], (*g.Cooldown.Range)[1]
}

// Defaults returns the built-in config template (same as the TS project).
func Defaults() Settings {
	return Settings{
		Typing: true,
		Prefix: "w",
		Status: StatusSettings{
			Hunt: true, Battle: true, Zoo: false, Pray: true, Curse: false,
			Lootbox: true, LootboxFabled: true, Crate: true, Cookie: false,
			Gems: true, Inventory: true, Quest: false,
		},
		Interval: Interval{
			SendMessage: 5000,
			Zoo:         300000,
			Pray:        305000,
			Curse:       305000,
			Hunt:        ActionDelay{MinDelay: 50000, MaxDelay: 200000},
			Battle:      ActionDelay{MinDelay: 50000, MaxDelay: 200000},
			Inventory:   300000,
			Checklist:   1000000,
			Quest:       QuestDelay{Owo: 32000, Check: 60000},
		},
		Channels: Channels{
			Hunt:  "1513744333579489310",
			Quest: "1513744333579489310",
		},
		Target: Targets{
			Cookie: "469369739131617291",
		},
		OwoID:              "408785106942164992",
		ChecklistCompleted: false,
		CashCheck:          true,
		AutoDaily:          true,
		AllowAutoQuest:     false,
		OCRApi:             "helloworld",
		AutoQuest: AutoQuest{
			Enabled: false,
			HelpChannel: AutoQuestHelpChan{
				PostInHelpChannel: false,
			},
			EnableCommandsToCompleteQuest: true,
			HelpOthers:                    true,
			CheckCooldown:                 jsonSecRange{Range: &[2]float64{10, 30}},
		},
		Gamble: Gamble{
			AllottedAmount: 10000,
			GoalSystem:     GambleGoalSystem{Enabled: true, Amount: 30000},
			Coinflip: CoinflipSettings{
				GambleGame: GambleGame{
					Enabled: false, StartValue: 200, MultiplierOnLose: 2,
					Cooldown: jsonSecRange{Range: &[2]float64{16, 18}},
				},
				Options: CoinflipOptions{Heads: true, Tails: false},
			},
			Slots: GambleGame{
				Enabled: false, StartValue: 200, MultiplierOnLose: 2,
				Cooldown: jsonSecRange{Range: &[2]float64{16, 18}},
			},
			Blackjack: GambleGame{
				Enabled: false, StartValue: 200, MultiplierOnLose: 2,
				Cooldown: jsonSecRange{Range: &[2]float64{16, 18}},
			},
		},
		Huntbot: Huntbot{
			Enabled:     false,
			CashToSpend: 10000,
			Upgrader: HuntbotUpgrader{
				Enabled: true,
				Sleeptime: jsonSleeptime{Range: &[2]float64{10, 15}},
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
	}
}
