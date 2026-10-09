package analysis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type AnalysisService interface {
	AnalyzeGame(ctx context.Context, gameID string) (*GameAnalysisResponse, error)
	GetAnalysis(ctx context.Context, gameID string) (*GameAnalysisResponse, error)
}

type GameAnalysisResponse struct {
	Analysis *GameAnalysis      `json:"analysis"`
	Moves    []GameMoveAnalysis `json:"moves"`
}

type Service struct {
	Engine     *AnalysisEngine
	Repo       AnalysisRepository
	GameRepo   game.GameRepository
	Thresholds ClassificationThresholds
}

func NewService(
	engine *AnalysisEngine,
	repo AnalysisRepository,
	gameRepo game.GameRepository,
) AnalysisService {
	return &Service{
		Engine:     engine,
		Repo:       repo,
		GameRepo:   gameRepo,
		Thresholds: DefaultThresholds(),
	}
}

func (s *Service) AnalyzeGame(ctx context.Context, gameID string) (*GameAnalysisResponse, error) {
	targetGame, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "find game for analysis failed", "game_id", gameID, "error", err)
		return nil, errors.New("game not found")
	}

	if targetGame.Status != game.StatusFinished {
		slog.WarnContext(ctx, "cannot analyze unfinished game", "game_id", gameID, "status", targetGame.Status)
		return nil, errors.New("game is not finished")
	}

	moves, err := s.GameRepo.GetMoves(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "fetch moves for analysis failed", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("get moves: %w", err)
	}

	analysisRecord, err := s.Repo.CreateAnalysis(ctx, gameID, "Stockfish 16", 12)
	if err != nil {
		return nil, fmt.Errorf("create analysis record: %w", err)
	}

	chessBoard := game.NewChessGame()
	opts := AnalysisOptions{
		Depth:      12,
		MoveTimeMs: 150,
	}

	var moveAnalyses []GameMoveAnalysis
	whiteLosses := []float64{}
	blackLosses := []float64{}

	for idx, moveRecord := range moves {
		playerColor := "white"
		if idx%2 != 0 {
			playerColor = "black"
		}

		posBeforeFEN := chessBoard.FEN()

		evalBeforeRes, err := s.Engine.Analyze(ctx, PositionOptions{FEN: posBeforeFEN}, opts)
		if err != nil {
			slog.WarnContext(ctx, "engine eval before move warning", "game_id", gameID, "move_number", moveRecord.MoveNumber, "error", err)
		}

		evalBeforeWhite := extractCentipawnScore(evalBeforeRes, playerColor == "white")

		bestMove := ""
		var pv []string
		if evalBeforeRes != nil {
			bestMove = evalBeforeRes.BestMove
			pv = evalBeforeRes.PV
		}

		err = chessBoard.MakeMove(moveRecord.Move)
		if err != nil {
			slog.ErrorContext(ctx, "reconstruct move failed", "game_id", gameID, "move", moveRecord.Move, "error", err)
		}

		posAfterFEN := chessBoard.FEN()
		evalAfterWhite := evalBeforeWhite

		if chessBoard.IsFinished() {
			if chessBoard.Outcome() == "1-0" {
				evalAfterWhite = 10000
			} else if chessBoard.Outcome() == "0-1" {
				evalAfterWhite = -10000
			} else {
				evalAfterWhite = 0
			}
		} else {
			evalAfterRes, err := s.Engine.Analyze(ctx, PositionOptions{FEN: posAfterFEN}, opts)
			if err != nil {
				slog.WarnContext(ctx, "engine eval after move warning", "game_id", gameID, "move_number", moveRecord.MoveNumber, "error", err)
			}
			evalAfterWhite = extractCentipawnScore(evalAfterRes, playerColor != "white")
		}

		var cpl int
		if playerColor == "white" {
			cpl = evalBeforeWhite - evalAfterWhite
		} else {
			cpl = evalAfterWhite - evalBeforeWhite
		}

		if cpl < 0 {
			cpl = 0
		}

		classification := classifyMoveWithThresholds(cpl, moveRecord.Move, bestMove, s.Thresholds)

		if playerColor == "white" {
			whiteLosses = append(whiteLosses, float64(cpl))
			switch classification {
			case ClassificationInaccuracy:
				analysisRecord.InaccuraciesWhite++
			case ClassificationMistake:
				analysisRecord.MistakesWhite++
			case ClassificationBlunder:
				analysisRecord.BlundersWhite++
			}
		} else {
			blackLosses = append(blackLosses, float64(cpl))
			switch classification {
			case ClassificationInaccuracy:
				analysisRecord.InaccuraciesBlack++
			case ClassificationMistake:
				analysisRecord.MistakesBlack++
			case ClassificationBlunder:
				analysisRecord.BlundersBlack++
			}
		}

		eb := evalBeforeWhite
		ea := evalAfterWhite

		moveAnalysis := GameMoveAnalysis{
			ID:             uuid.NewString(),
			AnalysisID:     analysisRecord.ID,
			GameID:         gameID,
			MoveNumber:     moveRecord.MoveNumber,
			PlayerColor:    playerColor,
			PlayedMove:     moveRecord.Move,
			PositionFEN:    posBeforeFEN,
			EvalBefore:     &eb,
			EvalAfter:      &ea,
			BestMove:       bestMove,
			PV:             pv,
			CentipawnLoss:  cpl,
			Classification: classification,
			CreatedAt:      time.Now(),
		}

		moveAnalyses = append(moveAnalyses, moveAnalysis)
	}

	accWhite := calculateAccuracy(whiteLosses)
	accBlack := calculateAccuracy(blackLosses)

	analysisRecord.AccuracyWhite = &accWhite
	analysisRecord.AccuracyBlack = &accBlack
	analysisRecord.Status = StatusCompleted

	err = s.Repo.SaveAnalysisResult(ctx, analysisRecord, moveAnalyses)
	if err != nil {
		slog.ErrorContext(ctx, "save move-by-move analysis failed", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("save analysis result: %w", err)
	}

	return &GameAnalysisResponse{
		Analysis: analysisRecord,
		Moves:    moveAnalyses,
	}, nil
}

func (s *Service) GetAnalysis(ctx context.Context, gameID string) (*GameAnalysisResponse, error) {
	analysisRecord, err := s.Repo.GetAnalysisByGameID(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "get analysis record failed", "game_id", gameID, "error", err)
		return nil, errors.New("analysis not found")
	}

	moves, err := s.Repo.GetMoveAnalyses(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "get move analyses failed", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("get move analyses: %w", err)
	}

	if moves == nil {
		moves = []GameMoveAnalysis{}
	}

	return &GameAnalysisResponse{
		Analysis: analysisRecord,
		Moves:    moves,
	}, nil
}

func extractCentipawnScore(res *AnalysisResult, isSideToMove bool) int {
	if res == nil {
		return 0
	}
	if res.MateIn != nil {
		m := *res.MateIn
		var val int
		if m > 0 {
			val = 10000 - (m * 10)
		} else {
			val = -10000 - (m * 10)
		}
		if !isSideToMove {
			return -val
		}
		return val
	}
	if res.EvaluationCentipawns != nil {
		val := *res.EvaluationCentipawns
		if !isSideToMove {
			return -val
		}
		return val
	}
	return 0
}

func classifyMoveWithThresholds(cpl int, playedMove, bestMove string, thresholds ClassificationThresholds) string {
	if playedMove != "" && bestMove != "" && playedMove == bestMove {
		return ClassificationBest
	}
	switch {
	case cpl <= thresholds.Best:
		return ClassificationBest
	case cpl <= thresholds.Good:
		return ClassificationGood
	case cpl <= thresholds.Inaccuracy:
		return ClassificationInaccuracy
	case cpl <= thresholds.Mistake:
		return ClassificationMistake
	default:
		return ClassificationBlunder
	}
}

func calculateAccuracy(losses []float64) float64 {
	if len(losses) == 0 {
		return 100.0
	}
	var total float64
	for _, l := range losses {
		total += l
	}
	avgLoss := total / float64(len(losses))
	accuracy := math.Max(0.0, math.Min(100.0, 100.0-(avgLoss*0.4)))
	return math.Round(accuracy*100) / 100
}
