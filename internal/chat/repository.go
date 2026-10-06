package chat

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	CreateMessage(ctx context.Context, msg *ChatMessage) error
	GetMessagesByGameID(ctx context.Context, gameID string) ([]ChatMessage, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) ChatRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateMessage(
	ctx context.Context,
	msg *ChatMessage,
) error {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}

	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO game_chat_messages (
			id,
			game_id,
			user_id,
			message,
			created_at
		)
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

func (r *Repository) GetMessagesByGameID(
	ctx context.Context,
	gameID string,
) ([]ChatMessage, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			id,
			game_id,
			user_id,
			message,
			created_at
		FROM game_chat_messages
		WHERE game_id = $1
		ORDER BY created_at ASC`,
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
		err := rows.Scan(
			&msg.ID,
			&msg.GameID,
			&msg.UserID,
			&msg.Message,
			&msg.CreatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan chat message row failed", "game_id", gameID, "error", err)
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get chat messages rows iteration error", "game_id", gameID, "error", err)
		return nil, err
	}

	return messages, nil
}
