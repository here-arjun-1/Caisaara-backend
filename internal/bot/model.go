package bot

import "time"

const (
	ColorWhite = "white"
	ColorBlack = "black"
)

const (
	StatusActive   = "active"
	StatusFinished = "finished"
)

const (
	ResultWhiteWin = "white_win"
	ResultBlackWin = "black_win"
	ResultDraw     = "draw"
)

const (
	EndReasonCheckmate            = "checkmate"
	EndReasonStalemate            = "stalemate"
	EndReasonResignation          = "resignation"
	EndReasonThreefoldRepetition  = "threefold_repetition"
	EndReason50MoveRule           = "50_move_rule"
	EndReason75MoveRule           = "75_move_rule"
	EndReasonInsufficientMaterial = "insufficient_material"
)

type Game struct {
	ID          string     `json:"id"`
	PlayerID    int64      `json:"player_id"`
	PlayerColor string     `json:"player_color"`
	BotLevel    string     `json:"bot_level"`
	BotRating   int        `json:"bot_rating"`
	Position    string     `json:"position"`
	Moves       []string   `json:"moves"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	EndReason   string     `json:"end_reason,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

func (g *Game) BotColor() string {
	if g.PlayerColor == ColorWhite {
		return ColorBlack
	}
	return ColorWhite
}
