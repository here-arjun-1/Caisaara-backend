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
	MaxPlayers   int       `json:"max_players"`
	Visibility   string    `json:"visibility"`
	InviteCode   string    `json:"invite_code,omitempty"`
	Status       string    `json:"status"`
	TotalRounds  int       `json:"total_rounds"`
	CurrentRound int       `json:"current_round"`
	CreatedBy    int64     `json:"created_by"`
	StartAt      time.Time `json:"start_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func ToTournamentResponse(t *Tournament) *TournamentResponse {
	inviteCode := ""
	if t.InviteCode != nil {
		inviteCode = *t.InviteCode
	}
	return &TournamentResponse{
		ID:           strconv.FormatInt(t.ID, 10),
		Name:         t.Name,
		Description:  t.Description,
		Format:       t.Format,
		TimeControl:  t.TimeControl,
		MaxPlayers:   t.MaxPlayers,
		Visibility:   t.Visibility,
		InviteCode:   inviteCode,
		Status:       t.Status,
		TotalRounds:  t.TotalRounds,
		CurrentRound: t.CurrentRound,
		CreatedBy:    t.CreatedBy,
		StartAt:      t.StartAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
