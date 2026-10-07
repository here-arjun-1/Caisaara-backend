package bot

const (
	LevelEasy   = "easy"
	LevelMedium = "medium"
	LevelHard   = "hard"
	LevelExpert = "expert"
	LevelCustom = "custom"
)

const (
	MinRating  = 1000
	MaxRating  = 3000
	minUCIElo  = 1320
	moveTimeMs = 500
)

var presetRatings = map[string]int{
	LevelEasy:   1000,
	LevelMedium: 1500,
	LevelHard:   2000,
	LevelExpert: 2500,
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

	step := (rating - MinRating) / 64

	return EngineSettings{
		SkillLevel: step,
		Depth:      step + 1,
		MoveTimeMs: moveTimeMs,
	}
}
