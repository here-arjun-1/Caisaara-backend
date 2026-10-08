package matchmaking

import "time"

type QueueEntry struct {
	UserID             int64     `json:"user_id"`
	Rating             int       `json:"rating"`
	TimeControlMinutes int       `json:"time_control_minutes"`
	IncrementSeconds   int       `json:"increment_seconds"`
	Rated              bool      `json:"rated"`
	JoinedAt           time.Time `json:"joined_at"`
}

const (
	StatusIdle      = "idle"
	StatusSearching = "searching"
	StatusMatched   = "matched"
)

type Status struct {
	State  string `json:"state"`
	GameID string `json:"game_id,omitempty"`
}
