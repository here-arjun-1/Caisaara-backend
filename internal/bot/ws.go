package bot

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	Type string `json:"type"`
	Move string `json:"move"`
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	gameID := c.Param("gameID")

	userID, ok := userIDFromToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or missing token",
		})
		return
	}

	game, err := h.Service.GetGame(ctx, gameID, userID)
	if errors.Is(err, ErrGameNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "get bot game for websocket failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	conn.SetReadLimit(1024)

	sendGame(conn, game)
	h.playBotTurn(ctx, conn, game)

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "move":
			game, err = h.Service.PlayMove(ctx, gameID, userID, msg.Move)
		case "resign":
			game, err = h.Service.Resign(ctx, gameID, userID)
		default:
			sendError(conn, "unsupported message type")
			continue
		}

		if err != nil {
			sendServiceError(ctx, conn, err)
			continue
		}

		sendGame(conn, game)
		h.playBotTurn(ctx, conn, game)
	}
}

func (h *Handler) playBotTurn(ctx context.Context, conn *websocket.Conn, game *Game) {
	if game.Status != StatusActive || !isBotTurn(game) {
		return
	}

	_ = conn.WriteJSON(gin.H{"type": "bot_thinking"})

	updatedGame, err := h.Service.PlayBotMove(ctx, game.ID, game.PlayerID)
	if err != nil {
		slog.ErrorContext(ctx, "bot move failed", "game_id", game.ID, "error", err)
		sendError(conn, "bot could not move, reconnect to try again")
		return
	}

	sendGame(conn, updatedGame)
}

func userIDFromToken(tokenString string) (int64, bool) {
	if tokenString == "" {
		return 0, false
	}

	claims, err := token.ValidateToken(tokenString)
	if err != nil || claims.IsGuest || claims.UserID == 0 {
		return 0, false
	}

	return claims.UserID, true
}

func isBotTurn(game *Game) bool {
	whiteToMove := len(game.Moves)%2 == 0
	if game.BotColor() == ColorWhite {
		return whiteToMove
	}
	return !whiteToMove
}

func sendGame(conn *websocket.Conn, game *Game) {
	_ = conn.WriteJSON(gin.H{"type": "game_state", "game": game})
}

func sendError(conn *websocket.Conn, message string) {
	_ = conn.WriteJSON(gin.H{"type": "error", "message": message})
}

func sendServiceError(ctx context.Context, conn *websocket.Conn, err error) {
	switch {
	case errors.Is(err, ErrInvalidMove),
		errors.Is(err, ErrNotYourTurn),
		errors.Is(err, ErrGameFinished),
		errors.Is(err, ErrGameChanged),
		errors.Is(err, ErrGameNotFound):
		sendError(conn, err.Error())
	default:
		slog.ErrorContext(ctx, "bot websocket action failed", "error", err)
		sendError(conn, "internal server error")
	}
}
