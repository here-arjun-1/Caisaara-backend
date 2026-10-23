package bot

const (
	LevelEasy   = "easy"
	LevelMedium = "medium"
	LevelHard   = "hard"
	LevelExpert = "expert"
	LevelCustom = "custom"
)

const (
	MinRating  = 400
	MaxRating  = 3000
	minUCIElo  = 1320
	moveTimeMs = 500
)

type EngineSettings struct {
	LimitStrength bool
	Elo           int
	SkillLevel    int
	Depth         int
	MoveTimeMs    int
}

func SettingsForRating(rating int) EngineSettings {
	if rating >= minUCIElo {
		engineElo := rating - 200
		if engineElo < minUCIElo {
			engineElo = minUCIElo
		}

		return EngineSettings{
			LimitStrength: true,
			Elo:           engineElo,
			SkillLevel:    20,
			MoveTimeMs:    moveTimeMs,
		}
	}

	skill := (rating - MinRating) / 50
	depth := (rating-MinRating)/200 + 1

	return EngineSettings{
		SkillLevel: skill,
		Depth:      depth,
		MoveTimeMs: moveTimeMs,
	}
}

func MistakeChance(rating int) int {
	if rating >= 2400 {
		return 0
	}

	if rating >= minUCIElo {
		return (2400 - rating) * 10 / (2400 - minUCIElo)
	}

	if rating < MinRating {
		rating = MinRating
	}

	return 10 + (minUCIElo-rating)*30/(minUCIElo-MinRating)
}
