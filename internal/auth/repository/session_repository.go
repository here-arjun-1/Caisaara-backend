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

func (r *SessionRepository) CreateSession(ctx context.Context, session *model.Session) error {

	_, err := r.DB.Exec(
		ctx,
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
	ctx context.Context,
	refreshTokenHash string,
) (*model.Session, error) {

	var session model.Session

	err := r.DB.QueryRow(
		ctx,
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

func (r *SessionRepository) RotateSession(
	ctx context.Context,
	oldRefreshTokenHash string,
	newSession *model.Session,
) error {

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()
	err = tx.QueryRow(
		ctx,
		`UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE refresh_token_hash = $1
		AND revoked_at IS NULL
		AND expires_at > CURRENT_TIMESTAMP
		RETURNING user_id`,
		oldRefreshTokenHash,
	).Scan(&newSession.UserID)

	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO sessions
		(user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		newSession.UserID,
		newSession.RefreshTokenHash,
		newSession.ExpiresAt,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SessionRepository) RevokeSession(
	ctx context.Context,
	refreshTokenHash string,
) error {

	result, err := r.DB.Exec(
		ctx,
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
	ctx context.Context,
	userID int64,
) error {

	_, err := r.DB.Exec(
		ctx,
		`UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		AND revoked_at IS NULL`,
		userID,
	)

	return err
}

func (r *SessionRepository) CleanExpiredSessions(ctx context.Context) (int64, error) {
	result, err := r.DB.Exec(
		ctx,
		`DELETE FROM sessions 
		WHERE expires_at < CURRENT_TIMESTAMP
		OR revoked_at IS NOT NULL`,
	)

	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
