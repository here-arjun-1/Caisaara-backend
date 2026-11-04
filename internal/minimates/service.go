package minimates

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/corentings/chess/v2"
)

var (
	ErrLevelLocked = errors.New("level is locked")
	ErrInvalidMove = errors.New("invalid move")
)

type MiniMatesService interface {
	GetOverview(ctx context.Context, userID int64) (*MiniMatesOverviewResponse, error)
	GetLevel(ctx context.Context, userID int64, levelNumber int) (*MiniMatesLevelResponse, error)
	SubmitMove(ctx context.Context, userID int64, levelNumber int, req SubmitMiniMatesMoveRequest) (*SubmitMiniMatesMoveResponse, error)
}

type Service struct {
	repo          *Repository
	streakService any
}

func NewService(repo *Repository, streakService any) *Service {
	return &Service{
		repo:          repo,
		streakService: streakService,
	}
}

func calculateStars(attempts int) int {
	if attempts <= 1 {
		return 3
	}
	if attempts <= 3 {
		return 2
	}
	return 1
}

func (s *Service) recordStreak(ctx context.Context, userID int64) {
	if s.streakService == nil || userID <= 0 {
		return
	}
	val := reflect.ValueOf(s.streakService)
	method := val.MethodByName("RecordActivity")
	if method.IsValid() {
		numIn := method.Type().NumIn()
		if numIn == 3 {
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID), reflect.ValueOf("minimates")})
			return
		} else if numIn == 2 {
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID)})
			return
		}
	}
	method = val.MethodByName("RecordActivityDay")
	if method.IsValid() {
		numIn := method.Type().NumIn()
		if numIn == 2 {
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID)})
			return
		} else if numIn == 3 {
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID), reflect.ValueOf("minimates")})
			return
		}
	}
}

func (s *Service) GetOverview(ctx context.Context, userID int64) (*MiniMatesOverviewResponse, error) {
	return s.repo.GetChapterOverview(ctx, userID)
}

func (s *Service) GetLevel(ctx context.Context, userID int64, levelNumber int) (*MiniMatesLevelResponse, error) {
	progress, err := s.repo.GetUserMiniMatesProgress(ctx, userID)
	if err != nil {
		return nil, err
	}
	if levelNumber > progress.CurrentLevel {
		return nil, ErrLevelLocked
	}

	lp, err := s.repo.GetMiniMatesLevel(ctx, levelNumber)
	if err != nil {
		return nil, err
	}

	movesList := strings.Fields(lp.Moves)
	setupMove := ""
	currentFEN := lp.FEN
	if len(movesList) > 0 {
		setupMove = movesList[0]
		fenOpt, err := chess.FEN(lp.FEN)
		if err == nil {
			g := chess.NewGame(fenOpt)
			if err := g.PushNotationMove(setupMove, chess.UCINotation{}, nil); err == nil {
				currentFEN = g.FEN()
			}
		}
	}

	playerColor := "white"
	fenParts := strings.Fields(currentFEN)
	if len(fenParts) > 1 && fenParts[1] == "b" {
		playerColor = "black"
	}

	chapterNames := map[int]string{
		1: "Novice Mates",
		2: "Tactical Traps",
		3: "Master Sacrifices",
		4: "Grandmaster Net",
	}
	chapterName := chapterNames[lp.Chapter]
	if chapterName == "" {
		chapterName = fmt.Sprintf("Chapter %d", lp.Chapter)
	}

	stars := 0
	completed := false
	history, _ := s.repo.GetUserLevelHistory(ctx, userID, levelNumber)
	if history != nil {
		stars = history.Stars
		completed = history.Completed
	}

	return &MiniMatesLevelResponse{
		LevelNumber: lp.LevelNumber,
		PuzzleID:    lp.PuzzleID,
		Chapter:     lp.Chapter,
		ChapterName: chapterName,
		TargetMoves: lp.TargetMoves,
		Title:       lp.Title,
		FEN:         currentFEN,
		InitialFEN:  lp.FEN,
		SetupMove:   setupMove,
		PlayerColor: playerColor,
		Stars:       stars,
		Completed:   completed,
	}, nil
}

func (s *Service) SubmitMove(ctx context.Context, userID int64, levelNumber int, req SubmitMiniMatesMoveRequest) (*SubmitMiniMatesMoveResponse, error) {
	progress, err := s.repo.GetUserMiniMatesProgress(ctx, userID)
	if err != nil {
		return nil, err
	}
	if levelNumber > progress.CurrentLevel {
		return nil, ErrLevelLocked
	}

	lp, err := s.repo.GetMiniMatesLevel(ctx, levelNumber)
	if err != nil {
		return nil, err
	}

	movesList := strings.Fields(lp.Moves)
	if len(movesList) < 2 {
		return nil, errors.New("invalid puzzle move sequence")
	}

	playerPly := 1 + req.MoveIndex*2
	if playerPly >= len(movesList) && req.MoveIndex > 0 {
		altPly := 1 + (req.MoveIndex-1)*2
		if altPly < len(movesList) {
			playerPly = altPly
		}
	}
	if playerPly >= len(movesList) {
		playerPly = len(movesList) - 1
		if playerPly%2 == 0 {
			playerPly--
		}
	}

	fenOpt, err := chess.FEN(lp.FEN)
	if err != nil {
		return nil, fmt.Errorf("parse fen: %w", err)
	}
	board := chess.NewGame(fenOpt)

	for i := 0; i < playerPly; i++ {
		if err := board.PushNotationMove(movesList[i], chess.UCINotation{}, nil); err != nil {
			return nil, fmt.Errorf("apply prior move %s: %w", movesList[i], err)
		}
	}

	expectedUCI := movesList[playerPly]
	userMove := strings.TrimSpace(req.Move)

	if strings.Contains(userMove, " ") {
		remainingMoves := strings.Join(movesList[playerPly:], " ")
		if strings.EqualFold(userMove, remainingMoves) {
			for i := playerPly; i < len(movesList); i++ {
				_ = board.PushNotationMove(movesList[i], chess.UCINotation{}, nil)
			}
			history, _ := s.repo.GetUserLevelHistory(ctx, userID, levelNumber)
			attempts := 1
			if history != nil && history.Attempts > 0 {
				attempts = history.Attempts
			}
			stars := calculateStars(attempts)
			updatedProgress, err := s.repo.SaveLevelCompletion(ctx, userID, levelNumber, stars, req.TimeTakenMs)
			if err != nil {
				return nil, fmt.Errorf("save level completion: %w", err)
			}
			s.recordStreak(ctx, userID)
			return &SubmitMiniMatesMoveResponse{
				Correct:           true,
				IsCheckmate:       true,
				Stars:             stars,
				TotalStars:        updatedProgress.TotalStars,
				CurrentLevel:      updatedProgress.CurrentLevel,
				NextUnlockedLevel: updatedProgress.CurrentLevel,
				FEN:               board.FEN(),
				Message:           "Checkmate!",
			}, nil
		}
	}

	moveErr := board.PushNotationMove(userMove, chess.UCINotation{}, nil)
	if moveErr != nil {
		moveErr = board.PushNotationMove(userMove, chess.AlgebraicNotation{}, nil)
	}
	if moveErr != nil {
		_ = s.repo.RecordFailedAttempt(ctx, userID, levelNumber)
		return &SubmitMiniMatesMoveResponse{
			Correct:     false,
			IsCheckmate: false,
			Message:     "illegal move",
		}, nil
	}

	lastMove := board.Moves()[len(board.Moves())-1]
	playedUCI := chess.UCINotation{}.Encode(nil, lastMove)

	if !strings.EqualFold(playedUCI, expectedUCI) {
		_ = s.repo.RecordFailedAttempt(ctx, userID, levelNumber)
		return &SubmitMiniMatesMoveResponse{
			Correct:     false,
			IsCheckmate: false,
			Message:     "incorrect move",
		}, nil
	}

	isFinalPlayerMove := (playerPly + 1 >= len(movesList))

	if isFinalPlayerMove {
		history, _ := s.repo.GetUserLevelHistory(ctx, userID, levelNumber)
		attempts := 1
		if history != nil && history.Attempts > 0 {
			attempts = history.Attempts
		}

		stars := calculateStars(attempts)

		updatedProgress, err := s.repo.SaveLevelCompletion(ctx, userID, levelNumber, stars, req.TimeTakenMs)
		if err != nil {
			return nil, fmt.Errorf("save level completion: %w", err)
		}

		s.recordStreak(ctx, userID)

		return &SubmitMiniMatesMoveResponse{
			Correct:           true,
			IsCheckmate:       true,
			Stars:             stars,
			TotalStars:        updatedProgress.TotalStars,
			CurrentLevel:      updatedProgress.CurrentLevel,
			NextUnlockedLevel: updatedProgress.CurrentLevel,
			FEN:               board.FEN(),
			Message:           "Checkmate!",
		}, nil
	}

	nextOpponentMove := movesList[playerPly+1]
	if err := board.PushNotationMove(nextOpponentMove, chess.UCINotation{}, nil); err != nil {
		return nil, fmt.Errorf("apply opponent reply %s: %w", nextOpponentMove, err)
	}

	return &SubmitMiniMatesMoveResponse{
		Correct:     true,
		NextMove:    nextOpponentMove,
		IsCheckmate: false,
		FEN:         board.FEN(),
		Message:     "correct move",
	}, nil
}

