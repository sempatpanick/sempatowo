package farm

import (
	"testing"

	discord "github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/types"
)

// nickname must never return the message author's member nick: every message
// the bot routes is authored by OwO, so msg.Member is OwO's member. Trusting it
// made nick resolve to OwO's server nick, which then failed to match our own
// embed author ("<us> goes into battle!") and dropped battle/quest summaries.
func TestNicknameIgnoresAuthorMember(t *testing.T) {
	var b Bot // nil client: no self member to resolve, so it must fall to username ("")
	msg := &discord.Message{
		GuildID: 123,
		Member:  &types.Member{Nick: "OwO"},
	}
	if got := b.nickname(msg); got == "OwO" {
		t.Fatalf("nickname returned the author's (OwO's) member nick %q", got)
	}
}
