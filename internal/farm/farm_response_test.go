package farm

import (
	"testing"

	discord "github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/types"
)

func TestFarmCmdsFromMessageHunt(t *testing.T) {
	cmds := farmCmdsFromMessage(nil, "**test** You found: :dog: dog", "test")
	if len(cmds) != 1 || cmds[0] != "hunt" {
		t.Fatalf("got %v, want [hunt]", cmds)
	}
}

func TestFarmCmdsFromMessageBattle(t *testing.T) {
	msg := &discord.Message{
		Embeds: []*discord.Embed{{
			Author: &types.EmbedAuthor{Name: "test goes into battle!"},
		}},
	}
	cmds := farmCmdsFromMessage(msg, "", "test")
	if len(cmds) != 1 || cmds[0] != "battle" {
		t.Fatalf("got %v, want [battle]", cmds)
	}
}

func TestFarmCmdsFromMessageQuestLog(t *testing.T) {
	msg := &discord.Message{
		Embeds: []*discord.Embed{{
			Author: &types.EmbedAuthor{Name: "test's Quest Log"},
		}},
	}
	cmds := farmCmdsFromMessage(msg, "", "test")
	if len(cmds) != 1 || cmds[0] != "quest" {
		t.Fatalf("got %v, want [quest]", cmds)
	}
}

func TestIsOwoCommandResponse(t *testing.T) {
	if !isOwoCommandResponse("Rank #1,234 • testuser", "testuser") {
		t.Fatal("rank message should count as owo response")
	}
}

func TestSignalFarmResponseReschedules(t *testing.T) {
	b := New("token")
	b.active = true
	b.ready = true
	b.farmAwaiting = map[string]struct{}{"hunt": {}}
	b.cmdSchedStop = make(chan struct{})
	b.cfg = nil // farmCmdByName enabled check uses settings

	// Without config, enabled checks may fail — test pending clear only.
	b.signalFarmResponse("hunt")
	if _, ok := b.farmAwaiting["hunt"]; ok {
		t.Fatal("expected hunt pending cleared")
	}
}
