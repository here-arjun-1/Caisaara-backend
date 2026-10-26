package privacy

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSettings(ctx context.Context, userID int64) (*Settings, error) {
	s := DefaultSettings()
	err := r.db.QueryRow(ctx, `
		SELECT direct_message, club, tournament, invite_friend
		FROM user_privacy_settings
		WHERE user_id = $1
	`, userID).Scan(&s.DirectMessage, &s.Club, &s.Tournament, &s.InviteFriend)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpsertSettings(ctx context.Context, userID int64, s Settings) (*Settings, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO user_privacy_settings (user_id, direct_message, club, tournament, invite_friend, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (user_id) DO UPDATE SET
			direct_message = EXCLUDED.direct_message,
			club = EXCLUDED.club,
			tournament = EXCLUDED.tournament,
			invite_friend = EXCLUDED.invite_friend,
			updated_at = now()
		RETURNING direct_message, club, tournament, invite_friend
	`, userID, s.DirectMessage, s.Club, s.Tournament, s.InviteFriend).Scan(
		&s.DirectMessage, &s.Club, &s.Tournament, &s.InviteFriend,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) AreFriends(ctx context.Context, user1ID, user2ID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM friend_requests
			WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))
			  AND status = 'accepted'
		)
	`, user1ID, user2ID).Scan(&exists)
	return exists, err
}
