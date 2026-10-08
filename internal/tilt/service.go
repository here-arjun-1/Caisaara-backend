package tilt

import (
	"context"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type Service struct {
	Repository GameRepository
}

func NewService(repository GameRepository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) Check(ctx context.Context, playerID int64) (*Status, error) {
	games, err := s.Repository.GetRecentRatedGames(ctx, playerID, recentGamesLimit)
	if err != nil {
		return nil, err
	}

	status := &Status{}

	if len(games) == 0 {
		return status, nil
	}

	losses := 0
	for _, g := range games {
		if !isLoss(g, playerID) {
			break
		}
		losses++
	}
	if losses >= lossStreakLimit {
		status.Reasons = append(status.Reasons, ReasonLossStreak)
	}

	lastGame := games[0]

	if isLoss(lastGame, playerID) && time.Since(lastGame.EndedAt) < requeueWindow {
		status.Reasons = append(status.Reasons, ReasonQuickQueue)
	}

	if lastGame.TimeControlMode != game.ModeBullet && lastGame.TimeControlMode != game.ModeDaily {
		fast, err := s.playedTooFast(ctx, lastGame, playerID)
		if err != nil {
			return nil, err
		}
		if fast {
			status.Reasons = append(status.Reasons, ReasonFastMoves)
		}
	}

	if len(status.Reasons) > 0 {
		status.Tilted = true
		status.Suggestions = defaultSuggestions
	}

	return status, nil
}

func (s *Service) playedTooFast(ctx context.Context, g RecentGame, playerID int64) (bool, error) {
	times, err := s.Repository.GetMoveTimes(ctx, g.ID)
	if err != nil {
		return false, err
	}

	if len(times) < minMovesToCheck {
		return false, nil
	}

	start := 2
	if g.BlackPlayerID == playerID {
		start = 1
	}

	var total time.Duration
	count := 0
	for i := start; i < len(times); i += 2 {
		total += times[i].Sub(times[i-1])
		count++
	}

	if count == 0 {
		return false, nil
	}

	average := total / time.Duration(count)
	return average < fastMoveLimit, nil
}

func isLoss(g RecentGame, playerID int64) bool {
	if g.WhitePlayerID == playerID {
		return g.Result == game.ResultBlackWin
	}
	return g.Result == game.ResultWhiteWin
}
