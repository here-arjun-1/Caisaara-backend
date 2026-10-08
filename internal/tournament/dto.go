package tournament

import (
	"strconv"
	"time"
)

type CreateTournamentRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Format      string     `json:"format" binding:"required"`
	TimeControl string     `json:"time_control" binding:"required"`
	MinPlayers  int        `json:"min_players"`
	MaxPlayers  int        `json:"max_players" binding:"required"`
	Visibility  string     `json:"visibility"`
	TotalRounds int        `json:"total_rounds"`
	StartAt     *time.Time `json:"start_at"`
}

type TournamentResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Format       string    `json:"format"`
	TimeControl  string    `json:"time_control"`
	MinPlayers   int       `json:"min_players"`
	MaxPlayers   int       `json:"max_players"`
	Visibility   string    `json:"visibility"`
	InviteCode   string    `json:"invite_code,omitempty"`
	InviteLink   string    `json:"invite_link,omitempty"`
	Status       string    `json:"status"`
	TotalRounds  int       `json:"total_rounds"`
	CurrentRound int       `json:"current_round"`
	CreatedBy    int64     `json:"created_by"`
	StartAt      time.Time `json:"start_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TournamentDetailsResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Format       string    `json:"format"`
	TimeControl  string    `json:"time_control"`
	Players      int       `json:"players"`
	MinPlayers   int       `json:"min_players"`
	MaxPlayers   int       `json:"max_players"`
	Visibility   string    `json:"visibility"`
	InviteCode   string    `json:"invite_code,omitempty"`
	InviteLink   string    `json:"invite_link,omitempty"`
	Status       string    `json:"status"`
	CurrentRound int       `json:"current_round"`
	TotalRounds  int       `json:"total_rounds"`
	CreatedBy    int64     `json:"created_by"`
	StartAt      time.Time `json:"start_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type JoinTournamentRequest struct {
	InviteCode string `json:"invite_code,omitempty"`
}

type JoinTournamentResponse struct {
	Message      string `json:"message"`
	TournamentID string `json:"tournament_id"`
	UserID       int64  `json:"user_id"`
	Status       string `json:"status"`
}

type LeaveTournamentResponse struct {
	Message      string `json:"message"`
	TournamentID string `json:"tournament_id"`
	UserID       int64  `json:"user_id"`
}

type StartTournamentResponse struct {
	Message      string `json:"message"`
	TournamentID string `json:"tournament_id"`
	Status       string `json:"status"`
	CurrentRound int    `json:"current_round"`
}

func ToTournamentResponse(t *Tournament) *TournamentResponse {
	inviteCode := ""
	inviteLink := ""
	if t.InviteCode != nil && *t.InviteCode != "" {
		inviteCode = *t.InviteCode
		inviteLink = "/tournaments/invite/" + inviteCode
	}
	return &TournamentResponse{
		ID:           strconv.FormatInt(t.ID, 10),
		Name:         t.Name,
		Description:  t.Description,
		Format:       t.Format,
		TimeControl:  t.TimeControl,
		MinPlayers:   t.MinPlayers,
		MaxPlayers:   t.MaxPlayers,
		Visibility:   t.Visibility,
		InviteCode:   inviteCode,
		InviteLink:   inviteLink,
		Status:       t.Status,
		TotalRounds:  t.TotalRounds,
		CurrentRound: t.CurrentRound,
		CreatedBy:    t.CreatedBy,
		StartAt:      t.StartAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func ToTournamentDetailsResponse(tw *TournamentWithPlayerCount) *TournamentDetailsResponse {
	inviteCode := ""
	inviteLink := ""
	if tw.InviteCode != nil && *tw.InviteCode != "" {
		inviteCode = *tw.InviteCode
		inviteLink = "/tournaments/invite/" + inviteCode
	}
	return &TournamentDetailsResponse{
		ID:           strconv.FormatInt(tw.ID, 10),
		Name:         tw.Name,
		Description:  tw.Description,
		Format:       tw.Format,
		TimeControl:  tw.TimeControl,
		Players:      tw.Players,
		MinPlayers:   tw.MinPlayers,
		MaxPlayers:   tw.MaxPlayers,
		Visibility:   tw.Visibility,
		InviteCode:   inviteCode,
		InviteLink:   inviteLink,
		Status:       tw.Status,
		TotalRounds:  tw.TotalRounds,
		CurrentRound: tw.CurrentRound,
		CreatedBy:    tw.CreatedBy,
		StartAt:      tw.StartAt,
		CreatedAt:    tw.CreatedAt,
		UpdatedAt:    tw.UpdatedAt,
	}
}

type StandingsPlayerResponse struct {
	Rank        int     `json:"rank"`
	PlayerID    int64   `json:"player_id"`
	Username    string  `json:"username"`
	Score       float64 `json:"score"`
	Buchholz    float64 `json:"buchholz"`
	Wins        int     `json:"wins"`
	Draws       int     `json:"draws"`
	Losses      int     `json:"losses"`
	GamesPlayed int     `json:"games_played"`
}

type TournamentStandingsResponse struct {
	TournamentID string                     `json:"tournament_id"`
	Standings    []*StandingsPlayerResponse `json:"standings"`
}

type PairingResponse struct {
	ID            string  `json:"id"`
	RoundID       string  `json:"round_id"`
	RoundNumber   int     `json:"round_number"`
	WhitePlayerID *int64  `json:"white_player_id"`
	WhiteUsername string  `json:"white_username,omitempty"`
	BlackPlayerID *int64  `json:"black_player_id"`
	BlackUsername string  `json:"black_username,omitempty"`
	GameID        *string `json:"game_id,omitempty"`
	Result        string  `json:"result"`
	Status        string  `json:"status"`
	IsBye         bool    `json:"is_bye"`
}

type RoundResponse struct {
	ID          string             `json:"id"`
	RoundNumber int                `json:"round_number"`
	Status      string             `json:"status"`
	Pairings    []*PairingResponse `json:"pairings"`
}

type TournamentRoundsResponse struct {
	TournamentID string           `json:"tournament_id"`
	Rounds       []*RoundResponse `json:"rounds"`
}

type TournamentGameItem struct {
	GameID        string `json:"game_id"`
	PairingID     string `json:"pairing_id"`
	RoundNumber   int    `json:"round_number"`
	WhitePlayerID int64  `json:"white_player_id"`
	WhiteUsername string `json:"white_username"`
	BlackPlayerID int64  `json:"black_player_id"`
	BlackUsername string `json:"black_username"`
	Result        string `json:"result"`
	Status        string `json:"status"`
}

type TournamentGamesResponse struct {
	TournamentID string                `json:"tournament_id"`
	Games        []*TournamentGameItem `json:"games"`
}
