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
	DeleteMessages(ctx context.Context, gameID string) error

	SendDirectMessage(ctx context.Context, senderID, receiverID int64, message string) (*DirectMessage, error)
	GetDirectMessages(ctx context.Context, currentUserID, otherUserID int64) ([]DirectMessage, error)
	GetConversations(ctx context.Context, userID int64) ([]Conversation, error)
}

type Service struct {
	ChatRepo    ChatRepository
	GameRepo    game.GameRepository
	RateLimiter RateLimiter
}

func NewService(chatRepo ChatRepository, gameRepo game.GameRepository, rateLimiter ...RateLimiter) ChatService {
	s := &Service{
		ChatRepo: chatRepo,
		GameRepo: gameRepo,
	}
	if len(rateLimiter) > 0 {
		s.RateLimiter = rateLimiter[0]
	}
	return s
}

func (s *Service) SendMessage(ctx context.Context, gameID string, userID int64, message string) (*ChatMessage, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return nil, errors.New("message cannot be empty")
	}
	if len(trimmed) > 500 {
		return nil, errors.New("message exceeds maximum length of 500 characters")
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

	if g.Status != game.StatusActive {
		slog.WarnContext(ctx, "cannot send chat message in finished game", "game_id", gameID, "status", g.Status)
		return nil, errors.New("cannot send chat messages in a finished game")
	}

	if s.RateLimiter != nil {
		allowed, err := s.RateLimiter.Allow(ctx, gameID, userID)
		if err == nil && !allowed {
			slog.WarnContext(ctx, "chat rate limit exceeded", "game_id", gameID, "user_id", userID)
			return nil, errors.New("rate limit exceeded, please wait")
		}
	}

	var username string
	switch userID {
	case g.WhitePlayerID:
		username = g.WhiteUsername
	case g.BlackPlayerID:
		username = g.BlackUsername
	}

	msg := &ChatMessage{
		ID:        uuid.NewString(),
		GameID:    gameID,
		UserID:    userID,
		Username:  username,
		Message:   trimmed,
		CreatedAt: time.Now(),
	}

	if err := s.ChatRepo.CreateMessage(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "create chat message failed in service", "game_id", gameID, "user_id", userID, "error", err)
		return nil, err
	}

	return msg, nil
}

func (s *Service) GetMessages(ctx context.Context, gameID string, userID int64) ([]ChatMessage, error) {
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

func (s *Service) DeleteMessages(ctx context.Context, gameID string) error {
	return s.ChatRepo.DeleteMessagesByGameID(ctx, gameID)
}

func (s *Service) SendDirectMessage(ctx context.Context, senderID, receiverID int64, message string) (*DirectMessage, error) {
	if senderID == receiverID {
		return nil, errors.New("cannot send message to yourself")
	}

	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return nil, errors.New("message cannot be empty")
	}
	if len(trimmed) > 500 {
		return nil, errors.New("message exceeds maximum length of 500 characters")
	}

	if s.RateLimiter != nil {
		allowed, err := s.RateLimiter.Allow(ctx, "dm", senderID)
		if err == nil && !allowed {
			slog.WarnContext(ctx, "direct message rate limit exceeded", "sender_id", senderID)
			return nil, errors.New("rate limit exceeded, please wait")
		}
	}

	exists, err := s.ChatRepo.UserExists(ctx, receiverID)
	if err != nil {
		slog.ErrorContext(ctx, "check receiver exists failed", "receiver_id", receiverID, "error", err)
		return nil, errors.New("internal server error")
	}
	if !exists {
		return nil, errors.New("recipient user not found")
	}

	msg := &DirectMessage{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Message:    trimmed,
	}

	if err := s.ChatRepo.CreateDirectMessage(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "create direct message failed", "sender_id", senderID, "receiver_id", receiverID, "error", err)
		return nil, err
	}

	return msg, nil
}

func (s *Service) GetDirectMessages(ctx context.Context, currentUserID, otherUserID int64) ([]DirectMessage, error) {
	if currentUserID == otherUserID {
		return nil, errors.New("invalid user id")
	}

	messages, err := s.ChatRepo.GetDirectMessages(ctx, currentUserID, otherUserID)
	if err != nil {
		slog.ErrorContext(ctx, "get direct messages failed", "user_id", currentUserID, "other_user_id", otherUserID, "error", err)
		return nil, err
	}

	if err := s.ChatRepo.MarkDirectMessagesAsRead(ctx, currentUserID, otherUserID); err != nil {
		slog.WarnContext(ctx, "mark direct messages as read failed", "user_id", currentUserID, "other_user_id", otherUserID, "error", err)
	}

	return messages, nil
}

func (s *Service) GetConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	conversations, err := s.ChatRepo.GetConversations(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get conversations failed", "user_id", userID, "error", err)
		return nil, err
	}
	return conversations, nil
}
