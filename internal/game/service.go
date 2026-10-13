package game

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type GameService interface {
	MakeMove(ctx context.Context, gameID string, playerID int64, move string) (*Game, error)
	ResignGame(ctx context.Context, gameID string, playerID int64) (*Game, error)
	DrawGame(ctx context.Context, gameID string, playerID int64) (*Game, error)
	OfferDraw(ctx context.Context, gameID string, playerID int64) (*Game, bool, int64, error)
	AcceptDraw(ctx context.Context, gameID string, playerID int64) (*Game, error)
	DeclineDraw(ctx context.Context, gameID string, playerID int64) (*Game, error)
	GetGame(ctx context.Context, gameID string) (*Game, error)
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
	GetPlayerGames(ctx context.Context, playerID int64) ([]Game, error)
	SetGameCompletionListener(listener func(ctx context.Context, gameID string, result string))
	SetGameStateUpdatedListener(listener func(ctx context.Context, g *Game))
	CheckGameTimeout(ctx context.Context, currentGame *Game) (bool, error)
	CheckAllTimeouts(ctx context.Context) ([]*Game, error)
	ClaimTimeout(ctx context.Context, gameID string) (*Game, error)
	StartTimeoutChecker(ctx context.Context, interval time.Duration)
	EndGameOnDisconnect(ctx context.Context, gameID string, disconnectedPlayerID int64) (*Game, error)
}

type Service struct {
	Repository         GameRepository
	OnGameCompleted    func(ctx context.Context, gameID string, result string)
	OnGameStateUpdated func(ctx context.Context, g *Game)
	drawOffers         sync.Map
}

func NewService(repository GameRepository) GameService {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SetGameCompletionListener(listener func(ctx context.Context, gameID string, result string)) {
	s.OnGameCompleted = listener
}

func (s *Service) notifyGameCompleted(ctx context.Context, gameID string, result string) {
	if s.OnGameCompleted != nil {
		s.OnGameCompleted(ctx, gameID, result)
	}
}

func (s *Service) SetGameStateUpdatedListener(listener func(ctx context.Context, g *Game)) {
	s.OnGameStateUpdated = listener
}

func (s *Service) notifyGameStateUpdated(ctx context.Context, g *Game) {
	if s.OnGameStateUpdated != nil {
		s.OnGameStateUpdated(ctx, g)
	}
}

func (s *Service) CheckGameTimeout(ctx context.Context, currentGame *Game) (bool, error) {
	if currentGame == nil || currentGame.Status != StatusActive {
		return false, nil
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
	var timedOut bool
	var result, endReason string
	newWhiteTimeMs := currentGame.WhiteTimeMs
	newBlackTimeMs := currentGame.BlackTimeMs

	if currentGame.TimeControlMode == ModeDaily {
		dailyLimitMs := currentGame.DailyMoveTimeMs
		if dailyLimitMs <= 0 {
			dailyLimitMs = 86400000
		}
		if elapsedMs > dailyLimitMs {
			timedOut = true
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
				timedOut = true
				endReason = EndReasonTimeout
				result = ResultBlackWin
			}
		} else {
			newBlackTimeMs = currentGame.BlackTimeMs - elapsedMs
			if newBlackTimeMs <= 0 {
				newBlackTimeMs = 0
				timedOut = true
				endReason = EndReasonTimeout
				result = ResultWhiteWin
			}
		}
	}

	if !timedOut {
		return false, nil
	}

	err := s.Repository.UpdateGameState(
		ctx,
		currentGame.ID,
		currentGame.Position,
		StatusFinished,
		result,
		endReason,
		currentGame.CurrentTurn,
		newWhiteTimeMs,
		newBlackTimeMs,
		&now,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update game state on timeout failed", "game_id", currentGame.ID, "error", err)
		return false, err
	}

	_ = s.Repository.EnforceRetention(ctx, currentGame.WhitePlayerID)
	_ = s.Repository.EnforceRetention(ctx, currentGame.BlackPlayerID)

	currentGame.Status = StatusFinished
	currentGame.Result = result
	currentGame.EndReason = endReason
	currentGame.WhiteTimeMs = newWhiteTimeMs
	currentGame.BlackTimeMs = newBlackTimeMs
	currentGame.TurnStartedAt = &now

	s.notifyGameCompleted(ctx, currentGame.ID, currentGame.Result)
	s.notifyGameStateUpdated(ctx, currentGame)

	slog.InfoContext(ctx, "game ended by timeout", "game_id", currentGame.ID, "result", result, "end_reason", endReason)
	return true, nil
}

func (s *Service) CheckAllTimeouts(ctx context.Context) ([]*Game, error) {
	activeGames, err := s.Repository.GetActiveGames(ctx)
	if err != nil {
		return nil, err
	}

	var timedOutGames []*Game
	for i := range activeGames {
		g := &activeGames[i]
		timedOut, err := s.CheckGameTimeout(ctx, g)
		if err != nil {
			slog.ErrorContext(ctx, "check active game timeout failed", "game_id", g.ID, "error", err)
			continue
		}
		if timedOut {
			timedOutGames = append(timedOutGames, g)
		}
	}

	return timedOutGames, nil
}

func (s *Service) ClaimTimeout(ctx context.Context, gameID string) (*Game, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if currentGame.Status != StatusActive {
		return currentGame, nil
	}
	timedOut, err := s.CheckGameTimeout(ctx, currentGame)
	if err != nil {
		return nil, err
	}
	if !timedOut {
		return currentGame, errors.New("time has not expired")
	}
	return currentGame, nil
}

func (s *Service) StartTimeoutChecker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.CheckAllTimeouts(ctx)
			}
		}
	}()
}

func (s *Service) MakeMove(
	ctx context.Context,
	gameID string,
	playerID int64,
	move string,
) (*Game, error) {
	s.drawOffers.Delete(gameID)

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

	timedOut, err := s.CheckGameTimeout(ctx, currentGame)
	if err != nil {
		return nil, err
	}
	if timedOut {
		return currentGame, errors.New("time has expired")
	}

	if isWhiteTurn {
		newWhiteTimeMs = currentGame.WhiteTimeMs - elapsedMs
	} else {
		newBlackTimeMs = currentGame.BlackTimeMs - elapsedMs
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
	currentGame.LastMove = move

	if status == StatusFinished {
		s.notifyGameCompleted(ctx, currentGame.ID, currentGame.Result)
	}

	slog.InfoContext(ctx, "move completed successfully", "game_id", gameID, "player_id", playerID, "move", move, "status", status)

	return currentGame, nil
}

func (s *Service) ResignGame(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	s.drawOffers.Delete(gameID)

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

	s.notifyGameCompleted(ctx, currentGame.ID, currentGame.Result)

	return currentGame, nil
}

func (s *Service) DrawGame(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	s.drawOffers.Delete(gameID)

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

	s.notifyGameCompleted(ctx, currentGame.ID, currentGame.Result)

	return currentGame, nil
}

func (s *Service) OfferDraw(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, bool, int64, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, false, 0, err
	}

	if currentGame.Status != StatusActive {
		return nil, false, 0, errors.New("game is already finished")
	}

	if playerID != currentGame.WhitePlayerID && playerID != currentGame.BlackPlayerID {
		return nil, false, 0, errors.New("player is not part of this game")
	}

	val, exists := s.drawOffers.Load(gameID)
	if exists {
		offererID := val.(int64)
		if offererID != playerID {
			s.drawOffers.Delete(gameID)
			g, err := s.DrawGame(ctx, gameID, playerID)
			if err != nil {
				return nil, false, 0, err
			}
			return g, true, offererID, nil
		}
		currentGame.DrawOfferedBy = offererID
		return currentGame, false, offererID, nil
	}

	s.drawOffers.Store(gameID, playerID)
	currentGame.DrawOfferedBy = playerID
	return currentGame, false, playerID, nil
}

func (s *Service) AcceptDraw(
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

	val, exists := s.drawOffers.Load(gameID)
	if !exists {
		return nil, errors.New("no pending draw offer to accept")
	}

	offererID := val.(int64)
	if offererID == playerID {
		return nil, errors.New("cannot accept your own draw offer")
	}

	s.drawOffers.Delete(gameID)
	return s.DrawGame(ctx, gameID, playerID)
}

func (s *Service) DeclineDraw(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if playerID != currentGame.WhitePlayerID && playerID != currentGame.BlackPlayerID {
		return nil, errors.New("player is not part of this game")
	}

	s.drawOffers.Delete(gameID)
	currentGame.DrawOfferedBy = 0
	return currentGame, nil
}

func (s *Service) GetGame(
	ctx context.Context,
	gameID string,
) (*Game, error) {
	g, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if g.Status == StatusActive {
		_, _ = s.CheckGameTimeout(ctx, g)
	}
	if val, ok := s.drawOffers.Load(gameID); ok {
		g.DrawOfferedBy = val.(int64)
	}
	moves, err := s.Repository.GetMoves(ctx, gameID)
	if err == nil && len(moves) > 0 {
		g.LastMove = moves[len(moves)-1].Move
	}
	return g, nil
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

func (s *Service) EndGameOnDisconnect(
	ctx context.Context,
	gameID string,
	disconnectedPlayerID int64,
) (*Game, error) {
	s.drawOffers.Delete(gameID)

	currentGame, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if currentGame.Status != StatusActive {
		return currentGame, nil
	}

	if disconnectedPlayerID != currentGame.WhitePlayerID && disconnectedPlayerID != currentGame.BlackPlayerID {
		return nil, errors.New("player is not part of this game")
	}

	now := time.Now()
	status := StatusFinished
	endReason := EndReasonAbandonment

	var result string
	if disconnectedPlayerID == currentGame.WhitePlayerID {
		result = ResultBlackWin
	} else {
		result = ResultWhiteWin
	}

	newWhiteTimeMs := currentGame.WhiteTimeMs
	newBlackTimeMs := currentGame.BlackTimeMs

	if currentGame.TimeControlMode != ModeDaily {
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
		if isWhiteTurn {
			newWhiteTimeMs -= elapsedMs
			if newWhiteTimeMs < 0 {
				newWhiteTimeMs = 0
			}
		} else {
			newBlackTimeMs -= elapsedMs
			if newBlackTimeMs < 0 {
				newBlackTimeMs = 0
			}
		}
	}

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

	s.notifyGameCompleted(ctx, currentGame.ID, currentGame.Result)
	s.notifyGameStateUpdated(ctx, currentGame)

	return currentGame, nil
}
