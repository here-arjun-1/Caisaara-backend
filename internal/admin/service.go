package admin

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Disconnector interface {
	DisconnectUser(userID int64)
}

type TournamentCanceller interface {
	ForceCancelTournament(ctx context.Context, tournamentID int64) error
}

type Service struct {
	Repository          *Repository
	Disconnector        Disconnector
	TournamentCanceller TournamentCanceller
}

func NewService(repository *Repository, tournamentCanceller TournamentCanceller) *Service {
	return &Service{
		Repository:          repository,
		TournamentCanceller: tournamentCanceller,
	}
}

func (s *Service) SetDisconnector(disconnector Disconnector) {
	s.Disconnector = disconnector
}

func fixPagination(limit int, offset int) (int, int) {
	if limit <= 0 {
		limit = DefaultLimit
	}

	if limit > MaxLimit {
		limit = MaxLimit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	return s.Repository.GetStats(ctx)
}

func (s *Service) SearchUsers(ctx context.Context, search string, limit int, offset int) ([]User, error) {
	search = strings.TrimSpace(search)
	limit, offset = fixPagination(limit, offset)

	return s.Repository.SearchUsers(ctx, search, limit, offset)
}

func (s *Service) GetUser(ctx context.Context, userID int64) (*UserDetail, error) {
	return s.Repository.GetUser(ctx, userID)
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

func (s *Service) UnbanUser(ctx context.Context, adminID int64, userID int64) error {
	_, err := s.Repository.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}

	return s.Repository.UnbanUser(ctx, adminID, userID)
}

func (s *Service) ForceLogout(ctx context.Context, adminID int64, userID int64) error {
	err := s.Repository.ForceLogout(ctx, adminID, userID)
	if err != nil {
		return err
	}

	if s.Disconnector != nil {
		s.Disconnector.DisconnectUser(userID)
	}

	return nil
}

func (s *Service) ListReports(ctx context.Context, status string, limit int, offset int) ([]Report, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		status = ReportPending
	}

	if status != ReportPending && status != ReportResolved && status != ReportDismissed {
		return nil, ErrInvalidReportStatus
	}

	limit, offset = fixPagination(limit, offset)

	return s.Repository.ListReports(ctx, status, limit, offset)
}

func (s *Service) CloseReport(ctx context.Context, adminID int64, reportID int64, status string, note string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	note = strings.TrimSpace(note)

	if status != ReportResolved && status != ReportDismissed {
		return ErrInvalidResolution
	}

	if utf8.RuneCountInString(note) > MaxNoteLength {
		return ErrNoteTooLong
	}

	return s.Repository.CloseReport(ctx, adminID, reportID, status, note)
}

func (s *Service) DeletePost(ctx context.Context, adminID int64, postID int64) error {
	return s.Repository.DeletePost(ctx, adminID, postID)
}

func (s *Service) DeleteComment(ctx context.Context, adminID int64, commentID int64) error {
	return s.Repository.DeleteComment(ctx, adminID, commentID)
}

func (s *Service) DeleteClub(ctx context.Context, adminID int64, clubID string) error {
	if _, err := uuid.Parse(clubID); err != nil {
		return ErrInvalidClubID
	}

	return s.Repository.DeleteClub(ctx, adminID, clubID)
}

func (s *Service) CancelTournament(ctx context.Context, adminID int64, tournamentID int64) error {
	err := s.TournamentCanceller.ForceCancelTournament(ctx, tournamentID)
	if err != nil {
		return err
	}

	err = s.Repository.LogAction(ctx, adminID, "cancel_tournament", "tournament", strconv.FormatInt(tournamentID, 10), "")
	if err != nil {
		slog.ErrorContext(ctx, "log cancel tournament action failed", "tournament_id", tournamentID, "error", err)
	}

	return nil
}

func (s *Service) ListActions(ctx context.Context, limit int, offset int) ([]Action, error) {
	limit, offset = fixPagination(limit, offset)

	return s.Repository.ListActions(ctx, limit, offset)
}
