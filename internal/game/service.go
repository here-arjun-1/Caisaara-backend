package game

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type GameService interface {
	MakeMove(ctx context.Context, gameID string, playerID int64, move string) (*Game, error)
	ResignGame(ctx context.Context, gameID string, playerID int64) (*Game, error)
	DrawGame(ctx context.Context, gameID string, playerID int64) (*Game, error)
	GetGame(ctx context.Context, gameID string) (*Game, error)
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
	GetPlayerGames(ctx context.Context, playerID int64) ([]Game, error)
}

type Service struct {
	Repository GameRepository
}

func NewService(repository GameRepository) GameService {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) MakeMove(
	ctx context.Context,
	gameID string,
	playerID int64,
	move string,
) (*Game, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "find game failed during make move", "game_id", gameID, "error", err)
		return nil, err
	}

	if currentGame.Status != StatusActive {
		slog.WarnContext(ctx, "game is not active", "game_id", gameID, "status", currentGame.Status)
		return nil, errors.New("game is already finished")
	}

	if playerID != currentGame.WhitePlayerID && playerID != currentGame.BlackPlayerID {
		slog.WarnContext(ctx, "player not in game", "game_id", gameID, "player_id", playerID)
		return nil, errors.New("player is not part of this game")
	}

	chessGame, err := NewChessGameFromFEN(currentGame.Position)
	if err != nil {
		slog.ErrorContext(ctx, "parse FEN position failed", "game_id", gameID, "position", currentGame.Position, "error", err)
		return nil, err
	}

	if !s.isPlayerTurn(chessGame, currentGame, playerID) {
		slog.WarnContext(ctx, "not player turn", "game_id", gameID, "player_id", playerID, "turn", chessGame.Turn())
		return nil, errors.New("not your turn")
	}

	now := time.Now()
	var turnStartedAt time.Time
	switch {
	case currentGame.TurnStartedAt != nil:
		turnStartedAt = *currentGame.TurnStartedAt
	case currentGame.StartedAt != nil:
		turnStartedAt = *currentGame.StartedAt
	default:
		turnStartedAt = now
	}

	elapsedMs := now.Sub(turnStartedAt).Milliseconds()
	if elapsedMs < 0 {
		elapsedMs = 0
	}

	isWhiteTurn := currentGame.CurrentTurn == "w" || currentGame.CurrentTurn == "White"
	status := StatusActive
	result := ""
	endReason := ""

	newWhiteTimeMs := currentGame.WhiteTimeMs
	newBlackTimeMs := currentGame.BlackTimeMs

	if currentGame.TimeControlMode == ModeDaily {
		dailyLimitMs := currentGame.DailyMoveTimeMs
		if dailyLimitMs <= 0 {
			dailyLimitMs = 86400000
		}
		if elapsedMs > dailyLimitMs {
			status = StatusFinished
			endReason = EndReasonDailyTimeout
			if isWhiteTurn {
				result = ResultBlackWin
			} else {
				result = ResultWhiteWin
			}
		}
	} else {
		if isWhiteTurn {
			newWhiteTimeMs = currentGame.WhiteTimeMs - elapsedMs
			if newWhiteTimeMs <= 0 {
				newWhiteTimeMs = 0
				status = StatusFinished
				endReason = EndReasonTimeout
				result = ResultBlackWin
			}
		} else {
			newBlackTimeMs = currentGame.BlackTimeMs - elapsedMs
			if newBlackTimeMs <= 0 {
				newBlackTimeMs = 0
				status = StatusFinished
				endReason = EndReasonTimeout
				result = ResultWhiteWin
			}
		}
	}

	if status == StatusFinished {
		err = s.Repository.UpdateGameState(
			ctx,
			gameID,
			currentGame.Position,
			status,
			result,
			endReason,
			currentGame.CurrentTurn,
			newWhiteTimeMs,
			newBlackTimeMs,
			&now,
		)
		if err != nil {
			slog.ErrorContext(ctx, "update game state on time expiry failed", "game_id", gameID, "error", err)
			return nil, err
		}

		_ = s.Repository.EnforceRetention(ctx, currentGame.WhitePlayerID)
		_ = s.Repository.EnforceRetention(ctx, currentGame.BlackPlayerID)

		currentGame.Status = status
		currentGame.Result = result
		currentGame.EndReason = endReason
		currentGame.WhiteTimeMs = newWhiteTimeMs
		currentGame.BlackTimeMs = newBlackTimeMs
		currentGame.TurnStartedAt = &now
		return currentGame, nil
	}

	if err := chessGame.MakeMove(move); err != nil {
		slog.WarnContext(ctx, "invalid move execution", "game_id", gameID, "player_id", playerID, "move", move, "error", err)
		return nil, errors.New("invalid move")
	}

	if currentGame.TimeControlMode != ModeDaily {
		if isWhiteTurn {
			newWhiteTimeMs += currentGame.IncrementMs
		} else {
			newBlackTimeMs += currentGame.IncrementMs
		}
	}

	existingMoves, err := s.Repository.GetMoves(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "fetch moves count failed", "game_id", gameID, "error", err)
		return nil, err
	}
	moveNumber := len(existingMoves) + 1

	position := chessGame.FEN()

	if chessGame.IsFinished() {
		status = StatusFinished
		result, endReason = getGameResultAndEndReason(chessGame)
	}

	nextTurn := chessGame.Turn()

	err = s.Repository.UpdateGameState(
		ctx,
		gameID,
		position,
		status,
		result,
		endReason,
		nextTurn,
		newWhiteTimeMs,
		newBlackTimeMs,
		&now,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update game state failed in make move", "game_id", gameID, "error", err)
		return nil, err
	}

	gameMove := NewGameMove(
		gameID,
		playerID,
		moveNumber,
		move,
		position,
	)
	gameMove.CreatedAt = now

	err = s.Repository.AddMove(ctx, gameMove)
	if err != nil {
		slog.ErrorContext(ctx, "add move failed in make move", "game_id", gameID, "error", err)
		return nil, err
	}

	if status == StatusFinished {
		_ = s.Repository.EnforceRetention(ctx, currentGame.WhitePlayerID)
		_ = s.Repository.EnforceRetention(ctx, currentGame.BlackPlayerID)
	}

	currentGame.Position = position
	currentGame.Status = status
	currentGame.Result = result
	currentGame.EndReason = endReason
	currentGame.CurrentTurn = nextTurn
	currentGame.WhiteTimeMs = newWhiteTimeMs
	currentGame.BlackTimeMs = newBlackTimeMs
	currentGame.TurnStartedAt = &now

	slog.InfoContext(ctx, "move completed successfully", "game_id", gameID, "player_id", playerID, "move", move, "status", status)

	return currentGame, nil
}

func (s *Service) ResignGame(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if currentGame.Status != StatusActive {
		return nil, errors.New("game is already finished")
	}

	if playerID != currentGame.WhitePlayerID && playerID != currentGame.BlackPlayerID {
		return nil, errors.New("player is not part of this game")
	}

	now := time.Now()
	status := StatusFinished
	endReason := EndReasonResignation

	var result string
	if playerID == currentGame.WhitePlayerID {
		result = ResultBlackWin
	} else {
		result = ResultWhiteWin
	}

	err = s.Repository.UpdateGameState(
		ctx,
		gameID,
		currentGame.Position,
		status,
		result,
		endReason,
		currentGame.CurrentTurn,
		currentGame.WhiteTimeMs,
		currentGame.BlackTimeMs,
		&now,
	)
	if err != nil {
		return nil, err
	}

	_ = s.Repository.EnforceRetention(ctx, currentGame.WhitePlayerID)
	_ = s.Repository.EnforceRetention(ctx, currentGame.BlackPlayerID)

	currentGame.Status = status
	currentGame.Result = result
	currentGame.EndReason = endReason
	currentGame.TurnStartedAt = &now

	return currentGame, nil
}

func (s *Service) DrawGame(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if currentGame.Status != StatusActive {
		return nil, errors.New("game is already finished")
	}

	if playerID != currentGame.WhitePlayerID && playerID != currentGame.BlackPlayerID {
		return nil, errors.New("player is not part of this game")
	}

	now := time.Now()
	status := StatusFinished
	result := ResultDraw
	endReason := EndReasonDrawAgreement

	err = s.Repository.UpdateGameState(
		ctx,
		gameID,
		currentGame.Position,
		status,
		result,
		endReason,
		currentGame.CurrentTurn,
		currentGame.WhiteTimeMs,
		currentGame.BlackTimeMs,
		&now,
	)
	if err != nil {
		return nil, err
	}

	_ = s.Repository.EnforceRetention(ctx, currentGame.WhitePlayerID)
	_ = s.Repository.EnforceRetention(ctx, currentGame.BlackPlayerID)

	currentGame.Status = status
	currentGame.Result = result
	currentGame.EndReason = endReason
	currentGame.TurnStartedAt = &now

	return currentGame, nil
}

func (s *Service) GetGame(
	ctx context.Context,
	gameID string,
) (*Game, error) {
	return s.Repository.FindGameByID(ctx, gameID)
}

func (s *Service) GetMoves(
	ctx context.Context,
	gameID string,
) ([]GameMove, error) {
	return s.Repository.GetMoves(ctx, gameID)
}

func (s *Service) GetPlayerGames(
	ctx context.Context,
	playerID int64,
) ([]Game, error) {
	_ = s.Repository.EnforceRetention(ctx, playerID)
	return s.Repository.GetPlayerGames(ctx, playerID)
}

func (s *Service) isPlayerTurn(
	chessGame *ChessGame,
	currentGame *Game,
	playerID int64,
) bool {
	turn := chessGame.Turn()
	if turn == "w" || turn == "White" {
		return playerID == currentGame.WhitePlayerID
	}
	return playerID == currentGame.BlackPlayerID
}

