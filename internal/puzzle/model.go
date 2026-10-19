package puzzle

import (
	"time"
)

type Puzzle struct {
	ID              string    `json:"id"`
	FEN             string    `json:"fen"`
	Moves           string    `json:"moves"`
	Rating          int       `json:"rating"`
	RatingDeviation int       `json:"rating_deviation"`
	Popularity      int       `json:"popularity"`
	NbPlays         int       `json:"nb_plays"`
	Themes          []string  `json:"themes"`
	GameURL         string    `json:"game_url,omitempty"`
	OpeningTags     string    `json:"opening_tags,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type UserPuzzleRating struct {
	UserID           int64     `json:"user_id"`
	Rating           int       `json:"rating"`
	RatingDeviation  float64   `json:"rating_deviation"`
	RatingVolatility float64   `json:"rating_volatility"`
	PuzzlesSolved    int       `json:"puzzles_solved"`
	PuzzlesFailed    int       `json:"puzzles_failed"`
	BestRating       int       `json:"best_rating"`
	BestRatingAt     time.Time `json:"best_rating_at"`
	CurrentStreak    int       `json:"current_streak"`
	MaxStreak        int       `json:"max_streak"`
	LastPlayedDate   string    `json:"last_played_date,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UserPuzzleAttempt struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	PuzzleID         string    `json:"puzzle_id"`
	Solved           bool      `json:"solved"`
	UserRatingBefore int       `json:"user_rating_before"`
	UserRatingAfter  int       `json:"user_rating_after"`
	TimeTakenMs      int       `json:"time_taken_ms"`
	AttemptedMoves   []string  `json:"attempted_moves"`
	CreatedAt        time.Time `json:"created_at"`
}

type DailyPuzzleSession struct {
	UserID         int64    `json:"user_id"`
	PuzzleID       string   `json:"puzzle_id"`
	Date           string   `json:"date"`
	AttemptsLeft   int      `json:"attempts_left"`
	HintUsed       bool     `json:"hint_used"`
	SolutionViewed bool     `json:"solution_viewed"`
	Solved         bool     `json:"solved"`
	RatingEarned   int      `json:"rating_earned"`
	AttemptedMoves []string `json:"attempted_moves"`
}

type DailyPuzzleResponse struct {
	PuzzleID       string   `json:"puzzle_id"`
	FEN            string   `json:"fen"`
	InitialMove    string   `json:"initial_move"`
	Rating         int      `json:"rating"`
	Themes         []string `json:"themes"`
	AttemptsLeft   int      `json:"attempts_left"`
	HintAvailable  bool     `json:"hint_available"`
	SolutionViewed bool     `json:"solution_viewed"`
	Solved         bool     `json:"solved"`
	CurrentStreak  int      `json:"current_streak"`
	PlayerRating   int      `json:"player_rating"`
	AttemptedMoves []string `json:"attempted_moves"`
}

type SubmitAttemptRequest struct {
	PuzzleID    string `json:"puzzle_id" binding:"required"`
	Move        string `json:"move" binding:"required"`
	TimeTakenMs int    `json:"time_taken_ms"`
}

type SubmitAttemptResponse struct {
	Correct          bool     `json:"correct"`
	Solved           bool     `json:"solved"`
	NextMove         string   `json:"next_move,omitempty"`
	AttemptsLeft     int      `json:"attempts_left"`
	RatingChange     int      `json:"rating_change"`
	UserRatingBefore int      `json:"user_rating_before"`
	UserRatingAfter  int      `json:"user_rating_after"`
	CurrentStreak    int      `json:"current_streak"`
	AttemptedMoves   []string `json:"attempted_moves"`
	Message          string   `json:"message"`
}

type HintRequest struct {
	PuzzleID string `json:"puzzle_id" binding:"required"`
}

type HintResponse struct {
	HintType   string `json:"hint_type"`
	FromSquare string `json:"from_square"`
	PieceType  string `json:"piece_type,omitempty"`
	Message    string `json:"message"`
	HintUsed   bool   `json:"hint_used"`
}

type SolutionRequest struct {
	PuzzleID string `json:"puzzle_id" binding:"required"`
}

type SolutionResponse struct {
	Moves          []string `json:"moves"`
	RatingChange   int      `json:"rating_change"`
	SolutionViewed bool     `json:"solution_viewed"`
	Message        string   `json:"message"`
}

type PuzzleStatsResponse struct {
	Rating        int        `json:"rating"`
	BestRating    int        `json:"best_rating"`
	BestRatingAt  *time.Time `json:"best_rating_at,omitempty"`
	PuzzlesSolved int        `json:"puzzles_solved"`
	PuzzlesFailed int        `json:"puzzles_failed"`
	WinRate       float64    `json:"win_rate"`
	CurrentStreak int        `json:"current_streak"`
	MaxStreak     int        `json:"max_streak"`
}

type PuzzleHistoryItem struct {
	ID               int64     `json:"id"`
	PuzzleID         string    `json:"puzzle_id"`
	Solved           bool      `json:"solved"`
	AttemptedMoves   []string  `json:"attempted_moves"`
	SolutionMoves    []string  `json:"solution_moves"`
	UserRatingBefore int       `json:"user_rating_before"`
	UserRatingAfter  int       `json:"user_rating_after"`
	RatingChange     int       `json:"rating_change"`
	TimeTakenMs      int       `json:"time_taken_ms"`
	CreatedAt        time.Time `json:"created_at"`
}

type PuzzleHistoryResponse struct {
	Total    int64               `json:"total"`
	Attempts []PuzzleHistoryItem `json:"attempts"`
}
