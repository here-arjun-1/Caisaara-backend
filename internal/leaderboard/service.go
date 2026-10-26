package leaderboard

import (
	"context"
	"strings"

	"github.com/here-arjun-1/Caisaara-backend/internal/rating/glicko2"
)

const (
	defaultMode  = "daily"
	defaultLimit = 50
	maxLimit     = 100
)

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) GetLeaderboard(ctx context.Context, mode string, limit int, offset int) ([]LeaderboardEntry, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = defaultMode
	}

	if mode != "bullet" && mode != "blitz" && mode != "rapid" && mode != "daily" {
		return nil, ErrInvalidMode
	}

	if limit <= 0 {
		limit = defaultLimit
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	if offset < 0 {
		offset = 0
	}

	return s.Repository.GetLeaderboard(ctx, mode, glicko2.ProvisionalRD, limit, offset)
}
