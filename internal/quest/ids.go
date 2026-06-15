package quest

import "strings"

type QuestMeta struct {
	ID       string
	Helpable bool
}

var questByText = map[string]QuestMeta{
	"receive an action from a friend": {ID: "action_receive", Helpable: true},
	"receive curses":                  {ID: "curse", Helpable: true},
	"receive prayers":                 {ID: "pray", Helpable: true},
	"receive cookies":                 {ID: "cookie", Helpable: true},
	"battle with a friend":            {ID: "battle_friend", Helpable: true},
	"earn battle xp":                  {ID: "battle_xp", Helpable: false},
	"gamble your cowoncy":             {ID: "gamble", Helpable: false},
	"defeat bosses":                   {ID: "boss", Helpable: false},
	"send an action to a friend":      {ID: "action_send", Helpable: false},
	"manually hunt":                   {ID: "hunt", Helpable: false},
	"battle":                          {ID: "battle", Helpable: false},
	"say owo":                         {ID: "owo", Helpable: false},
}

var animalRanks = []string{
	"common", "uncommon", "rare", "epic", "special",
	"mythical", "gem", "legendary", "fabled", "distorted", "hidden",
}

func init() {
	for _, rank := range animalRanks {
		questByText["find "+rank+" animals"] = QuestMeta{ID: "find_animal_" + rank, Helpable: false}
	}
}

func LookupQuest(text string) (QuestMeta, bool) {
	meta, ok := questByText[strings.ToLower(strings.TrimSpace(text))]
	return meta, ok
}
