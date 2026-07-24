package huntbot

import (
	"testing"

	"github.com/semptpanick/sempatowo/internal/config"
)

type fakeBot struct {
	settings  config.Huntbot
	slept     []float64
	noise     []float64
	sent      []string
	cancelled bool
}

func (f *fakeBot) HuntChannelID() string             { return "hunt" }
func (f *fakeBot) OwoBotID() string                  { return "owo" }
func (f *fakeBot) OwnUserID() string                 { return "111" }
func (f *fakeBot) Nickname() string                  { return "tester" }
func (f *fakeBot) Settings() config.Huntbot          { return f.settings }
func (f *fakeBot) RandomPrefix(cmds []string) string { return "owo " + cmds[0] }
func (f *fakeBot) CanSend() bool                     { return true }
func (f *fakeBot) Log(string)                        {}
func (f *fakeBot) Sleep(float64)                     {}
func (f *fakeBot) CancelSleep()                      { f.cancelled = true }

func (f *fakeBot) SendMessage(channelID, text string) error {
	f.sent = append(f.sent, text)
	return nil
}

// SleepUntil records the requested delay and returns immediately, as if the
// timer had elapsed.
func (f *fakeBot) SleepUntil(seconds, noise float64) bool {
	f.slept = append(f.slept, seconds)
	f.noise = append(f.noise, noise)
	return true
}

func newFake() *fakeBot {
	return &fakeBot{settings: config.Huntbot{Enabled: true, CashToSpend: 100}}
}

func hbEmbed(fields ...EmbedField) Message {
	return Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Embeds: []MessageEmbed{{
			Author: &EmbedAuthor{Name: "tester's HuntBot"},
			Fields: fields,
		}},
	}
}

func TestResendFollowsRemainingTimePlusPadding(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	fields := make([]EmbedField, 8)
	fields = append(fields, EmbedField{
		Name:  "HUNTBOT is currently hunting!",
		Value: "`2H 30M` remaining!",
	})
	h.HandleMessage(hbEmbed(fields...))

	want := float64(2*3600+30*60) + resendPadding // 9003
	if len(f.slept) == 0 || f.slept[0] != want {
		t.Fatalf("slept = %v, want first delay of %v", f.slept, want)
	}
	if f.noise[0] != 0 {
		t.Fatalf("noise = %v, want 0 so the delay tracks the reported time exactly", f.noise[0])
	}
	wantPair(t, f.sent)
}

func TestResendFollowsBackInMessage(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "tester I WILL BE BACK IN **1D 3H 20M**",
	})

	want := float64(86400+3*3600+20*60) + resendPadding // 96003
	if len(f.slept) == 0 || f.slept[0] != want {
		t.Fatalf("slept = %v, want first delay of %v", f.slept, want)
	}
	wantPair(t, f.sent)
}

// The completion push re-opens HuntBot even though it carries no nick and, in
// this case, no ping at all: channel+author already scope it to this account.
func TestReturnPushResendsWithAmount(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "BEEP BOOP. I AM BACK WITH 598 ANIMALS, 4725 ESSENCE, AND 5512 EXPERIENCE",
	})

	if len(f.slept) == 0 || f.slept[0] != briefCooldownMin {
		t.Fatalf("slept = %v, want first delay of %v", f.slept, briefCooldownMin)
	}
	wantPair(t, f.sent)
}

// A push arriving inside the pairEcho window is the answer to a resend that
// already went out (the remaining-time timer beat the push to it), so it must
// not queue a second pair.
func TestReturnPushAfterPairDoesNotResend(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "tester I WILL BE BACK IN **10M**",
	})
	wantPair(t, f.sent)

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "BEEP BOOP. I AM BACK WITH 1 ANIMALS, 1 ESSENCE, AND 1 EXPERIENCE",
	})
	wantPair(t, f.sent)
}

// A push whose only ping is another account must not cross-trigger a resend
// when several accounts share one hunt channel.
func TestReturnPushForAnotherAccountIgnored(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "<@999> BEEP BOOP. I AM BACK WITH 1 ANIMALS, 1 ESSENCE, AND 1 EXPERIENCE",
	})

	if len(f.sent) != 0 {
		t.Fatalf("sent = %v, want nothing for another account's return push", f.sent)
	}
}

// The bare command only prints HuntBot's status, so a resend that stops there
// leaves the farm idle: the amount has to follow it.
func wantPair(t *testing.T, sent []string) {
	t.Helper()
	if len(sent) != 2 || sent[0] != "owo huntbot" || sent[1] != "owo huntbot 100" {
		t.Fatalf("sent = %v, want the bare command followed by the amount", sent)
	}
}

// The status embed answering a paired resend must not queue a third send.
func TestStatusEmbedAfterPairDoesNotResend(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "tester I WILL BE BACK IN **10M**",
	})
	wantPair(t, f.sent)

	h.HandleMessage(hbEmbed(EmbedField{Name: "Efficiency", Value: "Lvl 1 [0/10]"}))
	wantPair(t, f.sent)
}

// A status embed with no resend behind it is the answer to Start()'s bare
// probe, and still drives the amount on its own.
func TestStatusEmbedOnItsOwnSendsAmount(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(hbEmbed(EmbedField{Name: "Efficiency", Value: "Lvl 1 [0/10]"}))
	if len(f.sent) != 1 || f.sent[0] != "owo huntbot 100" {
		t.Fatalf("sent = %v, want the amount after the status embed", f.sent)
	}
}

func TestUnparsableRemainingFallsBackToShortRetry(t *testing.T) {
	f := newFake()
	h := NewHandler(f, "token")

	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "tester I WILL BE BACK IN **soon**",
	})

	if len(f.slept) == 0 || f.slept[0] != briefCooldownMin {
		t.Fatalf("slept = %v, want fallback of %v rather than a 3s hammer", f.slept, briefCooldownMin)
	}
	if f.noise[0] == 0 {
		t.Fatal("fallback retry should keep its jitter")
	}
}

func TestCancelledWaitDoesNotSend(t *testing.T) {
	c := &cancelBot{fakeBot: newFake()}
	h := NewHandler(c, "token")
	h.HandleMessage(Message{
		ChannelID: "hunt",
		AuthorID:  "owo",
		Content:   "tester I WILL BE BACK IN **10M**",
	})

	if len(c.sent) != 0 {
		t.Fatalf("sent = %v, want nothing sent when the wait is cancelled", c.sent)
	}
}

// cancelBot simulates CancelSleep firing during the wait.
type cancelBot struct{ *fakeBot }

func (c *cancelBot) SleepUntil(seconds, noise float64) bool {
	c.slept = append(c.slept, seconds)
	c.noise = append(c.noise, noise)
	return false
}
