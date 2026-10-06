package game

import (
	"errors"

	"github.com/corentings/chess/v2"
)

type ChessGame struct {
	game *chess.Game
}

func NewChessGame() *ChessGame {
	return &ChessGame{
		game: chess.NewGame(),
	}
}

func NewChessGameFromFEN(fen string) (*ChessGame, error) {
	option, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}

	return &ChessGame{
		game: chess.NewGame(option),
	}, nil
}

func (g *ChessGame) MakeMove(move string) error {
	if g.game.Outcome() != chess.NoOutcome {
		return errors.New("game is already finished")
	}

	err := g.game.PushNotationMove(move, chess.UCINotation{}, nil)
	if err != nil {
		err = g.game.PushNotationMove(move, chess.AlgebraicNotation{}, nil)
	}
	if err != nil {
		return errors.New("invalid move")
	}

	return nil
}

func (g *ChessGame) FEN() string {
	return g.game.FEN()
}

func (g *ChessGame) Turn() string {
	return g.game.Position().Turn().String()
}

func (g *ChessGame) Outcome() string {
	return g.game.Outcome().String()
}

func (g *ChessGame) Method() string {
	return g.game.Method().String()
}

func (g *ChessGame) IsFinished() bool {
	return g.game.Outcome() != chess.NoOutcome
}

func (g *ChessGame) MoveCount() int {
	return len(g.game.Moves())
}

func (g *ChessGame) PGN() string {
	return g.game.String()
}

func getGameResultAndEndReason(chessGame *ChessGame) (string, string) {
	result := ResultDraw
	switch chessGame.Outcome() {
	case "1-0":
		result = ResultWhiteWin
	case "0-1":
		result = ResultBlackWin
	default:
		result = ResultDraw
	}

	endReason := EndReasonDrawAgreement
	switch chessGame.Method() {
	case "Checkmate":
		endReason = EndReasonCheckmate
	case "Stalemate":
		endReason = EndReasonStalemate
	case "ThreefoldRepetition", "FivefoldRepetition":
		endReason = EndReasonThreefoldRepetition
	case "FiftyMoveRule":
		endReason = EndReason50MoveRule
	case "SeventyFiveMoveRule":
		endReason = EndReason75MoveRule
	case "InsufficientMaterial":
		endReason = EndReasonInsufficientMaterial
	default:
		if result == ResultDraw {
			endReason = EndReasonDrawAgreement
		}
	}

	return result, endReason
}

