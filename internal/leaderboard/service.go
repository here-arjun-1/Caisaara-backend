package leaderboard

import "context"

const (
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

func (s *Service) GetDailyLeaderboard(ctx context.Context, limit int, offset int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	if offset < 0 {
		offset = 0
	}

	entries, err := s.Repository.GetDailyLeaderboard(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	for i := range entries {
		entries[i].Rank = offset + i + 1
	}

	return entries, nil
}
