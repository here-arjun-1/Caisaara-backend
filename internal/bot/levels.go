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

var presetRatings = map[string]int{
	LevelEasy:   400,
	LevelMedium: 800,
	LevelHard:   1350,
	LevelExpert: 2100,
}

func ResolveRating(level string, customRating int) (int, error) {
	if level == LevelCustom {
		if customRating < MinRating || customRating > MaxRating {
			return 0, ErrInvalidRating
		}
		return customRating, nil
	}

	rating, ok := presetRatings[level]
	if !ok {
		return 0, ErrInvalidLevel
	}

	return rating, nil
}

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
