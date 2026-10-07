package tournament

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"
)

type Service interface {
	CreateTournament(ctx context.Context, userID int64, req CreateTournamentRequest) (*Tournament, error)
}

type TournamentService struct {
	Repo TournamentRepository
}

func NewService(repo TournamentRepository) Service {
	return &TournamentService{
		Repo: repo,
	}
}

const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateInviteCode() string {
	b := make([]byte, 6)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			b[i] = inviteCodeChars[0]
			continue
		}
		b[i] = inviteCodeChars[num.Int64()]
	}
	return string(b)
}

func (s *TournamentService) CreateTournament(ctx context.Context, userID int64, req CreateTournamentRequest) (*Tournament, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	timeControl := strings.TrimSpace(req.TimeControl)
	if timeControl == "" {
		return nil, ErrInvalidTimeControl
	}

	if req.Format != FormatSwiss {
		return nil, ErrInvalidFormat
	}

	visibility := strings.TrimSpace(req.Visibility)
	if visibility == "" {
		visibility = VisibilityPublic
	}
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return nil, ErrInvalidVisibility
	}

	if req.MaxPlayers < 2 {
		return nil, ErrInvalidMaxPlayers
	}

	totalRounds := req.TotalRounds
	if totalRounds <= 0 {
		totalRounds = 5
	}

	inviteCode := generateInviteCode()

	now := time.Now()
	startAt := now
	if req.StartAt != nil && !req.StartAt.IsZero() {
		startAt = *req.StartAt
	}

	tournament := &Tournament{
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		Format:       req.Format,
		TimeControl:  timeControl,
		MaxPlayers:   req.MaxPlayers,
		Visibility:   visibility,
		InviteCode:   &inviteCode,
		Status:       StatusRegistration,
		TotalRounds:  totalRounds,
		CurrentRound: 0,
		CreatedBy:    userID,
		StartAt:      startAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.Repo.CreateTournament(ctx, tournament); err != nil {
		return nil, err
	}

	return tournament, nil
}
