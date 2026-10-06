package chat

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type ChatService interface {
	SendMessage(ctx context.Context, gameID string, userID int64, message string) (*ChatMessage, error)
	GetMessages(ctx context.Context, gameID string, userID int64) ([]ChatMessage, error)
}

type Service struct {
	ChatRepo ChatRepository
	GameRepo game.GameRepository
}

func NewService(chatRepo ChatRepository, gameRepo game.GameRepository) ChatService {
	return &Service{
		ChatRepo: chatRepo,
		GameRepo: gameRepo,
	}
}

func (s *Service) SendMessage(
	ctx context.Context,
	gameID string,
	userID int64,
	message string,
) (*ChatMessage, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return nil, errors.New("message cannot be empty")
	}

	if len(trimmed) > 1000 {
		return nil, errors.New("message exceeds maximum length of 1000 characters")
	}

	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "find game failed for chat message", "game_id", gameID, "error", err)
		return nil, errors.New("game not found")
	}

	if userID != g.WhitePlayerID && userID != g.BlackPlayerID {
		slog.WarnContext(ctx, "unauthorized chat message attempt", "game_id", gameID, "user_id", userID)
		return nil, errors.New("player is not part of this game")
	}

	msg := &ChatMessage{
		ID:        uuid.NewString(),
		GameID:    gameID,
		UserID:    userID,
		Message:   trimmed,
		CreatedAt: time.Now(),
	}

	err = s.ChatRepo.CreateMessage(ctx, msg)
	if err != nil {
		slog.ErrorContext(ctx, "create chat message failed in service", "game_id", gameID, "user_id", userID, "error", err)
		return nil, err
	}

	return msg, nil
}

func (s *Service) GetMessages(
	ctx context.Context,
	gameID string,
	userID int64,
) ([]ChatMessage, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "find game failed for get chat messages", "game_id", gameID, "error", err)
		return nil, errors.New("game not found")
	}

	if userID != g.WhitePlayerID && userID != g.BlackPlayerID {
		slog.WarnContext(ctx, "unauthorized get chat messages attempt", "game_id", gameID, "user_id", userID)
		return nil, errors.New("player is not part of this game")
	}

	messages, err := s.ChatRepo.GetMessagesByGameID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if messages == nil {
		messages = []ChatMessage{}
	}

	return messages, nil
}
