package game

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	CreateGame(ctx context.Context, whitePlayerID int64, blackPlayerID int64, timeControlMinutes int, incrementSeconds int, rated bool) (string, error)
	CreateGameWithID(ctx context.Context, gameID string, whitePlayerID int64, blackPlayerID int64, timeControlMinutes int, incrementSeconds int, rated bool) (string, error)
	FindGameByID(ctx context.Context, gameID string) (*Game, error)
	GetActiveGames(ctx context.Context) ([]Game, error)
	GetTimedOutGames(ctx context.Context, includeDaily bool, limit int) ([]Game, error)
	UpdateGameState(ctx context.Context, gameID string, position string, status string, result string, endReason string, currentTurn string, whiteTimeMs int64, blackTimeMs int64, turnStartedAt *time.Time) error
	AddMove(ctx context.Context, move *GameMove) error
	SaveMoveTx(ctx context.Context, gameID string, position string, status string, result string, endReason string, currentTurn string, whiteTimeMs int64, blackTimeMs int64, turnStartedAt *time.Time, move *GameMove, expectedTurn ...string) error
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
	GetPlayerGames(ctx context.Context, playerID int64) ([]Game, error)
	DeleteGameWithMoves(ctx context.Context, gameID string) error
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GameRepository {
	return &Repository{
		DB: db,
	}
}

func ResolveTimeControlMode(timeControlMinutes int, incrementSeconds int) (mode string, initialTimeMs int64, incrementMs int64, dailyMoveMs int64) {
	incrementMs = int64(incrementSeconds) * 1000

	if timeControlMinutes >= 1440 {
		dailyMs := int64(timeControlMinutes) * 60 * 1000
		return ModeDaily, dailyMs, 0, dailyMs
	}

	initialTimeMs = int64(timeControlMinutes) * 60 * 1000
	if initialTimeMs < 0 {
		initialTimeMs = 0
	}

	switch {
	case timeControlMinutes >= 1 && timeControlMinutes <= 2:
		mode = ModeBullet
	case timeControlMinutes >= 3 && timeControlMinutes <= 9:
		mode = ModeBlitz
	case timeControlMinutes >= 10 && timeControlMinutes <= 60:
		mode = ModeRapid
	default:
		if timeControlMinutes < 1 {
			mode = ModeBullet
		} else {
			mode = ModeRapid
		}
	}

	return mode, initialTimeMs, incrementMs, 0
}

func (r *Repository) CreateGame(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	incrementSeconds int,
	rated bool,
) (string, error) {
	return r.CreateGameWithID(ctx, uuid.New().String(), whitePlayerID, blackPlayerID, timeControlMinutes, incrementSeconds, rated)
}

func (r *Repository) CreateGameWithID(
	ctx context.Context,
	gameID string,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	incrementSeconds int,
	rated bool,
) (string, error) {
	mode, initialTimeMs, incrementMs, dailyMoveMs := ResolveTimeControlMode(timeControlMinutes, incrementSeconds)
	parsedID, err := uuid.Parse(gameID)
	if err != nil {
		parsedID = uuid.New()
	}
	return r.CreateGameWithSpecificIDAndDetails(ctx, parsedID, whitePlayerID, blackPlayerID, timeControlMinutes, rated, mode, initialTimeMs, incrementMs, dailyMoveMs)
}

func (r *Repository) CreateGameWithDetails(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	rated bool,
	mode string,
	initialTimeMs int64,
	incrementMs int64,
	dailyMoveMs int64,
) (string, error) {
	return r.CreateGameWithSpecificIDAndDetails(ctx, uuid.New(), whitePlayerID, blackPlayerID, timeControlMinutes, rated, mode, initialTimeMs, incrementMs, dailyMoveMs)
}

func (r *Repository) CreateGameWithSpecificIDAndDetails(
	ctx context.Context,
	gameID uuid.UUID,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	rated bool,
	mode string,
	initialTimeMs int64,
	incrementMs int64,
	dailyMoveMs int64,
) (string, error) {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO games (
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			rated,
			time_control_mode,
			daily_move_time_ms,
			position,
			status,
			started_at,
			initial_time_ms,
			increment_ms,
			white_time_ms,
			black_time_ms,
			current_turn,
			turn_started_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', NOW(), $9, $10, $11, $12, 'w', NOW())`,
		gameID,
		whitePlayerID,
		blackPlayerID,
		timeControlMinutes,
		rated,
		mode,
		dailyMoveMs,
		NewChessGame().FEN(),
		initialTimeMs,
		incrementMs,
		initialTimeMs,
		initialTimeMs,
	)

	if err != nil {
		slog.ErrorContext(ctx, "create game failed", "error", err)
		return "", err
	}

	return gameID.String(), nil
}

func (r *Repository) FindGameByID(
	ctx context.Context,
	gameID string,
) (*Game, error) {
	var g Game

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			g.id,
			g.white_player_id,
			g.black_player_id,
			COALESCE(uw.username, ''),
			COALESCE(ub.username, ''),
			g.time_control_minutes,
			g.rated,
			COALESCE(g.time_control_mode, 'rapid'),
			COALESCE(g.daily_move_time_ms, 86400000),
			g.position,
			g.status,
			COALESCE(g.result, ''),
			COALESCE(g.end_reason, ''),
			COALESCE(g.initial_time_ms, 600000),
			COALESCE(g.increment_ms, 0),
			COALESCE(g.white_time_ms, 600000),
			COALESCE(g.black_time_ms, 600000),
			COALESCE(g.current_turn, 'w'),
			g.turn_started_at,
			g.created_at,
			g.started_at,
			g.ended_at,
			g.white_rating_before,
			g.white_rating_after,
			g.black_rating_before,
			g.black_rating_after,
			g.white_provisional,
			g.black_provisional
		FROM games g
		LEFT JOIN users uw ON uw.id = g.white_player_id
		LEFT JOIN users ub ON ub.id = g.black_player_id
		WHERE g.id = $1`,
		gameID,
	).Scan(
		&g.ID,
		&g.WhitePlayerID,
		&g.BlackPlayerID,
		&g.WhiteUsername,
		&g.BlackUsername,
		&g.TimeControlMinutes,
		&g.Rated,
		&g.TimeControlMode,
		&g.DailyMoveTimeMs,
		&g.Position,
		&g.Status,
		&g.Result,
		&g.EndReason,
		&g.InitialTimeMs,
		&g.IncrementMs,
		&g.WhiteTimeMs,
		&g.BlackTimeMs,
		&g.CurrentTurn,
		&g.TurnStartedAt,
		&g.CreatedAt,
		&g.StartedAt,
		&g.EndedAt,
		&g.WhiteRatingBefore,
		&g.WhiteRatingAfter,
		&g.BlackRatingBefore,
		&g.BlackRatingAfter,
		&g.WhiteProvisional,
		&g.BlackProvisional,
	)

	if err != nil {
		slog.ErrorContext(ctx, "find game by id failed", "game_id", gameID, "error", err)
		return nil, err
	}

	return &g, nil
}

func (r *Repository) GetActiveGames(ctx context.Context) ([]Game, error) {
	return r.GetTimedOutGames(ctx, true, 100)
}

func (r *Repository) GetTimedOutGames(ctx context.Context, includeDaily bool, limit int) ([]Game, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT
			g.id,
			g.white_player_id,
			g.black_player_id,
			COALESCE(uw.username, ''),
			COALESCE(ub.username, ''),
			g.time_control_minutes,
			g.rated,
			COALESCE(g.time_control_mode, 'rapid'),
			COALESCE(g.daily_move_time_ms, 86400000),
			g.position,
			g.status,
			COALESCE(g.result, ''),
			COALESCE(g.end_reason, ''),
			COALESCE(g.initial_time_ms, 600000),
			COALESCE(g.increment_ms, 0),
			COALESCE(g.white_time_ms, 600000),
			COALESCE(g.black_time_ms, 600000),
			COALESCE(g.current_turn, 'w'),
			g.turn_started_at,
			g.created_at,
			g.started_at,
			g.ended_at
		FROM games g
		LEFT JOIN users uw ON uw.id = g.white_player_id
		LEFT JOIN users ub ON ub.id = g.black_player_id
		WHERE g.status = 'active'
		  AND (
		    (
		      COALESCE(g.time_control_mode, 'rapid') != 'daily'
		      AND COALESCE(g.turn_started_at, g.started_at, g.created_at) + (
		        CASE WHEN COALESCE(g.current_turn, 'w') IN ('w', 'White')
		             THEN COALESCE(g.white_time_ms, 600000)
		             ELSE COALESCE(g.black_time_ms, 600000)
		        END * interval '1 millisecond'
		      ) <= NOW()
		    )
		    OR
		    (
		      COALESCE(g.time_control_mode, 'rapid') != 'daily'
		      AND split_part(g.position, ' ', 6) = '1'
		      AND COALESCE(g.turn_started_at, g.started_at, g.created_at) + ($2::bigint * interval '1 millisecond') <= NOW()
		    )`

	if includeDaily {
		query += `
		    OR
		    (
		      COALESCE(g.time_control_mode, 'rapid') = 'daily'
		      AND COALESCE(g.turn_started_at, g.started_at, g.created_at) + (
		        COALESCE(g.daily_move_time_ms, 86400000) * interval '1 millisecond'
		      ) <= NOW()
		    )`
	}

	query += `
		  )
		ORDER BY COALESCE(g.turn_started_at, g.started_at, g.created_at) ASC
		LIMIT $1`

	rows, err := r.DB.Query(ctx, query, limit, firstMoveTimeoutMs)
	if err != nil {
		slog.ErrorContext(ctx, "get timed out games query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		err := rows.Scan(
			&g.ID,
			&g.WhitePlayerID,
			&g.BlackPlayerID,
			&g.WhiteUsername,
			&g.BlackUsername,
			&g.TimeControlMinutes,
			&g.Rated,
			&g.TimeControlMode,
			&g.DailyMoveTimeMs,
			&g.Position,
			&g.Status,
			&g.Result,
			&g.EndReason,
			&g.InitialTimeMs,
			&g.IncrementMs,
			&g.WhiteTimeMs,
			&g.BlackTimeMs,
			&g.CurrentTurn,
			&g.TurnStartedAt,
			&g.CreatedAt,
			&g.StartedAt,
			&g.EndedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan timed out game row failed", "error", err)
			return nil, err
		}
		games = append(games, g)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get timed out games rows iteration error", "error", err)
		return nil, err
	}

	return games, nil
}

func (r *Repository) UpdateGameState(
	ctx context.Context,
	gameID string,
	position string,
	status string,
	result string,
	endReason string,
	currentTurn string,
	whiteTimeMs int64,
	blackTimeMs int64,
	turnStartedAt *time.Time,
) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE games
		SET position = $1,
			status = $2::text,
			result = NULLIF($3, ''),
			end_reason = NULLIF($4, ''),
			current_turn = $5,
			white_time_ms = $6,
			black_time_ms = $7,
			turn_started_at = $8,
			ended_at = CASE
				WHEN $2::text = 'finished' THEN NOW()
				ELSE ended_at
			END
		WHERE id = $9`,
		position,
		status,
		result,
		endReason,
		currentTurn,
		whiteTimeMs,
		blackTimeMs,
		turnStartedAt,
		gameID,
	)

	if err != nil {
		slog.ErrorContext(ctx, "update game state failed", "game_id", gameID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) AddMove(
	ctx context.Context,
	move *GameMove,
) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO game_moves (
			id,
			game_id,
			move_number,
			player_id,
			move,
			position_after,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		move.ID,
		move.GameID,
		move.MoveNumber,
		move.PlayerID,
		move.Move,
		move.PositionAfter,
		move.CreatedAt,
	)

	if err != nil {
		slog.ErrorContext(ctx, "add move failed", "game_id", move.GameID, "move_number", move.MoveNumber, "error", err)
		return err
	}

	return nil
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.DB.Begin(ctx)
}

func (r *Repository) SaveMoveTx(
	ctx context.Context,
	gameID string,
	position string,
	status string,
	result string,
	endReason string,
	currentTurn string,
	whiteTimeMs int64,
	blackTimeMs int64,
	turnStartedAt *time.Time,
	move *GameMove,
	expectedTurn ...string,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin transaction failed for save move", "game_id", gameID, "error", err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	expected := ""
	if len(expectedTurn) > 0 && expectedTurn[0] != "" {
		expected = expectedTurn[0]
	} else {
		switch currentTurn {
		case "w":
			expected = "b"
		case "b":
			expected = "w"
		}
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE games
		SET position = $1,
			status = $2::text,
			result = NULLIF($3, ''),
			end_reason = NULLIF($4, ''),
			current_turn = $5,
			white_time_ms = $6,
			black_time_ms = $7,
			turn_started_at = $8,
			ended_at = CASE
				WHEN $2::text = 'finished' THEN NOW()
				ELSE ended_at
			END
		WHERE id = $9
		  AND ($10 = '' OR current_turn = $10 OR ($10 = 'w' AND current_turn = 'White') OR ($10 = 'b' AND current_turn = 'Black'))
		  AND status = 'active'`,
		position,
		status,
		result,
		endReason,
		currentTurn,
		whiteTimeMs,
		blackTimeMs,
		turnStartedAt,
		gameID,
		expected,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update game state in transaction failed", "game_id", gameID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("game not found or state changed")
	}

	if move != nil {
		if move.MoveNumber <= 0 {
			var moveNum int
			err = tx.QueryRow(
				ctx,
				`SELECT COALESCE(MAX(move_number), 0) + 1 FROM game_moves WHERE game_id = $1`,
				gameID,
			).Scan(&moveNum)
			if err != nil {
				slog.ErrorContext(ctx, "query move number in transaction failed", "game_id", gameID, "error", err)
				return err
			}
			move.MoveNumber = moveNum
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO game_moves (
				id,
				game_id,
				move_number,
				player_id,
				move,
				position_after,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			move.ID,
			move.GameID,
			move.MoveNumber,
			move.PlayerID,
			move.Move,
			move.PositionAfter,
			move.CreatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "add move in transaction failed", "game_id", move.GameID, "move_number", move.MoveNumber, "error", err)
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit transaction failed for save move", "game_id", gameID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) GetMoves(
	ctx context.Context,
	gameID string,
) ([]GameMove, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			id,
			game_id,
			move_number,
			player_id,
			move,
			position_after,
			created_at
		FROM game_moves
		WHERE game_id = $1
		ORDER BY move_number ASC`,
		gameID,
	)

	if err != nil {
		slog.ErrorContext(ctx, "get moves query failed", "game_id", gameID, "error", err)
		return nil, err
	}

	defer rows.Close()

	var moves []GameMove

	for rows.Next() {
		var move GameMove

		err := rows.Scan(
			&move.ID,
			&move.GameID,
			&move.MoveNumber,
			&move.PlayerID,
			&move.Move,
			&move.PositionAfter,
			&move.CreatedAt,
		)

		if err != nil {
			slog.ErrorContext(ctx, "scan move row failed", "game_id", gameID, "error", err)
			return nil, err
		}

		moves = append(moves, move)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get moves rows iteration error", "game_id", gameID, "error", err)
		return nil, err
	}

	return moves, nil
}

func FormatPlayerUUID(playerID int64) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strconv.FormatInt(playerID, 10))).String()
}

func NewGameMove(
	gameID string,
	playerID int64,
	moveNumber int,
	move string,
	position string,
) *GameMove {
	return &GameMove{
		ID:            uuid.NewString(),
		GameID:        gameID,
		PlayerID:      FormatPlayerUUID(playerID),
		MoveNumber:    moveNumber,
		Move:          move,
		PositionAfter: position,
	}
}

func (r *Repository) GetPlayerGames(
	ctx context.Context,
	playerID int64,
) ([]Game, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			g.id,
			g.white_player_id,
			g.black_player_id,
			COALESCE(uw.username, ''),
			COALESCE(ub.username, ''),
			g.time_control_minutes,
			g.rated,
			COALESCE(g.time_control_mode, 'rapid'),
			COALESCE(g.daily_move_time_ms, 86400000),
			g.position,
			g.status,
			COALESCE(g.result, ''),
			COALESCE(g.end_reason, ''),
			COALESCE(g.initial_time_ms, 600000),
			COALESCE(g.increment_ms, 0),
			COALESCE(g.white_time_ms, 600000),
			COALESCE(g.black_time_ms, 600000),
			COALESCE(g.current_turn, 'w'),
			g.turn_started_at,
			g.created_at,
			g.started_at,
			g.ended_at,
			g.white_rating_before,
			g.white_rating_after,
			g.black_rating_before,
			g.black_rating_after,
			g.white_provisional,
			g.black_provisional
		FROM games g
		LEFT JOIN users uw ON uw.id = g.white_player_id
		LEFT JOIN users ub ON ub.id = g.black_player_id
		WHERE g.white_player_id = $1 OR g.black_player_id = $1
		ORDER BY g.created_at DESC
		LIMIT 50`,
		playerID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get player games query failed", "player_id", playerID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		err := rows.Scan(
			&g.ID,
			&g.WhitePlayerID,
			&g.BlackPlayerID,
			&g.WhiteUsername,
			&g.BlackUsername,
			&g.TimeControlMinutes,
			&g.Rated,
			&g.TimeControlMode,
			&g.DailyMoveTimeMs,
			&g.Position,
			&g.Status,
			&g.Result,
			&g.EndReason,
			&g.InitialTimeMs,
			&g.IncrementMs,
			&g.WhiteTimeMs,
			&g.BlackTimeMs,
			&g.CurrentTurn,
			&g.TurnStartedAt,
			&g.CreatedAt,
			&g.StartedAt,
			&g.EndedAt,
			&g.WhiteRatingBefore,
			&g.WhiteRatingAfter,
			&g.BlackRatingBefore,
			&g.BlackRatingAfter,
			&g.WhiteProvisional,
			&g.BlackProvisional,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan player game row failed", "player_id", playerID, "error", err)
			return nil, err
		}
		games = append(games, g)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get player games rows iteration error", "player_id", playerID, "error", err)
		return nil, err
	}

	return games, nil
}

func (r *Repository) DeleteGameWithMoves(
	ctx context.Context,
	gameID string,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin transaction failed for delete game", "game_id", gameID, "error", err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(
		ctx,
		`DELETE FROM game_moves WHERE game_id = $1`,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete game moves in transaction failed", "game_id", gameID, "error", err)
		return err
	}

	_, err = tx.Exec(
		ctx,
		`DELETE FROM game_chat_messages WHERE game_id = $1`,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete game chat messages in transaction failed", "game_id", gameID, "error", err)
		return err
	}

	_, err = tx.Exec(
		ctx,
		`DELETE FROM games WHERE id = $1 AND status = 'finished'`,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete completed game in transaction failed", "game_id", gameID, "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit transaction failed for delete game", "game_id", gameID, "error", err)
		return err
	}

	return nil
}
