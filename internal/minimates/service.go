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
	GetHint(ctx context.Context, userID int64, levelNumber int, moveIndex int) (*MiniMatesHintResponse, error)
	SubmitMove(ctx context.Context, userID int64, levelNumber int, req SubmitMiniMatesMoveRequest) (*SubmitMiniMatesMoveResponse, error)
}

type MiniMatesRepository interface {
	GetMiniMatesLevel(ctx context.Context, levelNumber int) (*Level, error)
	GetLevel(ctx context.Context, levelNumber int) (*Level, error)
	GetUserMiniMatesProgress(ctx context.Context, userID int64) (*UserProgress, error)
	GetUserProgress(ctx context.Context, userID int64) (*UserProgress, error)
	SaveLevelCompletion(ctx context.Context, userID int64, levelNumber int, stars int, timeTakenMs int) (*UserProgress, error)
	GetUserLevelHistory(ctx context.Context, userID int64, levelNumber int) (*LevelHistory, error)
	MarkHintUsed(ctx context.Context, userID int64, levelNumber int) error
	RecordFailedAttempt(ctx context.Context, userID int64, levelNumber int) error
	GetChapterOverview(ctx context.Context, userID int64) (*MiniMatesOverviewResponse, error)
}

type Service struct {
	repo          MiniMatesRepository
	streakService any
}

func NewService(repo MiniMatesRepository, streakService any) *Service {
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
		switch method.Type().NumIn() {
		case 3:
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID), reflect.ValueOf("minimates")})
			return
		case 2:
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID)})
			return
		}
	}
	method = val.MethodByName("RecordActivityDay")
	if method.IsValid() {
		switch method.Type().NumIn() {
		case 2:
			method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(userID)})
			return
		case 3:
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

	data, ok := GetLevelData(levelNumber)
	if !ok {
		return nil, ErrLevelNotFound
	}

	chapterNames := map[int]string{
		1: "Chapter 1: Novice Mates",
		2: "Chapter 2: Tactical Traps",
		3: "Chapter 3: Master Sacrifices",
		4: "Chapter 4: Grandmaster Net",
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

	viewport := Viewport{
		StartFile: "a",
		EndFile:   fmt.Sprintf("%c", 'a'+data.Cols-1),
		StartRank: 1,
		EndRank:   data.Rows,
		Cols:      data.Cols,
		Rows:      data.Rows,
	}

	puzzleID := ""
	if lp.PuzzleID != nil {
		puzzleID = *lp.PuzzleID
	}

	return &MiniMatesLevelResponse{
		LevelNumber: lp.LevelNumber,
		PuzzleID:    puzzleID,
		Chapter:     lp.Chapter,
		ChapterName: chapterName,
		TargetMoves: lp.TargetMoves,
		Title:       lp.Title,
		FEN:         data.FEN,
		InitialFEN:  data.FEN,
		SetupMove:   "",
		PlayerColor: "white",
		Viewport:    viewport,
		Stars:       stars,
		Completed:   completed,
	}, nil
}

func getPieceNameFromFEN(fen string, square string) string {
	if len(square) < 2 {
		return "Piece"
	}
	parts := strings.Fields(fen)
	if len(parts) == 0 {
		return "Piece"
	}
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return "Piece"
	}
	targetFile := int(square[0] - 'a')
	targetRank := int(square[1] - '1')
	if targetFile < 0 || targetFile > 7 || targetRank < 0 || targetRank > 7 {
		return "Piece"
	}
	rankStr := ranks[7-targetRank]
	col := 0
	var pieceChar rune
	for _, ch := range rankStr {
		if ch >= '1' && ch <= '8' {
			col += int(ch - '0')
		} else {
			if col == targetFile {
				pieceChar = ch
				break
			}
			col++
		}
	}
	switch strings.ToUpper(string(pieceChar)) {
	case "Q":
		return "Queen"
	case "R":
		return "Rook"
	case "B":
		return "Bishop"
	case "N":
		return "Knight"
	case "K":
		return "King"
	case "P":
		return "Pawn"
	default:
		return "Piece"
	}
}

func (s *Service) GetHint(ctx context.Context, userID int64, levelNumber int, moveIndex int) (*MiniMatesHintResponse, error) {
	progress, err := s.repo.GetUserMiniMatesProgress(ctx, userID)
	if err != nil {
		return nil, err
	}
	if levelNumber > progress.CurrentLevel {
		return nil, ErrLevelLocked
	}

	data, ok := GetLevelData(levelNumber)
	if !ok {
		return nil, ErrLevelNotFound
	}

	stepIdx := moveIndex * 2
	if stepIdx < 0 || stepIdx >= len(data.Moves) {
		stepIdx = 0
	}
	playerMove := data.Moves[stepIdx]
	if len(playerMove) < 2 {
		return nil, ErrInvalidMove
	}
	fromSquare := playerMove[:2]

	fenOpt, err := chess.FEN(data.FEN)
	if err != nil {
		return nil, fmt.Errorf("parse fen: %w", err)
	}
	board := chess.NewGame(fenOpt)
	for i := 0; i < stepIdx; i++ {
		if err := board.PushNotationMove(data.Moves[i], chess.UCINotation{}, nil); err != nil {
			return nil, fmt.Errorf("apply prior move %s: %w", data.Moves[i], err)
		}
	}

	pieceName := getPieceNameFromFEN(board.FEN(), fromSquare)
	_ = s.repo.MarkHintUsed(ctx, userID, levelNumber)

	return &MiniMatesHintResponse{
		FromSquare: fromSquare,
		Piece:      pieceName,
		Message:    fmt.Sprintf("Move your %s on %s", pieceName, fromSquare),
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

	data, ok := GetLevelData(levelNumber)
	if !ok {
		return nil, ErrLevelNotFound
	}

	movesList := data.Moves
	if len(movesList) == 0 {
		return nil, errors.New("invalid puzzle move sequence")
	}

	moveIdx := req.MoveIndex
	if req.AltMoveIndex != nil && moveIdx == 0 {
		moveIdx = *req.AltMoveIndex
	}
	if req.Step != nil && moveIdx == 0 {
		if *req.Step >= 1 {
			moveIdx = *req.Step - 1
		}
	}

	userMove := strings.TrimSpace(req.Move)
	playerPly := moveIdx * 2
	if playerPly >= len(movesList) {
		playerPly = len(movesList) - 1
		if playerPly%2 != 0 {
			playerPly--
		}
	}

	fenOpt, err := chess.FEN(data.FEN)
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

	if strings.Contains(userMove, " ") {
		remainingMoves := strings.Join(movesList[playerPly:], " ")
		if strings.EqualFold(userMove, remainingMoves) {
			for i := playerPly; i < len(movesList); i++ {
				_ = board.PushNotationMove(movesList[i], chess.UCINotation{}, nil)
			}
			history, _ := s.repo.GetUserLevelHistory(ctx, userID, levelNumber)
			attempts := 1
			hintUsed := false
			if history != nil {
				if history.Attempts > 0 {
					attempts = history.Attempts + 1
				}
				hintUsed = history.HintUsed
			}
			stars := calculateStars(attempts)
			if hintUsed {
				stars = 1
			}
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
	toSq := lastMove.S2()
	toCol := int(toSq.File())
	toRow := int(toSq.Rank())
	if toCol >= data.Cols || toRow >= data.Rows {
		_ = s.repo.RecordFailedAttempt(ctx, userID, levelNumber)
		return &SubmitMiniMatesMoveResponse{
			Correct:     false,
			IsCheckmate: false,
			Message:     "illegal move",
		}, nil
	}

	playedUCI := chess.UCINotation{}.Encode(nil, lastMove)
	isFinalPlayerMove := (playerPly+1 >= len(movesList))
	isCheckmate := board.Method() == chess.Checkmate

	if !strings.EqualFold(playedUCI, expectedUCI) && !isCheckmate {
		_ = s.repo.RecordFailedAttempt(ctx, userID, levelNumber)
		return &SubmitMiniMatesMoveResponse{
			Correct:     false,
			IsCheckmate: false,
			Message:     "incorrect move",
		}, nil
	}

	if isFinalPlayerMove || isCheckmate {
		history, _ := s.repo.GetUserLevelHistory(ctx, userID, levelNumber)
		attempts := 1
		hintUsed := false
		if history != nil {
			if history.Attempts > 0 {
				attempts = history.Attempts + 1
			}
			hintUsed = history.HintUsed
		}

		stars := calculateStars(attempts)
		if hintUsed {
			stars = 1
		}

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
