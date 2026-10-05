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

func (g *ChessGame) IsFinished() bool {
	return g.game.Outcome() != chess.NoOutcome
}

func (g *ChessGame) MoveCount() int {
	return len(g.game.Moves())
}

func (g *ChessGame) PGN() string {
	return g.game.String()
}
