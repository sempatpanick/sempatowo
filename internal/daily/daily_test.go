package daily

import (
	"strings"
	"testing"
)

func TestDailyMessagePatterns(t *testing.T) {
	nick := "nggaweangger"
	claim := "💰 | nggaweangger, Here is your daily 775 Cowoncy! | You're on a 7 daily streak! | You received a lootbox! ⏱️ | Your next daily is in: 22H 55M 35S"
	cooldown := "⏱ | Nu! nggaweangger! You need to wait 22H 49M 32S"
	other := "💰 | nggaweangger, you currently have 12,000 cowoncy!"

	if !dailyClaimRe.MatchString(claim) {
		t.Fatal("expected claim message to match dailyClaimRe")
	}
	if reward, ok := parseReward(claim); !ok || reward != 775 {
		t.Fatalf("parseReward(claim) = (%d, %v), want (775, true)", reward, ok)
	}
	if !dailyCooldownRe.MatchString(cooldown) {
		t.Fatal("expected cooldown message to match dailyCooldownRe")
	}
	if dailyClaimRe.MatchString(cooldown) || dailyCooldownRe.MatchString(claim) {
		t.Fatal("claim and cooldown patterns should not cross-match")
	}
	if dailyClaimRe.MatchString(other) || dailyCooldownRe.MatchString(other) {
		t.Fatal("non-daily cash message should not match daily patterns")
	}
	if !strings.Contains(claim, nick) || !strings.Contains(cooldown, nick) {
		t.Fatal("nick should be present in daily result messages")
	}
}
