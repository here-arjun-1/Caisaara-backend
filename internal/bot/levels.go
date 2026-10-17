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
		return EngineSettings{
			LimitStrength: true,
			Elo:           rating,
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
