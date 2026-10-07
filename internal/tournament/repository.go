package tournament

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TournamentWithPlayerCount struct {
	Tournament
	Players int
}

type TournamentRepository interface {
	CreateTournament(ctx context.Context, tournament *Tournament) error
	FindByID(ctx context.Context, id int64) (*Tournament, error)
	ListPublicTournaments(ctx context.Context) ([]*TournamentWithPlayerCount, error)
	FindByIDWithPlayerCount(ctx context.Context, id int64) (*TournamentWithPlayerCount, error)
	FindByInviteCode(ctx context.Context, code string) (*TournamentWithPlayerCount, error)
	IsPlayerJoined(ctx context.Context, tournamentID int64, userID int64) (bool, error)
	AddPlayer(ctx context.Context, player *TournamentPlayer) error
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

func (r *Repository) ListPublicTournaments(ctx context.Context) ([]*TournamentWithPlayerCount, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			t.id,
			t.name,
			COALESCE(t.description, ''),
			t.format,
			t.time_control,
			t.max_players,
			t.visibility,
			t.invite_code,
			t.status,
			t.total_rounds,
			t.current_round,
			t.created_by,
			t.start_at,
			t.created_at,
			t.updated_at,
			COUNT(tp.id)::int AS players
		FROM tournaments t
		LEFT JOIN tournament_players tp ON t.id = tp.tournament_id
		WHERE t.visibility = 'public'
		GROUP BY t.id
		ORDER BY t.created_at DESC`,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list public tournaments repository failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	list := make([]*TournamentWithPlayerCount, 0)
	for rows.Next() {
		var tw TournamentWithPlayerCount
		err := rows.Scan(
			&tw.ID,
			&tw.Name,
			&tw.Description,
			&tw.Format,
			&tw.TimeControl,
			&tw.MaxPlayers,
			&tw.Visibility,
			&tw.InviteCode,
			&tw.Status,
			&tw.TotalRounds,
			&tw.CurrentRound,
			&tw.CreatedBy,
			&tw.StartAt,
			&tw.CreatedAt,
			&tw.UpdatedAt,
			&tw.Players,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan tournament with player count failed", "error", err)
			return nil, err
		}
		list = append(list, &tw)
	}

	return list, nil
}

func (r *Repository) FindByIDWithPlayerCount(ctx context.Context, id int64) (*TournamentWithPlayerCount, error) {
	row := r.DB.QueryRow(
		ctx,
		`SELECT
			t.id,
			t.name,
			COALESCE(t.description, ''),
			t.format,
			t.time_control,
			t.max_players,
			t.visibility,
			t.invite_code,
			t.status,
			t.total_rounds,
			t.current_round,
			t.created_by,
			t.start_at,
			t.created_at,
			t.updated_at,
			COUNT(tp.id)::int AS players
		FROM tournaments t
		LEFT JOIN tournament_players tp ON t.id = tp.tournament_id
		WHERE t.id = $1
		GROUP BY t.id`,
		id,
	)

	var tw TournamentWithPlayerCount
	err := row.Scan(
		&tw.ID,
		&tw.Name,
		&tw.Description,
		&tw.Format,
		&tw.TimeControl,
		&tw.MaxPlayers,
		&tw.Visibility,
		&tw.InviteCode,
		&tw.Status,
		&tw.TotalRounds,
		&tw.CurrentRound,
		&tw.CreatedBy,
		&tw.StartAt,
		&tw.CreatedAt,
		&tw.UpdatedAt,
		&tw.Players,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTournamentNotFound
		}
		slog.ErrorContext(ctx, "find tournament by id with player count failed", "id", id, "error", err)
		return nil, err
	}

	return &tw, nil
}

func (r *Repository) FindByInviteCode(ctx context.Context, code string) (*TournamentWithPlayerCount, error) {
	row := r.DB.QueryRow(
		ctx,
		`SELECT
			t.id,
			t.name,
			COALESCE(t.description, ''),
			t.format,
			t.time_control,
			t.max_players,
			t.visibility,
			t.invite_code,
			t.status,
			t.total_rounds,
			t.current_round,
			t.created_by,
			t.start_at,
			t.created_at,
			t.updated_at,
			COUNT(tp.id)::int AS players
		FROM tournaments t
		LEFT JOIN tournament_players tp ON t.id = tp.tournament_id
		WHERE LOWER(t.invite_code) = LOWER($1)
		GROUP BY t.id`,
		code,
	)

	var tw TournamentWithPlayerCount
	err := row.Scan(
		&tw.ID,
		&tw.Name,
		&tw.Description,
		&tw.Format,
		&tw.TimeControl,
		&tw.MaxPlayers,
		&tw.Visibility,
		&tw.InviteCode,
		&tw.Status,
		&tw.TotalRounds,
		&tw.CurrentRound,
		&tw.CreatedBy,
		&tw.StartAt,
		&tw.CreatedAt,
		&tw.UpdatedAt,
		&tw.Players,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTournamentNotFound
		}
		slog.ErrorContext(ctx, "find tournament by invite code failed", "code", code, "error", err)
		return nil, err
	}

	return &tw, nil
}

func (r *Repository) IsPlayerJoined(ctx context.Context, tournamentID int64, userID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM tournament_players
			WHERE tournament_id = $1 AND user_id = $2
		)`,
		tournamentID,
		userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "check is player joined failed", "tournament_id", tournamentID, "user_id", userID, "error", err)
		return false, err
	}
	return exists, nil
}

func (r *Repository) AddPlayer(ctx context.Context, player *TournamentPlayer) error {
	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO tournament_players (
			tournament_id,
			user_id,
			score,
			wins,
			draws,
			losses,
			games_played,
			joined_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, joined_at`,
		player.TournamentID,
		player.UserID,
		player.Score,
		player.Wins,
		player.Draws,
		player.Losses,
		player.GamesPlayed,
		player.JoinedAt,
	).Scan(&player.ID, &player.JoinedAt)

	if err != nil {
		slog.ErrorContext(ctx, "add tournament player failed", "tournament_id", player.TournamentID, "user_id", player.UserID, "error", err)
		return err
	}

	return nil
}
