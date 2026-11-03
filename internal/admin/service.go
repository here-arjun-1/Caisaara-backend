package admin

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Disconnector interface {
	DisconnectUser(userID int64)
}

type Service struct {
	Repository   *Repository
	Disconnector Disconnector
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SetDisconnector(disconnector Disconnector) {
	s.Disconnector = disconnector
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

func (s *Service) BanUser(ctx context.Context, adminID int64, userID int64, days int, reason string) error {
	reason = strings.TrimSpace(reason)

	if adminID == userID {
		return ErrCannotBanYourself
	}

	if reason == "" {
		return ErrReasonRequired
	}

	if utf8.RuneCountInString(reason) > MaxBanReasonLength {
		return ErrReasonTooLong
	}

	if days < 0 || days > MaxBanDays {
		return ErrInvalidBanDays
	}

	role, err := s.Repository.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}

	if role == "admin" {
		return ErrCannotBanAdmin
	}

	details := "permanent: " + reason
	if days > 0 {
		details = fmt.Sprintf("%d days: %s", days, reason)
	}

	err = s.Repository.BanUser(ctx, adminID, userID, days, reason, details)
	if err != nil {
		return err
	}

	if s.Disconnector != nil {
		s.Disconnector.DisconnectUser(userID)
	}

	return nil
}
