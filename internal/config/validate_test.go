package config

import (
	"strings"
	"testing"
)

func TestDefaultsValidate(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("the built-in defaults do not validate: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Settings)
		want string
	}{
		{
			name: "empty prefix",
			mut:  func(s *Settings) { s.Prefix = "" },
			want: "prefix",
		},
		{
			name: "owo id is not a snowflake",
			mut:  func(s *Settings) { s.OwoBotID = "owo" },
			want: "owoBotId",
		},
		{
			name: "zero send interval",
			mut:  func(s *Settings) { s.SendMessageInterval = 0 },
			want: "sendMessageInterval",
		},
		{
			// The most common silent failure: a feature is on but nothing tells
			// the bot where to send the command.
			name: "enabled feature with no channel",
			mut: func(s *Settings) {
				s.DefaultChannel = ""
				s.Features.Hunt.Enabled = true
			},
			want: "no channel is set",
		},
		{
			name: "inverted delay range",
			mut: func(s *Settings) {
				s.Features.Hunt.Delay = Range{Min: secs(30), Max: secs(5)}
			},
			want: "features.hunt.delay",
		},
		{
			name: "zero delay",
			mut: func(s *Settings) {
				s.Features.Battle.Delay = Range{}
			},
			want: "features.battle.delay",
		},
		{
			name: "target that is not a snowflake",
			mut:  func(s *Settings) { s.Features.Pray.Target = "@someone" },
			want: "features.pray.target",
		},
		{
			name: "coinflip with no side",
			mut: func(s *Settings) {
				s.Features.Gamble.Coinflip.Enabled = true
				s.Features.Gamble.Coinflip.Options = CoinflipOptions{}
			},
			want: "no side to bet on",
		},
		{
			name: "gamble with no stake",
			mut: func(s *Settings) {
				s.Features.Gamble.Slots.Enabled = true
				s.Features.Gamble.Slots.StartValue = 0
			},
			want: "startValue",
		},
		{
			name: "multiplier below one would shrink the martingale",
			mut: func(s *Settings) {
				s.Features.Gamble.Slots.Enabled = true
				s.Features.Gamble.Slots.MultiplierOnLose = 0.5
			},
			want: "multiplierOnLose",
		},
		{
			// Without balance tracking the allotted amount cannot be enforced,
			// so this combination spends without a ceiling.
			name: "gambling without balance tracking",
			mut: func(s *Settings) {
				s.Features.Gamble.Blackjack.Enabled = true
				s.TrackBalance = false
			},
			want: "trackBalance",
		},
		{
			name: "upgrader with every trait off",
			mut: func(s *Settings) {
				s.Features.Huntbot.Enabled = true
				s.Features.Huntbot.Upgrader.Traits = HuntbotTraits{}
			},
			want: "every trait is off",
		},
		{
			name: "enabled trait with zero weight never gets picked",
			mut: func(s *Settings) {
				s.Features.Huntbot.Enabled = true
				s.Features.Huntbot.Upgrader.Weights.Cost = 0
			},
			want: "weight is 0",
		},
		{
			name: "huntbot with nothing to spend",
			mut: func(s *Settings) {
				s.Features.Huntbot.Enabled = true
				s.Features.Huntbot.CashToSpend = 0
			},
			want: "cashToSpend",
		},
		{
			name: "help channel on with no channel id",
			mut: func(s *Settings) {
				s.Features.Quest.Auto.Enabled = true
				s.Features.Quest.Auto.HelpChannel.PostInHelpChannel = true
			},
			want: "channelId is empty",
		},
		{
			name: "config from a newer build",
			mut:  func(s *Settings) { s.SchemaVersion = SchemaVersion + 1 },
			want: "newer than this build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Defaults()
			tt.mut(&s)

			err := s.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want an error mentioning %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

// Validate reports everything at once so a hand-edited file can be fixed in one
// pass instead of one error per run.
func TestValidateReportsAllProblems(t *testing.T) {
	s := Defaults()
	s.Prefix = ""
	s.OwoBotID = "nope"
	s.SendMessageInterval = 0

	err := s.Validate()
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"prefix", "owoBotId", "sendMessageInterval"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q: %v", want, err)
		}
	}
}

// A disabled feature is not held to the same standard — you should be able to
// leave a half-filled block behind while it is switched off.
func TestValidateIgnoresDisabledFeatures(t *testing.T) {
	s := Defaults()
	s.Features.Zoo.Enabled = false
	s.Features.Zoo.Delay = Range{}
	s.Features.Gamble.Slots.Enabled = false
	s.Features.Gamble.Slots.StartValue = 0

	if err := s.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for disabled features", err)
	}
}

func TestWarnings(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Settings)
		want string
	}{
		{
			name: "auto quest not acknowledged",
			mut:  func(s *Settings) { s.Features.Quest.Auto.Enabled = true },
			want: "acknowledgeExperimental",
		},
		{
			name: "gems without inventory checks",
			mut:  func(s *Settings) { s.Features.Inventory.Enabled = false },
			want: "features.gems",
		},
		{
			name: "fixed hunt delay is a fingerprint",
			mut:  func(s *Settings) { s.Features.Hunt.Delay = Range{Min: secs(20), Max: secs(20)} },
			want: "perfectly periodic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Defaults()
			tt.mut(&s)

			joined := strings.Join(s.Warnings(), "\n")
			if !strings.Contains(joined, tt.want) {
				t.Errorf("Warnings() = %v, want one mentioning %q", s.Warnings(), tt.want)
			}
			// A warning must never be an error.
			if err := s.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil — this case should only warn", err)
			}
		})
	}
}

// HuntBot is its own OwO feature, not an alternative implementation of hunting:
// it runs on essence and reports back on its own schedule while `owo hunt`
// keeps working. The two only share the outgoing queue, so having both on is a
// normal setup and must not warn.
func TestWarningsAllowsHuntAndHuntbotTogether(t *testing.T) {
	s := Defaults()
	s.Features.Hunt.Enabled = true
	s.Features.Huntbot.Enabled = true

	for _, w := range s.Warnings() {
		if strings.Contains(w, "huntbot") {
			t.Errorf("Warnings() = %q, want nothing about huntbot vs hunt", w)
		}
	}
}

func TestIsSnowflake(t *testing.T) {
	valid := []string{"408785106942164992", "1513744333579489310", "123456789012345"}
	invalid := []string{"", "123", "12345678901234567890123", "12345678901234a", "<@408785106942164992>"}

	for _, id := range valid {
		if !isSnowflake(id) {
			t.Errorf("isSnowflake(%q) = false, want true", id)
		}
	}
	for _, id := range invalid {
		if isSnowflake(id) {
			t.Errorf("isSnowflake(%q) = true, want false", id)
		}
	}
}
