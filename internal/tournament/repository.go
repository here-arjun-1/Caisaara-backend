package tournament

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TournamentRepository interface {
	CreateTournament(ctx context.Context, tournament *Tournament) error
	FindByID(ctx context.Context, id int64) (*Tournament, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) TournamentRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateTournament(ctx context.Context, tournament *Tournament) error {
	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO tournaments (
			name,
			description,
			format,
			time_control,
			max_players,
			visibility,
			invite_code,
			status,
			total_rounds,
			current_round,
			created_by,
			start_at,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`,
		tournament.Name,
		tournament.Description,
		tournament.Format,
		tournament.TimeControl,
		tournament.MaxPlayers,
		tournament.Visibility,
		tournament.InviteCode,
		tournament.Status,
		tournament.TotalRounds,
		tournament.CurrentRound,
		tournament.CreatedBy,
		tournament.StartAt,
		tournament.CreatedAt,
		tournament.UpdatedAt,
	).Scan(&tournament.ID, &tournament.CreatedAt, &tournament.UpdatedAt)

	if err != nil {
		slog.ErrorContext(ctx, "create tournament repository failed", "error", err)
		return err
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Tournament, error) {
	row := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			name,
			COALESCE(description, ''),
			format,
			time_control,
			max_players,
			visibility,
			invite_code,
			status,
			total_rounds,
			current_round,
			created_by,
			start_at,
			created_at,
			updated_at
		FROM tournaments
		WHERE id = $1`,
		id,
	)

	var t Tournament
	err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.Format,
		&t.TimeControl,
		&t.MaxPlayers,
		&t.Visibility,
		&t.InviteCode,
		&t.Status,
		&t.TotalRounds,
		&t.CurrentRound,
		&t.CreatedBy,
		&t.StartAt,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTournamentNotFound
		}
		slog.ErrorContext(ctx, "find tournament by id failed", "id", id, "error", err)
		return nil, err
	}

	return &t, nil
}
