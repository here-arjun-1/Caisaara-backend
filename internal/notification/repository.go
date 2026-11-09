package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) SaveDeviceToken(ctx context.Context, userID int64, token string, platform string) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO device_tokens (user_id, token, platform) VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, updated_at = NOW()`,
		userID, token, platform,
	)
	if err != nil {
		slog.ErrorContext(ctx, "save device token failed", "user_id", userID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) DeleteDeviceToken(ctx context.Context, userID int64, token string) error {
	_, err := r.DB.Exec(ctx,
		`DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`,
		token, userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete device token failed", "user_id", userID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) DeleteDeadToken(ctx context.Context, token string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM device_tokens WHERE token = $1`, token)
	if err != nil {
		slog.ErrorContext(ctx, "delete dead device token failed", "error", err)
		return err
	}
	return nil
}

func (r *Repository) FindDeviceTokens(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT token FROM device_tokens WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find device tokens failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	tokens := []string{}
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			slog.ErrorContext(ctx, "scan device token failed", "user_id", userID, "error", err)
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "device tokens rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return tokens, nil
}

func (r *Repository) IsTypeEnabled(ctx context.Context, userID int64, notificationType string) (bool, error) {
	var enabled bool
	err := r.DB.QueryRow(ctx,
		`SELECT enabled FROM notification_preferences WHERE user_id = $1 AND type = $2`,
		userID, notificationType).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "check notification preference failed", "user_id", userID, "type", notificationType, "error", err)
		return false, err
	}
	return enabled, nil
}

func (r *Repository) FindPreferences(ctx context.Context, userID int64) (map[string]bool, error) {
	preferences := map[string]bool{}
	rows, err := r.DB.Query(ctx,
		`SELECT type, enabled FROM notification_preferences WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find notification preferences failed", "user_id", userID, "error", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var notificationType string
		var enabled bool

		if err := rows.Scan(&notificationType, &enabled); err != nil {
			slog.ErrorContext(ctx, "scan notification preference failed", "user_id", userID, "error", err)
			return nil, err
		}
		preferences[notificationType] = enabled
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "notification preferences rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return preferences, nil
}

func (r *Repository) SavePreference(ctx context.Context, userID int64, notificationType string, enabled bool) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO notification_preferences (user_id, type, enabled) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, type) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()`,
		userID, notificationType, enabled,
	)
	if err != nil {
		slog.ErrorContext(ctx, "save notification preference failed", "user_id", userID, "type", notificationType, "error", err)
		return err
	}
	return nil
}
