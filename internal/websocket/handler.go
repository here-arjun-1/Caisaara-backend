package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type Handler struct {
	Hub         *Hub
	GameService game.GameService
}

func NewHandler(hub *Hub, gameService game.GameService) *Handler {
	return &Handler{
		Hub:         hub,
		GameService: gameService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	gameID := c.Param("gameID")
	accessToken := c.Query("token")

	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "token required",
		})
		return
	}

	claims, err := token.ValidateToken(accessToken)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"websocket token validation failed",
			"game_id", gameID,
			"error", err,
		)

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"websocket upgrade failed",
			"game_id", gameID,
			"error", err,
		)
		return
	}

	client := &Client{
		Conn:   conn,
		UserID: claims.UserID,
		GameID: gameID,
		Send:   make(chan []byte, 256),
	}

	room := h.Hub.GetOrCreateRoom(gameID)
	room.AddClient(client)

	if room.Count() == 2 {
		h.sendGameStart(ctx, room)
	}

	go h.writePump(ctx, client)
	go h.readPump(ctx, client, room)
}

func (h *Handler) writePump(ctx context.Context, client *Client) {
	defer client.Close()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(
			websocket.TextMessage,
			message,
		); err != nil {
			slog.ErrorContext(
				ctx,
				"websocket write failed",
				"user_id", client.UserID,
				"game_id", client.GameID,
				"error", err,
			)
			return
		}
	}
}

func (h *Handler) readPump(
	ctx context.Context,
	client *Client,
	room *Room,
) {
	defer func() {
		room.RemoveClient(client.UserID)
		client.Close()

		if room.Count() == 0 {
			h.Hub.RemoveRoom(room.GameID)
		}
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.WarnContext(ctx, "websocket read error", "user_id", client.UserID, "game_id", client.GameID, "error", err)
			}
			return
		}

		var moveMessage game.MoveMessage

		if err := json.Unmarshal(message, &moveMessage); err != nil {
			sendError(ctx, client, "invalid message format")
			continue
		}

		if moveMessage.Type != "move" {
			sendError(ctx, client, "unsupported message type")
			continue
		}

		currentGame, err := h.GameService.MakeMove(
			ctx,
			client.GameID,
			client.UserID,
			moveMessage.Move,
		)
		if err != nil {
			slog.WarnContext(ctx, "make move failed", "game_id", client.GameID, "user_id", client.UserID, "move", moveMessage.Move, "error", err)
			sendError(ctx, client, err.Error())
			continue
		}

		response := game.GameStateMessage{
			Type:          "game_state",
			GameID:        currentGame.ID,
			Position:      currentGame.Position,
			Status:        currentGame.Status,
			Result:        currentGame.Result,
			InitialTimeMs: currentGame.InitialTimeMs,
			IncrementMs:   currentGame.IncrementMs,
			WhiteTimeMs:   currentGame.WhiteTimeMs,
			BlackTimeMs:   currentGame.BlackTimeMs,
			CurrentTurn:   currentGame.CurrentTurn,
			TurnStartedAt: currentGame.TurnStartedAt,
		}

		data, err := json.Marshal(response)
		if err != nil {
			slog.ErrorContext(ctx, "marshal game state failed", "game_id", client.GameID, "error", err)
			sendError(ctx, client, "failed to create game state")
			continue
		}

		room.Broadcast(data)
	}
}

func (h *Handler) sendGameStart(ctx context.Context, room *Room) {
	g, err := h.GameService.GetGame(ctx, room.GameID)
	if err != nil {
		slog.ErrorContext(ctx, "get game failed for game_start", "game_id", room.GameID, "error", err)
		return
	}

	response := game.GameStateMessage{
		Type:          "game_start",
		GameID:        g.ID,
		Position:      g.Position,
		Status:        g.Status,
		Result:        g.Result,
		InitialTimeMs: g.InitialTimeMs,
		IncrementMs:   g.IncrementMs,
		WhiteTimeMs:   g.WhiteTimeMs,
		BlackTimeMs:   g.BlackTimeMs,
		CurrentTurn:   g.CurrentTurn,
		TurnStartedAt: g.TurnStartedAt,
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"marshal game_start failed",
			"game_id", room.GameID,
			"error", err,
		)
		return
	}

	room.Broadcast(data)
}

func sendError(ctx context.Context, client *Client, message string) {
	response := game.ErrorMessage{
		Type:    "error",
		Message: message,
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(ctx, "marshal error message failed", "user_id", client.UserID, "error", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		slog.WarnContext(ctx, "client send buffer full, dropped error message", "user_id", client.UserID)
	}
}
