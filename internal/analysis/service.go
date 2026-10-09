package analysis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/hibiken/asynq"
)

type AnalysisService interface {
	TriggerAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysis, error)
	GetAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysisResponse, error)
	GetMoveAnalyses(ctx context.Context, gameID string, userID int64) ([]GameMoveAnalysis, error)
	GetMoveAnalysisByNumber(ctx context.Context, gameID string, moveNumber int, userID int64) (*GameMoveAnalysis, error)
	RetryAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysis, error)
}

type GameAnalysisResponse struct {
	Analysis     *GameAnalysis      `json:"analysis"`
	SummaryStats SummaryStatistics  `json:"summary_stats"`
	EvalHistory  []EvalHistoryPoint `json:"eval_history"`
}

type Service struct {
	Repo            AnalysisRepository
	GameRepo        game.GameRepository
	TaskDistributor *asynq.Client
	StatsService    *StatisticsService
}

func NewService(
	repo AnalysisRepository,
	gameRepo game.GameRepository,
	taskDistributor *asynq.Client,
) AnalysisService {
	return &Service{
		Repo:            repo,
		GameRepo:        gameRepo,
		TaskDistributor: taskDistributor,
		StatsService:    NewStatisticsService(),
	}
}

func (s *Service) checkUserGamePermission(ctx context.Context, g *game.Game, userID int64) error {
	if userID <= 0 {
		return nil
	}
	if userID == g.WhitePlayerID || userID == g.BlackPlayerID {
		return nil
	}
	if g.Status == game.StatusFinished {
		return nil
	}
	return errors.New("forbidden: not part of this game")
}

func (s *Service) TriggerAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysis, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "trigger analysis game check failed", "game_id", gameID, "error", err)
		return nil, errors.New("game not found")
	}

	if err := s.checkUserGamePermission(ctx, g, userID); err != nil {
		return nil, err
	}

	if g.Status != game.StatusFinished {
		slog.WarnContext(ctx, "attempted to analyze unfinished game", "game_id", gameID, "status", g.Status)
		return nil, errors.New("game is not finished")
	}

	existing, err := s.Repo.GetAnalysisByGameID(ctx, gameID)
	if err == nil && existing != nil {
		if existing.Status == StatusPending || existing.Status == StatusProcessing || existing.Status == StatusCompleted {
			slog.InfoContext(ctx, "preventing duplicate analysis request", "game_id", gameID, "status", existing.Status)
			return existing, nil
		}
	}

	analysisRecord, err := s.Repo.CreateAnalysis(ctx, gameID, "Stockfish 16", 12)
	if err != nil {
		slog.ErrorContext(ctx, "create analysis DB record failed", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("create analysis record: %w", err)
	}

	if s.TaskDistributor != nil {
		task, err := NewGameAnalysisTask(gameID)
		if err != nil {
			slog.ErrorContext(ctx, "create analysis task failed", "game_id", gameID, "error", err)
			return nil, fmt.Errorf("create analysis task: %w", err)
		}

		info, err := s.TaskDistributor.EnqueueContext(ctx, task)
		if err != nil {
			slog.ErrorContext(ctx, "enqueue analysis task failed", "game_id", gameID, "error", err)
			return nil, fmt.Errorf("enqueue analysis task: %w", err)
		}

		slog.InfoContext(ctx, "enqueued background game analysis task", "game_id", gameID, "task_id", info.ID)
	}

	return analysisRecord, nil
}

func (s *Service) GetAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysisResponse, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, errors.New("game not found")
	}

	if err := s.checkUserGamePermission(ctx, g, userID); err != nil {
		return nil, err
	}

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

	stats, evalHistory := s.StatsService.CalculateStatistics(moves)

	return &GameAnalysisResponse{
		Analysis:     analysisRecord,
		SummaryStats: stats,
		EvalHistory:  evalHistory,
	}, nil
}

func (s *Service) GetMoveAnalyses(ctx context.Context, gameID string, userID int64) ([]GameMoveAnalysis, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, errors.New("game not found")
	}

	if err := s.checkUserGamePermission(ctx, g, userID); err != nil {
		return nil, err
	}

	moves, err := s.Repo.GetMoveAnalyses(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("get move analyses: %w", err)
	}

	if moves == nil {
		moves = []GameMoveAnalysis{}
	}

	return moves, nil
}

func (s *Service) GetMoveAnalysisByNumber(ctx context.Context, gameID string, moveNumber int, userID int64) (*GameMoveAnalysis, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, errors.New("game not found")
	}

	if err := s.checkUserGamePermission(ctx, g, userID); err != nil {
		return nil, err
	}

	moves, err := s.Repo.GetMoveAnalyses(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("get move analyses: %w", err)
	}

	for _, m := range moves {
		if m.MoveNumber == moveNumber {
			return &m, nil
		}
	}

	return nil, errors.New("move analysis not found")
}

func (s *Service) RetryAnalysis(ctx context.Context, gameID string, userID int64) (*GameAnalysis, error) {
	g, err := s.GameRepo.FindGameByID(ctx, gameID)
	if err != nil {
		return nil, errors.New("game not found")
	}

	if err := s.checkUserGamePermission(ctx, g, userID); err != nil {
		return nil, err
	}

	if g.Status != game.StatusFinished {
		return nil, errors.New("game is not finished")
	}

	_ = s.Repo.UpdateStatus(ctx, gameID, StatusPending, "")

	if s.TaskDistributor != nil {
		task, err := NewGameAnalysisTask(gameID)
		if err != nil {
			return nil, fmt.Errorf("create retry analysis task: %w", err)
		}
		_, err = s.TaskDistributor.EnqueueContext(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("enqueue retry task: %w", err)
		}
	}

	return s.Repo.GetAnalysisByGameID(ctx, gameID)
}
