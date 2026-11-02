package streak

import (
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

const (
	StatusPlayed = "played"
	StatusFrozen = "frozen"
	StatusMissed = "missed"
	StatusEmpty  = "empty"
)

const (
	StreakActive = "active"
	StreakPaused = "paused"
	StreakBroken = "broken"
)

type UserStreak struct {
	UserID           int64     `json:"user_id"`
	CurrentStreak    int       `json:"current_streak"`
	BestStreak       int       `json:"best_streak"`
	MaxStreak        int       `json:"max_streak"`
	FreezesAvailable int       `json:"freezes_available"`
	LastActivityDate string    `json:"last_activity_date,omitempty"`
	StreakStatus     string    `json:"streak_status"`
	Timezone         string    `json:"timezone"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type WeekDayItem struct {
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	Status  string `json:"status"`
}

type StreakResponse struct {
	CurrentStreak    int           `json:"current_streak"`
	MaxStreak        int           `json:"max_streak"`
	BestStreak       int           `json:"best_streak"`
	TodayCompleted   bool          `json:"today_completed"`
	StreakStatus     string        `json:"streak_status"`
	Timezone         string        `json:"timezone"`
	FreezesAvailable int           `json:"freezes_available"`
	Week             []WeekDayItem `json:"week"`
}

type StreakUpdate = game.StreakUpdate
