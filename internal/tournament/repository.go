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
	RemovePlayer(ctx context.Context, tournamentID int64, userID int64) error
	StartTournament(ctx context.Context, tournamentID int64) error
	GetSwissPlayers(ctx context.Context, tournamentID int64) ([]*SwissPlayer, error)
	SaveRoundPairings(ctx context.Context, tournamentID int64, roundNumber int, pairings []SwissPairingResult) ([]*TournamentPairing, error)
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

func (r *Repository) RemovePlayer(ctx context.Context, tournamentID int64, userID int64) error {
	tag, err := r.DB.Exec(
		ctx,
		`DELETE FROM tournament_players
		WHERE tournament_id = $1 AND user_id = $2`,
		tournamentID,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "remove tournament player failed", "tournament_id", tournamentID, "user_id", userID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotJoined
	}
	return nil
}

func (r *Repository) StartTournament(ctx context.Context, tournamentID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin transaction failed for start tournament", "tournament_id", tournamentID, "error", err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(
		ctx,
		`UPDATE tournaments
		SET status = 'ongoing', current_round = 1, updated_at = NOW()
		WHERE id = $1 AND status = 'registration'`,
		tournamentID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update tournament status to ongoing failed", "tournament_id", tournamentID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotRegistration
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO tournament_rounds (tournament_id, round_number, status, started_at)
		VALUES ($1, 1, 'ongoing', NOW())
		ON CONFLICT (tournament_id, round_number)
		DO UPDATE SET status = 'ongoing', started_at = NOW()`,
		tournamentID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "insert round 1 failed", "tournament_id", tournamentID, "error", err)
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetSwissPlayers(ctx context.Context, tournamentID int64) ([]*SwissPlayer, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT tp.user_id, COALESCE(u.rating, 1200) AS rating, tp.score, tp.wins, tp.draws, tp.losses, tp.games_played
		FROM tournament_players tp
		LEFT JOIN users u ON tp.user_id = u.id
		WHERE tp.tournament_id = $1`,
		tournamentID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get swiss players query failed", "tournament_id", tournamentID, "error", err)
		return nil, err
	}
	defer rows.Close()

	playerMap := make(map[int64]*SwissPlayer)
	for rows.Next() {
		var p SwissPlayer
		p.PreviousOpponents = make(map[int64]bool)
		if err := rows.Scan(&p.UserID, &p.Rating, &p.Score, &p.Wins, &p.Draws, &p.Losses, &p.GamesPlayed); err != nil {
			return nil, err
		}
		playerMap[p.UserID] = &p
	}
	rows.Close()

	pairingRows, err := r.DB.Query(
		ctx,
		`SELECT white_player_id, black_player_id, is_bye, status
		FROM tournament_pairings
		WHERE tournament_id = $1
		ORDER BY created_at ASC`,
		tournamentID,
	)
	if err == nil {
		defer pairingRows.Close()
		for pairingRows.Next() {
			var whiteID, blackID *int64
			var isBye bool
			var status string
			if err := pairingRows.Scan(&whiteID, &blackID, &isBye, &status); err == nil {
				if isBye && whiteID != nil {
					if sp, ok := playerMap[*whiteID]; ok {
						sp.ReceivedBye = true
					}
				} else if whiteID != nil && blackID != nil {
					wSp, wOk := playerMap[*whiteID]
					bSp, bOk := playerMap[*blackID]
					if wOk {
						wSp.PreviousOpponents[*blackID] = true
						wSp.WhiteCount++
						if wSp.LastColor == "W" {
							wSp.ColorStreak++
						} else {
							wSp.LastColor = "W"
							wSp.ColorStreak = 1
						}
					}
					if bOk {
						bSp.PreviousOpponents[*whiteID] = true
						bSp.BlackCount++
						if bSp.LastColor == "B" {
							bSp.ColorStreak++
						} else {
							bSp.LastColor = "B"
							bSp.ColorStreak = 1
						}
					}
				}
			}
		}
	}

	for _, sp := range playerMap {
		for oppID := range sp.PreviousOpponents {
			if opp, ok := playerMap[oppID]; ok {
				sp.Buchholz += opp.Score
			}
		}
	}

	players := make([]*SwissPlayer, 0, len(playerMap))
	for _, p := range playerMap {
		players = append(players, p)
	}

	return players, nil
}

func (r *Repository) SaveRoundPairings(ctx context.Context, tournamentID int64, roundNumber int, pairings []SwissPairingResult) ([]*TournamentPairing, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var roundID int64
	err = tx.QueryRow(
		ctx,
		`INSERT INTO tournament_rounds (tournament_id, round_number, status, started_at)
		VALUES ($1, $2, 'ongoing', NOW())
		ON CONFLICT (tournament_id, round_number)
		DO UPDATE SET status = 'ongoing', started_at = NOW()
		RETURNING id`,
		tournamentID,
		roundNumber,
	).Scan(&roundID)
	if err != nil {
		slog.ErrorContext(ctx, "insert or update round failed", "tournament_id", tournamentID, "round", roundNumber, "error", err)
		return nil, err
	}

	resultPairings := make([]*TournamentPairing, 0, len(pairings))
	for _, p := range pairings {
		var tp TournamentPairing
		tp.TournamentID = tournamentID
		tp.RoundID = roundID
		tp.WhitePlayerID = p.WhitePlayerID
		tp.BlackPlayerID = p.BlackPlayerID
		tp.IsBye = p.IsBye
		tp.Status = "pending"

		if p.IsBye {
			resultStr := ResultBye
			tp.Result = &resultStr
			tp.Status = "completed"

			err = tx.QueryRow(
				ctx,
				`INSERT INTO tournament_pairings (
					tournament_id, round_id, white_player_id, black_player_id, result, status, is_bye, created_at, completed_at
				) VALUES ($1, $2, $3, NULL, $4, $5, TRUE, NOW(), NOW())
				RETURNING id, created_at, completed_at`,
				tournamentID, roundID, p.WhitePlayerID, resultStr, tp.Status,
			).Scan(&tp.ID, &tp.CreatedAt, &tp.CompletedAt)

			if err != nil {
				return nil, err
			}

			if p.WhitePlayerID != nil {
				_, err = tx.Exec(
					ctx,
					`UPDATE tournament_players
					SET score = score + 1.0, wins = wins + 1, games_played = games_played + 1
					WHERE tournament_id = $1 AND user_id = $2`,
					tournamentID, *p.WhitePlayerID,
				)
				if err != nil {
					return nil, err
				}
			}
		} else {
			err = tx.QueryRow(
				ctx,
				`INSERT INTO tournament_pairings (
					tournament_id, round_id, white_player_id, black_player_id, status, is_bye, created_at
				) VALUES ($1, $2, $3, $4, 'pending', FALSE, NOW())
				RETURNING id, created_at`,
				tournamentID, roundID, p.WhitePlayerID, p.BlackPlayerID,
			).Scan(&tp.ID, &tp.CreatedAt)

			if err != nil {
				return nil, err
			}
		}
		resultPairings = append(resultPairings, &tp)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return resultPairings, nil
}
