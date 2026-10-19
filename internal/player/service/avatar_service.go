package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/jackc/pgx/v5"
)

var defaultAvatarIDs = []string{"king", "queen", "rook", "bishop", "knight", "pawn"}

type AvatarRepository interface {
	GetAvatarURL(ctx context.Context, userID int64) (*string, error)
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL *string) error
}

type AvatarStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
	KeyFromURL(url string) (string, bool)
}

type AvatarService struct {
	AvatarRepository AvatarRepository
	AvatarStorage    AvatarStorage
}

func NewAvatarService(avatarRepository AvatarRepository, avatarStorage AvatarStorage) *AvatarService {
	return &AvatarService{
		AvatarRepository: avatarRepository,
		AvatarStorage:    avatarStorage,
	}
}

func (s *AvatarService) UploadPhoto(ctx context.Context, userID int64, data []byte) (string, error) {
	avatar, err := makeAvatar(data)
	if errors.Is(err, ErrInvalidPhoto) || errors.Is(err, ErrPhotoTooLarge) {
		return "", err
	}
	if err != nil {
		slog.Error("make avatar failed", "error", err)
		return "", ErrInternal
	}

	oldURL, err := s.AvatarRepository.GetAvatarURL(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		slog.Error("get avatar url failed", "error", err)
		return "", ErrInternal
	}

	key := fmt.Sprintf("avatars/users/%d/%s.jpg", userID, uuid.NewString())
	newURL, err := s.AvatarStorage.Upload(ctx, key, avatar, "image/jpeg")
	if err != nil {
		slog.Error("upload avatar failed", "error", err)
		return "", ErrInternal
	}

	if err := s.AvatarRepository.UpdateAvatarURL(ctx, userID, &newURL); err != nil {
		slog.Error("save avatar url failed", "error", err)
		s.deletePhoto(ctx, newURL)
		return "", ErrInternal
	}

	if oldURL != nil {
		s.deletePhoto(ctx, *oldURL)
	}

	return newURL, nil
}

func (s *AvatarService) ListDefaultAvatars() []dto.DefaultAvatarResponse {
	avatars := make([]dto.DefaultAvatarResponse, 0, len(defaultAvatarIDs))
	for _, id := range defaultAvatarIDs {
		avatars = append(avatars, dto.DefaultAvatarResponse{
			ID:  id,
			URL: s.AvatarStorage.URL("avatars/defaults/" + id + ".png"),
		})
	}
	return avatars
}

func (s *AvatarService) ChooseDefault(ctx context.Context, userID int64, avatarID string) (string, error) {
	avatarID = strings.ToLower(strings.TrimSpace(avatarID))
	if !isDefaultAvatar(avatarID) {
		return "", ErrInvalidAvatarID
	}

	oldURL, err := s.AvatarRepository.GetAvatarURL(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		slog.Error("get avatar url failed", "error", err)
		return "", ErrInternal
	}

	newURL := s.AvatarStorage.URL("avatars/defaults/" + avatarID + ".png")
	if err := s.AvatarRepository.UpdateAvatarURL(ctx, userID, &newURL); err != nil {
		slog.Error("save avatar url failed", "error", err)
		return "", ErrInternal
	}

	if oldURL != nil {
		s.deletePhoto(ctx, *oldURL)
	}

	return newURL, nil
}

func (s *AvatarService) RemoveAvatar(ctx context.Context, userID int64) error {
	oldURL, err := s.AvatarRepository.GetAvatarURL(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		slog.Error("get avatar url failed", "error", err)
		return ErrInternal
	}

	if err := s.AvatarRepository.UpdateAvatarURL(ctx, userID, nil); err != nil {
		slog.Error("remove avatar url failed", "error", err)
		return ErrInternal
	}

	if oldURL != nil {
		s.deletePhoto(ctx, *oldURL)
	}

	return nil
}

func (s *AvatarService) deletePhoto(ctx context.Context, url string) {
	key, ok := s.AvatarStorage.KeyFromURL(url)
	if !ok || !strings.HasPrefix(key, "avatars/users/") {
		return
	}

	if err := s.AvatarStorage.Delete(ctx, key); err != nil {
		slog.Error("delete avatar failed", "key", key, "error", err)
	}
}

func isDefaultAvatar(avatarID string) bool {
	for _, id := range defaultAvatarIDs {
		if id == avatarID {
			return true
		}
	}
	return false
}
