package quest

import "testing"

func TestLookupQuest(t *testing.T) {
	tests := []struct {
		text     string
		wantID   string
		helpable bool
	}{
		{"receive curses", "curse", true},
		{"Receive Prayers", "pray", true},
		{"  battle with a friend  ", "battle_friend", true},
		{"manually hunt", "hunt", false},
		{"say owo", "owo", false},
		// Rank variants are generated in init rather than listed by hand.
		{"find fabled animals", "find_animal_fabled", false},
		{"find common animals", "find_animal_common", false},
	}

	for _, tt := range tests {
		meta, ok := LookupQuest(tt.text)
		if !ok {
			t.Errorf("LookupQuest(%q) not found", tt.text)
			continue
		}
		if meta.ID != tt.wantID {
			t.Errorf("LookupQuest(%q).ID = %q, want %q", tt.text, meta.ID, tt.wantID)
		}
		if meta.Helpable != tt.helpable {
			t.Errorf("LookupQuest(%q).Helpable = %v, want %v", tt.text, meta.Helpable, tt.helpable)
		}
	}
}

// An unrecognised quest must report not-found rather than a zero QuestMeta that
// reads as a real quest with an empty ID.
func TestLookupQuestUnknown(t *testing.T) {
	for _, text := range []string{"", "become a millionaire", "find sparkly animals"} {
		if meta, ok := LookupQuest(text); ok {
			t.Errorf("LookupQuest(%q) = %+v, want not found", text, meta)
		}
	}
}

// Every rank gets its own quest ID; a collision would make two quests
// indistinguishable in the progress map.
func TestAnimalRankQuestIDsAreDistinct(t *testing.T) {
	seen := make(map[string]string)
	for _, rank := range animalRanks {
		meta, ok := LookupQuest("find " + rank + " animals")
		if !ok {
			t.Fatalf("rank %q has no quest entry", rank)
		}
		if prev, dup := seen[meta.ID]; dup {
			t.Errorf("ranks %q and %q share the ID %q", prev, rank, meta.ID)
		}
		seen[meta.ID] = rank
	}
}
