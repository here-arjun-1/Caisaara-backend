package admin

import (
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Rating    int       `json:"rating"`
	IsBanned  bool      `json:"is_banned"`
	BanReason string    `json:"ban_reason"`
	CreatedAt time.Time `json:"created_at"`
}

type UserDetail struct {
	ID             int64      `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	Rating         int        `json:"rating"`
	EmailVerified  bool       `json:"email_verified"`
	IsBanned       bool       `json:"is_banned"`
	IsPermanentBan bool       `json:"is_permanent_ban"`
	BannedUntil    *time.Time `json:"banned_until"`
	BanReason      string     `json:"ban_reason"`
	GamesPlayed    int        `json:"games_played"`
	ReportsAgainst int        `json:"reports_against"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Report struct {
	ID               int64     `json:"id"`
	ReporterID       int64     `json:"reporter_id"`
	ReporterUsername string    `json:"reporter_username"`
	ReportedID       int64     `json:"reported_id"`
	ReportedUsername string    `json:"reported_username"`
	Reason           string    `json:"reason"`
	Details          string    `json:"details"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type Action struct {
	ID            int64     `json:"id"`
	AdminID       *int64    `json:"admin_id"`
	AdminUsername string    `json:"admin_username"`
	Action        string    `json:"action"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	Details       string    `json:"details"`
	CreatedAt     time.Time `json:"created_at"`
}

type Stats struct {
	TotalUsers         int `json:"total_users"`
	NewUsersToday      int `json:"new_users_today"`
	BannedUsers        int `json:"banned_users"`
	ActiveGames        int `json:"active_games"`
	PendingReports     int `json:"pending_reports"`
	OngoingTournaments int `json:"ongoing_tournaments"`
}

const (
	DefaultLimit       = 20
	MaxLimit           = 50
	MaxBanReasonLength = 500
	MaxBanDays         = 365
	MaxNoteLength      = 500
)

const (
	ReportPending   = "pending"
	ReportResolved  = "resolved"
	ReportDismissed = "dismissed"
)

type LiveGame struct {
	ID                 string     `json:"id"`
	WhitePlayerID      int64      `json:"white_player_id"`
	WhiteUsername      string     `json:"white_username"`
	BlackPlayerID      int64      `json:"black_player_id"`
	BlackUsername      string     `json:"black_username"`
	TimeControlMode    string     `json:"time_control_mode"`
	TimeControlMinutes int        `json:"time_control_minutes"`
	Rated              bool       `json:"rated"`
	MoveCount          int        `json:"move_count"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at"`
}

type MatchmakingQueue struct {
	Queue   string `json:"queue"`
	Players int64  `json:"players"`
}

type Health struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

const (
	HealthOK   = "ok"
	HealthDown = "down"
)

type GameDetail struct {
	Game  *game.Game      `json:"game"`
	Moves []game.GameMove `json:"moves"`
}
