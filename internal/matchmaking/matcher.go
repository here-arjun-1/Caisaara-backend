package matchmaking

import (
	"context"
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

const (
	matchInterval   = 1 * time.Second
	matchTTL        = 5 * time.Minute
	baseRatingRange = 100
	rangeStep       = 50
	maxRatingRange  = 400
)

type BlockChecker interface {
	IsBlockedEitherWay(ctx context.Context, userID int64, otherID int64) (bool, error)
}

type Matcher struct {
	QueueRepo      QueueRepository
	GameRepository game.GameRepository
	OnMatchFound   func(userID int64, gameID string)
	BlockChecker   BlockChecker
}

func NewMatcher(queueRepo QueueRepository, gameRepository game.GameRepository, blockChecker BlockChecker) *Matcher {
	return &Matcher{
		QueueRepo:      queueRepo,
		GameRepository: gameRepository,
		BlockChecker:   blockChecker,
	}
}

func (m *Matcher) SetOnMatchFound(onMatchFound func(userID int64, gameID string)) {
	m.OnMatchFound = onMatchFound
}

func (m *Matcher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(matchInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.runOnce(ctx)
			}
		}
	}()
}

func (m *Matcher) runOnce(ctx context.Context) {
	for _, tc := range AllowedTimeControls {
		for _, rated := range []bool{false, true} {
			m.matchQueue(ctx, tc.Minutes, tc.Increment, rated)
		}
	}
}

func (m *Matcher) matchQueue(ctx context.Context, timeControlMinutes int, incrementSeconds int, rated bool) {
	entries, err := m.QueueRepo.GetQueue(ctx, timeControlMinutes, incrementSeconds, rated)
	if err != nil {
		slog.ErrorContext(ctx, "get queue failed", "error", err, "time_control", timeControlMinutes, "increment", incrementSeconds, "rated", rated)
		return
	}

	now := time.Now()
	i := 0
	for i+1 < len(entries) {
		a := entries[i]
		b := entries[i+1]

		if !canMatch(a, b, now) {
			i++
			continue
		}

		blocked, err := m.BlockChecker.IsBlockedEitherWay(ctx, a.UserID, b.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "check block for matchmaking failed", "error", err, "user_id", a.UserID, "other_id", b.UserID)
			i++
			continue
		}
		if blocked {
			i++
			continue
		}

		m.pair(ctx, a, b)
		i += 2
	}
}

func canMatch(a *QueueEntry, b *QueueEntry, now time.Time) bool {
	oldest := a.JoinedAt
	if b.JoinedAt.Before(oldest) {
		oldest = b.JoinedAt
	}

	return b.Rating-a.Rating <= ratingRange(oldest, now)
}

func ratingRange(joinedAt time.Time, now time.Time) int {
	waitedSeconds := int(now.Sub(joinedAt).Seconds())
	r := baseRatingRange + (waitedSeconds/10)*rangeStep
	return min(r, maxRatingRange)
}

func (m *Matcher) pair(ctx context.Context, a *QueueEntry, b *QueueEntry) {
	entryA, err := m.QueueRepo.RemoveFromQueue(ctx, a.UserID)
	if err != nil {
		return
	}

	entryB, err := m.QueueRepo.RemoveFromQueue(ctx, b.UserID)
	if err != nil {
		m.putBack(ctx, entryA)
		return
	}

	whitePlayerID, blackPlayerID, err := randomColors(entryA.UserID, entryB.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "generate random color failed", "error", err)
		m.putBack(ctx, entryA)
		m.putBack(ctx, entryB)
		return
	}

	gameID, err := m.GameRepository.CreateGame(
		ctx,
		whitePlayerID,
		blackPlayerID,
		entryA.TimeControlMinutes,
		entryA.IncrementSeconds,
		entryA.Rated,
	)
	if err != nil {
		slog.ErrorContext(ctx, "create game from matchmaking failed", "error", err)
		m.putBack(ctx, entryA)
		m.putBack(ctx, entryB)
		return
	}

	for _, userID := range []int64{entryA.UserID, entryB.UserID} {
		if err := m.QueueRepo.SaveMatch(ctx, userID, gameID, matchTTL); err != nil {
			slog.ErrorContext(ctx, "save match failed", "error", err, "user_id", userID)
		}

		if m.OnMatchFound != nil {
			m.OnMatchFound(userID, gameID)
		}
	}

	slog.InfoContext(ctx, "players matched",
		"game_id", gameID,
		"white_player_id", whitePlayerID,
		"black_player_id", blackPlayerID,
	)
}

func (m *Matcher) putBack(ctx context.Context, entry *QueueEntry) {
	if err := m.QueueRepo.AddToQueue(ctx, entry, queueTTL); err != nil {
		slog.ErrorContext(ctx, "put player back in queue failed", "error", err, "user_id", entry.UserID)
	}
}

func randomColors(playerA int64, playerB int64) (int64, int64, error) {
	random := make([]byte, 1)
	if _, err := rand.Read(random); err != nil {
		return 0, 0, err
	}

	if random[0]%2 == 0 {
		return playerA, playerB, nil
	}
	return playerB, playerA, nil
}
