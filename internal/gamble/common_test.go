package gamble

import "testing"

func TestItoaPlainForCommands(t *testing.T) {
	if got := itoa(3000); got != "3000" {
		t.Fatalf("itoa(3000) = %q, want plain number for commands", got)
	}
	if got := itoa(250000); got != "250000" {
		t.Fatalf("itoa(250000) = %q, want no comma separator", got)
	}
}

func TestLogAmtFormattedForDisplay(t *testing.T) {
	if got := logAmt(114308); got != "114,308" {
		t.Fatalf("logAmt(114308) = %q, want formatted display", got)
	}
}
