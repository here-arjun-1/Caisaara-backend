package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"github.com/here-arjun-1/Caisaara-backend/internal/bot"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"golang.org/x/crypto/bcrypt"
)

type GameResigner interface {
	ResignGame(ctx context.Context, gameID string, playerID int64) (*game.Game, error)
}

type BotGameResigner interface {
	Resign(ctx context.Context, gameID string, playerID int64) (*bot.Game, error)
}

type TournamentRemover interface {
	RemoveUserFromTournaments(ctx context.Context, userID int64) error
}

type QueueLeaver interface {
	Cancel(ctx context.Context, userID int64) error
}

type Disconnector interface {
	DisconnectUser(userID int64)
}

type AvatarStorage interface {
	Delete(ctx context.Context, key string) error
	KeyFromURL(url string) (string, bool)
}

type Service struct {
	Repository    *Repository
	Games         GameResigner
	BotGames      BotGameResigner
	Tournaments   TournamentRemover
	Matchmaking   QueueLeaver
	Disconnector  Disconnector
	AvatarStorage AvatarStorage
}

func NewService(
	repository *Repository,
	games GameResigner,
	botGames BotGameResigner,
	tournaments TournamentRemover,
	matchmaking QueueLeaver,
	disconnector Disconnector,
	avatarStorage AvatarStorage,
) *Service {
	return &Service{
		Repository:    repository,
		Games:         games,
		BotGames:      botGames,
		Tournaments:   tournaments,
		Matchmaking:   matchmaking,
		Disconnector:  disconnector,
		AvatarStorage: avatarStorage,
	}
}

func (s *Service) DeleteAccount(ctx context.Context, userID int64, password string) error {
	passwordHash, err := s.Repository.FindPasswordHash(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return ErrWrongPassword
	}

	avatarURL, err := s.Repository.FindAvatarURL(ctx, userID)
	if err != nil {
		return ErrInternal
	}

	if err := s.Tournaments.RemoveUserFromTournaments(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "remove user from tournaments failed", "user_id", userID, "error", err)
		return ErrInternal
	}

	newPasswordHash, err := randomPasswordHash()
	if err != nil {
		slog.ErrorContext(ctx, "make random password failed", "user_id", userID, "error", err)
		return ErrInternal
	}

	err = s.Repository.DeleteUserData(ctx, userID, newPasswordHash)
	if errors.Is(err, ErrUserNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return ErrInternal
	}

	if avatarURL != nil {
		s.deleteAvatar(ctx, userID, *avatarURL)
	}

	if err := s.Matchmaking.Cancel(ctx, userID); err != nil {
		slog.InfoContext(ctx, "remove deleted user from queue skipped", "user_id", userID, "error", err)
	}

	s.resignActiveGames(ctx, userID)
	s.resignBotGames(ctx, userID)

	if s.Disconnector != nil {
		s.Disconnector.DisconnectUser(userID)
	}

	slog.InfoContext(ctx, "account deleted", "user_id", userID)
	return nil
}

func (s *Service) resignActiveGames(ctx context.Context, userID int64) {
	gameIDs, err := s.Repository.FindActiveGameIDs(ctx, userID)
	if err != nil {
		return
	}

	for _, gameID := range gameIDs {
		if _, err := s.Games.ResignGame(ctx, gameID, userID); err != nil {
			slog.ErrorContext(ctx, "resign game for deleted user failed", "user_id", userID, "game_id", gameID, "error", err)
		}
	}
}

func (s *Service) resignBotGames(ctx context.Context, userID int64) {
	gameIDs, err := s.Repository.FindActiveBotGameIDs(ctx, userID)
	if err != nil {
		return
	}

	for _, gameID := range gameIDs {
		if _, err := s.BotGames.Resign(ctx, gameID, userID); err != nil {
			slog.ErrorContext(ctx, "resign bot game for deleted user failed", "user_id", userID, "game_id", gameID, "error", err)
		}
	}
}

func (s *Service) deleteAvatar(ctx context.Context, userID int64, url string) {
	key, ok := s.AvatarStorage.KeyFromURL(url)
	if !ok || !strings.HasPrefix(key, "avatars/users/") {
		return
	}

	if err := s.AvatarStorage.Delete(ctx, key); err != nil {
		slog.ErrorContext(ctx, "delete avatar for deleted user failed", "user_id", userID, "key", key, "error", err)
	}
}

func randomPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(b)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
