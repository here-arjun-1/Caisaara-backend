package tournament

import "time"

const (
	FormatSwiss      = "swiss"
	FormatKnockout   = "knockout"
	FormatRoundRobin = "round_robin"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

const (
	StatusRegistration = "registration"
	StatusUpcoming     = "upcoming"
	StatusOngoing      = "ongoing"
	StatusCompleted    = "completed"
	StatusCancelled    = "cancelled"
)

const (
	RoundStatusPending   = "pending"
	RoundStatusOngoing   = "ongoing"
	RoundStatusCompleted = "completed"
)

const (
	PairingStatusPending   = "pending"
	PairingStatusOngoing   = "ongoing"
	PairingStatusCompleted = "completed"
)

const (
	ResultWhiteWin = "white_win"
	ResultBlackWin = "black_win"
	ResultDraw     = "draw"
	ResultBye      = "bye"
)

type Tournament struct {
	ID           int64      `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Description  string     `json:"description,omitempty" db:"description"`
	Format       string     `json:"format" db:"format"`
	TimeControl  string     `json:"time_control" db:"time_control"`
	MinPlayers   int        `json:"min_players" db:"min_players"`
	MaxPlayers   int        `json:"max_players" db:"max_players"`
	Visibility   string     `json:"visibility" db:"visibility"`
	InviteCode   *string    `json:"invite_code,omitempty" db:"invite_code"`
	Status       string     `json:"status" db:"status"`
	TotalRounds  int        `json:"total_rounds" db:"total_rounds"`
	CurrentRound int        `json:"current_round" db:"current_round"`
	CreatedBy    int64      `json:"created_by" db:"created_by"`
	StartAt      time.Time  `json:"start_at" db:"start_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type TournamentPlayer struct {
	ID           int64     `json:"id" db:"id"`
	TournamentID int64     `json:"tournament_id" db:"tournament_id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	Score        float64   `json:"score" db:"score"`
	Wins         int       `json:"wins" db:"wins"`
	Draws        int       `json:"draws" db:"draws"`
	Losses       int       `json:"losses" db:"losses"`
	GamesPlayed  int       `json:"games_played" db:"games_played"`
	JoinedAt     time.Time `json:"joined_at" db:"joined_at"`
}

type TournamentRound struct {
	ID           int64      `json:"id" db:"id"`
	TournamentID int64      `json:"tournament_id" db:"tournament_id"`
	RoundNumber  int        `json:"round_number" db:"round_number"`
	Status       string     `json:"status" db:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

type TournamentPairing struct {
	ID            int64      `json:"id" db:"id"`
	TournamentID  int64      `json:"tournament_id" db:"tournament_id"`
	RoundID       int64      `json:"round_id" db:"round_id"`
	WhitePlayerID *int64     `json:"white_player_id,omitempty" db:"white_player_id"`
	BlackPlayerID *int64     `json:"black_player_id,omitempty" db:"black_player_id"`
	GameID        *string    `json:"game_id,omitempty" db:"game_id"`
	Result        *string    `json:"result,omitempty" db:"result"`
	Status        string     `json:"status" db:"status"`
	IsBye         bool       `json:"is_bye" db:"is_bye"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}
