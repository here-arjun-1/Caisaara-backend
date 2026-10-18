package repository

import "context"

func (r *ProfileRepository) GetAvatarURL(ctx context.Context, userID int64) (*string, error) {
	var avatarURL *string

	err := r.DB.QueryRow(
		ctx,
		`SELECT p.avatar_url FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.id = $1`,
		userID,
	).Scan(&avatarURL)

	if err != nil {
		return nil, err
	}

	return avatarURL, nil
}

func (r *ProfileRepository) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL *string) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO player_profiles (user_id, avatar_url)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET
			avatar_url = EXCLUDED.avatar_url,
			updated_at = now()`,
		userID,
		avatarURL,
	)
	return err
}
