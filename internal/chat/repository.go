package chat

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	CreateMessage(ctx context.Context, msg *ChatMessage) error
	GetMessagesByGameID(ctx context.Context, gameID string) ([]ChatMessage, error)
	DeleteMessagesByGameID(ctx context.Context, gameID string) error

	CreateDirectMessage(ctx context.Context, msg *DirectMessage) error
	GetDirectMessages(ctx context.Context, user1ID, user2ID int64) ([]DirectMessage, error)
	MarkDirectMessagesAsRead(ctx context.Context, receiverID, senderID int64) error
	GetConversations(ctx context.Context, userID int64) ([]Conversation, error)
	UserExists(ctx context.Context, userID int64) (bool, error)
	FindUsernameByID(ctx context.Context, userID int64) (string, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) ChatRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateMessage(ctx context.Context, msg *ChatMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}

	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO game_chat_messages (id, game_id, user_id, message, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		msg.ID,
		msg.GameID,
		msg.UserID,
		msg.Message,
		msg.CreatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "create chat message failed", "game_id", msg.GameID, "user_id", msg.UserID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) GetMessagesByGameID(ctx context.Context, gameID string) ([]ChatMessage, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT m.id, m.game_id, m.user_id, COALESCE(u.username, ''), m.message, m.created_at
		FROM game_chat_messages m
		LEFT JOIN users u ON u.id = m.user_id
		WHERE m.game_id = $1
		ORDER BY m.created_at ASC`,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get chat messages query failed", "game_id", gameID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.ID, &msg.GameID, &msg.UserID, &msg.Username, &msg.Message, &msg.CreatedAt); err != nil {
			slog.ErrorContext(ctx, "scan chat message row failed", "game_id", gameID, "error", err)
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *Repository) DeleteMessagesByGameID(ctx context.Context, gameID string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM game_chat_messages WHERE game_id = $1`, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "delete chat messages failed", "game_id", gameID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) CreateDirectMessage(ctx context.Context, msg *DirectMessage) error {
	return r.DB.QueryRow(
		ctx,
		`INSERT INTO direct_messages (sender_id, receiver_id, message)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		msg.SenderID,
		msg.ReceiverID,
		msg.Message,
	).Scan(&msg.ID, &msg.CreatedAt)
}

func (r *Repository) GetDirectMessages(ctx context.Context, user1ID, user2ID int64) ([]DirectMessage, error) {
	// inner query picks the newest 100, outer query puts them back in old -> new order for the chat screen
	rows, err := r.DB.Query(
		ctx,
		`SELECT id, sender_id, receiver_id, message, created_at
		FROM (
			SELECT id, sender_id, receiver_id, message, created_at
			FROM direct_messages
			WHERE (sender_id = $1 AND receiver_id = $2)
			   OR (sender_id = $2 AND receiver_id = $1)
			ORDER BY created_at DESC
			LIMIT 100
		) latest
		ORDER BY created_at ASC`,
		user1ID,
		user2ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]DirectMessage, 0)
	for rows.Next() {
		var msg DirectMessage
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Message, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *Repository) MarkDirectMessagesAsRead(ctx context.Context, receiverID, senderID int64) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE direct_messages
		SET is_read = true
		WHERE receiver_id = $1 AND sender_id = $2 AND is_read = false`,
		receiverID,
		senderID,
	)
	return err
}

func (r *Repository) GetConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := r.DB.Query(
		ctx,
		`WITH user_conversations AS (
			SELECT DISTINCT ON (
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END
			)
				message,
				created_at,
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END AS other_user_id
			FROM direct_messages
			WHERE sender_id = $1 OR receiver_id = $1
			ORDER BY
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END,
				created_at DESC
		),
		unread_counts AS (
			SELECT sender_id AS other_user_id, COUNT(*) AS unread_count
			FROM direct_messages
			WHERE receiver_id = $1 AND is_read = false
			GROUP BY sender_id
		)
		SELECT
			uc.other_user_id,
			u.username,
			p.display_name,
			p.avatar_url,
			uc.message,
			uc.created_at,
			COALESCE(un.unread_count, 0)
		FROM user_conversations uc
		JOIN users u ON u.id = uc.other_user_id
		LEFT JOIN player_profiles p ON p.user_id = uc.other_user_id
		LEFT JOIN unread_counts un ON un.other_user_id = uc.other_user_id
		ORDER BY uc.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]Conversation, 0)
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(
			&conv.UserID,
			&conv.Username,
			&conv.DisplayName,
			&conv.AvatarURL,
			&conv.LastMessage,
			&conv.LastMessageAt,
			&conv.UnreadCount,
		); err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

func (r *Repository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var id int64
	err := r.DB.QueryRow(ctx, `SELECT id FROM users WHERE id = $1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) FindUsernameByID(ctx context.Context, userID int64) (string, error) {
	var username string
	err := r.DB.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, userID).Scan(&username)
	if err != nil {
		slog.ErrorContext(ctx, "find username failed", "user_id", userID, "error", err)
		return "", err
	}
	return username, nil
}
