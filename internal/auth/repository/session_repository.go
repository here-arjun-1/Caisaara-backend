package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	DB *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		DB: db,
	}
}

func (r *SessionRepository) CreateSession(session *model.Session) error {

	_, err := r.DB.Exec(
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

func (r *SessionRepository) FindSessionByRefreshTokenHash(
	refreshTokenHash string,
) (*model.Session, error) {

	var session model.Session

	err := r.DB.QueryRow(
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

func (r *SessionRepository) RevokeSession(
	refreshTokenHash string,
) error {

	result, err := r.DB.Exec(
		context.Background(),
		`UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE refresh_token_hash = $1
		AND revoked_at IS NULL`,
		refreshTokenHash,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *SessionRepository) RevokeAllSessions(
	userID int64,
) error {

	_, err := r.DB.Exec(
		context.Background(),
		`UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		AND revoked_at IS NULL`,
		userID,
	)

	return err
}
