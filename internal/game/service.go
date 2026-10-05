package game

import (
	"context"
	"errors"
	"time"
)

type Service struct {
	Repository GameRepository
}

func NewService(repository GameRepository) *Service {
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
		return nil, err
	}

	if currentGame.Status != StatusActive {
		return nil, errors.New("game is not active")
	}

	if playerID != currentGame.WhitePlayerID &&
		playerID != currentGame.BlackPlayerID {
		return nil, errors.New("player is not part of this game")
	}

	chessGame, err := NewChessGameFromFEN(currentGame.Position)
	if err != nil {
		return nil, err
	}

	if !s.isPlayerTurn(chessGame, currentGame, playerID) {
		return nil, errors.New("not your turn")
	}

	if err := chessGame.MakeMove(move); err != nil {
		return nil, errors.New("invalid move")
	}

	position := chessGame.FEN()
	status := StatusActive
	result := ""

	if chessGame.Outcome() != "*" {
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
		return nil, err
	}

	gameMove := NewGameMove(
		gameID,
		playerID,
		chessGame.MoveCount(),
		move,
		position,
	)

	gameMove.CreatedAt = time.Now()

	err = s.Repository.AddMove(ctx, gameMove)
	if err != nil {
		return nil, err
	}

	currentGame.Position = position
	currentGame.Status = status
	currentGame.Result = result

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
