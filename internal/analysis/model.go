package analysis

import "time"

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const (
	ClassificationBrilliant  = "brilliant"
	ClassificationGreat      = "great"
	ClassificationBest       = "best"
	ClassificationGood       = "good"
	ClassificationInaccuracy = "inaccuracy"
	ClassificationMistake    = "mistake"
	ClassificationMiss       = "miss"
	ClassificationBlunder    = "blunder"
)

const (
	PhaseOpening    = "opening"
	PhaseMiddlegame = "middlegame"
	PhaseEndgame    = "endgame"
)

type ClassificationThresholds struct {
	Brilliant  int `json:"brilliant"`
	Great      int `json:"great"`
	Best       int `json:"best"`
	Good       int `json:"good"`
	Inaccuracy int `json:"inaccuracy"`
	Mistake    int `json:"mistake"`
	Miss       int `json:"miss"`
}

func DefaultThresholds() ClassificationThresholds {
	return ClassificationThresholds{
		Brilliant:  0,
		Great:      5,
		Best:       15,
		Good:       40,
		Inaccuracy: 90,
		Mistake:    180,
		Miss:       300,
	}
}

type GameAnalysis struct {
	ID                      string    `json:"id"`
	GameID                  string    `json:"game_id"`
	Status                  string    `json:"status"`
	AccuracyWhite           *float64  `json:"accuracy_white,omitempty"`
	AccuracyBlack           *float64  `json:"accuracy_black,omitempty"`
	RatingWhite             int       `json:"rating_white"`
	RatingBlack             int       `json:"rating_black"`
	EngineVersion           string    `json:"engine_version"`
	Depth                   int       `json:"depth"`
	BrilliantWhite          int       `json:"brilliant_white"`
	GreatWhite              int       `json:"great_white"`
	BestWhite               int       `json:"best_white"`
	GoodWhite               int       `json:"good_white"`
	InaccuraciesWhite       int       `json:"inaccuracies_white"`
	MistakesWhite           int       `json:"mistakes_white"`
	MissesWhite             int       `json:"misses_white"`
	BlundersWhite           int       `json:"blunders_white"`
	BrilliantBlack          int       `json:"brilliant_black"`
	GreatBlack              int       `json:"great_black"`
	BestBlack               int       `json:"best_black"`
	GoodBlack               int       `json:"good_black"`
	InaccuraciesBlack       int       `json:"inaccuracies_black"`
	MistakesBlack           int       `json:"mistakes_black"`
	MissesBlack             int       `json:"misses_black"`
	BlundersBlack           int       `json:"blunders_black"`
	OpeningAccuracyWhite    *float64  `json:"opening_accuracy_white,omitempty"`
	OpeningAccuracyBlack    *float64  `json:"opening_accuracy_black,omitempty"`
	MiddlegameAccuracyWhite *float64  `json:"middlegame_accuracy_white,omitempty"`
	MiddlegameAccuracyBlack *float64  `json:"middlegame_accuracy_black,omitempty"`
	EndgameAccuracyWhite    *float64  `json:"endgame_accuracy_white,omitempty"`
	EndgameAccuracyBlack    *float64  `json:"endgame_accuracy_black,omitempty"`
	ErrorMessage            string    `json:"error_message,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type GameMoveAnalysis struct {
	ID             string    `json:"id"`
	AnalysisID     string    `json:"analysis_id"`
	GameID         string    `json:"game_id"`
	MoveNumber     int       `json:"move_number"`
	PlayerColor    string    `json:"player_color"`
	PlayedMove     string    `json:"played_move"`
	PositionFEN    string    `json:"position_fen"`
	EvalBefore     *int      `json:"eval_before,omitempty"`
	EvalAfter      *int      `json:"eval_after,omitempty"`
	BestMove       string    `json:"best_move,omitempty"`
	PV             []string  `json:"pv,omitempty"`
	CentipawnLoss  int       `json:"centipawn_loss"`
	Classification string    `json:"classification"`
	Phase          string    `json:"phase"`
	Explanation    string    `json:"explanation"`
	CreatedAt      time.Time `json:"created_at"`
}
