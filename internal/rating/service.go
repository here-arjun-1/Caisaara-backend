package rating

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/rating/glicko2"
)

const minRating = 100

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) UpdateRatingsAfterGame(ctx context.Context, gameID string) error {
	tx, err := s.Repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	gameInfo, err := s.Repository.ClaimGame(ctx, tx, gameID)
	if err != nil {
		return err
	}
	if gameInfo == nil || gameInfo.WhitePlayerID == gameInfo.BlackPlayerID {
		return nil
	}

	whiteRating, blackRating, err := s.Repository.LockRatings(ctx, tx, gameInfo.Mode, gameInfo.WhitePlayerID, gameInfo.BlackPlayerID)
	if err != nil {
		return err
	}

	whiteScore, blackScore := getScores(gameInfo.Result)

	now := time.Now()
	addInactiveDays(whiteRating, now)
	addInactiveDays(blackRating, now)

	oldWhiteRating := *whiteRating
	oldBlackRating := *blackRating

	updatePlayerRating(whiteRating, oldBlackRating, whiteScore, now)
	updatePlayerRating(blackRating, oldWhiteRating, blackScore, now)

	if err := s.Repository.SaveRating(ctx, tx, whiteRating); err != nil {
		return err
	}
	if err := s.Repository.SaveRating(ctx, tx, blackRating); err != nil {
		return err
	}

	if err := s.Repository.SaveOverallRating(ctx, tx, whiteRating.UserID); err != nil {
		return err
	}
	if err := s.Repository.SaveOverallRating(ctx, tx, blackRating.UserID); err != nil {
		return err
	}

	whiteProv := glicko2.IsProvisional(whiteRating.RatingDeviation)
	blackProv := glicko2.IsProvisional(blackRating.RatingDeviation)
	if err := s.Repository.SaveGameRatings(ctx, tx, gameID, oldWhiteRating.Rating, whiteRating.Rating, whiteProv, oldBlackRating.Rating, blackRating.Rating, blackProv); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) ApplyMissedRatings(ctx context.Context) {
	gameIDs, err := s.Repository.FindGamesWithoutRating(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "find games without rating failed", "error", err)
		return
	}

	for _, gameID := range gameIDs {
		if err := s.UpdateRatingsAfterGame(ctx, gameID); err != nil {
			slog.ErrorContext(ctx, "apply missed rating failed", "game_id", gameID, "error", err)
		}
	}
}

func (s *Service) StartMissedRatingsChecker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.ApplyMissedRatings(ctx)
			}
		}
	}()
}

func getScores(result string) (float64, float64) {
	switch result {
	case game.ResultWhiteWin:
		return 1, 0
	case game.ResultBlackWin:
		return 0, 1
	default:
		return 0.5, 0.5
	}
}

func addInactiveDays(player *PlayerRating, now time.Time) {
	days := int(now.Sub(player.UpdatedAt).Hours() / 24)
	if days <= 0 {
		return
	}

	updated := glicko2.AddInactiveDays(
		glicko2.Player{
			Rating:     float64(player.Rating),
			RD:         player.RatingDeviation,
			Volatility: player.RatingVolatility,
		},
		days,
	)

	player.RatingDeviation = updated.RD
}

func updatePlayerRating(player *PlayerRating, opponent PlayerRating, score float64, now time.Time) {
	newRating := glicko2.Update(
		glicko2.Player{
			Rating:     float64(player.Rating),
			RD:         player.RatingDeviation,
			Volatility: player.RatingVolatility,
		},
		[]glicko2.Result{
			{
				Opponent: glicko2.Player{
					Rating:     float64(opponent.Rating),
					RD:         opponent.RatingDeviation,
					Volatility: opponent.RatingVolatility,
				},
				Score: score,
			},
		},
	)

	player.Rating = int(math.Round(newRating.Rating))
	if player.Rating < minRating {
		player.Rating = minRating
	}
	player.RatingDeviation = newRating.RD
	player.RatingVolatility = newRating.Volatility
	player.GamesPlayed++

	switch score {
	case 1:
		player.Wins++
	case 0:
		player.Losses++
	default:
		player.Draws++
	}

	if player.BestRating == nil || player.Rating > *player.BestRating {
		bestRating := player.Rating
		player.BestRating = &bestRating
		player.BestRatingAt = &now
	}
}
