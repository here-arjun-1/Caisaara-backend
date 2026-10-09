package analysis

import "time"

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const (
	ClassificationBest       = "best"
	ClassificationGood       = "good"
	ClassificationInaccuracy = "inaccuracy"
	ClassificationMistake    = "mistake"
	ClassificationBlunder    = "blunder"
)

type ClassificationThresholds struct {
	Best       int `json:"best"`
	Good       int `json:"good"`
	Inaccuracy int `json:"inaccuracy"`
	Mistake    int `json:"mistake"`
}

func DefaultThresholds() ClassificationThresholds {
	return ClassificationThresholds{
		Best:       15,
		Good:       50,
		Inaccuracy: 100,
		Mistake:    200,
	}
}

type GameAnalysis struct {
	ID                string    `json:"id"`
	GameID            string    `json:"game_id"`
	Status            string    `json:"status"`
	AccuracyWhite     *float64  `json:"accuracy_white,omitempty"`
	AccuracyBlack     *float64  `json:"accuracy_black,omitempty"`
	EngineVersion     string    `json:"engine_version"`
	Depth             int       `json:"depth"`
	InaccuraciesWhite int       `json:"inaccuracies_white"`
	MistakesWhite     int       `json:"mistakes_white"`
	BlundersWhite     int       `json:"blunders_white"`
	InaccuraciesBlack int       `json:"inaccuracies_black"`
	MistakesBlack     int       `json:"mistakes_black"`
	BlundersBlack     int       `json:"blunders_black"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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
	CreatedAt      time.Time `json:"created_at"`
}
