package account

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

func (r *Repository) FindPasswordHash(ctx context.Context, userID int64) (string, error) {
	var passwordHash string
	err := r.DB.QueryRow(
		ctx,
		`SELECT password FROM users WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find password hash failed", "user_id", userID, "error", err)
		return "", err
	}
	return passwordHash, nil
}

func (r *Repository) FindAvatarURL(ctx context.Context, userID int64) (*string, error) {
	var avatarURL *string
	err := r.DB.QueryRow(
		ctx,
		`SELECT avatar_url FROM player_profiles WHERE user_id = $1`,
		userID,
	).Scan(&avatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "find avatar url failed", "user_id", userID, "error", err)
		return nil, err
	}
	return avatarURL, nil
}

func (r *Repository) FindActiveGameIDs(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT id::text FROM games
		WHERE status = 'active' AND (white_player_id = $1 OR black_player_id = $1)`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find active games failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	gameIDs := []string{}
	for rows.Next() {
		var gameID string
		if err := rows.Scan(&gameID); err != nil {
			return nil, err
		}
		gameIDs = append(gameIDs, gameID)
	}

	return gameIDs, rows.Err()
}

func (r *Repository) FindActiveBotGameIDs(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT id::text FROM bot_games WHERE player_id = $1 AND status = 'active'`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find active bot games failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	gameIDs := []string{}
	for rows.Next() {
		var gameID string
		if err := rows.Scan(&gameID); err != nil {
			return nil, err
		}
		gameIDs = append(gameIDs, gameID)
	}

	return gameIDs, rows.Err()
}

func (r *Repository) DeleteUserData(ctx context.Context, userID int64, newPasswordHash string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(
		ctx,
		`UPDATE users
		SET username = 'deleted_user_' || id,
			email = NULL,
			password = $2,
			email_verification_token = NULL,
			email_verification_expires_at = NULL,
			password_version = COALESCE(password_version, 1) + 1,
			deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
		newPasswordHash,
	)
	if err != nil {
		slog.ErrorContext(ctx, "anonymize user failed", "user_id", userID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if err := handOverClubs(ctx, tx, userID); err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE player_profiles
		SET display_name = NULL, country = NULL, bio = NULL, avatar_url = NULL, updated_at = NOW()
		WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "clear player profile failed", "user_id", userID, "error", err)
		return err
	}

	deleteQueries := []string{
		`DELETE FROM sessions WHERE user_id = $1`,
		`DELETE FROM device_tokens WHERE user_id = $1`,
		`DELETE FROM notifications WHERE user_id = $1`,
		`DELETE FROM player_locations WHERE user_id = $1`,
		`DELETE FROM friend_requests WHERE sender_id = $1 OR receiver_id = $1`,
		`DELETE FROM user_blocks WHERE blocker_id = $1 OR blocked_id = $1`,
		`DELETE FROM club_members WHERE user_id = $1`,
		`DELETE FROM post_reactions WHERE user_id = $1`,
		`DELETE FROM post_comments WHERE user_id = $1`,
		`DELETE FROM posts WHERE user_id = $1`,
	}
	for _, query := range deleteQueries {
		if _, err := tx.Exec(ctx, query, userID); err != nil {
			slog.ErrorContext(ctx, "delete user data failed", "user_id", userID, "query", query, "error", err)
			return err
		}
	}

	return tx.Commit(ctx)
}

func handOverClubs(ctx context.Context, tx pgx.Tx, userID int64) error {
	rows, err := tx.Query(ctx, `SELECT id::text FROM clubs WHERE owner_id = $1`, userID)
	if err != nil {
		slog.ErrorContext(ctx, "find owned clubs failed", "user_id", userID, "error", err)
		return err
	}

	clubIDs := []string{}
	for rows.Next() {
		var clubID string
		if err := rows.Scan(&clubID); err != nil {
			rows.Close()
			return err
		}
		clubIDs = append(clubIDs, clubID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, clubID := range clubIDs {
		var newOwnerID int64
		err := tx.QueryRow(
			ctx,
			`SELECT user_id FROM club_members
			WHERE club_id = $1 AND user_id != $2
			ORDER BY CASE WHEN role = 'admin' THEN 0 ELSE 1 END, joined_at ASC
			LIMIT 1`,
			clubID,
			userID,
		).Scan(&newOwnerID)

		if errors.Is(err, pgx.ErrNoRows) {
			_, err = tx.Exec(ctx, `DELETE FROM clubs WHERE id = $1`, clubID)
			if err != nil {
				slog.ErrorContext(ctx, "delete empty club failed", "club_id", clubID, "error", err)
				return err
			}
			continue
		}
		if err != nil {
			slog.ErrorContext(ctx, "find new club owner failed", "club_id", clubID, "error", err)
			return err
		}

		_, err = tx.Exec(ctx, `DELETE FROM club_members WHERE club_id = $1 AND user_id = $2`, clubID, userID)
		if err != nil {
			slog.ErrorContext(ctx, "remove old club owner failed", "club_id", clubID, "user_id", userID, "error", err)
			return err
		}

		_, err = tx.Exec(ctx, `UPDATE club_members SET role = 'owner' WHERE club_id = $1 AND user_id = $2`, clubID, newOwnerID)
		if err != nil {
			slog.ErrorContext(ctx, "promote new club owner failed", "club_id", clubID, "user_id", newOwnerID, "error", err)
			return err
		}

		_, err = tx.Exec(ctx, `UPDATE clubs SET owner_id = $2 WHERE id = $1`, clubID, newOwnerID)
		if err != nil {
			slog.ErrorContext(ctx, "update club owner failed", "club_id", clubID, "error", err)
			return err
		}
	}

	return nil
}
