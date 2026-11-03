package report

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`,
		userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "check user exists failed", "user_id", userID, "error", err)
		return false, err
	}

	return exists, nil
}

func (r *Repository) CreateReport(ctx context.Context, reporterID int64, reportedID int64, reason string, details string) (int64, error) {
	var reportID int64
	err := r.DB.QueryRow(ctx,
		`INSERT INTO player_reports (reporter_id, reported_id, reason, details)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		reporterID, reportedID, reason, details,
	).Scan(&reportID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrReportExists
		}

		slog.ErrorContext(ctx, "create player report failed", "reporter_id", reporterID, "reported_id", reportedID, "error", err)
		return 0, err
	}
	return reportID, nil
}
