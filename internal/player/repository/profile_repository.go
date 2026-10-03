package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/player/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileRepository struct {
	DB *pgxpool.Pool
}

func NewProfileRepository(db *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{
		DB: db,
	}
}

func (r *ProfileRepository) FindByUserID(userID int64) (*model.Profile, error) {
	var p model.Profile

	err := r.DB.QueryRow(
		context.Background(),
		`SELECT
			u.id,
			u.username,
			u.email,
			u.rating,
			u.skill_level,
			u.created_at,
			p.display_name,
			p.country,
			p.bio,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.id = $1`,
		userID,
	).Scan(
		&p.UserID,
		&p.Username,
		&p.Email,
		&p.Rating,
		&p.SkillLevel,
		&p.CreatedAt,
		&p.DisplayName,
		&p.Country,
		&p.Bio,
		&p.AvatarURL,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *ProfileRepository) FindByUsername(username string) (*model.Profile, error) {
	var p model.Profile

	err := r.DB.QueryRow(
		context.Background(),
		`SELECT
			u.id,
			u.username,
			u.email,
			u.rating,
			u.skill_level,
			u.created_at,
			p.display_name,
			p.country,
			p.bio,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.username = $1`,
		username,
	).Scan(
		&p.UserID,
		&p.Username,
		&p.Email,
		&p.Rating,
		&p.SkillLevel,
		&p.CreatedAt,
		&p.DisplayName,
		&p.Country,
		&p.Bio,
		&p.AvatarURL,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *ProfileRepository) SaveProfile(p *model.Profile) error {
	_, err := r.DB.Exec(
		context.Background(),
		`INSERT INTO player_profiles (user_id, display_name, country, bio, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			country      = EXCLUDED.country,
			bio          = EXCLUDED.bio,
			avatar_url   = EXCLUDED.avatar_url,
			updated_at   = now()`,
		p.UserID,
		p.DisplayName,
		p.Country,
		p.Bio,
		p.AvatarURL,
	)
	return err
}
