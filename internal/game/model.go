package game

import "time"

const (
	ModeBullet = "bullet"
	ModeBlitz  = "blitz"
	ModeRapid  = "rapid"
	ModeCustom = "custom"
	ModeDaily  = "daily"
)

type Game struct {
	ID                 string     `json:"id"`
	WhitePlayerID      int64      `json:"white_player_id"`
	BlackPlayerID      int64      `json:"black_player_id"`
	TimeControlMinutes int        `json:"time_control_minutes"`
	Rated              bool       `json:"rated"`
	TimeControlMode    string     `json:"time_control_mode"`
	DailyMoveTimeMs    int64      `json:"daily_move_time_ms,omitempty"`
	Position           string     `json:"position"`
	Status             string     `json:"status"`
	Result             string     `json:"result,omitempty"`
	EndReason          string     `json:"end_reason,omitempty"`
	InitialTimeMs      int64      `json:"initial_time_ms"`
	IncrementMs        int64      `json:"increment_ms"`
	WhiteTimeMs        int64      `json:"white_time_ms"`
	BlackTimeMs        int64      `json:"black_time_ms"`
	CurrentTurn        string     `json:"current_turn"`
	TurnStartedAt      *time.Time `json:"turn_started_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
}

const (
	StatusWaiting  = "waiting"
	StatusActive   = "active"
	StatusFinished = "finished"
)

const (
	ResultWhiteWin = "white_win"
	ResultBlackWin = "black_win"
	ResultDraw     = "draw"
)

const (
	EndReasonCheckmate            = "checkmate"
	EndReasonStalemate            = "stalemate"
	EndReasonResignation          = "resignation"
	EndReasonTimeout              = "timeout"
	EndReasonDrawAgreement        = "draw_agreement"
	EndReasonThreefoldRepetition  = "threefold_repetition"
	EndReason50MoveRule           = "50_move_rule"
	EndReason75MoveRule           = "75_move_rule"
	EndReasonInsufficientMaterial = "insufficient_material"
	EndReasonDailyTimeout         = "daily_timeout"
)

type GameMove struct {
	ID            string    `json:"id"`
	GameID        string    `json:"game_id"`
	MoveNumber    int       `json:"move_number"`
	PlayerID      string    `json:"player_id"`
	Move          string    `json:"move"`
	PositionAfter string    `json:"position_after"`
	CreatedAt     time.Time `json:"created_at"`
}

type MoveMessage struct {
	Type string `json:"type"`
	Move string `json:"move,omitempty"`
}

type GameStateMessage struct {
	Type            string     `json:"type"`
	GameID          string     `json:"game_id"`
	Position        string     `json:"position"`
	Status          string     `json:"status"`
	Result          string     `json:"result,omitempty"`
	EndReason       string     `json:"end_reason,omitempty"`
	TimeControlMode string     `json:"time_control_mode"`
	DailyMoveTimeMs int64      `json:"daily_move_time_ms,omitempty"`
	InitialTimeMs   int64      `json:"initial_time_ms"`
	IncrementMs     int64      `json:"increment_ms"`
	WhiteTimeMs     int64      `json:"white_time_ms"`
	BlackTimeMs     int64      `json:"black_time_ms"`
	CurrentTurn     string     `json:"current_turn"`
	TurnStartedAt   *time.Time `json:"turn_started_at,omitempty"`
	Moves           []GameMove `json:"moves,omitempty"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
