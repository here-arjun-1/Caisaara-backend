package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/tilt"
)

type TimeControl struct {
	Minutes   int `json:"time_control_minutes"`
	Increment int `json:"increment_seconds"`
}

var AllowedTimeControls = []TimeControl{
	{Minutes: 1, Increment: 0},
	{Minutes: 1, Increment: 1},
	{Minutes: 2, Increment: 1},
	{Minutes: 3, Increment: 0},
	{Minutes: 3, Increment: 2},
	{Minutes: 5, Increment: 0},
	{Minutes: 5, Increment: 2},
	{Minutes: 5, Increment: 5},
	{Minutes: 10, Increment: 0},
	{Minutes: 10, Increment: 5},
	{Minutes: 15, Increment: 10},
	{Minutes: 20, Increment: 0},
	{Minutes: 30, Increment: 0},
	{Minutes: 60, Increment: 0},
	{Minutes: 1440, Increment: 0},
	{Minutes: 2880, Increment: 0},
	{Minutes: 4320, Increment: 0},
	{Minutes: 7200, Increment: 0},
	{Minutes: 10080, Increment: 0},
	{Minutes: 20160, Increment: 0},
}

const queueTTL = 10 * time.Minute

type UserFinder interface {
	FindUserByID(ctx context.Context, id int64) (*model.User, error)
}

type TiltChecker interface {
	Check(ctx context.Context, playerID int64) (*tilt.Status, error)
}

type MatchmakingService interface {
	Join(ctx context.Context, userID int64, timeControlMinutes int, incrementSeconds int, rated bool, ignoreTilt bool) (*tilt.Status, error)
	Cancel(ctx context.Context, userID int64) error
	GetStatus(ctx context.Context, userID int64) (*Status, error)
}

type Service struct {
	QueueRepo   QueueRepository
	UserFinder  UserFinder
	TiltChecker TiltChecker
}

func NewService(queueRepo QueueRepository, userFinder UserFinder, tiltChecker TiltChecker) *Service {
	return &Service{
		QueueRepo:   queueRepo,
		UserFinder:  userFinder,
		TiltChecker: tiltChecker,
	}
}

func isAllowedTimeControl(timeControlMinutes int, incrementSeconds int) bool {
	for _, tc := range AllowedTimeControls {
		if tc.Minutes == timeControlMinutes && tc.Increment == incrementSeconds {
			return true
		}
	}

	if timeControlMinutes >= 0 && timeControlMinutes <= 120 && incrementSeconds >= 0 && incrementSeconds <= 60 && (timeControlMinutes > 0 || incrementSeconds > 0) {
		return true
	}

	return false
}

func (s *Service) Join(
	ctx context.Context,
	userID int64,
	timeControlMinutes int,
	incrementSeconds int,
	rated bool,
	ignoreTilt bool,
) (*tilt.Status, error) {

	if !isAllowedTimeControl(timeControlMinutes, incrementSeconds) {
		return nil, ErrInvalidTimeControl
	}

	if rated && !ignoreTilt {
		tiltStatus, err := s.TiltChecker.Check(ctx, userID)
		if err != nil {
			slog.ErrorContext(ctx, "tilt check failed", "error", err, "user_id", userID)
		} else if tiltStatus != nil && tiltStatus.Tilted {
			return tiltStatus, ErrTilted
		}
	}

	user, err := s.UserFinder.FindUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "find user for matchmaking failed", "error", err, "user_id", userID)
		return nil, ErrInternal
	}

	entry := &QueueEntry{
		UserID:             userID,
		Rating:             user.Rating,
		TimeControlMinutes: timeControlMinutes,
		IncrementSeconds:   incrementSeconds,
		Rated:              rated,
		JoinedAt:           time.Now(),
	}

	err = s.QueueRepo.AddToQueue(ctx, entry, queueTTL)
	if err != nil {
		if errors.Is(err, ErrAlreadyInQueue) {
			return nil, ErrAlreadyInQueue
		}
		slog.ErrorContext(ctx, "add to queue failed", "error", err, "user_id", userID)
		return nil, ErrInternal
	}

	return nil, nil
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
