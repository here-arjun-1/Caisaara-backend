package analysis

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalysisRepository interface {
	CreateAnalysis(ctx context.Context, gameID string, engineVersion string, depth int) (*GameAnalysis, error)
	UpdateStatus(ctx context.Context, gameID string, status string, errorMsg string) error
	SaveAnalysisResult(ctx context.Context, analysis *GameAnalysis, moveAnalyses []GameMoveAnalysis) error
	GetAnalysisByGameID(ctx context.Context, gameID string) (*GameAnalysis, error)
	GetMoveAnalyses(ctx context.Context, gameID string) ([]GameMoveAnalysis, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) AnalysisRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateAnalysis(ctx context.Context, gameID string, engineVersion string, depth int) (*GameAnalysis, error) {
	analysisID := uuid.NewString()
	now := time.Now()

	if engineVersion == "" {
		engineVersion = "Stockfish 16"
	}
	if depth <= 0 {
		depth = 12
	}

	analysis := &GameAnalysis{
		ID:            analysisID,
		GameID:        gameID,
		Status:        StatusPending,
		RatingWhite:   100,
		RatingBlack:   100,
		EngineVersion: engineVersion,
		Depth:         depth,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO game_analyses (
			id,
			game_id,
			status,
			engine_version,
			depth,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (game_id) DO UPDATE SET
			status = EXCLUDED.status,
			engine_version = EXCLUDED.engine_version,
			depth = EXCLUDED.depth,
			updated_at = NOW()`,
		analysis.ID,
		analysis.GameID,
		analysis.Status,
		analysis.EngineVersion,
		analysis.Depth,
		analysis.CreatedAt,
		analysis.UpdatedAt,
	)

	if err != nil {
		slog.ErrorContext(ctx, "create analysis record failed", "game_id", gameID, "error", err)
		return nil, err
	}

	return r.GetAnalysisByGameID(ctx, gameID)
}

func (r *Repository) UpdateStatus(ctx context.Context, gameID string, status string, errorMsg string) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE game_analyses
		SET status = $1,
			error_message = $2,
			updated_at = NOW()
		WHERE game_id = $3`,
		status,
		errorMsg,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update analysis status failed", "game_id", gameID, "status", status, "error", err)
		return err
	}
	return nil
}

func (r *Repository) SaveAnalysisResult(ctx context.Context, analysis *GameAnalysis, moveAnalyses []GameMoveAnalysis) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin transaction for save analysis failed", "game_id", analysis.GameID, "error", err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(
		ctx,
		`UPDATE game_analyses
		SET status = $1,
			accuracy_white = $2,
			accuracy_black = $3,
			rating_white = $4,
			rating_black = $5,
			engine_version = $6,
			depth = $7,
			brilliant_white = $8,
			great_white = $9,
			best_white = $10,
			good_white = $11,
			inaccuracies_white = $12,
			mistakes_white = $13,
			misses_white = $14,
			blunders_white = $15,
			brilliant_black = $16,
			great_black = $17,
			best_black = $18,
			good_black = $19,
			inaccuracies_black = $20,
			mistakes_black = $21,
			misses_black = $22,
			blunders_black = $23,
			opening_accuracy_white = $24,
			opening_accuracy_black = $25,
			middlegame_accuracy_white = $26,
			middlegame_accuracy_black = $27,
			endgame_accuracy_white = $28,
			endgame_accuracy_black = $29,
			error_message = '',
			updated_at = NOW()
		WHERE game_id = $30`,
		StatusCompleted,
		analysis.AccuracyWhite,
		analysis.AccuracyBlack,
		analysis.RatingWhite,
		analysis.RatingBlack,
		analysis.EngineVersion,
		analysis.Depth,
		analysis.BrilliantWhite,
		analysis.GreatWhite,
		analysis.BestWhite,
		analysis.GoodWhite,
		analysis.InaccuraciesWhite,
		analysis.MistakesWhite,
		analysis.MissesWhite,
		analysis.BlundersWhite,
		analysis.BrilliantBlack,
		analysis.GreatBlack,
		analysis.BestBlack,
		analysis.GoodBlack,
		analysis.InaccuraciesBlack,
		analysis.MistakesBlack,
		analysis.MissesBlack,
		analysis.BlundersBlack,
		analysis.OpeningAccuracyWhite,
		analysis.OpeningAccuracyBlack,
		analysis.MiddlegameAccuracyWhite,
		analysis.MiddlegameAccuracyBlack,
		analysis.EndgameAccuracyWhite,
		analysis.EndgameAccuracyBlack,
		analysis.GameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update completed analysis failed", "game_id", analysis.GameID, "error", err)
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM game_move_analyses WHERE game_id = $1`, analysis.GameID)
	if err != nil {
		slog.ErrorContext(ctx, "delete old move analyses failed", "game_id", analysis.GameID, "error", err)
		return err
	}

	for _, moveAnalysis := range moveAnalyses {
		mID := moveAnalysis.ID
		if mID == "" {
			mID = uuid.NewString()
		}
		_, err = tx.Exec(
			ctx,
			`INSERT INTO game_move_analyses (
				id,
				analysis_id,
				game_id,
				move_number,
				player_color,
				played_move,
				position_fen,
				eval_before,
				eval_after,
				best_move,
				pv,
				centipawn_loss,
				classification,
				phase,
				explanation,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW())`,
			mID,
			analysis.ID,
			analysis.GameID,
			moveAnalysis.MoveNumber,
			moveAnalysis.PlayerColor,
			moveAnalysis.PlayedMove,
			moveAnalysis.PositionFEN,
			moveAnalysis.EvalBefore,
			moveAnalysis.EvalAfter,
			moveAnalysis.BestMove,
			moveAnalysis.PV,
			moveAnalysis.CentipawnLoss,
			moveAnalysis.Classification,
			moveAnalysis.Phase,
			moveAnalysis.Explanation,
		)
		if err != nil {
			slog.ErrorContext(ctx, "insert move analysis failed", "game_id", analysis.GameID, "move_number", moveAnalysis.MoveNumber, "error", err)
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit analysis transaction failed", "game_id", analysis.GameID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) GetAnalysisByGameID(ctx context.Context, gameID string) (*GameAnalysis, error) {
	var a GameAnalysis
	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			game_id,
			status,
			accuracy_white,
			accuracy_black,
			COALESCE(rating_white, 100),
			COALESCE(rating_black, 100),
			COALESCE(engine_version, 'Stockfish 16'),
			COALESCE(depth, 12),
			brilliant_white,
			great_white,
			best_white,
			good_white,
			inaccuracies_white,
			mistakes_white,
			misses_white,
			blunders_white,
			brilliant_black,
			great_black,
			best_black,
			good_black,
			inaccuracies_black,
			mistakes_black,
			misses_black,
			blunders_black,
			opening_accuracy_white,
			opening_accuracy_black,
			middlegame_accuracy_white,
			middlegame_accuracy_black,
			endgame_accuracy_white,
			endgame_accuracy_black,
			COALESCE(error_message, ''),
			created_at,
			updated_at
		FROM game_analyses
		WHERE game_id = $1`,
		gameID,
	).Scan(
		&a.ID,
		&a.GameID,
		&a.Status,
		&a.AccuracyWhite,
		&a.AccuracyBlack,
		&a.RatingWhite,
		&a.RatingBlack,
		&a.EngineVersion,
		&a.Depth,
		&a.BrilliantWhite,
		&a.GreatWhite,
		&a.BestWhite,
		&a.GoodWhite,
		&a.InaccuraciesWhite,
		&a.MistakesWhite,
		&a.MissesWhite,
		&a.BlundersWhite,
		&a.BrilliantBlack,
		&a.GreatBlack,
		&a.BestBlack,
		&a.GoodBlack,
		&a.InaccuraciesBlack,
		&a.MistakesBlack,
		&a.MissesBlack,
		&a.BlundersBlack,
		&a.OpeningAccuracyWhite,
		&a.OpeningAccuracyBlack,
		&a.MiddlegameAccuracyWhite,
		&a.MiddlegameAccuracyBlack,
		&a.EndgameAccuracyWhite,
		&a.EndgameAccuracyBlack,
		&a.ErrorMessage,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetMoveAnalyses(ctx context.Context, gameID string) ([]GameMoveAnalysis, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			id,
			analysis_id,
			game_id,
			move_number,
			player_color,
			played_move,
			position_fen,
			eval_before,
			eval_after,
			COALESCE(best_move, ''),
			COALESCE(pv, '{}'),
			centipawn_loss,
			classification,
			COALESCE(phase, 'opening'),
			COALESCE(explanation, ''),
			created_at
		FROM game_move_analyses
		WHERE game_id = $1
		ORDER BY move_number ASC`,
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moves []GameMoveAnalysis
	for rows.Next() {
		var m GameMoveAnalysis
		err := rows.Scan(
			&m.ID,
			&m.AnalysisID,
			&m.GameID,
			&m.MoveNumber,
			&m.PlayerColor,
			&m.PlayedMove,
			&m.PositionFEN,
			&m.EvalBefore,
			&m.EvalAfter,
			&m.BestMove,
			&m.PV,
			&m.CentipawnLoss,
			&m.Classification,
			&m.Phase,
			&m.Explanation,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		moves = append(moves, m)
	}

	return moves, nil
}
