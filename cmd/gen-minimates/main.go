package main

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/corentings/chess/v2"
	"github.com/here-arjun-1/Caisaara-backend/internal/minimates"
)

type LevelDef struct {
	Number      int
	Chapter     int
	TargetMoves int
	Cols        int
	Rows        int
	FEN         string
	Moves       []string
}

func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

func randFloat32() float32 {
	return float32(randIntn(1000)) / 1000.0
}

func main() {
	sizes := [][2]int{
		{3, 4}, {4, 3}, {4, 4}, {5, 3}, {3, 5},
		{5, 4}, {4, 5}, {5, 5}, {6, 4}, {4, 6},
		{6, 5}, {5, 6}, {6, 6},
	}

	targets := []struct {
		chapter     int
		targetMoves int
		count       int
	}{
		{1, 1, 35},
		{2, 2, 50},
		{3, 3, 50},
		{4, 4, 15},
	}

	var allLevels []LevelDef
	currentNum := 1

	for _, tg := range targets {
		fmt.Printf("Generating Chapter %d (Mate in %d): %d levels...\n", tg.chapter, tg.targetMoves, tg.count)
		found := 0
		startCh := time.Now()
		for found < tg.count {
			sz := sizes[randIntn(len(sizes))]
			cols, rows := sz[0], sz[1]

			pos := generateCandidate(cols, rows, tg.targetMoves)
			if !isValidCandidate(pos) {
				continue
			}

			solver := minimates.NewSolver()
			if !solver.ExactMateIn(pos, tg.targetMoves) {
				continue
			}

			line, ok := solver.UniqueLine(pos, tg.targetMoves)
			if !ok || len(line) != tg.targetMoves*2-1 {
				continue
			}

			if !solver.IsMinimal(pos, tg.targetMoves) {
				continue
			}

			fen := pos.ToFEN()
			if !validateInChessV2(fen, line) {
				continue
			}

			allLevels = append(allLevels, LevelDef{
				Number:      currentNum,
				Chapter:     tg.chapter,
				TargetMoves: tg.targetMoves,
				Cols:        cols,
				Rows:        rows,
				FEN:         fen,
				Moves:       line,
			})
			currentNum++
			found++
			fmt.Printf("  [%d/%d] Level %d (Mate in %d, %dx%d) generated in %s\n", found, tg.count, currentNum-1, tg.targetMoves, cols, rows, time.Since(startCh).Round(time.Millisecond))
		}
	}

	writeLevelsData(allLevels)
	writeMigrationSQL(allLevels)
	fmt.Println("All 150 levels generated successfully!")
}

func generateCandidate(cols, rows, targetMoves int) minimates.Position {
	pos := minimates.NewPosition(cols, rows, minimates.White)

	bkc, bkr := randIntn(cols), randIntn(rows)
	pos.SetPiece(bkc, bkr, minimates.BlackKing)

	var wkc, wkr int
	for a := 0; a < 50; a++ {
		c, r := randIntn(cols), randIntn(rows)
		dc := c - bkc
		if dc < 0 {
			dc = -dc
		}
		dr := r - bkr
		if dr < 0 {
			dr = -dr
		}
		if dc > 1 || dr > 1 {
			if targetMoves <= 2 && (dc > 2 || dr > 2) {
				continue
			}
			wkc, wkr = c, r
			break
		}
	}
	pos.SetPiece(wkc, wkr, minimates.WhiteKing)

	pools := [][]minimates.Piece{
		{minimates.WhiteRook},
		{minimates.WhiteQueen},
		{minimates.WhiteRook, minimates.WhiteBishop},
		{minimates.WhiteRook, minimates.WhiteKnight},
		{minimates.WhiteBishop, minimates.WhiteKnight},
		{minimates.WhiteQueen, minimates.WhiteKnight},
	}
	combo := pools[randIntn(len(pools))]
	for _, pc := range combo {
		for a := 0; a < 30; a++ {
			c, r := randIntn(cols), randIntn(rows)
			if pos.PieceAt(c, r) == minimates.Empty {
				pos.SetPiece(c, r, pc)
				break
			}
		}
	}

	if randFloat32() < 0.25 {
		bPieces := []minimates.Piece{minimates.BlackKnight, minimates.BlackBishop, minimates.BlackRook}
		bpc := bPieces[randIntn(len(bPieces))]
		for a := 0; a < 30; a++ {
			c, r := randIntn(cols), randIntn(rows)
			if pos.PieceAt(c, r) == minimates.Empty {
				pos.SetPiece(c, r, bpc)
				break
			}
		}
	}

	return pos
}

func isValidCandidate(pos minimates.Position) bool {
	if pos.InCheck(minimates.White) || pos.InCheck(minimates.Black) {
		return false
	}
	return true
}

func validateInChessV2(fen string, moves []string) bool {
	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return false
	}
	game := chess.NewGame(fenOpt)
	for _, m := range moves {
		if err := game.PushNotationMove(m, chess.UCINotation{}, nil); err != nil {
			return false
		}
	}
	return game.Method() == chess.Checkmate
}

func writeLevelsData(levels []LevelDef) {
	var sb strings.Builder
	sb.WriteString("package minimates\n\n")
	sb.WriteString("type LevelData struct {\n")
	sb.WriteString("\tNumber      int      `json:\"number\"`\n")
	sb.WriteString("\tChapter     int      `json:\"chapter\"`\n")
	sb.WriteString("\tTargetMoves int      `json:\"target_moves\"`\n")
	sb.WriteString("\tCols        int      `json:\"cols\"`\n")
	sb.WriteString("\tRows        int      `json:\"rows\"`\n")
	sb.WriteString("\tFEN         string   `json:\"fen\"`\n")
	sb.WriteString("\tMoves       []string `json:\"moves\"`\n")
	sb.WriteString("}\n\n")
	sb.WriteString("func GetLevelData(levelNumber int) (*LevelData, bool) {\n")
	sb.WriteString("\tif levelNumber <= 0 || levelNumber > len(Levels) {\n")
	sb.WriteString("\t\treturn nil, false\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &Levels[levelNumber-1], true\n")
	sb.WriteString("}\n\n")
	sb.WriteString("var Levels = []LevelData{\n")
	for _, l := range levels {
		movesQuoted := make([]string, len(l.Moves))
		for i, m := range l.Moves {
			movesQuoted[i] = fmt.Sprintf("%q", m)
		}
		sb.WriteString("\t{\n")
		fmt.Fprintf(&sb, "\t\tNumber:      %d,\n", l.Number)
		fmt.Fprintf(&sb, "\t\tChapter:     %d,\n", l.Chapter)
		fmt.Fprintf(&sb, "\t\tTargetMoves: %d,\n", l.TargetMoves)
		fmt.Fprintf(&sb, "\t\tCols:        %d,\n", l.Cols)
		fmt.Fprintf(&sb, "\t\tRows:        %d,\n", l.Rows)
		fmt.Fprintf(&sb, "\t\tFEN:         %q,\n", l.FEN)
		fmt.Fprintf(&sb, "\t\tMoves:       []string{%s},\n", strings.Join(movesQuoted, ", "))
		sb.WriteString("\t},\n")
	}
	sb.WriteString("}\n")

	path := filepath.Join("internal", "minimates", "levels_data.go")
	_ = os.WriteFile(path, []byte(sb.String()), 0600)
}

func writeMigrationSQL(levels []LevelDef) {
	var sb strings.Builder
	sb.WriteString("ALTER TABLE mini_mates_levels DROP CONSTRAINT IF EXISTS mini_mates_levels_puzzle_id_fkey;\n")
	sb.WriteString("ALTER TABLE mini_mates_levels ALTER COLUMN puzzle_id DROP NOT NULL;\n\n")

	sb.WriteString("INSERT INTO mini_mates_levels (level_number, puzzle_id, chapter, target_moves, title)\nVALUES\n")
	for i, l := range levels {
		sep := ",\n"
		if i == len(levels)-1 {
			sep = "\n"
		}
		fmt.Fprintf(&sb, "    (%d, NULL, %d, %d, 'Level %d')%s", l.Number, l.Chapter, l.TargetMoves, l.Number, sep)
	}
	sb.WriteString("ON CONFLICT (level_number) DO UPDATE SET\n")
	sb.WriteString("    puzzle_id = EXCLUDED.puzzle_id,\n")
	sb.WriteString("    chapter = EXCLUDED.chapter,\n")
	sb.WriteString("    target_moves = EXCLUDED.target_moves,\n")
	sb.WriteString("    title = EXCLUDED.title;\n\n")
	sb.WriteString("DELETE FROM mini_mates_levels WHERE level_number > 150;\n")

	pathUp := filepath.Join("migrations", "sql", "000053_pocket_mini_mates.up.sql")
	_ = os.WriteFile(pathUp, []byte(sb.String()), 0600)

	var downSb strings.Builder
	downSb.WriteString("DELETE FROM mini_mates_levels WHERE level_number <= 150;\n")
	pathDown := filepath.Join("migrations", "sql", "000053_pocket_mini_mates.down.sql")
	_ = os.WriteFile(pathDown, []byte(downSb.String()), 0600)
}
