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
	GameService *game.Service
}

func NewHandler(hub *Hub, gameService *game.Service) *Handler {
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
		slog.Error(
			"websocket token validation failed",
			"error", err,
		)

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error(
			"websocket upgrade failed",
			"error", err,
		)
		return
	}

	client := &Client{
		Conn:   conn,
		UserID: claims.UserID,
		GameID: gameID,
		Send:   make(chan []byte, 16),
	}

	room := h.Hub.GetOrCreateRoom(gameID)
	room.AddClient(client)

	if room.Count() == 2 {
		sendGameStart(room)
	}

	go h.writePump(client)
	go h.readPump(c.Request.Context(), client, room)
}

func (h *Handler) writePump(client *Client) {
	defer client.Close()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(
			websocket.TextMessage,
			message,
		); err != nil {
			slog.Error(
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
			return
		}

		var moveMessage game.MoveMessage

		if err := json.Unmarshal(message, &moveMessage); err != nil {
			sendError(client, "invalid message")
			continue
		}

		if moveMessage.Type != "move" {
			sendError(client, "unsupported message type")
			continue
		}

		currentGame, err := h.GameService.MakeMove(
			ctx,
			client.GameID,
			client.UserID,
			moveMessage.Move,
		)
		if err != nil {
			sendError(client, err.Error())
			continue
		}

		response := game.GameStateMessage{
			Type:     "game_state",
			GameID:   currentGame.ID,
			Position: currentGame.Position,
			Status:   currentGame.Status,
			Result:   currentGame.Result,
		}

		data, err := json.Marshal(response)
		if err != nil {
			sendError(client, "failed to create game state")
			continue
		}

		room.Broadcast(data)
	}
}

func sendGameStart(room *Room) {
	message := map[string]string{
		"type":    "game_start",
		"game_id": room.GameID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		slog.Error(
			"marshal game_start failed",
			"error", err,
		)
		return
	}

	room.Broadcast(data)
}

func sendError(client *Client, message string) {
	response := game.ErrorMessage{
		Type:    "error",
		Message: message,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}
