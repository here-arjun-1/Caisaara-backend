package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AvatarRepository interface {
	GetAvatarURL(ctx context.Context, userID int64) (*string, error)
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL *string) error
}

type AvatarStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
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

func (s *AvatarService) deletePhoto(ctx context.Context, url string) {
	key, ok := s.AvatarStorage.KeyFromURL(url)
	if !ok || !strings.HasPrefix(key, "avatars/users/") {
		return
	}

	if err := s.AvatarStorage.Delete(ctx, key); err != nil {
		slog.Error("delete avatar failed", "key", key, "error", err)
	}
}
