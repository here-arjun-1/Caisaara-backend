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
	GetMiniMatesLevel(ctx context.Context, levelNumber int) (*LevelWithPuzzle, error)
	GetLevel(ctx context.Context, levelNumber int) (*LevelWithPuzzle, error)
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

	solutionMoves := movesList
	if len(movesList) > 1 {
		solutionMoves = movesList[1:]
	}

	pocketFEN, viewport, _ := minimizeBoard(currentFEN, solutionMoves)

	return &MiniMatesLevelResponse{
		LevelNumber: lp.LevelNumber,
		PuzzleID:    lp.PuzzleID,
		Chapter:     lp.Chapter,
		ChapterName: chapterName,
		TargetMoves: lp.TargetMoves,
		Title:       lp.Title,
		FEN:         pocketFEN,
		InitialFEN:  lp.FEN,
		SetupMove:   setupMove,
		PlayerColor: playerColor,
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

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

func minimizeBoard(fen string, moves []string) (string, Viewport, []string) {
	defaultViewport := Viewport{
		StartFile: "a",
		EndFile:   "h",
		StartRank: 1,
		EndRank:   8,
		Cols:      8,
		Rows:      8,
	}

	parts := strings.Fields(fen)
	var ranks []string
	if len(parts) > 0 {
		ranks = strings.Split(parts[0], "/")
	}
	if len(ranks) != 8 {
		return fen, defaultViewport, moves
	}

	turn := "w"
	if len(parts) > 1 {
		turn = parts[1]
	}

	var grid [8][8]rune
	for rIdx, rStr := range ranks {
		actualRank := 7 - rIdx
		fIdx := 0
		for _, ch := range rStr {
			if ch >= '1' && ch <= '8' {
				for i := 0; i < int(ch-'0'); i++ {
					grid[actualRank][fIdx] = '.'
					fIdx++
				}
			} else {
				grid[actualRank][fIdx] = ch
				fIdx++
			}
		}
	}

	buildFEN := func(g [8][8]rune) string {
		var rankStrings []string
		for r := 7; r >= 0; r-- {
			var sb strings.Builder
			emptyCount := 0
			for f := 0; f < 8; f++ {
				ch := g[r][f]
				if ch == '.' || ch == 0 {
					emptyCount++
				} else {
					if emptyCount > 0 {
						fmt.Fprintf(&sb, "%d", emptyCount)
						emptyCount = 0
					}
					sb.WriteRune(ch)
				}
			}
			if emptyCount > 0 {
				fmt.Fprintf(&sb, "%d", emptyCount)
			}
			rankStrings = append(rankStrings, sb.String())
		}
		res := strings.Join(rankStrings, "/")
		res += " " + turn + " - - 0 1"
		return res
	}

	checkSolution := func(testFEN string, mvs []string) bool {
		fenOpt, err := chess.FEN(testFEN)
		if err != nil {
			return false
		}
		game := chess.NewGame(fenOpt)
		for _, m := range mvs {
			if err := game.PushNotationMove(m, chess.UCINotation{}, nil); err != nil {
				return false
			}
		}
		return game.Method() == chess.Checkmate
	}

	isLocked := func(r, f int) bool {
		ch := grid[r][f]
		if ch == 'k' || ch == 'K' {
			return true
		}
		sqName := fmt.Sprintf("%c%d", 'a'+f, r+1)
		for _, m := range moves {
			if len(m) >= 4 {
				if m[:2] == sqName || m[2:4] == sqName {
					return true
				}
			}
		}
		return false
	}

	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			if grid[r][f] == '.' || isLocked(r, f) {
				continue
			}
			saved := grid[r][f]
			grid[r][f] = '.'
			testFEN := buildFEN(grid)
			if !checkSolution(testFEN, moves) {
				grid[r][f] = saved
			}
		}
	}

	compressedMoves := make([]string, len(moves))
	copy(compressedMoves, moves)

	for i, mv := range compressedMoves {
		if len(mv) < 4 {
			continue
		}
		f1, r1 := int(mv[0]-'a'), int(mv[1]-'1')
		f2, r2 := int(mv[2]-'a'), int(mv[3]-'1')
		p := grid[r1][f1]
		if p == '.' {
			continue
		}
		pLower := strings.ToLower(string(p))
		if pLower != "q" && pLower != "r" && pLower != "b" {
			continue
		}

		df := f2 - f1
		dr := r2 - r1
		stepF := sign(df)
		stepR := sign(dr)

		isDiagonal := stepF != 0 && stepR != 0 && (df*stepF == dr*stepR)
		isStraight := (stepF == 0 && stepR != 0) || (stepF != 0 && stepR == 0)

		if !isDiagonal && !isStraight {
			continue
		}

		dist := df * stepF
		if dist == 0 {
			dist = dr * stepR
		}
		if dist < 0 {
			dist = -dist
		}

		if dist > 2 {
			for targetDist := 2; targetDist < dist; targetDist++ {
				candF := f2 - stepF*targetDist
				candR := r2 - stepR*targetDist
				if candF >= 0 && candF < 8 && candR >= 0 && candR < 8 {
					pathClear := true
					currF := candF + stepF
					currR := candR + stepR
					for currF != f2 || currR != r2 {
						if grid[currR][currF] != '.' {
							pathClear = false
							break
						}
						currF += stepF
						currR += stepR
					}
					if !pathClear {
						continue
					}

					if grid[candR][candF] == '.' {
						grid[r1][f1] = '.'
						grid[candR][candF] = p
						candMove := fmt.Sprintf("%c%d%s", 'a'+candF, candR+1, mv[2:])
						testMoves := make([]string, len(compressedMoves))
						copy(testMoves, compressedMoves)
						testMoves[i] = candMove
						testFEN := buildFEN(grid)
						if checkSolution(testFEN, testMoves) {
							compressedMoves[i] = candMove
							break
						}
						grid[candR][candF] = '.'
						grid[r1][f1] = p
					}
				}
			}
		}
	}

	friendlyKingChar := 'K'
	if turn == "b" {
		friendlyKingChar = 'k'
	}
	fKingF, fKingR := -1, -1
	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			if grid[r][f] == friendlyKingChar {
				fKingF, fKingR = f, r
				break
			}
		}
	}

	kingInMoves := false
	if fKingF >= 0 {
		sq := fmt.Sprintf("%c%d", 'a'+fKingF, fKingR+1)
		for _, m := range compressedMoves {
			if len(m) >= 4 && (m[:2] == sq || m[2:4] == sq) {
				kingInMoves = true
				break
			}
		}
	}

	minF, maxF := 7, 0
	minR, maxR := 7, 0
	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			if grid[r][f] != '.' && (r != fKingR || f != fKingF || kingInMoves) {
				if f < minF {
					minF = f
				}
				if f > maxF {
					maxF = f
				}
				if r < minR {
					minR = r
				}
				if r > maxR {
					maxR = r
				}
			}
		}
	}

	for _, mv := range compressedMoves {
		if len(mv) >= 4 {
			f1, r1 := int(mv[0]-'a'), int(mv[1]-'1')
			f2, r2 := int(mv[2]-'a'), int(mv[3]-'1')
			if f1 < minF {
				minF = f1
			}
			if f1 > maxF {
				maxF = f1
			}
			if r1 < minR {
				minR = r1
			}
			if r1 > maxR {
				maxR = r1
			}
			if f2 < minF {
				minF = f2
			}
			if f2 > maxF {
				maxF = f2
			}
			if r2 < minR {
				minR = r2
			}
			if r2 > maxR {
				maxR = r2
			}
		}
	}

	if !kingInMoves && fKingF >= 0 {
		if fKingF < minF || fKingF > maxF || fKingR < minR || fKingR > maxR {
			placed := false
			for r := minR; r <= maxR; r++ {
				for f := minF; f <= maxF; f++ {
					if grid[r][f] == '.' {
						grid[fKingR][fKingF] = '.'
						grid[r][f] = friendlyKingChar
						testFEN := buildFEN(grid)
						if checkSolution(testFEN, compressedMoves) {
							placed = true
							fKingF, fKingR = f, r
							break
						}
						grid[r][f] = '.'
						grid[fKingR][fKingF] = friendlyKingChar
					}
				}
				if placed {
					break
				}
			}
		}
	}

	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			if grid[r][f] != '.' {
				if f < minF {
					minF = f
				}
				if f > maxF {
					maxF = f
				}
				if r < minR {
					minR = r
				}
				if r > maxR {
					maxR = r
				}
			}
		}
	}

	if minF > maxF {
		minF, maxF = 0, 7
	}
	if minR > maxR {
		minR, maxR = 0, 7
	}

expandFiles:
	for (maxF - minF + 1) < 3 {
		switch {
		case minF > 0:
			minF--
		case maxF < 7:
			maxF++
		default:
			break expandFiles
		}
	}

expandRanks:
	for (maxR - minR + 1) < 3 {
		switch {
		case minR > 0:
			minR--
		case maxR < 7:
			maxR++
		default:
			break expandRanks
		}
	}

	if (maxF-minF+1) > 6 || (maxR-minR+1) > 6 {
		return buildFEN(grid), defaultViewport, compressedMoves
	}

	viewport := Viewport{
		StartFile: string(rune('a' + minF)),
		EndFile:   string(rune('a' + maxF)),
		StartRank: minR + 1,
		EndRank:   maxR + 1,
		Cols:      maxF - minF + 1,
		Rows:      maxR - minR + 1,
	}

	return buildFEN(grid), viewport, compressedMoves
}

func (s *Service) GetHint(ctx context.Context, userID int64, levelNumber int, moveIndex int) (*MiniMatesHintResponse, error) {
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

	playerPly := 1 + moveIndex*2
	if playerPly >= len(movesList) && moveIndex > 0 {
		altPly := 1 + (moveIndex-1)*2
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
	if playerPly < 1 {
		playerPly = 1
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

	currentFEN := board.FEN()
	pocketFEN, _, compressedMoves := minimizeBoard(currentFEN, movesList[playerPly:])
	playerMove := movesList[playerPly]
	if len(compressedMoves) > 0 && len(compressedMoves[0]) >= 4 {
		playerMove = compressedMoves[0]
	}
	fromSquare := playerMove[:2]
	pieceName := getPieceNameFromFEN(pocketFEN, fromSquare)

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

	solutionMoves := movesList[playerPly:]
	pocketFEN, _, compressedMoves := minimizeBoard(board.FEN(), solutionMoves)
	if len(compressedMoves) > 0 && strings.EqualFold(userMove, compressedMoves[0]) {
		expectedUCI = userMove
	}

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
		if pOpt, pErr := chess.FEN(pocketFEN); pErr == nil {
			pBoard := chess.NewGame(pOpt)
			if errP := pBoard.PushNotationMove(userMove, chess.UCINotation{}, nil); errP == nil {
				board = pBoard
				moveErr = nil
			}
		}
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

	isFinalPlayerMove := (playerPly+1 >= len(movesList))
	isAlternateCheckmate := isFinalPlayerMove && board.Method() == chess.Checkmate

	if !strings.EqualFold(playedUCI, expectedUCI) && !isAlternateCheckmate {
		_ = s.repo.RecordFailedAttempt(ctx, userID, levelNumber)
		return &SubmitMiniMatesMoveResponse{
			Correct:     false,
			IsCheckmate: false,
			Message:     "incorrect move",
		}, nil
	}

	if isFinalPlayerMove || isAlternateCheckmate {
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
