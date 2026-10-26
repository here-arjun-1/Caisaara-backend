package notification

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	if n.Data == nil {
		n.Data = map[string]any{}
	}

	err := r.DB.QueryRow(ctx,
		`INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_read, created_at`,
		n.UserID, n.Type, n.Title, n.Body, n.Data,
	).Scan(&n.ID, &n.IsRead, &n.CreatedAt)
	if err != nil {
		slog.ErrorContext(ctx, "create notification failed", "user_id", n.UserID, "type", n.Type, "error", err)
		return err
	}
	return nil
}

func (r *Repository) FindByUser(ctx context.Context, userID int64, beforeID int64, limit int) ([]Notification, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT id, user_id, type, title, body, data, is_read, created_at
		FROM notifications
		WHERE user_id = $1 AND id < $2
		ORDER BY id DESC
		LIMIT $3`,
		userID, beforeID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find notifications failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []Notification{}
	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt)
		if err != nil {
			slog.ErrorContext(ctx, "scan notification row failed", "user_id", userID, "error", err)
			return nil, err
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "notifications rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return notifications, nil
}

func (r *Repository) CountUnread(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`,
		userID,
	).Scan(&count)
	if err != nil {
		slog.ErrorContext(ctx, "count unread notifications failed", "user_id", userID, "error", err)
		return 0, err
	}
	return count, nil
}

func (r *Repository) MarkRead(ctx context.Context, notificationID int64, userID int64) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`,
		notificationID, userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "mark notification read failed", "notification_id", notificationID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := r.DB.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "mark all notifications read failed", "user_id", userID, "error", err)
		return err
	}
	return nil
}
