package leaderboard

import (
	"context"
	"strings"

	"github.com/here-arjun-1/Caisaara-backend/internal/rating/glicko2"
)

const (
	defaultMode        = "daily"
	defaultLimit       = 50
	maxLimit           = 100
	defaultRadiusKm    = 25
	minRadiusKm        = 1
	maxRadiusKm        = 100
	locationMaxAgeDays = 7
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
	mode, err := cleanMode(mode)
	if err != nil {
		return nil, err
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

func (s *Service) GetLocalLeaderboard(ctx context.Context, userID int64, mode string, radiusKm int, limit int, offset int) ([]LocalLeaderboardEntry, error) {
	mode, err := cleanMode(mode)
	if err != nil {
		return nil, err
	}

	if radiusKm == 0 {
		radiusKm = defaultRadiusKm
	}

	if radiusKm < minRadiusKm || radiusKm > maxRadiusKm {
		return nil, ErrInvalidRadius
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

	saved, nearbyEnabled, locationFresh, err := s.Repository.GetLocationStatus(ctx, userID, locationMaxAgeDays)
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, ErrLocationNotSaved
	}
	if !nearbyEnabled {
		return nil, ErrNearbyOff
	}
	if !locationFresh {
		return nil, ErrLocationTooOld
	}

	return s.Repository.GetLocalLeaderboard(ctx, userID, mode, radiusKm*1000, locationMaxAgeDays, glicko2.ProvisionalRD, limit, offset)
}

func cleanMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return defaultMode, nil
	}

	if mode != "bullet" && mode != "blitz" && mode != "rapid" && mode != "daily" {
		return "", ErrInvalidMode
	}

	return mode, nil
}
