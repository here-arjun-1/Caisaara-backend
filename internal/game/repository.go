package game

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	CreateGame(ctx context.Context, whitePlayerID int64, blackPlayerID int64, timeControlMinutes int, rated bool) (string, error)
	FindGameByID(ctx context.Context, gameID string) (*Game, error)
	UpdateGameState(ctx context.Context, gameID string, position string, status string, result string, currentTurn string, whiteTimeMs int64, blackTimeMs int64, turnStartedAt *time.Time) error
	AddMove(ctx context.Context, move *GameMove) error
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GameRepository {
	return &Repository{
		DB: db,
	}
}

func ResolveTimeControlMode(timeControlMinutes int) (mode string, initialTimeMs int64, incrementMs int64, dailyMoveMs int64) {
	switch timeControlMinutes {
	case 1:
		return ModeBullet, 60000, 0, 0
	case 5:
		return ModeBlitz, 300000, 0, 0
	case 10:
		return ModeRapid, 600000, 0, 0
	case 1440:
		return ModeDaily, 86400000, 0, 86400000
	default:
		if timeControlMinutes > 180 {
			dailyMs := int64(timeControlMinutes) * 60 * 1000
			return ModeDaily, dailyMs, 0, dailyMs
		}
		initialMs := int64(timeControlMinutes) * 60 * 1000
		if initialMs <= 0 {
			initialMs = 600000
		}
		return ModeCustom, initialMs, 0, 0
	}
}

func (r *Repository) CreateGame(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	rated bool,
) (string, error) {
	mode, initialTimeMs, incrementMs, dailyMoveMs := ResolveTimeControlMode(timeControlMinutes)
	return r.CreateGameWithDetails(ctx, whitePlayerID, blackPlayerID, timeControlMinutes, mode, initialTimeMs, incrementMs, dailyMoveMs)
}

func (r *Repository) CreateGameWithDetails(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	mode string,
	initialTimeMs int64,
	incrementMs int64,
	dailyMoveMs int64,
) (string, error) {
	gameID := uuid.New()

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
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			rated,
			COALESCE(time_control_mode, 'rapid'),
			COALESCE(daily_move_time_ms, 86400000),
			position,
			status,
			COALESCE(result, ''),
			COALESCE(initial_time_ms, 600000),
			COALESCE(increment_ms, 0),
			COALESCE(white_time_ms, 600000),
			COALESCE(black_time_ms, 600000),
			COALESCE(current_turn, 'w'),
			turn_started_at,
			created_at,
			started_at,
			ended_at
		FROM games
		WHERE id = $1`,
		gameID,
	).Scan(
		&g.ID,
		&g.WhitePlayerID,
		&g.BlackPlayerID,
		&g.TimeControlMinutes,
		&g.Rated,
		&g.TimeControlMode,
		&g.DailyMoveTimeMs,
		&g.Position,
		&g.Status,
		&g.Result,
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
		slog.ErrorContext(ctx, "find game by id failed", "game_id", gameID, "error", err)
		return nil, err
	}

	return &g, nil
}

func (r *Repository) UpdateGameState(
	ctx context.Context,
	gameID string,
	position string,
	status string,
	result string,
	currentTurn string,
	whiteTimeMs int64,
	blackTimeMs int64,
	turnStartedAt *time.Time,
) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE games
		SET position = $1,
			status = $2,
			result = $3,
			current_turn = $4,
			white_time_ms = $5,
			black_time_ms = $6,
			turn_started_at = $7,
			ended_at = CASE
				WHEN $2 = 'finished' THEN NOW()
				ELSE ended_at
			END
		WHERE id = $8`,
		position,
		status,
		result,
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
