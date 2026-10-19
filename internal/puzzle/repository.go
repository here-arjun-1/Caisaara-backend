package puzzle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PuzzleRepository interface {
	FindMatchingPuzzle(ctx context.Context, userID int64, minRating, maxRating int, theme string) (*Puzzle, error)
	FindPuzzleByID(ctx context.Context, id string) (*Puzzle, error)
	GetOrCreateUserRating(ctx context.Context, userID int64) (*UserPuzzleRating, error)
	UpdateUserRating(ctx context.Context, rating *UserPuzzleRating) error
	RecordAttempt(ctx context.Context, attempt *UserPuzzleAttempt) error
	GetDailySession(ctx context.Context, userID int64, date string) (*DailyPuzzleSession, error)
	CreateDailySession(ctx context.Context, session *DailyPuzzleSession) error
	UpdateDailySession(ctx context.Context, session *DailyPuzzleSession) error
	GetUserStats(ctx context.Context, userID int64) (*PuzzleStatsResponse, error)
	GetUserHistory(ctx context.Context, userID int64, limit, offset int) (*PuzzleHistoryResponse, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) PuzzleRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queries := []string{
		`ALTER TABLE user_puzzle_ratings ADD COLUMN IF NOT EXISTS current_streak INT NOT NULL DEFAULT 0;`,
		`ALTER TABLE user_puzzle_ratings ADD COLUMN IF NOT EXISTS max_streak INT NOT NULL DEFAULT 0;`,
		`ALTER TABLE user_puzzle_ratings ADD COLUMN IF NOT EXISTS last_played_date DATE;`,
		`CREATE TABLE IF NOT EXISTS daily_puzzle_sessions (
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			puzzle_id VARCHAR(16) NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
			date DATE NOT NULL,
			attempts_left INT NOT NULL DEFAULT 3,
			hint_used BOOLEAN NOT NULL DEFAULT FALSE,
			solution_viewed BOOLEAN NOT NULL DEFAULT FALSE,
			solved BOOLEAN NOT NULL DEFAULT FALSE,
			rating_earned INT NOT NULL DEFAULT 0,
			attempted_moves TEXT[] NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, date)
		);`,
		`ALTER TABLE daily_puzzle_sessions ADD COLUMN IF NOT EXISTS attempted_moves TEXT[] NOT NULL DEFAULT '{}';`,
		`ALTER TABLE user_puzzle_attempts ADD COLUMN IF NOT EXISTS attempted_moves TEXT[] NOT NULL DEFAULT '{}';`,
	}

	for _, q := range queries {
		_, _ = db.Exec(ctx, q)
	}

	return &Repository{DB: db}
}

func (r *Repository) FindMatchingPuzzle(ctx context.Context, userID int64, minRating, maxRating int, theme string) (*Puzzle, error) {
	query := `
		SELECT id, fen, moves, rating, rating_deviation, popularity, nb_plays, themes, COALESCE(game_url, ''), COALESCE(opening_tags, ''), created_at
		FROM puzzles
		WHERE rating BETWEEN $1 AND $2
		  AND ($3 = '' OR $3 = ANY(themes))
		  AND id NOT IN (
		      SELECT puzzle_id FROM user_puzzle_attempts WHERE user_id = $4
		  )
		ORDER BY RANDOM()
		LIMIT 1
	`

	p := &Puzzle{}
	err := r.DB.QueryRow(ctx, query, minRating, maxRating, theme, userID).Scan(
		&p.ID,
		&p.FEN,
		&p.Moves,
		&p.Rating,
		&p.RatingDeviation,
		&p.Popularity,
		&p.NbPlays,
		&p.Themes,
		&p.GameURL,
		&p.OpeningTags,
		&p.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		fallbackQuery := `
			SELECT id, fen, moves, rating, rating_deviation, popularity, nb_plays, themes, COALESCE(game_url, ''), COALESCE(opening_tags, ''), created_at
			FROM puzzles
			WHERE rating BETWEEN $1 AND $2
			ORDER BY RANDOM()
			LIMIT 1
		`
		err = r.DB.QueryRow(ctx, fallbackQuery, minRating-150, maxRating+150).Scan(
			&p.ID,
			&p.FEN,
			&p.Moves,
			&p.Rating,
			&p.RatingDeviation,
			&p.Popularity,
			&p.NbPlays,
			&p.Themes,
			&p.GameURL,
			&p.OpeningTags,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return p, nil
}

func (r *Repository) FindPuzzleByID(ctx context.Context, id string) (*Puzzle, error) {
	query := `
		SELECT id, fen, moves, rating, rating_deviation, popularity, nb_plays, themes, COALESCE(game_url, ''), COALESCE(opening_tags, ''), created_at
		FROM puzzles
		WHERE id = $1
	`

	p := &Puzzle{}
	err := r.DB.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.FEN,
		&p.Moves,
		&p.Rating,
		&p.RatingDeviation,
		&p.Popularity,
		&p.NbPlays,
		&p.Themes,
		&p.GameURL,
		&p.OpeningTags,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (r *Repository) GetOrCreateUserRating(ctx context.Context, userID int64) (*UserPuzzleRating, error) {
	query := `
		SELECT user_id, rating, rating_deviation, rating_volatility, puzzles_solved, puzzles_failed, best_rating, best_rating_at, current_streak, max_streak, last_played_date, updated_at
		FROM user_puzzle_ratings
		WHERE user_id = $1
	`

	u := &UserPuzzleRating{}
	var lastPlayedDate *time.Time

	err := r.DB.QueryRow(ctx, query, userID).Scan(
		&u.UserID,
		&u.Rating,
		&u.RatingDeviation,
		&u.RatingVolatility,
		&u.PuzzlesSolved,
		&u.PuzzlesFailed,
		&u.BestRating,
		&u.BestRatingAt,
		&u.CurrentStreak,
		&u.MaxStreak,
		&lastPlayedDate,
		&u.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		insertQuery := `
			INSERT INTO user_puzzle_ratings (
				user_id, rating, rating_deviation, rating_volatility, puzzles_solved, puzzles_failed, best_rating, best_rating_at, current_streak, max_streak, updated_at
			)
			VALUES ($1, 1500, 350, 0.06, 0, 0, 1500, NOW(), 0, 0, NOW())
			ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
			RETURNING user_id, rating, rating_deviation, rating_volatility, puzzles_solved, puzzles_failed, best_rating, best_rating_at, current_streak, max_streak, last_played_date, updated_at
		`
		err = r.DB.QueryRow(ctx, insertQuery, userID).Scan(
			&u.UserID,
			&u.Rating,
			&u.RatingDeviation,
			&u.RatingVolatility,
			&u.PuzzlesSolved,
			&u.PuzzlesFailed,
			&u.BestRating,
			&u.BestRatingAt,
			&u.CurrentStreak,
			&u.MaxStreak,
			&lastPlayedDate,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if lastPlayedDate != nil {
		u.LastPlayedDate = lastPlayedDate.Format("2006-01-02")
	}

	return u, nil
}

func (r *Repository) UpdateUserRating(ctx context.Context, rating *UserPuzzleRating) error {
	var lpd *time.Time
	if rating.LastPlayedDate != "" {
		parsed, err := time.Parse("2006-01-02", rating.LastPlayedDate)
		if err == nil {
			lpd = &parsed
		}
	}

	query := `
		UPDATE user_puzzle_ratings
		SET rating = $1,
			rating_deviation = $2,
			rating_volatility = $3,
			puzzles_solved = $4,
			puzzles_failed = $5,
			best_rating = $6,
			best_rating_at = $7,
			current_streak = $8,
			max_streak = $9,
			last_played_date = $10,
			updated_at = NOW()
		WHERE user_id = $11
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		rating.Rating,
		rating.RatingDeviation,
		rating.RatingVolatility,
		rating.PuzzlesSolved,
		rating.PuzzlesFailed,
		rating.BestRating,
		rating.BestRatingAt,
		rating.CurrentStreak,
		rating.MaxStreak,
		lpd,
		rating.UserID,
	)

	return err
}

func (r *Repository) RecordAttempt(ctx context.Context, attempt *UserPuzzleAttempt) error {
	query := `
		INSERT INTO user_puzzle_attempts (
			user_id, puzzle_id, solved, user_rating_before, user_rating_after, time_taken_ms, attempted_moves, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`

	if attempt.AttemptedMoves == nil {
		attempt.AttemptedMoves = []string{}
	}

	return r.DB.QueryRow(
		ctx,
		query,
		attempt.UserID,
		attempt.PuzzleID,
		attempt.Solved,
		attempt.UserRatingBefore,
		attempt.UserRatingAfter,
		attempt.TimeTakenMs,
		attempt.AttemptedMoves,
	).Scan(&attempt.ID, &attempt.CreatedAt)
}

func (r *Repository) GetDailySession(ctx context.Context, userID int64, date string) (*DailyPuzzleSession, error) {
	query := `
		SELECT user_id, puzzle_id, TO_CHAR(date, 'YYYY-MM-DD'), attempts_left, hint_used, solution_viewed, solved, rating_earned, COALESCE(attempted_moves, '{}')
		FROM daily_puzzle_sessions
		WHERE user_id = $1 AND date = $2::date
	`

	session := &DailyPuzzleSession{}
	err := r.DB.QueryRow(ctx, query, userID, date).Scan(
		&session.UserID,
		&session.PuzzleID,
		&session.Date,
		&session.AttemptsLeft,
		&session.HintUsed,
		&session.SolutionViewed,
		&session.Solved,
		&session.RatingEarned,
		&session.AttemptedMoves,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if session.AttemptedMoves == nil {
		session.AttemptedMoves = []string{}
	}

	return session, nil
}

func (r *Repository) CreateDailySession(ctx context.Context, session *DailyPuzzleSession) error {
	query := `
		INSERT INTO daily_puzzle_sessions (
			user_id, puzzle_id, date, attempts_left, hint_used, solution_viewed, solved, rating_earned, attempted_moves, created_at, updated_at
		)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (user_id, date) DO NOTHING
	`

	if session.AttemptedMoves == nil {
		session.AttemptedMoves = []string{}
	}

	_, err := r.DB.Exec(
		ctx,
		query,
		session.UserID,
		session.PuzzleID,
		session.Date,
		session.AttemptsLeft,
		session.HintUsed,
		session.SolutionViewed,
		session.Solved,
		session.RatingEarned,
		session.AttemptedMoves,
	)

	return err
}

func (r *Repository) UpdateDailySession(ctx context.Context, session *DailyPuzzleSession) error {
	query := `
		UPDATE daily_puzzle_sessions
		SET attempts_left = $1,
			hint_used = $2,
			solution_viewed = $3,
			solved = $4,
			rating_earned = $5,
			attempted_moves = $6,
			updated_at = NOW()
		WHERE user_id = $7 AND date = $8::date
	`

	if session.AttemptedMoves == nil {
		session.AttemptedMoves = []string{}
	}

	_, err := r.DB.Exec(
		ctx,
		query,
		session.AttemptsLeft,
		session.HintUsed,
		session.SolutionViewed,
		session.Solved,
		session.RatingEarned,
		session.AttemptedMoves,
		session.UserID,
		session.Date,
	)

	return err
}

func (r *Repository) GetUserStats(ctx context.Context, userID int64) (*PuzzleStatsResponse, error) {
	rating, err := r.GetOrCreateUserRating(ctx, userID)
	if err != nil {
		return nil, err
	}

	total := rating.PuzzlesSolved + rating.PuzzlesFailed
	winRate := 0.0
	if total > 0 {
		winRate = (float64(rating.PuzzlesSolved) / float64(total)) * 100.0
	}

	stats := &PuzzleStatsResponse{
		Rating:        rating.Rating,
		BestRating:    rating.BestRating,
		BestRatingAt:  &rating.BestRatingAt,
		PuzzlesSolved: rating.PuzzlesSolved,
		PuzzlesFailed: rating.PuzzlesFailed,
		WinRate:       winRate,
		CurrentStreak: rating.CurrentStreak,
		MaxStreak:     rating.MaxStreak,
	}

	return stats, nil
}

func (r *Repository) GetUserHistory(ctx context.Context, userID int64, limit, offset int) (*PuzzleHistoryResponse, error) {
	var total int64
	countQuery := `SELECT COUNT(*) FROM user_puzzle_attempts WHERE user_id = $1`
	if err := r.DB.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, err
	}

	query := `
		SELECT a.id, a.puzzle_id, a.solved, COALESCE(a.attempted_moves, '{}'), COALESCE(p.moves, ''),
		       a.user_rating_before, a.user_rating_after, (a.user_rating_after - a.user_rating_before) AS rating_change,
		       a.time_taken_ms, a.created_at
		FROM user_puzzle_attempts a
		LEFT JOIN puzzles p ON a.puzzle_id = p.id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PuzzleHistoryItem, 0)
	for rows.Next() {
		var item PuzzleHistoryItem
		var rawMoves string
		if err := rows.Scan(
			&item.ID,
			&item.PuzzleID,
			&item.Solved,
			&item.AttemptedMoves,
			&rawMoves,
			&item.UserRatingBefore,
			&item.UserRatingAfter,
			&item.RatingChange,
			&item.TimeTakenMs,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		if item.AttemptedMoves == nil {
			item.AttemptedMoves = []string{}
		}

		parts := strings.Fields(rawMoves)
		if len(parts) > 1 {
			item.SolutionMoves = parts[1:]
		} else if len(parts) == 1 {
			item.SolutionMoves = parts
		} else {
			item.SolutionMoves = []string{}
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &PuzzleHistoryResponse{
		Total:    total,
		Attempts: items,
	}, nil
}
