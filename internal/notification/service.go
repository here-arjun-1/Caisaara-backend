package notification

import (
	"context"
	"log/slog"
	"math"

	"github.com/hibiken/asynq"
)

type Service struct {
	Repository  *Repository
	Broadcaster func(userID int64, n *Notification)
	IsOnline    func(userID int64) bool
	TaskClient  *asynq.Client
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SetBroadcaster(broadcaster func(userID int64, n *Notification)) {
	s.Broadcaster = broadcaster
}

func (s *Service) SetOnlineChecker(isOnline func(userID int64) bool) {
	s.IsOnline = isOnline
}

func (s *Service) SetTaskClient(client *asynq.Client) {
	s.TaskClient = client
}

func (s *Service) Send(ctx context.Context, userID int64, notificationType string, title string, body string, data map[string]any) error {
	n := &Notification{
		UserID: userID,
		Type:   notificationType,
		Title:  title,
		Body:   body,
		Data:   data,
	}

	if err := s.Repository.Create(ctx, n); err != nil {
		return err
	}

	if s.IsOnline != nil && s.IsOnline(userID) {
		if s.Broadcaster != nil {
			s.Broadcaster(userID, n)
		}
		return nil
	}

	if s.TaskClient == nil {
		return nil
	}

	task, err := NewPushTask(n)
	if err != nil {
		slog.ErrorContext(ctx, "create push task failed", "user_id", userID, "error", err)
		return nil
	}
	if _, err := s.TaskClient.EnqueueContext(ctx, task); err != nil {
		slog.ErrorContext(ctx, "enqueue push task failed", "user_id", userID, "error", err)
	}
	return nil
}

func (s *Service) GetNotifications(ctx context.Context, userID int64, beforeID int64, limit int) ([]Notification, error) {
	if beforeID <= 0 {
		beforeID = math.MaxInt64
	}
	if limit <= 0 || limit > 50 {
		limit = 30
	}
	return s.Repository.FindByUser(ctx, userID, beforeID, limit)
}

func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.Repository.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID int64, notificationID int64) error {
	found, err := s.Repository.MarkRead(ctx, notificationID, userID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotificationNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	return s.Repository.MarkAllRead(ctx, userID)
}

func (s *Service) RegisterDevice(ctx context.Context, userID int64, token string, platform string) error {
	return s.Repository.SaveDeviceToken(ctx, userID, token, platform)
}

func (s *Service) UnregisterDevice(ctx context.Context, userID int64, token string) error {
	return s.Repository.DeleteDeviceToken(ctx, userID, token)
}
