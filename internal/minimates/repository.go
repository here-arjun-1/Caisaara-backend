package minimates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLevelNotFound = errors.New("level not found")

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) GetMiniMatesLevel(ctx context.Context, levelNumber int) (*LevelWithPuzzle, error) {
	var lp LevelWithPuzzle
	err := r.DB.QueryRow(ctx,
		`SELECT 
			l.level_number,
			l.puzzle_id,
			l.chapter,
			l.target_moves,
			l.title,
			p.fen,
			p.moves,
			p.rating
		FROM mini_mates_levels l
		JOIN puzzles p ON l.puzzle_id = p.id
		WHERE l.level_number = $1`,
		levelNumber,
	).Scan(
		&lp.LevelNumber,
		&lp.PuzzleID,
		&lp.Chapter,
		&lp.TargetMoves,
		&lp.Title,
		&lp.FEN,
		&lp.Moves,
		&lp.Rating,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLevelNotFound
		}
		return nil, fmt.Errorf("get mini mates level: %w", err)
	}
	return &lp, nil
}

func (r *Repository) GetLevel(ctx context.Context, levelNumber int) (*LevelWithPuzzle, error) {
	return r.GetMiniMatesLevel(ctx, levelNumber)
}

func (r *Repository) GetUserMiniMatesProgress(ctx context.Context, userID int64) (*UserProgress, error) {
	var p UserProgress
	err := r.DB.QueryRow(ctx,
		`SELECT user_id, current_level, total_stars, completed_levels, created_at, updated_at
		FROM user_mini_mates_progress
		WHERE user_id = $1`,
		userID,
	).Scan(
		&p.UserID,
		&p.CurrentLevel,
		&p.TotalStars,
		&p.CompletedLevels,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			now := time.Now().UTC()
			return &UserProgress{
				UserID:          userID,
				CurrentLevel:    1,
				TotalStars:      0,
				CompletedLevels: 0,
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		}
		return nil, fmt.Errorf("get user mini mates progress: %w", err)
	}
	return &p, nil
}

func (r *Repository) GetUserProgress(ctx context.Context, userID int64) (*UserProgress, error) {
	return r.GetUserMiniMatesProgress(ctx, userID)
}

func (r *Repository) SaveLevelCompletion(ctx context.Context, userID int64, levelNumber int, stars int, timeTakenMs int) (*UserProgress, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var prevStars int
	var wasCompleted bool
	historyErr := tx.QueryRow(ctx,
		`SELECT stars, completed FROM user_mini_mates_level_history WHERE user_id = $1 AND level_number = $2`,
		userID, levelNumber,
	).Scan(&prevStars, &wasCompleted)

	isFirstTime := false
	if errors.Is(historyErr, pgx.ErrNoRows) {
		isFirstTime = true
		prevStars = 0
		wasCompleted = false
	} else if historyErr != nil {
		return nil, fmt.Errorf("query level history: %w", historyErr)
	}

	bestStars := stars
	if prevStars > bestStars {
		bestStars = prevStars
	}

	starsDelta := 0
	if stars > prevStars {
		starsDelta = stars - prevStars
	}

	completedDelta := 0
	if !wasCompleted {
		completedDelta = 1
	}

	if isFirstTime {
		_, err = tx.Exec(ctx,
			`INSERT INTO user_mini_mates_level_history 
				(user_id, level_number, stars, completed, time_taken_ms, attempts, completed_at, created_at, updated_at)
			VALUES ($1, $2, $3, true, $4, 1, NOW(), NOW(), NOW())`,
			userID, levelNumber, bestStars, timeTakenMs,
		)
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE user_mini_mates_level_history
			SET stars = $1,
			    completed = true,
			    time_taken_ms = CASE WHEN time_taken_ms = 0 OR $2 < time_taken_ms THEN $2 ELSE time_taken_ms END,
			    attempts = attempts + 1,
			    completed_at = COALESCE(completed_at, NOW()),
			    updated_at = NOW()
			WHERE user_id = $3 AND level_number = $4`,
			bestStars, timeTakenMs, userID, levelNumber,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("save level history: %w", err)
	}

	var maxLevel int
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(level_number), 150) FROM mini_mates_levels`).Scan(&maxLevel)
	if err != nil {
		maxLevel = 150
	}

	nextLevel := levelNumber + 1
	if nextLevel > maxLevel {
		nextLevel = maxLevel
	}

	var progress UserProgress
	err = tx.QueryRow(ctx,
		`INSERT INTO user_mini_mates_progress (user_id, current_level, total_stars, completed_levels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			current_level = GREATEST(user_mini_mates_progress.current_level, EXCLUDED.current_level),
			total_stars = user_mini_mates_progress.total_stars + $5,
			completed_levels = user_mini_mates_progress.completed_levels + $6,
			updated_at = NOW()
		RETURNING user_id, current_level, total_stars, completed_levels, created_at, updated_at`,
		userID, nextLevel, bestStars, completedDelta, starsDelta, completedDelta,
	).Scan(
		&progress.UserID,
		&progress.CurrentLevel,
		&progress.TotalStars,
		&progress.CompletedLevels,
		&progress.CreatedAt,
		&progress.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update user progress: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &progress, nil
}

func (r *Repository) GetUserLevelHistory(ctx context.Context, userID int64, levelNumber int) (*LevelHistory, error) {
	var h LevelHistory
	err := r.DB.QueryRow(ctx,
		`SELECT id, user_id, level_number, stars, completed, hint_used, time_taken_ms, attempts, completed_at, created_at, updated_at
		FROM user_mini_mates_level_history
		WHERE user_id = $1 AND level_number = $2`,
		userID, levelNumber,
	).Scan(
		&h.ID,
		&h.UserID,
		&h.LevelNumber,
		&h.Stars,
		&h.Completed,
		&h.HintUsed,
		&h.TimeTakenMs,
		&h.Attempts,
		&h.CompletedAt,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user level history: %w", err)
	}
	return &h, nil
}

func (r *Repository) MarkHintUsed(ctx context.Context, userID int64, levelNumber int) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO user_mini_mates_level_history (user_id, level_number, stars, completed, hint_used, attempts, created_at, updated_at)
		VALUES ($1, $2, 0, false, true, 1, NOW(), NOW())
		ON CONFLICT (user_id, level_number) DO UPDATE SET
			hint_used = true,
			updated_at = NOW()`,
		userID, levelNumber,
	)
	if err != nil {
		return fmt.Errorf("mark hint used: %w", err)
	}
	return nil
}

func (r *Repository) RecordFailedAttempt(ctx context.Context, userID int64, levelNumber int) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO user_mini_mates_level_history (user_id, level_number, stars, completed, attempts, created_at, updated_at)
		VALUES ($1, $2, 0, false, 1, NOW(), NOW())
		ON CONFLICT (user_id, level_number) DO UPDATE SET
			attempts = user_mini_mates_level_history.attempts + 1,
			updated_at = NOW()`,
		userID, levelNumber,
	)
	if err != nil {
		return fmt.Errorf("record failed attempt: %w", err)
	}
	return nil
}

func (r *Repository) GetChapterOverview(ctx context.Context, userID int64) (*MiniMatesOverviewResponse, error) {
	progress, err := r.GetUserMiniMatesProgress(ctx, userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.Query(ctx,
		`SELECT 
			l.level_number,
			l.chapter,
			l.target_moves,
			l.title,
			COALESCE(h.stars, 0) AS stars,
			COALESCE(h.completed, false) AS completed
		FROM mini_mates_levels l
		LEFT JOIN user_mini_mates_level_history h 
			ON h.user_id = $1 AND h.level_number = l.level_number
		ORDER BY l.level_number ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query levels for chapter overview: %w", err)
	}
	defer rows.Close()

	chaptersMap := map[int]*ChapterInfo{
		1: {
			ID:          1,
			Name:        "Chapter 1: Novice Mates",
			Description: "Master fundamental 1-move mating patterns",
			TargetMoves: 1,
			Levels:      make([]LevelSummary, 0),
		},
		2: {
			ID:          2,
			Name:        "Chapter 2: Tactical Traps",
			Description: "Forced checks, discoveries, and deflections",
			TargetMoves: 2,
			Levels:      make([]LevelSummary, 0),
		},
		3: {
			ID:          3,
			Name:        "Chapter 3: Master Sacrifices",
			Description: "Queen and rook sacrifices to lure the king",
			TargetMoves: 3,
			Levels:      make([]LevelSummary, 0),
		},
		4: {
			ID:          4,
			Name:        "Chapter 4: Grandmaster Net",
			Description: "Deep calculations and tight mating nets",
			TargetMoves: 4,
			Levels:      make([]LevelSummary, 0),
		},
	}

	for rows.Next() {
		var s LevelSummary
		if err := rows.Scan(
			&s.LevelNumber,
			&s.Chapter,
			&s.TargetMoves,
			&s.Title,
			&s.Stars,
			&s.Completed,
		); err != nil {
			return nil, fmt.Errorf("scan level summary: %w", err)
		}

		s.Unlocked = s.LevelNumber <= progress.CurrentLevel

		ch, exists := chaptersMap[s.Chapter]
		if !exists {
			ch = &ChapterInfo{
				ID:          s.Chapter,
				Name:        fmt.Sprintf("Chapter %d", s.Chapter),
				TargetMoves: s.TargetMoves,
				Levels:      make([]LevelSummary, 0),
			}
			chaptersMap[s.Chapter] = ch
		}

		if ch.StartLevel == 0 || s.LevelNumber < ch.StartLevel {
			ch.StartLevel = s.LevelNumber
		}
		if s.LevelNumber > ch.EndLevel {
			ch.EndLevel = s.LevelNumber
		}

		ch.Levels = append(ch.Levels, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	orderedChapters := make([]ChapterInfo, 0, len(chaptersMap))
	for i := 1; i <= len(chaptersMap); i++ {
		if ch, ok := chaptersMap[i]; ok {
			orderedChapters = append(orderedChapters, *ch)
		}
	}

	return &MiniMatesOverviewResponse{
		CurrentLevel:    progress.CurrentLevel,
		TotalStars:      progress.TotalStars,
		CompletedLevels: progress.CompletedLevels,
		Chapters:        orderedChapters,
	}, nil
}
