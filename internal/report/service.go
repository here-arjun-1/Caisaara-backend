package report

import (
	"context"
	"strings"
	"unicode/utf8"
)

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) ReportPlayer(ctx context.Context, reporterID int64, reportedID int64, reason string, details string) (int64, error) {
	reason = strings.ToLower(strings.TrimSpace(reason))
	details = strings.TrimSpace(details)

	if reporterID == reportedID {
		return 0, ErrCannotReportYourself
	}

	if !validReasons[reason] {
		return 0, ErrInvalidReason
	}

	if utf8.RuneCountInString(details) > MaxDetailsLength {
		return 0, ErrDetailsTooLong
	}

	exists, err := s.Repository.UserExists(ctx, reportedID)
	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, ErrUserNotFound
	}
	return s.Repository.CreateReport(ctx, reporterID, reportedID, reason, details)
}
