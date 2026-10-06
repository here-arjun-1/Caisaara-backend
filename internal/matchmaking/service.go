package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
)

var AllowedTimeControls = []int{1, 3, 5, 10, 15, 30}

const queueTTL = 10 * time.Minute

type UserFinder interface {
	FindUserByID(ctx context.Context, id int64) (*model.User, error)
}

type MatchmakingService interface {
	Join(ctx context.Context, userID int64, timeControlMinutes int, rated bool) error
	Cancel(ctx context.Context, userID int64) error
	GetStatus(ctx context.Context, userID int64) (*Status, error)
}

type Service struct {
	QueueRepo  QueueRepository
	UserFinder UserFinder
}

func NewService(queueRepo QueueRepository, userFinder UserFinder) *Service {
	return &Service{
		QueueRepo:  queueRepo,
		UserFinder: userFinder,
	}
}

func (s *Service) Join(
	ctx context.Context,
	userID int64,
	timeControlMinutes int,
	rated bool,
) error {

	if !slices.Contains(AllowedTimeControls, timeControlMinutes) {
		return ErrInvalidTimeControl
	}

	user, err := s.UserFinder.FindUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "find user for matchmaking failed", "error", err, "user_id", userID)
		return ErrInternal
	}

	entry := &QueueEntry{
		UserID:             userID,
		Rating:             user.Rating,
		TimeControlMinutes: timeControlMinutes,
		Rated:              rated,
		JoinedAt:           time.Now(),
	}

	err = s.QueueRepo.AddToQueue(ctx, entry, queueTTL)
	if err != nil {
		if errors.Is(err, ErrAlreadyInQueue) {
			return ErrAlreadyInQueue
		}
		slog.ErrorContext(ctx, "add to queue failed", "error", err, "user_id", userID)
		return ErrInternal
	}

	return nil
}

func (s *Service) Cancel(ctx context.Context, userID int64) error {
	_, err := s.QueueRepo.RemoveFromQueue(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotInQueue) {
			return ErrNotInQueue
		}
		slog.ErrorContext(ctx, "remove from queue failed", "error", err, "user_id", userID)
		return ErrInternal
	}

	return nil
}

func (s *Service) GetStatus(ctx context.Context, userID int64) (*Status, error) {
	_, err := s.QueueRepo.GetEntry(ctx, userID)
	if err == nil {
		return &Status{State: StatusSearching}, nil
	}
	if !errors.Is(err, ErrNotInQueue) {
		slog.ErrorContext(ctx, "get queue entry failed", "error", err, "user_id", userID)
		return nil, ErrInternal
	}

	gameID, err := s.QueueRepo.GetMatch(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get match failed", "error", err, "user_id", userID)
		return nil, ErrInternal
	}

	if gameID != "" {
		return &Status{State: StatusMatched, GameID: gameID}, nil
	}

	return &Status{State: StatusIdle}, nil
}
