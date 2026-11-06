package minimates

import "time"

type Level struct {
	LevelNumber int       `json:"level_number"`
	PuzzleID    string    `json:"puzzle_id"`
	Chapter     int       `json:"chapter"`
	TargetMoves int       `json:"target_moves"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserProgress struct {
	UserID          int64     `json:"user_id"`
	CurrentLevel    int       `json:"current_level"`
	TotalStars      int       `json:"total_stars"`
	CompletedLevels int       `json:"completed_levels"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LevelHistory struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	LevelNumber int        `json:"level_number"`
	Stars       int        `json:"stars"`
	Completed   bool       `json:"completed"`
	HintUsed    bool       `json:"hint_used"`
	TimeTakenMs int        `json:"time_taken_ms"`
	Attempts    int        `json:"attempts"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type LevelSummary struct {
	LevelNumber int    `json:"level_number"`
	Chapter     int    `json:"chapter"`
	TargetMoves int    `json:"target_moves"`
	Title       string `json:"title"`
	Stars       int    `json:"stars"`
	Completed   bool   `json:"completed"`
	Unlocked    bool   `json:"unlocked"`
}

type ChapterInfo struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	StartLevel  int            `json:"start_level"`
	EndLevel    int            `json:"end_level"`
	TargetMoves int            `json:"target_moves"`
	Levels      []LevelSummary `json:"levels"`
}

type MiniMatesOverviewResponse struct {
	CurrentLevel    int           `json:"current_level"`
	TotalStars      int           `json:"total_stars"`
	CompletedLevels int           `json:"completed_levels"`
	Chapters        []ChapterInfo `json:"chapters"`
}

type Viewport struct {
	StartFile string `json:"start_file"`
	EndFile   string `json:"end_file"`
	StartRank int    `json:"start_rank"`
	EndRank   int    `json:"end_rank"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

type MiniMatesLevelResponse struct {
	LevelNumber int      `json:"level_number"`
	PuzzleID    string   `json:"puzzle_id"`
	Chapter     int      `json:"chapter"`
	ChapterName string   `json:"chapter_name"`
	TargetMoves int      `json:"target_moves"`
	Title       string   `json:"title"`
	FEN         string   `json:"fen"`
	InitialFEN  string   `json:"initial_fen"`
	SetupMove   string   `json:"setup_move"`
	PlayerColor string   `json:"player_color"`
	Viewport    Viewport `json:"viewport"`
	Stars       int      `json:"stars"`
	Completed   bool     `json:"completed"`
}

type MiniMatesHintRequest struct {
	MoveIndex int `json:"move_index"`
}

type SubmitMiniMatesMoveRequest struct {
	Move        string `json:"move"`
	MoveIndex   int    `json:"move_index"`
	TimeTakenMs int    `json:"time_taken_ms"`
}

type SubmitMiniMatesMoveResponse struct {
	Correct           bool   `json:"correct"`
	NextMove          string `json:"next_move,omitempty"`
	IsCheckmate       bool   `json:"is_checkmate"`
	Stars             int    `json:"stars"`
	TotalStars        int    `json:"total_stars"`
	CurrentLevel      int    `json:"current_level"`
	NextUnlockedLevel int    `json:"next_unlocked_level"`
	FEN               string `json:"fen,omitempty"`
	Message           string `json:"message,omitempty"`
}

type MiniMatesHintResponse struct {
	FromSquare string `json:"from_square"`
	Piece      string `json:"piece"`
	Message    string `json:"message"`
}

type LevelWithPuzzle struct {
	LevelNumber int
	PuzzleID    string
	Chapter     int
	TargetMoves int
	Title       string
	FEN         string
	Moves       string
	Rating      int
}
