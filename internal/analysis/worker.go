package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/hibiken/asynq"
)

const (
	TypeGameAnalysis = "game:analysis"
)

type GameAnalysisPayload struct {
	GameID string `json:"game_id"`
}

func NewGameAnalysisTask(gameID string) (*asynq.Task, error) {
	payload, err := json.Marshal(GameAnalysisPayload{GameID: gameID})
	if err != nil {
		return nil, fmt.Errorf("marshal analysis payload: %w", err)
	}
	return asynq.NewTask(TypeGameAnalysis, payload), nil
}

type AnalysisProcessor struct {
	Engine       *AnalysisEngine
	AnalysisRepo AnalysisRepository
	GameRepo     game.GameRepository
	StatsService *StatisticsService
	Thresholds   ClassificationThresholds
}

func NewAnalysisProcessor(
	engine *AnalysisEngine,
	analysisRepo AnalysisRepository,
	gameRepo game.GameRepository,
) *AnalysisProcessor {
	return &AnalysisProcessor{
		Engine:       engine,
		AnalysisRepo: analysisRepo,
		GameRepo:     gameRepo,
		StatsService: NewStatisticsService(),
		Thresholds:   DefaultThresholds(),
	}
}

func StartAnalysisServer(
	redisOpt asynq.RedisClientOpt,
	processor *AnalysisProcessor,
	concurrency int,
) *asynq.Server {
	if concurrency <= 0 {
		concurrency = 2
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return 1 * time.Minute
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeGameAnalysis, processor.ProcessTaskGameAnalysis)

	if err := server.Start(mux); err != nil {
		slog.Error("could not start asynq game analysis server", "error", err)
		os.Exit(1)
	}

	return server
}

func (p *AnalysisProcessor) ProcessTaskGameAnalysis(ctx context.Context, t *asynq.Task) error {
	var pld GameAnalysisPayload
	if err := json.Unmarshal(t.Payload(), &pld); err != nil {
		return fmt.Errorf("unmarshal game analysis payload: %w: %w", err, asynq.SkipRetry)
	}

	gameID := pld.GameID
	slog.InfoContext(ctx, "starting background move-by-move game analysis task", "game_id", gameID)

	targetGame, err := p.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "find game for analysis failed", "game_id", gameID, "error", err)
		_ = p.AnalysisRepo.UpdateStatus(ctx, gameID, StatusFailed, "game not found: "+err.Error())
		return fmt.Errorf("find game: %w: %w", err, asynq.SkipRetry)
	}

	if targetGame.Status != game.StatusFinished {
		slog.WarnContext(ctx, "cannot analyze unfinished game", "game_id", gameID, "status", targetGame.Status)
		_ = p.AnalysisRepo.UpdateStatus(ctx, gameID, StatusFailed, "game is not finished")
		return nil
	}

	moves, err := p.GameRepo.GetMoves(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "fetch moves for analysis failed", "game_id", gameID, "error", err)
		_ = p.AnalysisRepo.UpdateStatus(ctx, gameID, StatusFailed, "failed to load moves: "+err.Error())
		return fmt.Errorf("get moves: %w", err)
	}

	_ = p.AnalysisRepo.UpdateStatus(ctx, gameID, StatusProcessing, "")

	analysisRecord, err := p.AnalysisRepo.GetAnalysisByGameID(ctx, gameID)
	if err != nil || analysisRecord == nil {
		analysisRecord, err = p.AnalysisRepo.CreateAnalysis(ctx, gameID, "Stockfish 16", 12)
		if err != nil {
			return fmt.Errorf("create analysis record: %w", err)
		}
	}

	chessBoard := game.NewChessGame()
	opts := AnalysisOptions{
		Depth:      12,
		MoveTimeMs: 150,
	}

	var moveAnalyses []GameMoveAnalysis
	whiteLosses := []float64{}
	blackLosses := []float64{}
	whiteOpeningLosses := []float64{}
	blackOpeningLosses := []float64{}
	whiteMiddleLosses := []float64{}
	blackMiddleLosses := []float64{}
	whiteEndLosses := []float64{}
	blackEndLosses := []float64{}

	for idx, moveRecord := range moves {
		playerColor := "white"
		if idx%2 != 0 {
			playerColor = "black"
		}

		posBeforeFEN := chessBoard.FEN()

		evalBeforeRes, err := p.Engine.Analyze(ctx, PositionOptions{FEN: posBeforeFEN}, opts)
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
		var evalAfterWhite int

		if chessBoard.IsFinished() {
			switch chessBoard.Outcome() {
			case "1-0":
				evalAfterWhite = 10000
			case "0-1":
				evalAfterWhite = -10000
			default:
				evalAfterWhite = 0
			}
		} else {
			evalAfterRes, err := p.Engine.Analyze(ctx, PositionOptions{FEN: posAfterFEN}, opts)
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

		phase := DeterminePhase(posBeforeFEN, moveRecord.MoveNumber)
		classification := classifyMoveWithThresholds(cpl, moveRecord.Move, bestMove, p.Thresholds)
		explanation := GenerateMoveExplanation(classification, moveRecord.Move, bestMove, cpl, evalBeforeWhite, evalAfterWhite)

		cplF := float64(cpl)

		if playerColor == "white" {
			whiteLosses = append(whiteLosses, cplF)
			switch phase {
			case PhaseOpening:
				whiteOpeningLosses = append(whiteOpeningLosses, cplF)
			case PhaseMiddlegame:
				whiteMiddleLosses = append(whiteMiddleLosses, cplF)
			case PhaseEndgame:
				whiteEndLosses = append(whiteEndLosses, cplF)
			}

			switch classification {
			case ClassificationBrilliant:
				analysisRecord.BrilliantWhite++
			case ClassificationGreat:
				analysisRecord.GreatWhite++
			case ClassificationBest:
				analysisRecord.BestWhite++
			case ClassificationGood:
				analysisRecord.GoodWhite++
			case ClassificationInaccuracy:
				analysisRecord.InaccuraciesWhite++
			case ClassificationMistake:
				analysisRecord.MistakesWhite++
			case ClassificationMiss:
				analysisRecord.MissesWhite++
			case ClassificationBlunder:
				analysisRecord.BlundersWhite++
			}
		} else {
			blackLosses = append(blackLosses, cplF)
			switch phase {
			case PhaseOpening:
				blackOpeningLosses = append(blackOpeningLosses, cplF)
			case PhaseMiddlegame:
				blackMiddleLosses = append(blackMiddleLosses, cplF)
			case PhaseEndgame:
				blackEndLosses = append(blackEndLosses, cplF)
			}

			switch classification {
			case ClassificationBrilliant:
				analysisRecord.BrilliantBlack++
			case ClassificationGreat:
				analysisRecord.GreatBlack++
			case ClassificationBest:
				analysisRecord.BestBlack++
			case ClassificationGood:
				analysisRecord.GoodBlack++
			case ClassificationInaccuracy:
				analysisRecord.InaccuraciesBlack++
			case ClassificationMistake:
				analysisRecord.MistakesBlack++
			case ClassificationMiss:
				analysisRecord.MissesBlack++
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
			Phase:          phase,
			Explanation:    explanation,
			CreatedAt:      time.Now(),
		}

		moveAnalyses = append(moveAnalyses, moveAnalysis)
	}

	accWhite := CalculateAccuracyFormula(whiteLosses)
	accBlack := CalculateAccuracyFormula(blackLosses)

	openWhite := CalculateAccuracyFormula(whiteOpeningLosses)
	openBlack := CalculateAccuracyFormula(blackOpeningLosses)
	midWhite := CalculateAccuracyFormula(whiteMiddleLosses)
	midBlack := CalculateAccuracyFormula(blackMiddleLosses)
	endWhite := CalculateAccuracyFormula(whiteEndLosses)
	endBlack := CalculateAccuracyFormula(blackEndLosses)

	analysisRecord.AccuracyWhite = &accWhite
	analysisRecord.AccuracyBlack = &accBlack
	analysisRecord.OpeningAccuracyWhite = &openWhite
	analysisRecord.OpeningAccuracyBlack = &openBlack
	analysisRecord.MiddlegameAccuracyWhite = &midWhite
	analysisRecord.MiddlegameAccuracyBlack = &midBlack
	analysisRecord.EndgameAccuracyWhite = &endWhite
	analysisRecord.EndgameAccuracyBlack = &endBlack
	analysisRecord.RatingWhite = CalculatePerformanceRating(accWhite)
	analysisRecord.RatingBlack = CalculatePerformanceRating(accBlack)
	analysisRecord.Status = StatusCompleted

	err = p.AnalysisRepo.SaveAnalysisResult(ctx, analysisRecord, moveAnalyses)
	if err != nil {
		slog.ErrorContext(ctx, "save move-by-move analysis failed", "game_id", gameID, "error", err)
		_ = p.AnalysisRepo.UpdateStatus(ctx, gameID, StatusFailed, "save analysis result failed: "+err.Error())
		return fmt.Errorf("save analysis result: %w", err)
	}

	slog.InfoContext(ctx, "background game analysis task completed", "game_id", gameID, "accuracy_white", accWhite, "accuracy_black", accBlack)

	return nil
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
	case cpl <= thresholds.Great:
		return ClassificationGreat
	case cpl <= thresholds.Best:
		return ClassificationBest
	case cpl <= thresholds.Good:
		return ClassificationGood
	case cpl <= thresholds.Inaccuracy:
		return ClassificationInaccuracy
	case cpl <= thresholds.Mistake:
		return ClassificationMistake
	case cpl <= thresholds.Miss:
		return ClassificationMiss
	default:
		return ClassificationBlunder
	}
}
