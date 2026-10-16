package bot

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

type wsMessage struct {
	Type string `json:"type"`
	Move string `json:"move"`
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	gameID := c.Param("gameID")

	claims, ok := middleware.ClaimsFromToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or missing token",
		})
		return
	}
	userID := claims.UserID

	if h.UserRepo != nil && !claims.IsGuest && claims.UserID > 0 {
		currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, claims.UserID)
		if err != nil || claims.PasswordVersion != currentVersion {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or revoked token",
			})
			return
		}
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

	conn, err := websocket.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	roomKey := "bot:" + gameID
	cl := websocket.NewClient(conn, claims.UserID, roomKey, claims.PasswordVersion, claims.IsGuest)
	defer cl.Close()

	conn.SetReadLimit(websocket.MaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))
		return nil
	})

	if h.Hub != nil {
		room := h.Hub.GetOrCreateRoom(roomKey)
		room.AddClient(cl)
		defer func() {
			room.RemoveClient(cl.UserID, cl)
			if room.Count() == 0 {
				h.Hub.RemoveRoom(roomKey)
			}
		}()
	}

	go cl.WritePump()

	sendGame(cl, game)
	h.playBotTurn(ctx, cl, game)

	limiter := websocket.NewRateLimiter(10, time.Second)

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))

		if !limiter.Allow() {
			sendError(cl, "rate limit exceeded")
			continue
		}

		if h.UserRepo != nil && userID > 0 {
			currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, userID)
			if err != nil || claims.PasswordVersion != currentVersion {
				sendError(cl, "invalid or revoked token")
				return
			}
		}

		switch msg.Type {
		case "move":
			game, err = h.Service.PlayMove(ctx, gameID, userID, msg.Move)
		case "resign":
			game, err = h.Service.Resign(ctx, gameID, userID)
		default:
			sendError(cl, "unsupported message type")
			continue
		}

		if err != nil {
			sendServiceError(ctx, cl, err)
			continue
		}

		sendGame(cl, game)
		h.playBotTurn(ctx, cl, game)
	}
}

func (h *Handler) playBotTurn(ctx context.Context, cl *websocket.Client, game *Game) {
	if game.Status != StatusActive || !isBotTurn(game) {
		return
	}

	cl.SendJSON(gin.H{"type": "bot_thinking"})

	updatedGame, err := h.Service.PlayBotMove(ctx, game.ID, game.PlayerID)
	if err != nil {
		slog.ErrorContext(ctx, "bot move failed", "game_id", game.ID, "error", err)
		sendError(cl, "bot could not move, reconnect to try again")
		return
	}

	sendGame(cl, updatedGame)
}

func isBotTurn(game *Game) bool {
	whiteToMove := len(game.Moves)%2 == 0
	if game.BotColor() == ColorWhite {
		return whiteToMove
	}
	return !whiteToMove
}

func sendGame(cl *websocket.Client, game *Game) {
	cl.SendJSON(gin.H{"type": "game_state", "game": game})
}

func sendError(cl *websocket.Client, message string) {
	cl.SendJSON(gin.H{"type": "error", "message": message})
}

func sendServiceError(ctx context.Context, cl *websocket.Client, err error) {
	switch {
	case errors.Is(err, ErrInvalidMove),
		errors.Is(err, ErrNotYourTurn),
		errors.Is(err, ErrGameFinished),
		errors.Is(err, ErrGameChanged),
		errors.Is(err, ErrGameNotFound):
		sendError(cl, err.Error())
	default:
		slog.ErrorContext(ctx, "bot websocket action failed", "error", err)
		sendError(cl, "internal server error")
	}
}
