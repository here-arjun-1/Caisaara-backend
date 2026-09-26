package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
)

type SessionRepository struct {
	DB *pgx.Conn
}

func NewSessionRepository(db *pgx.Conn) *SessionRepository {
	return &SessionRepository{
		DB: db,
	}
}

func (h *SessionRepository) CreateSession(session *model.Session) error {
	_, err := h.DB.Exec(
		context.Background(),
		`INSERT INTO sessions
		(user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
	)
	return err
}
func (h *SessionRepository) FindSessionByRefreshTokenHash(refreshTokenHash string) (*model.Session, error) {
	var session model.Session
	err := h.DB.QueryRow(
		context.Background(),
		`SELECT
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			created_at,
			revoked_at
		FROM sessions
		WHERE refresh_token_hash = $1`,
		refreshTokenHash,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
