package admin

import (
	"context"
	"strings"
)

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SearchUsers(ctx context.Context, search string, limit int, offset int) ([]User, error) {
	search = strings.TrimSpace(search)

	if limit <= 0 {
		limit = DefaultUsersLimit
	}

	if limit > MaxUsersLimit {
		limit = MaxUsersLimit
	}

	if offset < 0 {
		offset = 0
	}

	return s.Repository.SearchUsers(ctx, search, limit, offset)
}
