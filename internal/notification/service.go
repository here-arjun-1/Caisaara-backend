package notification

import (
	"context"
	"math"
)

type Service struct {
	Repository  *Repository
	Broadcaster func(userID int64, n *Notification)
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SetBroadcaster(broadcaster func(userID int64, n *Notification)) {
	s.Broadcaster = broadcaster
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

	if s.Broadcaster != nil {
		s.Broadcaster(userID, n)
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
