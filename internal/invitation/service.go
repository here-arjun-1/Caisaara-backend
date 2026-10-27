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
	"github.com/here-arjun-1/Caisaara-backend/internal/notification"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type UserFinder interface {
	FindUserByID(ctx context.Context, id int64) (*model.User, error)
}

type BlockChecker interface {
	IsBlockedEitherWay(ctx context.Context, userID int64, otherID int64) (bool, error)
}

type Notifier interface {
	Send(ctx context.Context, userID int64, notificationType string, title string, body string, data map[string]any) error
}

type PrivacyChecker interface {
	CheckInviteFriend(ctx context.Context, senderID, opponentID int64) error
}

type InvitationService interface {
	CreateInvite(ctx context.Context, userID int64, timeControlMinutes int, incrementSeconds int, color string, opponentID int64, rated ...bool) (*Invite, error)
	GetInvite(ctx context.Context, code string) (*Invite, error)
	GetInviteByGameID(ctx context.Context, gameID string) (*Invite, error)
	JoinInvite(ctx context.Context, code string, player2ID int64) (string, error)
	FindInviteCreator(ctx context.Context, creatorID int64) (*model.User, error)
	GetMyChallenges(ctx context.Context, userID int64) ([]Challenge, error)
	DeclineChallenge(ctx context.Context, userID int64, code string) error
	CreateWatchInvite(ctx context.Context, userID int64, gameID string) (*WatchInvite, error)
	GetWatchInvite(ctx context.Context, code string) (*WatchInvite, error)
	DeleteWatchInvite(ctx context.Context, gameID string) error
	SetNotifier(notifier Notifier)
	SetPrivacyChecker(checker PrivacyChecker)
}

type Service struct {
	InviteRepo     InviteRepository
	GameRepository game.GameRepository
	UserFinder     UserFinder
	BlockChecker   BlockChecker
	Notifier       Notifier
	PrivacyChecker PrivacyChecker
}

func (s *Service) SetPrivacyChecker(checker PrivacyChecker) {
	s.PrivacyChecker = checker
}

func NewService(
	inviteRepo InviteRepository,
	gameRepository game.GameRepository,
	userFinder UserFinder,
	blockChecker BlockChecker,
) *Service {
	return &Service{
		InviteRepo:     inviteRepo,
		GameRepository: gameRepository,
		UserFinder:     userFinder,
		BlockChecker:   blockChecker,
	}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.Notifier = notifier
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
	rated ...bool,
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

		blocked, err := s.BlockChecker.IsBlockedEitherWay(ctx, userID, opponentID)
		if err != nil {
			slog.ErrorContext(ctx, "check block for challenge failed", "error", err, "opponent_id", opponentID)
			return nil, ErrInternal
		}
		if blocked {
			return nil, ErrPlayerBlocked
		}

		if s.PrivacyChecker != nil {
			if err := s.PrivacyChecker.CheckInviteFriend(ctx, userID, opponentID); err != nil {
				return nil, err
			}
		}
	}

	isRated := true
	if len(rated) > 0 {
		isRated = rated[0]
	}

	if isRated {
		creator, err := s.UserFinder.FindUserByID(ctx, userID)
		if err != nil {
			slog.ErrorContext(ctx, "find invite creator failed", "error", err, "user_id", userID)
			return nil, ErrInternal
		}
		if creator.SkillLevel == nil {
			return nil, ErrLevelNotPicked
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
		Rated:              isRated,
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

		s.notifyChallenge(ctx, invite)
	}

	return invite, nil
}

func (s *Service) notifyChallenge(ctx context.Context, invite *Invite) {
	if s.Notifier == nil {
		return
	}

	creator, err := s.UserFinder.FindUserByID(ctx, invite.CreatorID)
	if err != nil {
		return
	}

	timeControl := fmt.Sprintf("%d+%d", invite.TimeControlMinutes, invite.IncrementSeconds)
	err = s.Notifier.Send(ctx, invite.OpponentID, notification.TypeGameChallenge,
		"New challenge",
		creator.Username+" challenged you to a "+timeControl+" game",
		map[string]any{"code": invite.Code, "game_id": invite.GameID, "user_id": invite.CreatorID, "username": creator.Username},
	)
	if err != nil {
		slog.ErrorContext(ctx, "send challenge notification failed", "opponent_id", invite.OpponentID, "error", err)
	}
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

	if invite.OpponentID != 0 && invite.OpponentID != player2ID {
		return "", ErrNotYourChallenge
	}

	blocked, err := s.BlockChecker.IsBlockedEitherWay(ctx, invite.CreatorID, player2ID)
	if err != nil {
		slog.ErrorContext(ctx, "check block for join failed", "error", err, "code", code)
		return "", ErrInternal
	}
	if blocked {
		return "", ErrPlayerBlocked
	}

	if invite.OpponentID != 0 && s.PrivacyChecker != nil {
		if err := s.PrivacyChecker.CheckInviteFriend(ctx, player2ID, invite.CreatorID); err != nil {
			return "", err
		}
	}

	if invite.Rated {
		player2, err := s.UserFinder.FindUserByID(ctx, player2ID)
		if err != nil {
			slog.ErrorContext(ctx, "find invite player failed", "error", err, "user_id", player2ID)
			return "", ErrInternal
		}
		if player2.SkillLevel == nil {
			return "", ErrLevelNotPicked
		}
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
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if releaseErr := s.InviteRepo.ReleaseLock(releaseCtx, code, player2ID); releaseErr != nil {
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
		invite.Rated,
	)

	if err != nil {
		slog.ErrorContext(ctx, "create game from invite failed", "error", err)
		return "", fmt.Errorf("failed to create game: %w", err)
	}

	invite.Status = "active"
	_ = s.InviteRepo.SaveInvite(ctx, invite, 2*time.Minute)

	if invite.OpponentID != 0 {
		err = s.InviteRepo.RemoveChallenge(ctx, invite.OpponentID, code)
		if err != nil {
			slog.ErrorContext(ctx, "remove challenge after join failed", "error", err, "code", code)
		}
	}

	slog.InfoContext(ctx, "invite joined successfully",
		"code", code,
		"game_id", gameID,
		"player2_id", player2ID,
	)

	return gameID, nil
}

func (s *Service) GetMyChallenges(ctx context.Context, userID int64) ([]Challenge, error) {
	codes, err := s.InviteRepo.GetChallengeCodes(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get challenge codes failed", "error", err, "user_id", userID)
		return nil, ErrInternal
	}

	challenges := []Challenge{}
	for _, code := range codes {
		invite, err := s.InviteRepo.GetInvite(ctx, code)
		if errors.Is(err, redis.Nil) {
			_ = s.InviteRepo.RemoveChallenge(ctx, userID, code)
			continue
		}
		if err != nil {
			slog.ErrorContext(ctx, "get challenge invite failed", "error", err, "code", code)
			return nil, ErrInternal
		}

		if invite.Status == "active" {
			continue
		}

		blocked, err := s.BlockChecker.IsBlockedEitherWay(ctx, invite.CreatorID, userID)
		if err != nil {
			slog.ErrorContext(ctx, "check block for challenge list failed", "error", err, "code", code)
			continue
		}
		if blocked {
			continue
		}

		challenger, err := s.UserFinder.FindUserByID(ctx, invite.CreatorID)
		if err != nil {
			slog.ErrorContext(ctx, "find challenger failed", "error", err, "creator_id", invite.CreatorID)
			continue
		}

		challenges = append(challenges, Challenge{
			Code:               invite.Code,
			GameID:             invite.GameID,
			ChallengerID:       challenger.ID,
			ChallengerUsername: challenger.Username,
			ChallengerColor:    invite.Color,
			TimeControlMinutes: invite.TimeControlMinutes,
			IncrementSeconds:   invite.IncrementSeconds,
			Rated:              invite.Rated,
		})
	}

	return challenges, nil
}

func (s *Service) DeclineChallenge(ctx context.Context, userID int64, code string) error {
	invite, err := s.InviteRepo.GetInvite(ctx, code)
	if errors.Is(err, redis.Nil) {
		_ = s.InviteRepo.RemoveChallenge(ctx, userID, code)
		return ErrInviteNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "get invite for decline failed", "error", err, "code", code)
		return ErrInternal
	}

	if invite.OpponentID != userID {
		return ErrInviteNotFound
	}

	if invite.Status == "active" {
		return ErrInviteUsed
	}

	err = s.InviteRepo.DeleteInvite(ctx, code)
	if err != nil {
		slog.ErrorContext(ctx, "delete declined invite failed", "error", err, "code", code)
		return ErrInternal
	}

	err = s.InviteRepo.RemoveChallenge(ctx, userID, code)
	if err != nil {
		slog.ErrorContext(ctx, "remove declined challenge failed", "error", err, "code", code)
		return ErrInternal
	}

	return nil
}

func (s *Service) CreateWatchInvite(ctx context.Context, userID int64, gameID string) (*WatchInvite, error) {
	currentGame, err := s.GameRepository.FindGameByID(ctx, gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find game for watch invite failed", "error", err, "game_id", gameID)
		return nil, ErrInternal
	}

	if userID != currentGame.WhitePlayerID && userID != currentGame.BlackPlayerID {
		return nil, ErrNotYourGame
	}

	if currentGame.Status != game.StatusActive {
		return nil, ErrGameNotActive
	}

	existing, err := s.InviteRepo.GetWatchInviteByGameID(ctx, gameID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, redis.Nil) {
		slog.ErrorContext(ctx, "get watch invite by game id failed", "error", err, "game_id", gameID)
		return nil, ErrInternal
	}

	code, err := generateInviteCode()
	if err != nil {
		slog.ErrorContext(ctx, "generate watch code failed", "error", err)
		return nil, ErrInternal
	}

	invite := &WatchInvite{
		Code:      code,
		GameID:    gameID,
		CreatedBy: userID,
	}

	err = s.InviteRepo.SaveWatchInvite(ctx, invite, 24*time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "save watch invite failed", "error", err, "game_id", gameID)
		return nil, ErrInternal
	}

	return invite, nil
}

func (s *Service) GetWatchInvite(ctx context.Context, code string) (*WatchInvite, error) {
	invite, err := s.InviteRepo.GetWatchInvite(ctx, code)
	if errors.Is(err, redis.Nil) {
		return nil, ErrWatchLinkNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "get watch invite failed", "error", err)
		return nil, ErrInternal
	}

	return invite, nil
}

func (s *Service) DeleteWatchInvite(ctx context.Context, gameID string) error {
	err := s.InviteRepo.DeleteWatchInvite(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "delete watch invite failed", "error", err, "game_id", gameID)
		return ErrInternal
	}
	return nil
}
