package gamble

import (
	"strings"
	"testing"
)

func TestSlotsLoseRegex(t *testing.T) {
	msg := "**  `___SLOTS___`**\n` ` <:heart:417475705899712522> <:cowoncy:417475705912426496> <:heart:417475705899712522> ` ` sempatpanick bet <:cowoncy:416043450337853441> 3,000\n  `|         |`   and won nothing... :c\n  `|         |`"
	bet, ok := parseRegexAmount(slotsBetRe, msg)
	if !ok || bet != 3000 {
		t.Fatalf("want bet 3000, got %d ok=%v", bet, ok)
	}
	if !strings.Contains(strings.ToLower(msg), "and won nothing") {
		t.Fatal("expected lose message")
	}
}

func TestSlotsWinRegex(t *testing.T) {
	msg := "**  `___SLOTS___`**\n` ` <:heart:417475705899712522> <:heart:417475705899712522> <:heart:417475705899712522> ` ` sempatpanick bet <:cowoncy:416043450337853441> 3,000\n  `|         |`   and won <:cowoncy:416043450337853441> 6,000\n  `|         |`"
	won, ok1 := parseRegexAmount(slotsWonRe, msg)
	bet, ok2 := parseRegexAmount(slotsBetRe, msg)
	if !ok1 || won != 6000 {
		t.Fatalf("want won 6000, got %d ok=%v", won, ok1)
	}
	if !ok2 || bet != 3000 {
		t.Fatalf("want bet 3000, got %d ok=%v", bet, ok2)
	}
}

func TestSlotsDrawRegex(t *testing.T) {
	msg := "**  `___SLOTS___`**\n` ` sempatpanick bet <:cowoncy:416043450337853441> 3,000\n  `|         |`   and won <:cowoncy:416043450337853441> 3,000\n  `|         |`"
	won, ok1 := parseRegexAmount(slotsWonRe, msg)
	bet, ok2 := parseRegexAmount(slotsBetRe, msg)
	if !ok1 || !ok2 || won != bet {
		t.Fatalf("want draw 3000/3000, got won=%d bet=%d ok=%v/%v", won, bet, ok1, ok2)
	}
}
