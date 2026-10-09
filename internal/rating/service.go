package rating

import (
	"context"
	"math"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/rating/glicko2"
)

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

	oldWhiteRating := *whiteRating
	oldBlackRating := *blackRating
	now := time.Now()

	updatePlayerRating(whiteRating, oldBlackRating, whiteScore, now)
	updatePlayerRating(blackRating, oldWhiteRating, blackScore, now)

	if err := s.Repository.SaveRating(ctx, tx, whiteRating); err != nil {
		return err
	}
	if err := s.Repository.SaveRating(ctx, tx, blackRating); err != nil {
		return err
	}

	return tx.Commit(ctx)
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
