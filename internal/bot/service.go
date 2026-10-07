package bot

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/corentings/chess/v2"
)

const (
	MaxActiveGames = 5
	ColorRandom    = "random"
	botMoveTimeout = 10 * time.Second
)

type MoveEngine interface {
	BestMove(ctx context.Context, fen string, settings EngineSettings) (string, error)
}

type GameService interface {
	CreateGame(ctx context.Context, playerID int64, level string, rating int, color string) (*Game, error)
	GetGame(ctx context.Context, gameID string, playerID int64) (*Game, error)
	PlayMove(ctx context.Context, gameID string, playerID int64, move string) (*Game, error)
	PlayBotMove(ctx context.Context, gameID string, playerID int64) (*Game, error)
	Resign(ctx context.Context, gameID string, playerID int64) (*Game, error)
}

type Service struct {
	Repository GameRepository
	Engine     MoveEngine
}

func NewService(repository GameRepository, engine MoveEngine) GameService {
	return &Service{
		Repository: repository,
		Engine:     engine,
	}
}

func (s *Service) CreateGame(
	ctx context.Context,
	playerID int64,
	level string,
	rating int,
	color string,
) (*Game, error) {
	botRating, err := ResolveRating(level, rating)
	if err != nil {
		return nil, err
	}

	playerColor, err := resolveColor(color)
	if err != nil {
		return nil, err
	}

	activeGames, err := s.Repository.CountActiveGames(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if activeGames >= MaxActiveGames {
		return nil, ErrTooManyActiveGames
	}

	game := &Game{
		PlayerID:    playerID,
		PlayerColor: playerColor,
		BotLevel:    level,
		BotRating:   botRating,
		Position:    chess.NewGame().FEN(),
		Moves:       []string{},
		Status:      StatusActive,
	}

	if err := s.Repository.CreateGame(ctx, game); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "bot game created", "game_id", game.ID, "player_id", playerID, "level", level, "rating", botRating)

	return game, nil
}

func (s *Service) GetGame(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	game, err := s.Repository.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if game.PlayerID != playerID {
		return nil, ErrGameNotFound
	}

	return game, nil
}

func (s *Service) PlayMove(
	ctx context.Context,
	gameID string,
	playerID int64,
	move string,
) (*Game, error) {
	game, err := s.GetGame(ctx, gameID, playerID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusActive {
		return nil, ErrGameFinished
	}

	board, err := replayMoves(game.Moves)
	if err != nil {
		return nil, err
	}

	if !isColorTurn(board, game.PlayerColor) {
		return nil, ErrNotYourTurn
	}

	if err := board.PushNotationMove(move, chess.UCINotation{}, nil); err != nil {
		return nil, ErrInvalidMove
	}

	return s.saveMove(ctx, game, board, move)
}

func (s *Service) PlayBotMove(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	game, err := s.GetGame(ctx, gameID, playerID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusActive {
		return game, nil
	}

	board, err := replayMoves(game.Moves)
	if err != nil {
		return nil, err
	}

	if !isColorTurn(board, game.BotColor()) {
		return game, nil
	}

	engineCtx, cancel := context.WithTimeout(ctx, botMoveTimeout)
	defer cancel()

	move, err := s.Engine.BestMove(engineCtx, board.FEN(), SettingsForRating(game.BotRating))
	if err != nil {
		slog.ErrorContext(ctx, "bot engine failed", "game_id", gameID, "error", err)
		return nil, err
	}

	if err := board.PushNotationMove(move, chess.UCINotation{}, nil); err != nil {
		return nil, fmt.Errorf("engine played illegal move %q: %w", move, err)
	}

	return s.saveMove(ctx, game, board, move)
}

func (s *Service) Resign(
	ctx context.Context,
	gameID string,
	playerID int64,
) (*Game, error) {
	game, err := s.GetGame(ctx, gameID, playerID)
	if err != nil {
		return nil, err
	}

	if game.Status != StatusActive {
		return nil, ErrGameFinished
	}

	now := time.Now()
	game.Status = StatusFinished
	game.EndReason = EndReasonResignation
	game.EndedAt = &now

	if game.PlayerColor == ColorWhite {
		game.Result = ResultBlackWin
	} else {
		game.Result = ResultWhiteWin
	}

	if err := s.Repository.UpdateGame(ctx, game, game.Moves); err != nil {
		return nil, err
	}

	return game, nil
}

func (s *Service) saveMove(
	ctx context.Context,
	game *Game,
	board *chess.Game,
	move string,
) (*Game, error) {
	previousMoves := slices.Clone(game.Moves)

	game.Moves = append(game.Moves, move)
	game.Position = board.FEN()
	finishIfOver(game, board)

	if err := s.Repository.UpdateGame(ctx, game, previousMoves); err != nil {
		return nil, err
	}

	return game, nil
}

func replayMoves(moves []string) (*chess.Game, error) {
	board := chess.NewGame()

	for _, move := range moves {
		if err := board.PushNotationMove(move, chess.UCINotation{}, nil); err != nil {
			return nil, fmt.Errorf("replay move %q: %w", move, err)
		}
	}

	return board, nil
}

func isColorTurn(board *chess.Game, color string) bool {
	if color == ColorWhite {
		return board.Position().Turn() == chess.White
	}
	return board.Position().Turn() == chess.Black
}

func resolveColor(color string) (string, error) {
	switch color {
	case "", ColorWhite:
		return ColorWhite, nil
	case ColorBlack:
		return ColorBlack, nil
	case ColorRandom:
		if rand.IntN(2) == 0 { //nolint:gosec // picking a side does not need secure randomness
			return ColorWhite, nil
		}
		return ColorBlack, nil
	default:
		return "", ErrInvalidColor
	}
}

func finishIfOver(game *Game, board *chess.Game) {
	for _, method := range board.EligibleDraws() {
		if method == chess.ThreefoldRepetition || method == chess.FiftyMoveRule {
			_ = board.Draw(method)
			break
		}
	}

	if board.Outcome() == chess.NoOutcome {
		return
	}

	now := time.Now()
	game.Status = StatusFinished
	game.EndedAt = &now
	game.EndReason = endReasonFor(board.Method())

	switch board.Outcome() {
	case chess.WhiteWon:
		game.Result = ResultWhiteWin
	case chess.BlackWon:
		game.Result = ResultBlackWin
	default:
		game.Result = ResultDraw
	}
}

func endReasonFor(method chess.Method) string {
	switch method {
	case chess.Checkmate:
		return EndReasonCheckmate
	case chess.Stalemate:
		return EndReasonStalemate
	case chess.ThreefoldRepetition, chess.FivefoldRepetition:
		return EndReasonThreefoldRepetition
	case chess.FiftyMoveRule:
		return EndReason50MoveRule
	case chess.SeventyFiveMoveRule:
		return EndReason75MoveRule
	case chess.InsufficientMaterial:
		return EndReasonInsufficientMaterial
	default:
		return ""
	}
}
