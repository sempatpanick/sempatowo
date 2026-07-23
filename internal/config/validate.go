package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validate reports every problem it can find at once, rather than the first,
// so a user fixing a hand-edited file does not have to play whack-a-mole.
//
// Anything that would make the bot silently do nothing (an enabled feature with
// no channel), send nonsense (an inverted delay range), or lose money in a way
// the safety limits cannot catch (a gamble game with no bet) is an error here.
// Anything merely suspicious is a Warning instead.
func (s Settings) Validate() error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if s.SchemaVersion > SchemaVersion {
		add("schemaVersion %d is newer than this build understands (max %d) — upgrade sempatowo", s.SchemaVersion, SchemaVersion)
	}
	if strings.TrimSpace(s.Prefix) == "" {
		add("prefix must not be empty")
	}
	if !isSnowflake(s.OwoBotID) {
		add("owoBotId %q is not a Discord ID", s.OwoBotID)
	}
	if s.DefaultChannel != "" && !isSnowflake(s.DefaultChannel) {
		add("defaultChannel %q is not a Discord ID", s.DefaultChannel)
	}
	if s.SendMessageInterval <= 0 {
		add("sendMessageInterval must be positive, got %s", s.SendMessageInterval)
	}

	f := s.Features
	scheduled := []struct {
		name string
		feat ScheduledFeature
	}{
		{"hunt", f.Hunt},
		{"battle", f.Battle},
		{"pray", f.Pray.ScheduledFeature},
		{"curse", f.Curse.ScheduledFeature},
		{"zoo", f.Zoo},
		{"inventory", f.Inventory},
		{"checklist", f.Checklist.ScheduledFeature},
		{"quest", ScheduledFeature{Enabled: f.Quest.Enabled, Channel: f.Quest.Channel, Delay: f.Quest.Delay}},
	}
	for _, sc := range scheduled {
		if !sc.feat.Enabled {
			continue
		}
		if ch := s.ChannelFor(sc.feat); ch == "" {
			add("features.%s is enabled but no channel is set — set defaultChannel or features.%s.channel", sc.name, sc.name)
		} else if !isSnowflake(ch) {
			add("features.%s channel %q is not a Discord ID", sc.name, ch)
		}
		if err := checkRange(sc.feat.Delay); err != nil {
			add("features.%s.delay: %v", sc.name, err)
		}
	}

	for _, t := range []struct {
		name, id string
	}{
		{"pray", f.Pray.Target},
		{"curse", f.Curse.Target},
		{"cookie", f.Cookie.Target},
	} {
		if t.id != "" && !isSnowflake(t.id) {
			add("features.%s.target %q is not a Discord ID", t.name, t.id)
		}
	}

	if f.Quest.Enabled {
		if err := checkRange(f.Quest.OwoDelay); err != nil {
			add("features.quest.owoDelay: %v", err)
		}
	}
	if f.Quest.Auto.Enabled {
		if err := checkRange(f.Quest.Auto.CheckCooldown); err != nil {
			add("features.quest.auto.checkCooldown: %v", err)
		}
		hc := f.Quest.Auto.HelpChannel
		if hc.PostInHelpChannel && hc.ChannelID == "" {
			add("features.quest.auto.helpChannel.postInHelpChannel is on but channelId is empty")
		}
		if hc.ChannelID != "" && !isSnowflake(hc.ChannelID) {
			add("features.quest.auto.helpChannel.channelId %q is not a Discord ID", hc.ChannelID)
		}
	}

	if f.Huntbot.Enabled && f.Huntbot.CashToSpend <= 0 {
		add("features.huntbot.cashToSpend must be positive when huntbot is enabled, got %d", f.Huntbot.CashToSpend)
	}
	if f.Huntbot.Enabled && f.Huntbot.Upgrader.Enabled {
		u := f.Huntbot.Upgrader
		if err := checkRange(u.Cooldown); err != nil {
			add("features.huntbot.upgrader.cooldown: %v", err)
		}
		if !anyTrait(u.Traits) {
			add("features.huntbot.upgrader is enabled but every trait is off — nothing to upgrade")
		}
		for _, w := range []struct {
			name string
			on   bool
			val  float64
		}{
			{"efficiency", u.Traits.Efficiency, u.Weights.Efficiency},
			{"duration", u.Traits.Duration, u.Weights.Duration},
			{"cost", u.Traits.Cost, u.Weights.Cost},
			{"gain", u.Traits.Gain, u.Weights.Gain},
			{"exp", u.Traits.Exp, u.Weights.Exp},
			{"radar", u.Traits.Radar, u.Weights.Radar},
		} {
			if w.val < 0 {
				add("features.huntbot.upgrader.weights.%s must not be negative, got %g", w.name, w.val)
			}
			if w.on && w.val == 0 {
				add("features.huntbot.upgrader trait %s is on but its weight is 0 — it will never be picked", w.name)
			}
		}
	}

	g := f.Gamble
	if s.AnyGambleEnabled() {
		if !s.TrackBalance {
			add("gambling is enabled but trackBalance is off — the allottedAmount and goal limits cannot be enforced without it")
		}
		if g.AllottedAmount <= 0 {
			add("features.gamble.allottedAmount must be positive when a game is enabled, got %d", g.AllottedAmount)
		}
		if g.GoalSystem.Enabled && g.GoalSystem.Amount <= 0 {
			add("features.gamble.goalSystem.amount must be positive when the goal system is on, got %d", g.GoalSystem.Amount)
		}
	}
	for _, game := range []struct {
		name string
		g    GambleGame
	}{
		{"coinflip", g.Coinflip.GambleGame},
		{"slots", g.Slots},
		{"blackjack", g.Blackjack},
	} {
		if !game.g.Enabled {
			continue
		}
		if game.g.StartValue <= 0 {
			add("features.gamble.%s.startValue must be positive, got %d", game.name, game.g.StartValue)
		}
		if game.g.MultiplierOnLose < 1 {
			add("features.gamble.%s.multiplierOnLose must be at least 1, got %g", game.name, game.g.MultiplierOnLose)
		}
		if err := checkRange(game.g.Cooldown); err != nil {
			add("features.gamble.%s.cooldown: %v", game.name, err)
		}
	}
	if g.Coinflip.Enabled && !g.Coinflip.Options.Heads && !g.Coinflip.Options.Tails {
		add("features.gamble.coinflip is enabled but both heads and tails are false — there is no side to bet on")
	}

	return errors.Join(errs...)
}

// Warnings returns settings that are legal but probably not what was intended.
func (s Settings) Warnings() []string {
	var out []string
	f := s.Features

	if f.Quest.Auto.Enabled && !f.Quest.Auto.AcknowledgeExperimental {
		out = append(out, "features.quest.auto is enabled but acknowledgeExperimental is false — auto-quest will not run")
	}
	if !s.TrackBalance && f.Daily.Enabled {
		out = append(out, "trackBalance is off — daily rewards will not be counted towards the gamble budget")
	}
	if f.Gems.Enabled && !f.Inventory.Enabled {
		out = append(out, "features.gems is on but features.inventory is off — gem stock is only learned from inventory checks")
	}
	if f.Cookie.Enabled && !f.Checklist.Enabled {
		out = append(out, "features.cookie is on but features.checklist is off — cookies are sent from the checklist reply")
	}
	for _, sc := range []struct {
		name  string
		delay Range
		on    bool
	}{
		{"hunt", f.Hunt.Delay, f.Hunt.Enabled},
		{"battle", f.Battle.Delay, f.Battle.Enabled},
	} {
		if sc.on && sc.delay.Min == sc.delay.Max {
			out = append(out, "features."+sc.name+".delay has min == max — a perfectly periodic command is easy to spot; consider a range")
		}
	}
	return out
}

func checkRange(r Range) error {
	if r.Min <= 0 {
		return fmt.Errorf("min must be positive, got %s", r.Min)
	}
	if r.Max < r.Min {
		return fmt.Errorf("max (%s) must not be less than min (%s)", r.Max, r.Min)
	}
	return nil
}

func anyTrait(t HuntbotTraits) bool {
	return t.Efficiency || t.Duration || t.Cost || t.Gain || t.Exp || t.Radar
}

// isSnowflake reports whether id looks like a Discord ID: digits only, and long
// enough to be a real snowflake.
func isSnowflake(id string) bool {
	if len(id) < 15 || len(id) > 21 {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
