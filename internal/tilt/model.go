package tilt

import "time"

const (
	recentGamesLimit = 5
	lossStreakLimit  = 3
	requeueWindow    = 60 * time.Second
	fastMoveLimit    = 2 * time.Second
	minMovesToCheck  = 10
)

const (
	ReasonLossStreak = "loss_streak"
	ReasonFastMoves  = "fast_moves"
	ReasonQuickQueue = "quick_requeue"
)

type RecentGame struct {
	ID              string
	WhitePlayerID   int64
	BlackPlayerID   int64
	TimeControlMode string
	Result          string
	EndedAt         time.Time
}

type Suggestion struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Status struct {
	Tilted      bool         `json:"tilted"`
	Reasons     []string     `json:"reasons,omitempty"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
}

var defaultSuggestions = []Suggestion{
	{Type: "bot_easy", Message: "Play a relaxed game against the easy bot"},
	{Type: "unrated", Message: "Play an unrated game instead"},
}
