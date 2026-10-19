package puzzle

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type PuzzleService interface {
	GetDailyPuzzle(ctx context.Context, userID int64) (*DailyPuzzleResponse, error)
	SubmitAttempt(ctx context.Context, userID int64, req SubmitAttemptRequest) (*SubmitAttemptResponse, error)
	GetHint(ctx context.Context, userID int64, req HintRequest) (*HintResponse, error)
	GetSolution(ctx context.Context, userID int64, req SolutionRequest) (*SolutionResponse, error)
	GetNextPuzzle(ctx context.Context, userID int64, theme string) (*Puzzle, error)
	GetUserStats(ctx context.Context, userID int64) (*PuzzleStatsResponse, error)
}

type Service struct {
	repo PuzzleRepository
}

func NewService(repo PuzzleRepository) PuzzleService {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetDailyPuzzle(ctx context.Context, userID int64) (*DailyPuzzleResponse, error) {
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	ratingProfile, err := s.repo.GetOrCreateUserRating(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	if ratingProfile.LastPlayedDate != "" && ratingProfile.LastPlayedDate != today && ratingProfile.LastPlayedDate != yesterday {
		ratingProfile.CurrentStreak = 0
		_ = s.repo.UpdateUserRating(ctx, ratingProfile)
	}

	session, err := s.repo.GetDailySession(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get daily session: %w", err)
	}

	var puzzleRecord *Puzzle
	if session != nil {
		puzzleRecord, err = s.repo.FindPuzzleByID(ctx, session.PuzzleID)
		if err != nil {
			return nil, fmt.Errorf("find session puzzle: %w", err)
		}
	} else {
		targetRating := ratingProfile.Rating + (ratingProfile.CurrentStreak * 10)
		if targetRating > 2800 {
			targetRating = 2800
		}

		puzzleRecord, err = s.repo.FindMatchingPuzzle(ctx, userID, targetRating-60, targetRating+60, "")
		if err != nil {
			return nil, fmt.Errorf("find daily puzzle: %w", err)
		}

		session = &DailyPuzzleSession{
			UserID:         userID,
			PuzzleID:       puzzleRecord.ID,
			Date:           today,
			AttemptsLeft:   3,
			HintUsed:       false,
			SolutionViewed: false,
			Solved:         false,
			RatingEarned:   0,
		}

		if err := s.repo.CreateDailySession(ctx, session); err != nil {
			return nil, fmt.Errorf("create daily session: %w", err)
		}
	}

	moves := strings.Fields(puzzleRecord.Moves)
	initialMove := ""
	if len(moves) > 0 {
		initialMove = moves[0]
	}

	return &DailyPuzzleResponse{
		PuzzleID:       puzzleRecord.ID,
		FEN:            puzzleRecord.FEN,
		InitialMove:    initialMove,
		Rating:         puzzleRecord.Rating,
		Themes:         puzzleRecord.Themes,
		AttemptsLeft:   session.AttemptsLeft,
		HintAvailable:  !session.HintUsed,
		SolutionViewed: session.SolutionViewed,
		Solved:         session.Solved,
		CurrentStreak:  ratingProfile.CurrentStreak,
		PlayerRating:   ratingProfile.Rating,
	}, nil
}

func (s *Service) SubmitAttempt(ctx context.Context, userID int64, req SubmitAttemptRequest) (*SubmitAttemptResponse, error) {
	today := time.Now().UTC().Format("2006-01-02")

	ratingProfile, err := s.repo.GetOrCreateUserRating(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	session, err := s.repo.GetDailySession(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get daily session: %w", err)
	}

	if session != nil && session.PuzzleID == req.PuzzleID {
		if session.Solved {
			return &SubmitAttemptResponse{
				Correct:          true,
				Solved:           true,
				AttemptsLeft:     session.AttemptsLeft,
				RatingChange:     0,
				UserRatingBefore: ratingProfile.Rating,
				UserRatingAfter:  ratingProfile.Rating,
				CurrentStreak:    ratingProfile.CurrentStreak,
				Message:          "Puzzle already solved for today.",
			}, nil
		}

		if session.AttemptsLeft <= 0 {
			return &SubmitAttemptResponse{
				Correct:          false,
				Solved:           false,
				AttemptsLeft:     0,
				RatingChange:     0,
				UserRatingBefore: ratingProfile.Rating,
				UserRatingAfter:  ratingProfile.Rating,
				CurrentStreak:    ratingProfile.CurrentStreak,
				Message:          "No attempts remaining. You can view the solution.",
			}, nil
		}
	}

	puzzleRecord, err := s.repo.FindPuzzleByID(ctx, req.PuzzleID)
	if err != nil {
		return nil, fmt.Errorf("find puzzle: %w", err)
	}

	moves := strings.Fields(puzzleRecord.Moves)
	if len(moves) < 2 {
		return nil, errors.New("invalid puzzle moves sequence")
	}

	playedMove := strings.TrimSpace(req.Move)
	correctMove := moves[1]
	isCorrect := strings.EqualFold(playedMove, correctMove)

	if !isCorrect && strings.Contains(playedMove, " ") {
		fullExpected := strings.Join(moves[1:], " ")
		isCorrect = strings.EqualFold(playedMove, fullExpected)
	}

	attemptsLeft := 3
	if session != nil {
		attemptsLeft = session.AttemptsLeft
	}

	if !isCorrect {
		if session != nil {
			session.AttemptsLeft--
			attemptsLeft = session.AttemptsLeft
			_ = s.repo.UpdateDailySession(ctx, session)
		}

		if attemptsLeft <= 0 {
			ratingChange := calculateRatingChange(ratingProfile.Rating, puzzleRecord.Rating, false, attemptsLeft)
			beforeRating := ratingProfile.Rating
			ratingProfile.Rating += ratingChange
			if ratingProfile.Rating < 100 {
				ratingProfile.Rating = 100
			}
			ratingProfile.PuzzlesFailed++

			_ = s.repo.UpdateUserRating(ctx, ratingProfile)

			attempt := &UserPuzzleAttempt{
				UserID:           userID,
				PuzzleID:         puzzleRecord.ID,
				Solved:           false,
				UserRatingBefore: beforeRating,
				UserRatingAfter:  ratingProfile.Rating,
				TimeTakenMs:      req.TimeTakenMs,
			}
			_ = s.repo.RecordAttempt(ctx, attempt)

			return &SubmitAttemptResponse{
				Correct:          false,
				Solved:           false,
				AttemptsLeft:     0,
				RatingChange:     ratingChange,
				UserRatingBefore: beforeRating,
				UserRatingAfter:  ratingProfile.Rating,
				CurrentStreak:    ratingProfile.CurrentStreak,
				Message:          "No attempts remaining. You can view the solution.",
			}, nil
		}

		return &SubmitAttemptResponse{
			Correct:          false,
			Solved:           false,
			AttemptsLeft:     attemptsLeft,
			RatingChange:     0,
			UserRatingBefore: ratingProfile.Rating,
			UserRatingAfter:  ratingProfile.Rating,
			CurrentStreak:    ratingProfile.CurrentStreak,
			Message:          fmt.Sprintf("Incorrect move. %d attempts remaining.", attemptsLeft),
		}, nil
	}

	nextMove := ""
	isFullySolved := true
	if len(moves) > 2 && !strings.Contains(playedMove, " ") {
		nextMove = moves[2]
		if len(moves) > 3 {
			isFullySolved = true
		}
	}

	ratingChange := 0
	beforeRating := ratingProfile.Rating

	if session != nil && session.SolutionViewed {
		ratingChange = 0
		session.Solved = true
		session.RatingEarned = 0
		_ = s.repo.UpdateDailySession(ctx, session)

		attempt := &UserPuzzleAttempt{
			UserID:           userID,
			PuzzleID:         puzzleRecord.ID,
			Solved:           true,
			UserRatingBefore: beforeRating,
			UserRatingAfter:  beforeRating,
			TimeTakenMs:      req.TimeTakenMs,
		}
		_ = s.repo.RecordAttempt(ctx, attempt)

		return &SubmitAttemptResponse{
			Correct:          true,
			Solved:           true,
			NextMove:         nextMove,
			AttemptsLeft:     attemptsLeft,
			RatingChange:     0,
			UserRatingBefore: beforeRating,
			UserRatingAfter:  beforeRating,
			CurrentStreak:    ratingProfile.CurrentStreak,
			Message:          "Correct move! However, no rating points are awarded because the solution was viewed.",
		}, nil
	}

	ratingChange = calculateRatingChange(ratingProfile.Rating, puzzleRecord.Rating, true, attemptsLeft)
	ratingProfile.Rating += ratingChange
	if ratingProfile.Rating > ratingProfile.BestRating {
		ratingProfile.BestRating = ratingProfile.Rating
		now := time.Now()
		ratingProfile.BestRatingAt = now
	}
	ratingProfile.PuzzlesSolved++

	if ratingProfile.LastPlayedDate != today {
		ratingProfile.CurrentStreak++
		if ratingProfile.CurrentStreak > ratingProfile.MaxStreak {
			ratingProfile.MaxStreak = ratingProfile.CurrentStreak
		}
		ratingProfile.LastPlayedDate = today
	}

	_ = s.repo.UpdateUserRating(ctx, ratingProfile)

	if session != nil {
		session.Solved = true
		session.RatingEarned = ratingChange
		_ = s.repo.UpdateDailySession(ctx, session)
	}

	attempt := &UserPuzzleAttempt{
		UserID:           userID,
		PuzzleID:         puzzleRecord.ID,
		Solved:           true,
		UserRatingBefore: beforeRating,
		UserRatingAfter:  ratingProfile.Rating,
		TimeTakenMs:      req.TimeTakenMs,
	}
	_ = s.repo.RecordAttempt(ctx, attempt)

	return &SubmitAttemptResponse{
		Correct:          true,
		Solved:           isFullySolved,
		NextMove:         nextMove,
		AttemptsLeft:     attemptsLeft,
		RatingChange:     ratingChange,
		UserRatingBefore: beforeRating,
		UserRatingAfter:  ratingProfile.Rating,
		CurrentStreak:    ratingProfile.CurrentStreak,
		Message:          "Puzzle solved successfully!",
	}, nil
}

func (s *Service) GetHint(ctx context.Context, userID int64, req HintRequest) (*HintResponse, error) {
	today := time.Now().UTC().Format("2006-01-02")

	session, err := s.repo.GetDailySession(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get daily session: %w", err)
	}

	puzzleRecord, err := s.repo.FindPuzzleByID(ctx, req.PuzzleID)
	if err != nil {
		return nil, fmt.Errorf("find puzzle: %w", err)
	}

	moves := strings.Fields(puzzleRecord.Moves)
	if len(moves) < 2 {
		return nil, errors.New("invalid puzzle moves sequence")
	}

	correctMove := moves[1]
	fromSquare := ""
	if len(correctMove) >= 2 {
		fromSquare = correctMove[:2]
	}

	if session != nil && !session.HintUsed {
		session.HintUsed = true
		_ = s.repo.UpdateDailySession(ctx, session)
	}

	return &HintResponse{
		HintType:   "piece",
		FromSquare: fromSquare,
		Message:    fmt.Sprintf("Focus on the piece on %s!", fromSquare),
		HintUsed:   true,
	}, nil
}

func (s *Service) GetSolution(ctx context.Context, userID int64, req SolutionRequest) (*SolutionResponse, error) {
	today := time.Now().UTC().Format("2006-01-02")

	session, err := s.repo.GetDailySession(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get daily session: %w", err)
	}

	puzzleRecord, err := s.repo.FindPuzzleByID(ctx, req.PuzzleID)
	if err != nil {
		return nil, fmt.Errorf("find puzzle: %w", err)
	}

	moves := strings.Fields(puzzleRecord.Moves)
	solutionMoves := []string{}
	if len(moves) > 1 {
		solutionMoves = moves[1:]
	}

	if session != nil {
		session.SolutionViewed = true
		_ = s.repo.UpdateDailySession(ctx, session)
	}

	return &SolutionResponse{
		Moves:          solutionMoves,
		RatingChange:   0,
		SolutionViewed: true,
		Message:        "Solution revealed. No rating increase will be awarded.",
	}, nil
}

func (s *Service) GetNextPuzzle(ctx context.Context, userID int64, theme string) (*Puzzle, error) {
	ratingProfile, err := s.repo.GetOrCreateUserRating(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	minRating := ratingProfile.Rating - 75
	maxRating := ratingProfile.Rating + 75

	return s.repo.FindMatchingPuzzle(ctx, userID, minRating, maxRating, theme)
}

func (s *Service) GetUserStats(ctx context.Context, userID int64) (*PuzzleStatsResponse, error) {
	return s.repo.GetUserStats(ctx, userID)
}

func calculateRatingChange(playerRating, puzzleRating int, solved bool, attemptsLeft int) int {
	diff := float64(puzzleRating - playerRating)
	expected := 1.0 / (1.0 + math.Pow(10, -diff/400.0))

	if solved {
		k := 32.0
		gain := k * (1.0 - expected)
		if attemptsLeft == 2 {
			gain *= 0.8
		} else if attemptsLeft == 1 {
			gain *= 0.6
		}
		if gain < 5 {
			gain = 5
		}
		if gain > 32 {
			gain = 32
		}
		return int(math.Round(gain))
	}

	k := 16.0
	loss := k * expected
	if loss < 5 {
		loss = 5
	}
	if loss > 20 {
		loss = 20
	}
	return -int(math.Round(loss))
}
