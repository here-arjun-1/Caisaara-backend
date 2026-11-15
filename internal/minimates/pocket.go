package minimates

import (
	"fmt"
	"strings"
)

type Piece byte

const (
	Empty       Piece = 0
	WhiteKing   Piece = 'K'
	WhiteQueen  Piece = 'Q'
	WhiteRook   Piece = 'R'
	WhiteBishop Piece = 'B'
	WhiteKnight Piece = 'N'
	BlackKing   Piece = 'k'
	BlackQueen  Piece = 'q'
	BlackRook   Piece = 'r'
	BlackBishop Piece = 'b'
	BlackKnight Piece = 'n'
)

type Color byte

const (
	White Color = 'w'
	Black Color = 'b'
)

type Move struct {
	FromCol int
	FromRow int
	ToCol   int
	ToRow   int
}

func (m Move) UCI() string {
	return fmt.Sprintf("%c%c%c%c", 'a'+m.FromCol, '1'+m.FromRow, 'a'+m.ToCol, '1'+m.ToRow)
}

func ParseMove(s string) (Move, bool) {
	if len(s) < 4 {
		return Move{}, false
	}
	fc := int(s[0] - 'a')
	fr := int(s[1] - '1')
	tc := int(s[2] - 'a')
	tr := int(s[3] - '1')
	return Move{FromCol: fc, FromRow: fr, ToCol: tc, ToRow: tr}, true
}

type Position struct {
	Cols  int
	Rows  int
	Turn  Color
	Board [6][6]Piece
}

func NewPosition(cols, rows int, turn Color) Position {
	return Position{
		Cols: cols,
		Rows: rows,
		Turn: turn,
	}
}

func (p Position) Clone() Position {
	return p
}

func (p Position) InBounds(col, row int) bool {
	return col >= 0 && col < p.Cols && row >= 0 && row < p.Rows
}

func (p Position) PieceAt(col, row int) Piece {
	if !p.InBounds(col, row) {
		return Empty
	}
	return p.Board[row][col]
}

func (p *Position) SetPiece(col, row int, pc Piece) {
	if p.InBounds(col, row) {
		p.Board[row][col] = pc
	}
}

func IsWhitePiece(pc Piece) bool {
	return pc == WhiteKing || pc == WhiteQueen || pc == WhiteRook || pc == WhiteBishop || pc == WhiteKnight
}

func IsBlackPiece(pc Piece) bool {
	return pc == BlackKing || pc == BlackQueen || pc == BlackRook || pc == BlackBishop || pc == BlackKnight
}

func PieceColor(pc Piece) (Color, bool) {
	if IsWhitePiece(pc) {
		return White, true
	}
	if IsBlackPiece(pc) {
		return Black, true
	}
	return ' ', false
}

func (p Position) KingSquare(c Color) (int, int, bool) {
	target := WhiteKing
	if c == Black {
		target = BlackKing
	}
	for r := 0; r < p.Rows; r++ {
		for f := 0; f < p.Cols; f++ {
			if p.Board[r][f] == target {
				return f, r, true
			}
		}
	}
	return -1, -1, false
}

func (p Position) IsSquareAttackedBy(col, row int, attacker Color) bool {
	knightDeltas := [][2]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}
	kPiece := BlackKnight
	if attacker == White {
		kPiece = WhiteKnight
	}
	for _, d := range knightDeltas {
		nc, nr := col+d[0], row+d[1]
		if p.InBounds(nc, nr) && p.Board[nr][nc] == kPiece {
			return true
		}
	}

	kingDeltas := [][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1}, {0, 1},
		{1, -1}, {1, 0}, {1, 1},
	}
	kingTarget := BlackKing
	if attacker == White {
		kingTarget = WhiteKing
	}
	for _, d := range kingDeltas {
		nc, nr := col+d[0], row+d[1]
		if p.InBounds(nc, nr) && p.Board[nr][nc] == kingTarget {
			return true
		}
	}

	orthoDeltas := [][2]int{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}
	for _, d := range orthoDeltas {
		nc, nr := col+d[0], row+d[1]
		for p.InBounds(nc, nr) {
			pc := p.Board[nr][nc]
			if pc != Empty {
				if attacker == White && (pc == WhiteRook || pc == WhiteQueen) {
					return true
				}
				if attacker == Black && (pc == BlackRook || pc == BlackQueen) {
					return true
				}
				break
			}
			nc += d[0]
			nr += d[1]
		}
	}

	diagDeltas := [][2]int{
		{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
	}
	for _, d := range diagDeltas {
		nc, nr := col+d[0], row+d[1]
		for p.InBounds(nc, nr) {
			pc := p.Board[nr][nc]
			if pc != Empty {
				if attacker == White && (pc == WhiteBishop || pc == WhiteQueen) {
					return true
				}
				if attacker == Black && (pc == BlackBishop || pc == BlackQueen) {
					return true
				}
				break
			}
			nc += d[0]
			nr += d[1]
		}
	}

	return false
}

func (p Position) InCheck(c Color) bool {
	kc, kr, ok := p.KingSquare(c)
	if !ok {
		return true
	}
	opponent := White
	if c == White {
		opponent = Black
	}
	return p.IsSquareAttackedBy(kc, kr, opponent)
}

func (p Position) Apply(m Move) Position {
	next := p
	pc := next.Board[m.FromRow][m.FromCol]
	next.Board[m.FromRow][m.FromCol] = Empty
	next.Board[m.ToRow][m.ToCol] = pc
	if next.Turn == White {
		next.Turn = Black
	} else {
		next.Turn = White
	}
	return next
}

func (p Position) PseudoLegalMoves() []Move {
	var moves []Move
	for r := 0; r < p.Rows; r++ {
		for c := 0; c < p.Cols; c++ {
			pc := p.Board[r][c]
			color, ok := PieceColor(pc)
			if !ok || color != p.Turn {
				continue
			}

			switch pc {
			case WhiteKing, BlackKing:
				deltas := [][2]int{
					{-1, -1}, {-1, 0}, {-1, 1},
					{0, -1}, {0, 1},
					{1, -1}, {1, 0}, {1, 1},
				}
				for _, d := range deltas {
					nc, nr := c+d[0], r+d[1]
					if p.InBounds(nc, nr) {
						destColor, destOccupied := PieceColor(p.Board[nr][nc])
						if !destOccupied || destColor != p.Turn {
							moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
						}
					}
				}

			case WhiteKnight, BlackKnight:
				deltas := [][2]int{
					{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
					{1, -2}, {1, 2}, {2, -1}, {2, 1},
				}
				for _, d := range deltas {
					nc, nr := c+d[0], r+d[1]
					if p.InBounds(nc, nr) {
						destColor, destOccupied := PieceColor(p.Board[nr][nc])
						if !destOccupied || destColor != p.Turn {
							moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
						}
					}
				}

			case WhiteRook, BlackRook:
				deltas := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
				for _, d := range deltas {
					nc, nr := c+d[0], r+d[1]
					for p.InBounds(nc, nr) {
						destColor, destOccupied := PieceColor(p.Board[nr][nc])
						if !destOccupied {
							moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
						} else {
							if destColor != p.Turn {
								moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
							}
							break
						}
						nc += d[0]
						nr += d[1]
					}
				}

			case WhiteBishop, BlackBishop:
				deltas := [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
				for _, d := range deltas {
					nc, nr := c+d[0], r+d[1]
					for p.InBounds(nc, nr) {
						destColor, destOccupied := PieceColor(p.Board[nr][nc])
						if !destOccupied {
							moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
						} else {
							if destColor != p.Turn {
								moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
							}
							break
						}
						nc += d[0]
						nr += d[1]
					}
				}

			case WhiteQueen, BlackQueen:
				deltas := [][2]int{
					{-1, 0}, {1, 0}, {0, -1}, {0, 1},
					{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
				}
				for _, d := range deltas {
					nc, nr := c+d[0], r+d[1]
					for p.InBounds(nc, nr) {
						destColor, destOccupied := PieceColor(p.Board[nr][nc])
						if !destOccupied {
							moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
						} else {
							if destColor != p.Turn {
								moves = append(moves, Move{FromCol: c, FromRow: r, ToCol: nc, ToRow: nr})
							}
							break
						}
						nc += d[0]
						nr += d[1]
					}
				}
			}
		}
	}
	return moves
}

func (p Position) LegalMoves() []Move {
	var legal []Move
	for _, m := range p.PseudoLegalMoves() {
		next := p.Apply(m)
		if !next.InCheck(p.Turn) {
			legal = append(legal, m)
		}
	}
	return legal
}

func (p Position) IsCheckmate() bool {
	return p.InCheck(p.Turn) && len(p.LegalMoves()) == 0
}

func (p Position) IsStalemate() bool {
	return !p.InCheck(p.Turn) && len(p.LegalMoves()) == 0
}

func (p Position) ToFEN() string {
	var ranks []string
	for r := 7; r >= 0; r-- {
		if r >= p.Rows {
			ranks = append(ranks, "8")
			continue
		}
		var sb strings.Builder
		emptyCount := 0
		for c := 0; c < 8; c++ {
			if c >= p.Cols {
				emptyCount++
				continue
			}
			pc := p.Board[r][c]
			if pc == Empty {
				emptyCount++
			} else {
				if emptyCount > 0 {
					fmt.Fprintf(&sb, "%d", emptyCount)
					emptyCount = 0
				}
				sb.WriteByte(byte(pc))
			}
		}
		if emptyCount > 0 {
			fmt.Fprintf(&sb, "%d", emptyCount)
		}
		ranks = append(ranks, sb.String())
	}
	return fmt.Sprintf("%s %c - - 0 1", strings.Join(ranks, "/"), p.Turn)
}

func ParseFEN(fen string, cols, rows int) (Position, error) {
	parts := strings.Fields(fen)
	if len(parts) == 0 {
		return Position{}, fmt.Errorf("empty fen")
	}
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return Position{}, fmt.Errorf("invalid ranks count: %d", len(ranks))
	}
	turn := White
	if len(parts) > 1 && parts[1] == "b" {
		turn = Black
	}
	pos := NewPosition(cols, rows, turn)
	for r := 0; r < rows; r++ {
		rankStr := ranks[7-r]
		c := 0
		for i := 0; i < len(rankStr); i++ {
			ch := rankStr[i]
			if ch >= '1' && ch <= '8' {
				c += int(ch - '0')
			} else {
				if c < cols {
					pos.Board[r][c] = Piece(ch)
				}
				c++
			}
		}
	}
	return pos, nil
}

type Solver struct {
	minMoves  map[Position]int
	maxFailed map[Position]int
}

func NewSolver() *Solver {
	return &Solver{
		minMoves:  make(map[Position]int),
		maxFailed: make(map[Position]int),
	}
}

func (s *Solver) MateIn(pos Position, movesLeft int) bool {
	if movesLeft <= 0 {
		return false
	}
	if minM, ok := s.minMoves[pos]; ok {
		if minM <= movesLeft {
			return true
		}
	}
	if maxF, ok := s.maxFailed[pos]; ok {
		if movesLeft <= maxF {
			return false
		}
	}

	legal := pos.LegalMoves()
	if pos.Turn == White {
		if len(legal) == 0 {
			s.maxFailed[pos] = movesLeft
			return false
		}
		for _, m := range legal {
			next := pos.Apply(m)
			if s.blackMated(next, movesLeft) {
				s.minMoves[pos] = movesLeft
				return true
			}
		}
		s.maxFailed[pos] = movesLeft
		return false
	}

	res := s.blackMated(pos, movesLeft)
	if res {
		s.minMoves[pos] = movesLeft
	} else {
		s.maxFailed[pos] = movesLeft
	}
	return res
}

func (s *Solver) blackMated(pos Position, movesLeft int) bool {
	legal := pos.LegalMoves()
	if len(legal) == 0 {
		return pos.InCheck(Black)
	}
	if movesLeft <= 1 {
		return false
	}
	for _, m := range legal {
		next := pos.Apply(m)
		if !s.MateIn(next, movesLeft-1) {
			return false
		}
	}
	return true
}

func (s *Solver) ExactMateIn(pos Position, n int) bool {
	if !s.MateIn(pos, n) {
		return false
	}
	fresh := NewSolver()
	if n > 1 && fresh.MateIn(pos, n-1) {
		return false
	}
	return true
}

func (s *Solver) UniqueLine(pos Position, n int) ([]string, bool) {
	if n <= 0 || pos.Turn != White {
		return nil, false
	}
	legal := pos.LegalMoves()
	var winningMoves []Move
	for _, m := range legal {
		next := pos.Apply(m)
		checkSolver := NewSolver()
		if checkSolver.blackMated(next, n) {
			if n == 1 || !checkSolver.blackMated(next, n-1) {
				winningMoves = append(winningMoves, m)
			}
		}
	}
	if len(winningMoves) != 1 {
		return nil, false
	}

	bestMove := winningMoves[0]
	afterWhite := pos.Apply(bestMove)

	if n == 1 {
		if afterWhite.IsCheckmate() {
			return []string{bestMove.UCI()}, true
		}
		return nil, false
	}

	bLegal := afterWhite.LegalMoves()
	if len(bLegal) == 0 {
		return nil, false
	}

	var bestReply Move
	bestReplyFound := false
	maxDepthResist := -1

	for _, bm := range bLegal {
		afterBlack := afterWhite.Apply(bm)
		subSolver := NewSolver()
		if subSolver.MateIn(afterBlack, n-1) {
			needed := n - 1
			if n-1 > 1 && subSolver.MateIn(afterBlack, n-2) {
				needed = n - 2
			}
			if needed > maxDepthResist {
				maxDepthResist = needed
				bestReply = bm
				bestReplyFound = true
			}
		}
	}

	if !bestReplyFound || maxDepthResist < n-1 {
		return nil, false
	}

	subLine, ok := s.UniqueLine(afterWhite.Apply(bestReply), n-1)
	if !ok {
		return nil, false
	}

	line := append([]string{bestMove.UCI(), bestReply.UCI()}, subLine...)
	return line, true
}

func (s *Solver) IsMinimal(pos Position, n int) bool {
	for r := 0; r < pos.Rows; r++ {
		for c := 0; c < pos.Cols; c++ {
			pc := pos.Board[r][c]
			if pc == Empty || pc == WhiteKing || pc == BlackKing {
				continue
			}
			reduced := pos
			reduced.Board[r][c] = Empty
			reducedSolver := NewSolver()
			if reducedSolver.MateIn(reduced, n) {
				return false
			}
		}
	}
	return true
}
