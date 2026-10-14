package bot

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
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

type client struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case <-c.done:
			return
		case data, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *client) sendJSON(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	select {
	case <-c.done:
	case c.send <- data:
	}
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	gameID := c.Param("gameID")

	claims, ok := claimsFromToken(c.Query("token"))
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := &client{
		conn: conn,
		send: make(chan []byte, 64),
		done: make(chan struct{}),
	}
	defer cl.close()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go cl.writePump()

	sendGame(cl, game)
	h.playBotTurn(ctx, cl, game)

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))

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

func (h *Handler) playBotTurn(ctx context.Context, cl *client, game *Game) {
	if game.Status != StatusActive || !isBotTurn(game) {
		return
	}

	cl.sendJSON(gin.H{"type": "bot_thinking"})

	updatedGame, err := h.Service.PlayBotMove(ctx, game.ID, game.PlayerID)
	if err != nil {
		slog.ErrorContext(ctx, "bot move failed", "game_id", game.ID, "error", err)
		sendError(cl, "bot could not move, reconnect to try again")
		return
	}

	sendGame(cl, updatedGame)
}

func claimsFromToken(tokenString string) (*token.AccessTokenClaims, bool) {
	if tokenString == "" {
		return nil, false
	}

	claims, err := token.ValidateToken(tokenString)
	if err != nil || claims.IsGuest || claims.UserID == 0 {
		return nil, false
	}

	return claims, true
}

func isBotTurn(game *Game) bool {
	whiteToMove := len(game.Moves)%2 == 0
	if game.BotColor() == ColorWhite {
		return whiteToMove
	}
	return !whiteToMove
}

func sendGame(cl *client, game *Game) {
	cl.sendJSON(gin.H{"type": "game_state", "game": game})
}

func sendError(cl *client, message string) {
	cl.sendJSON(gin.H{"type": "error", "message": message})
}

func sendServiceError(ctx context.Context, cl *client, err error) {
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
