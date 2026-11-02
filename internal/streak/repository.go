package streak

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

func (r *Repository) GetOrCreateStreak(ctx context.Context, userID int64, defaultTz string) (*UserStreak, error) {
	if defaultTz == "" {
		defaultTz = "Asia/Kolkata"
	}

	u := &UserStreak{}
	query := `
		SELECT user_id, current_streak, best_streak, COALESCE(TO_CHAR(last_activity_date, 'YYYY-MM-DD'), ''), streak_status, freezes_available, timezone, updated_at
		FROM user_streaks
		WHERE user_id = $1
	`
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&u.UserID,
		&u.CurrentStreak,
		&u.BestStreak,
		&u.LastActivityDate,
		&u.StreakStatus,
		&u.FreezesAvailable,
		&u.Timezone,
		&u.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		insertQuery := `
			INSERT INTO user_streaks (user_id, current_streak, best_streak, streak_status, freezes_available, timezone, updated_at)
			VALUES ($1, 0, 0, 'active', 2, $2, NOW())
			ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
			RETURNING user_id, current_streak, best_streak, COALESCE(TO_CHAR(last_activity_date, 'YYYY-MM-DD'), ''), streak_status, freezes_available, timezone, updated_at
		`
		err = r.db.QueryRow(ctx, insertQuery, userID, defaultTz).Scan(
			&u.UserID,
			&u.CurrentStreak,
			&u.BestStreak,
			&u.LastActivityDate,
			&u.StreakStatus,
			&u.FreezesAvailable,
			&u.Timezone,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	u.MaxStreak = u.BestStreak
	return u, nil
}

func (r *Repository) GetStreakByUserID(ctx context.Context, userID int64) (*UserStreak, error) {
	u := &UserStreak{}
	query := `
		SELECT user_id, current_streak, best_streak, COALESCE(TO_CHAR(last_activity_date, 'YYYY-MM-DD'), ''), streak_status, freezes_available, timezone, updated_at
		FROM user_streaks
		WHERE user_id = $1
	`
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&u.UserID,
		&u.CurrentStreak,
		&u.BestStreak,
		&u.LastActivityDate,
		&u.StreakStatus,
		&u.FreezesAvailable,
		&u.Timezone,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.MaxStreak = u.BestStreak
	return u, nil
}

func (r *Repository) UpdateStreak(ctx context.Context, s *UserStreak) error {
	if s.BestStreak < s.CurrentStreak {
		s.BestStreak = s.CurrentStreak
	}
	s.MaxStreak = s.BestStreak

	_, err := r.db.Exec(
		ctx,
		`UPDATE user_streaks
		 SET current_streak = $1, best_streak = $2, last_activity_date = NULLIF($3, '')::date, streak_status = $4, freezes_available = $5, timezone = $6, updated_at = NOW()
		 WHERE user_id = $7`,
		s.CurrentStreak,
		s.BestStreak,
		s.LastActivityDate,
		s.StreakStatus,
		s.FreezesAvailable,
		s.Timezone,
		s.UserID,
	)
	return err
}

func (r *Repository) RecordActivityDay(ctx context.Context, userID int64, date string, activityType string) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO user_activity_days (user_id, activity_date, activity_type, created_at)
		 VALUES ($1, $2::date, $3, NOW())
		 ON CONFLICT (user_id, activity_date)
		 DO UPDATE SET activity_type = EXCLUDED.activity_type, created_at = NOW()`,
		userID,
		date,
		activityType,
	)
	return err
}

func (r *Repository) GetActivityDaysInRange(ctx context.Context, userID int64, startDate, endDate string) (map[string]string, error) {
	activities := make(map[string]string)

	gameRows, err := r.db.Query(
		ctx,
		`SELECT DISTINCT TO_CHAR(ended_at, 'YYYY-MM-DD')
		 FROM games
		 WHERE (white_player_id = $1 OR black_player_id = $1)
		   AND status = 'finished'
		   AND ended_at >= $2::date AND ended_at < ($3::date + INTERVAL '1 day')`,
		userID,
		startDate,
		endDate,
	)
	if err == nil {
		defer gameRows.Close()
		for gameRows.Next() {
			var d string
			if scanErr := gameRows.Scan(&d); scanErr == nil {
				activities[d] = "game"
			}
		}
	}

	activityRows, err := r.db.Query(
		ctx,
		`SELECT TO_CHAR(activity_date, 'YYYY-MM-DD'), activity_type
		 FROM user_activity_days
		 WHERE user_id = $1 AND activity_date >= $2::date AND activity_date <= $3::date`,
		userID,
		startDate,
		endDate,
	)
	if err == nil {
		defer activityRows.Close()
		for activityRows.Next() {
			var d, at string
			if scanErr := activityRows.Scan(&d, &at); scanErr == nil {
				activities[d] = at
			}
		}
	}

	return activities, nil
}
