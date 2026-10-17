package bot

type Character struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Level  string `json:"level"`
	Order  int    `json:"order"`
	Rating int    `json:"rating"`
}

type BotInfo struct {
	Character
	Unlocked bool `json:"unlocked"`
	Beaten   bool `json:"beaten"`
}

var characters = []Character{
	{ID: "easy_1", Name: "Martin", Level: LevelEasy, Order: 1, Rating: 400},
	{ID: "easy_2", Name: "Wayne", Level: LevelEasy, Order: 2, Rating: 500},
	{ID: "easy_3", Name: "Fabian", Level: LevelEasy, Order: 3, Rating: 600},
	{ID: "easy_4", Name: "Juan", Level: LevelEasy, Order: 4, Rating: 700},

	{ID: "medium_1", Name: "Maria", Level: LevelMedium, Order: 1, Rating: 800},
	{ID: "medium_2", Name: "Maxim", Level: LevelMedium, Order: 2, Rating: 950},
	{ID: "medium_3", Name: "Hans", Level: LevelMedium, Order: 3, Rating: 1100},
	{ID: "medium_4", Name: "Azeez", Level: LevelMedium, Order: 4, Rating: 1200},

	{ID: "hard_1", Name: "Wendy", Level: LevelHard, Order: 1, Rating: 1350},
	{ID: "hard_2", Name: "Antonio", Level: LevelHard, Order: 2, Rating: 1500},
	{ID: "hard_3", Name: "Pierre", Level: LevelHard, Order: 3, Rating: 1650},
	{ID: "hard_4", Name: "Pablo", Level: LevelHard, Order: 4, Rating: 1800},

	{ID: "expert_1", Name: "Nora", Level: LevelExpert, Order: 1, Rating: 2100},
	{ID: "expert_2", Name: "Noam", Level: LevelExpert, Order: 2, Rating: 2400},
	{ID: "expert_3", Name: "Ahmed", Level: LevelExpert, Order: 3, Rating: 2700},
	{ID: "expert_4", Name: "Sakura", Level: LevelExpert, Order: 4, Rating: 3000},
}

func FindCharacter(id string) (Character, bool) {
	for _, character := range characters {
		if character.ID == id {
			return character, true
		}
	}
	return Character{}, false
}

func NextCharacter(id string) (Character, bool) {
	for i, character := range characters {
		if character.ID == id && i+1 < len(characters) {
			return characters[i+1], true
		}
	}
	return Character{}, false
}

func IsUnlocked(id string, beatenBots map[string]bool) bool {
	for i, character := range characters {
		if character.ID != id {
			continue
		}

		if character.Order == 1 {
			return true
		}

		previous := characters[i-1]
		return beatenBots[previous.ID]
	}
	return false
}
