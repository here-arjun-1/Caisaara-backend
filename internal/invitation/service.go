package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type UserFinder interface {
	FindUserByID(ctx context.Context, id int64) (*model.User, error)
}

type InvitationService interface {
	CreateInvite(ctx context.Context, userID int64, timeControlMinutes int, incrementSeconds int, color string, opponentID int64) (*Invite, error)
	GetInvite(ctx context.Context, code string) (*Invite, error)
	GetInviteByGameID(ctx context.Context, gameID string) (*Invite, error)
	JoinInvite(ctx context.Context, code string, player2ID int64) (string, error)
	FindInviteCreator(ctx context.Context, creatorID int64) (*model.User, error)
}

type Service struct {
	InviteRepo     InviteRepository
	GameRepository game.GameRepository
	UserFinder     UserFinder
}

func NewService(
	inviteRepo InviteRepository,
	gameRepository game.GameRepository,
	userFinder UserFinder,
) *Service {
	return &Service{
		InviteRepo:     inviteRepo,
		GameRepository: gameRepository,
		UserFinder:     userFinder,
	}
}

func generateInviteCode() (string, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) CreateInvite(
	ctx context.Context,
	userID int64,
	timeControlMinutes int,
	incrementSeconds int,
	color string,
	opponentID int64,
) (*Invite, error) {

	if timeControlMinutes <= 0 && incrementSeconds <= 0 {
		return nil, ErrInvalidTimeControl
	}

	if color != "white" && color != "black" && color != "random" {
		return nil, ErrInvalidColor
	}

	if opponentID != 0 {
		if opponentID == userID {
			return nil, ErrChallengeYourself
		}

		_, err := s.UserFinder.FindUserByID(ctx, opponentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOpponentNotFound
		}
		if err != nil {
			slog.ErrorContext(ctx, "find opponent failed", "error", err, "opponent_id", opponentID)
			return nil, ErrInternal
		}
	}

	code, err := generateInviteCode()
	if err != nil {
		slog.ErrorContext(ctx, "generate invite code failed", "error", err)
		return nil, ErrInternal
	}

	gameID := uuid.New().String()

	invite := &Invite{
		Code:               code,
		GameID:             gameID,
		CreatorID:          userID,
		OpponentID:         opponentID,
		TimeControlMinutes: timeControlMinutes,
		IncrementSeconds:   incrementSeconds,
		Color:              color,
		Status:             "waiting",
	}

	err = s.InviteRepo.SaveInvite(ctx, invite, 10*time.Minute)
	if err != nil {
		slog.ErrorContext(ctx, "save invite failed", "error", err)
		return nil, fmt.Errorf("failed to store invite: %w", err)
	}

	if opponentID != 0 {
		err = s.InviteRepo.AddChallenge(ctx, opponentID, code, 10*time.Minute)
		if err != nil {
			slog.ErrorContext(ctx, "save challenge failed", "error", err, "opponent_id", opponentID)
			return nil, ErrInternal
		}
	}

	return invite, nil
}

func (s *Service) GetInvite(
	ctx context.Context,
	code string,
) (*Invite, error) {

	invite, err := s.InviteRepo.GetInvite(ctx, code)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInviteNotFound
		}
		slog.ErrorContext(ctx, "get invite failed", "error", err)
		return nil, ErrInternal
	}

	return invite, nil
}

func (s *Service) GetInviteByGameID(
	ctx context.Context,
	gameID string,
) (*Invite, error) {

	invite, err := s.InviteRepo.GetInviteByGameID(ctx, gameID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInviteNotFound
		}
		slog.ErrorContext(ctx, "get invite by game id failed", "error", err)
		return nil, ErrInternal
	}

	return invite, nil
}

func (s *Service) FindInviteCreator(ctx context.Context, creatorID int64) (*model.User, error) {
	user, err := s.UserFinder.FindUserByID(ctx, creatorID)
	if err != nil {
		slog.ErrorContext(ctx, "find invite creator failed", "error", err, "creator_id", creatorID)
		return nil, err
	}
	return user, nil
}

func (s *Service) JoinInvite(
	ctx context.Context,
	code string,
	player2ID int64,
) (string, error) {

	invite, err := s.InviteRepo.GetInvite(ctx, code)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrInviteNotFound
		}
		slog.ErrorContext(ctx, "get invite for join failed", "error", err)
		return "", ErrInternal
	}

	if invite.CreatorID == player2ID {
		return "", ErrSelfJoin
	}

	locked, err := s.InviteRepo.AcquireLock(ctx, code, player2ID, 10*time.Second)
	if err != nil {
		slog.ErrorContext(ctx, "acquire invite lock failed", "error", err)
		return "", ErrInternal
	}

	if !locked {
		return "", ErrAlreadyJoining
	}

	defer func() {
		if releaseErr := s.InviteRepo.ReleaseLock(ctx, code); releaseErr != nil {
			slog.ErrorContext(ctx, "release invite lock failed", "error", releaseErr)
		}
	}()

	invite, err = s.InviteRepo.GetInvite(ctx, code)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrInviteUsed
		}
		slog.ErrorContext(ctx, "re-fetch invite failed", "error", err)
		return "", ErrInternal
	}

	if invite.Status == "active" {
		return "", ErrInviteUsed
	}

	var whitePlayerID int64
	var blackPlayerID int64

	switch invite.Color {
	case "white":
		whitePlayerID = invite.CreatorID
		blackPlayerID = player2ID
	case "black":
		blackPlayerID = invite.CreatorID
		whitePlayerID = player2ID
	case "random":
		random := make([]byte, 1)
		_, err := rand.Read(random)
		if err != nil {
			slog.ErrorContext(ctx, "generate random color failed", "error", err)
			return "", ErrInternal
		}
		if random[0]%2 == 0 {
			whitePlayerID = invite.CreatorID
			blackPlayerID = player2ID
		} else {
			whitePlayerID = player2ID
			blackPlayerID = invite.CreatorID
		}
	}

	gameID, err := s.GameRepository.CreateGameWithID(
		ctx,
		invite.GameID,
		whitePlayerID,
		blackPlayerID,
		invite.TimeControlMinutes,
		invite.IncrementSeconds,
		false,
	)

	if err != nil {
		slog.ErrorContext(ctx, "create game from invite failed", "error", err)
		return "", fmt.Errorf("failed to create game: %w", err)
	}

	invite.Status = "active"
	_ = s.InviteRepo.SaveInvite(ctx, invite, 2*time.Minute)

	slog.InfoContext(ctx, "invite joined successfully",
		"code", code,
		"game_id", gameID,
		"player2_id", player2ID,
	)

	return gameID, nil
}
