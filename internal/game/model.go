package game

import "time"

type Game struct {
	ID                 string     `json:"id"`
	WhitePlayerID      int64      `json:"white_player_id"`
	BlackPlayerID      int64      `json:"black_player_id"`
	TimeControlMinutes int        `json:"time_control_minutes"`
	Position           string     `json:"position"`
	Status             string     `json:"status"`
	Result             string     `json:"result,omitempty"`
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
	Move string `json:"move"`
}

type GameStateMessage struct {
	Type     string `json:"type"`
	GameID   string `json:"game_id"`
	Position string `json:"position"`
	Status   string `json:"status"`
	Result   string `json:"result,omitempty"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
