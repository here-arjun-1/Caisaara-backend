package game

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type GameService interface {
	MakeMove(ctx context.Context, gameID string, playerID int64, move string) (*Game, error)
	GetGame(ctx context.Context, gameID string) (*Game, error)
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
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
		return nil, errors.New("game is not active")
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

	if err := chessGame.MakeMove(move); err != nil {
		slog.WarnContext(ctx, "invalid move execution", "game_id", gameID, "player_id", playerID, "move", move, "error", err)
		return nil, errors.New("invalid move")
	}

	existingMoves, err := s.Repository.GetMoves(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "fetch moves count failed", "game_id", gameID, "error", err)
		return nil, err
	}
	moveNumber := len(existingMoves) + 1

	position := chessGame.FEN()
	status := StatusActive
	result := ""

	if chessGame.IsFinished() {
		status = StatusFinished
		result = getGameResult(chessGame)
	}

	err = s.Repository.UpdateGameState(
		ctx,
		gameID,
		position,
		status,
		result,
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
	gameMove.CreatedAt = time.Now()

	err = s.Repository.AddMove(ctx, gameMove)
	if err != nil {
		slog.ErrorContext(ctx, "add move failed in make move", "game_id", gameID, "error", err)
		return nil, err
	}

	currentGame.Position = position
	currentGame.Status = status
	currentGame.Result = result

	slog.InfoContext(ctx, "move completed successfully", "game_id", gameID, "player_id", playerID, "move", move, "status", status)

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

func (s *Service) isPlayerTurn(
	chessGame *ChessGame,
	currentGame *Game,
	playerID int64,
) bool {
	if chessGame.Turn() == "White" {
		return playerID == currentGame.WhitePlayerID
	}
	return playerID == currentGame.BlackPlayerID
}

func getGameResult(chessGame *ChessGame) string {
	switch chessGame.Outcome() {
	case "1-0":
		return ResultWhiteWin
	case "0-1":
		return ResultBlackWin
	default:
		return ResultDraw
	}
}
