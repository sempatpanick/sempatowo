package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// Everything in this file exists to read config files written before
// SchemaVersion 1 and convert them in place. It is self-contained so it can be
// deleted wholesale once no old files remain in the wild.

type legacySettings struct {
	Typing             bool           `json:"typing"`
	Prefix             string         `json:"prefix"`
	Status             legacyStatus   `json:"status"`
	Interval           legacyInterval `json:"interval"`
	Channels           legacyChannels `json:"channels"`
	Target             legacyTargets  `json:"target"`
	OwoID              string         `json:"owoId"`
	ChecklistCompleted bool           `json:"checklist_completed"`
	CashCheck          bool           `json:"cashCheck"`
	AutoDaily          bool           `json:"autoDaily"`
	AllowAutoQuest     bool           `json:"allowAutoQuest"`
	OCRApi             string         `json:"ocrApi"`
	Huntbot            legacyHuntbot  `json:"huntbot"`
	Gamble             legacyGamble   `json:"gamble"`
	AutoQuest          legacyAutoQ    `json:"autoQuest"`
}

type legacyStatus struct {
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
	Checklist     bool `json:"checklist"`
}

type legacyActionDelay struct {
	MinDelay    int `json:"minDelay"`
	MaxDelay    int `json:"maxDelay"`
	SlowestTime int `json:"slowestTime"`
	FastestTime int `json:"fastestTime"`
}

type legacyInterval struct {
	SendMessage int               `json:"send_message"`
	Zoo         int               `json:"zoo"`
	Pray        int               `json:"pray"`
	Curse       int               `json:"curse"`
	Hunt        legacyActionDelay `json:"hunt"`
	Battle      legacyActionDelay `json:"battle"`
	Inventory   int               `json:"inventory"`
	Checklist   int               `json:"checklist"`
	Quest       legacyQuestDelay  `json:"quest"`
}

type legacyQuestDelay struct {
	Owo   int `json:"owo"`
	Check int `json:"check"`
}

type legacyChannels struct {
	Hunt  string `json:"hunt"`
	Quest string `json:"quest"`
}

type legacyTargets struct {
	Pray   string `json:"pray"`
	Curse  string `json:"curse"`
	Cookie string `json:"cookie"`
}

type legacyHuntbot struct {
	Enabled     bool           `json:"enabled"`
	CashToSpend int            `json:"cashToSpend"`
	Upgrader    legacyUpgrader `json:"upgrader"`
}

type legacyUpgrader struct {
	Enabled   bool            `json:"enabled"`
	Sleeptime legacySleeptime `json:"sleeptime"`
	Traits    HuntbotTraits   `json:"traits"`
	Weights   HuntbotWeights  `json:"weights"`
}

// legacySleeptime accepted either a number or [min, max], both in seconds.
type legacySleeptime struct {
	Single *float64
	Range  *[2]float64
}

func (j *legacySleeptime) UnmarshalJSON(data []byte) error {
	// Reset both: the value being decoded into may already hold the default
	// range, and a scalar in the file has to displace it rather than lose to it.
	j.Single, j.Range = nil, nil

	var single float64
	if err := json.Unmarshal(data, &single); err == nil {
		j.Single = &single
		return nil
	}
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err == nil {
		j.Range = &arr
		return nil
	}
	return fmt.Errorf("invalid sleeptime: %s", string(data))
}

func (j legacySleeptime) toRange(def Range) Range {
	if j.Range != nil {
		return rangeSecs(j.Range[0], j.Range[1])
	}
	if j.Single != nil {
		return rangeSecs(*j.Single, *j.Single)
	}
	return def
}

// legacySecRange accepted [min, max] in seconds.
type legacySecRange struct {
	Range *[2]float64
}

func (j *legacySecRange) UnmarshalJSON(data []byte) error {
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("invalid cooldown: %s", string(data))
	}
	j.Range = &arr
	return nil
}

func (j legacySecRange) toRange(def Range) Range {
	if j.Range == nil {
		return def
	}
	return rangeSecs(j.Range[0], j.Range[1])
}

type legacyGambleGame struct {
	Enabled          bool           `json:"enabled"`
	StartValue       int            `json:"startValue"`
	MultiplierOnLose float64        `json:"multiplierOnLose"`
	Cooldown         legacySecRange `json:"cooldown"`
}

type legacyCoinflip struct {
	legacyGambleGame
	Options CoinflipOptions `json:"options"`
}

type legacyGamble struct {
	AllottedAmount int              `json:"allottedAmount"`
	GoalSystem     GambleGoalSystem `json:"goalSystem"`
	Coinflip       legacyCoinflip   `json:"coinflip"`
	Slots          legacyGambleGame `json:"slots"`
	Blackjack      legacyGambleGame `json:"blackjack"`
}

type legacyAutoQ struct {
	Enabled                       bool              `json:"enabled"`
	HelpChannel                   AutoQuestHelpChan `json:"helpChannel"`
	EnableCommandsToCompleteQuest bool              `json:"enableCommandsToCompleteQuest"`
	HelpOthers                    bool              `json:"helpOthers"`
	CheckCooldown                 legacySecRange    `json:"checkCooldown"`
}

// legacyDefaults mirrors the pre-1.0 Defaults() so a partial old file merges
// against the values it was written to sit on top of.
func legacyDefaults() legacySettings {
	return legacySettings{
		Typing: true,
		Prefix: "w",
		Status: legacyStatus{
			Hunt: true, Battle: true, Zoo: false, Pray: true, Curse: false,
			Lootbox: true, LootboxFabled: true, Crate: true, Cookie: false,
			Gems: true, Inventory: true, Quest: false, Checklist: false,
		},
		Interval: legacyInterval{
			SendMessage: 5000,
			Zoo:         300000,
			Pray:        305000,
			Curse:       305000,
			// Hunt and Battle are deliberately left zero here: legacyDelay
			// needs to tell "the user wrote minDelay" from "this is the
			// default", so that the older slowestTime/fastestTime spelling can
			// still win when the user used that instead.
			Inventory: 300000,
			Checklist: 1000000,
			Quest:     legacyQuestDelay{Owo: 32000, Check: 60000},
		},
		Channels: legacyChannels{
			Hunt:  "1513744333579489310",
			Quest: "1513744333579489310",
		},
		Target:         legacyTargets{Cookie: "469369739131617291"},
		OwoID:          "408785106942164992",
		CashCheck:      true,
		AutoDaily:      true,
		AllowAutoQuest: false,
		OCRApi:         "helloworld",
		AutoQuest: legacyAutoQ{
			EnableCommandsToCompleteQuest: true,
			HelpOthers:                    true,
			CheckCooldown:                 legacySecRange{Range: &[2]float64{10, 30}},
		},
		Gamble: legacyGamble{
			AllottedAmount: 10000,
			GoalSystem:     GambleGoalSystem{Enabled: true, Amount: 30000},
			Coinflip: legacyCoinflip{
				legacyGambleGame: legacyGambleGame{
					StartValue: 200, MultiplierOnLose: 2,
					Cooldown: legacySecRange{Range: &[2]float64{16, 18}},
				},
				Options: CoinflipOptions{Heads: true},
			},
			Slots: legacyGambleGame{
				StartValue: 200, MultiplierOnLose: 2,
				Cooldown: legacySecRange{Range: &[2]float64{16, 18}},
			},
			Blackjack: legacyGambleGame{
				StartValue: 200, MultiplierOnLose: 2,
				Cooldown: legacySecRange{Range: &[2]float64{16, 18}},
			},
		},
		Huntbot: legacyHuntbot{
			CashToSpend: 10000,
			Upgrader: legacyUpgrader{
				Enabled:   true,
				Sleeptime: legacySleeptime{Range: &[2]float64{10, 15}},
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

// isLegacy reports whether raw JSON predates SchemaVersion 1. The absence of
// the key is the signal — an old file has no way to claim a version.
func isLegacy(data []byte) bool {
	var probe struct {
		SchemaVersion *int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	return probe.SchemaVersion == nil
}

// migrateLegacy converts a pre-1.0 file into current Settings. It also returns
// human-readable notes about anything that moved somewhere the user has to know
// about (currently only ocrApi, which became an environment variable).
func migrateLegacy(data []byte) (Settings, []string, error) {
	old := legacyDefaults()
	if err := json.Unmarshal(data, &old); err != nil {
		return Settings{}, nil, fmt.Errorf("reading legacy config: %w", err)
	}

	var notes []string
	def := Defaults()
	s := def

	s.Typing = old.Typing
	s.Prefix = old.Prefix
	s.OwoBotID = old.OwoID
	s.DefaultChannel = old.Channels.Hunt
	s.SendMessageInterval = millis(old.Interval.SendMessage, def.SendMessageInterval)
	s.StopWhenChecklistDone = old.ChecklistCompleted
	s.TrackBalance = old.CashCheck

	f := &s.Features

	f.Hunt = ScheduledFeature{
		Enabled: old.Status.Hunt,
		Delay:   legacyDelay(old.Interval.Hunt, def.Features.Hunt.Delay),
	}
	f.Battle = ScheduledFeature{
		Enabled: old.Status.Battle,
		Delay:   legacyDelay(old.Interval.Battle, def.Features.Battle.Delay),
	}
	f.Pray = TargetedFeature{
		ScheduledFeature: ScheduledFeature{
			Enabled: old.Status.Pray,
			Delay:   fixedMillis(old.Interval.Pray, def.Features.Pray.Delay),
		},
		Target: old.Target.Pray,
	}
	f.Curse = TargetedFeature{
		ScheduledFeature: ScheduledFeature{
			Enabled: old.Status.Curse,
			Delay:   fixedMillis(old.Interval.Curse, def.Features.Curse.Delay),
		},
		Target: old.Target.Curse,
	}
	f.Zoo = ScheduledFeature{
		Enabled: old.Status.Zoo,
		Delay:   fixedMillis(old.Interval.Zoo, def.Features.Zoo.Delay),
	}
	f.Inventory = ScheduledFeature{
		Enabled: old.Status.Inventory,
		Delay:   fixedMillis(old.Interval.Inventory, def.Features.Inventory.Delay),
	}
	f.Checklist = ScheduledFeature{
		Enabled: old.Status.Checklist,
		Delay:   fixedMillis(old.Interval.Checklist, def.Features.Checklist.Delay),
	}
	f.Cookie = CookieFeature{Enabled: old.Status.Cookie, Target: old.Target.Cookie}
	f.Lootbox = LootboxFeature{Enabled: old.Status.Lootbox, Fabled: old.Status.LootboxFabled}
	f.Crate = Toggle{Enabled: old.Status.Crate}
	f.Gems = Toggle{Enabled: old.Status.Gems}
	f.Daily = Toggle{Enabled: old.AutoDaily}

	questChannel := ""
	if old.Channels.Quest != "" && old.Channels.Quest != old.Channels.Hunt {
		questChannel = old.Channels.Quest
	}
	f.Quest = QuestFeature{
		Enabled:  old.Status.Quest,
		Channel:  questChannel,
		Delay:    fixedMillis(old.Interval.Quest.Check, def.Features.Quest.Delay),
		OwoDelay: fixedMillis(old.Interval.Quest.Owo, def.Features.Quest.OwoDelay),
		Auto: AutoQuest{
			Enabled:                       old.AutoQuest.Enabled,
			AcknowledgeExperimental:       old.AllowAutoQuest,
			HelpChannel:                   old.AutoQuest.HelpChannel,
			EnableCommandsToCompleteQuest: old.AutoQuest.EnableCommandsToCompleteQuest,
			HelpOthers:                    old.AutoQuest.HelpOthers,
			CheckCooldown:                 old.AutoQuest.CheckCooldown.toRange(def.Features.Quest.Auto.CheckCooldown),
		},
	}

	f.Huntbot = Huntbot{
		Enabled:     old.Huntbot.Enabled,
		CashToSpend: old.Huntbot.CashToSpend,
		Upgrader: HuntbotUpgrader{
			Enabled:  old.Huntbot.Upgrader.Enabled,
			Cooldown: old.Huntbot.Upgrader.Sleeptime.toRange(def.Features.Huntbot.Upgrader.Cooldown),
			Traits:   old.Huntbot.Upgrader.Traits,
			Weights:  old.Huntbot.Upgrader.Weights,
		},
	}

	f.Gamble = Gamble{
		AllottedAmount: old.Gamble.AllottedAmount,
		GoalSystem:     old.Gamble.GoalSystem,
		Coinflip: CoinflipSettings{
			GambleGame: legacyGame(old.Gamble.Coinflip.legacyGambleGame, def.Features.Gamble.Coinflip.GambleGame),
			Options:    old.Gamble.Coinflip.Options,
		},
		Slots:     legacyGame(old.Gamble.Slots, def.Features.Gamble.Slots),
		Blackjack: legacyGame(old.Gamble.Blackjack, def.Features.Gamble.Blackjack),
	}

	if key := old.OCRApi; key != "" && key != "helloworld" {
		notes = append(notes, "ocrApi moved out of the config file — set OCR_API_KEY=\""+key+"\" in .env and delete the old key")
	}

	return s, notes, nil
}

func legacyGame(old legacyGambleGame, def GambleGame) GambleGame {
	return GambleGame{
		Enabled:          old.Enabled,
		StartValue:       old.StartValue,
		MultiplierOnLose: old.MultiplierOnLose,
		Cooldown:         old.Cooldown.toRange(def.Cooldown),
	}
}

func millis(ms int, def Duration) Duration {
	if ms <= 0 {
		return def
	}
	return Duration(time.Duration(ms) * time.Millisecond)
}

func fixedMillis(ms int, def Range) Range {
	if ms <= 0 {
		return def
	}
	d := Duration(time.Duration(ms) * time.Millisecond)
	return Range{Min: d, Max: d}
}

// legacyDelay folds in the even older slowestTime/fastestTime spelling that
// normalizeDelays used to patch up at load time.
func legacyDelay(d legacyActionDelay, def Range) Range {
	min, max := d.MinDelay, d.MaxDelay
	if min == 0 && d.SlowestTime > 0 {
		min = d.SlowestTime
	}
	if max == 0 && d.FastestTime > 0 {
		max = d.FastestTime
	}
	out := def
	if min > 0 {
		out.Min = Duration(time.Duration(min) * time.Millisecond)
	}
	if max > 0 {
		out.Max = Duration(time.Duration(max) * time.Millisecond)
	}
	return out
}
