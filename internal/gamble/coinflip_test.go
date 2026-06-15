package gamble

import (
	"strings"
	"testing"
)

func TestCoinflipLoseRegex(t *testing.T) {
	msg := "**sempatpanick** spent **<:cowoncy:416043450337853441> 2,000** and chose **heads**\nThe coin spins... <:tail:436677926398853120> and you lost it all..."
	lower := strings.ToLower(msg)
	if !strings.Contains(lower, "and you lost it all") {
		t.Fatal("expected lost it all")
	}
	lose, ok := parseRegexAmount(cfLoseRe, msg)
	if !ok || lose != 2000 {
		t.Fatalf("want 2000, got %d ok=%v", lose, ok)
	}
}

func TestCoinflipWinRegex(t *testing.T) {
	msg := "**sempatpanick** spent **<:cowoncy:416043450337853441> 2,000** and chose **heads**\nThe coin spins... <:head:123> and you won **<:cowoncy:416043450337853441> 4,000**"
	won, ok1 := parseRegexAmount(cfWonRe, msg)
	lose, ok2 := parseRegexAmount(cfLoseRe, msg)
	if !ok1 || won != 4000 {
		t.Fatalf("want won 4000, got %d ok=%v", won, ok1)
	}
	if !ok2 || lose != 2000 {
		t.Fatalf("want lose 2000, got %d ok=%v", lose, ok2)
	}
}
