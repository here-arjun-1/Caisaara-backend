package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type Handler struct {
	Hub         *Hub
	GameService game.GameService
	ChatService chat.ChatService
}

func NewHandler(hub *Hub, gameService game.GameService, chatService ...chat.ChatService) *Handler {
	h := &Handler{
		Hub:         hub,
		GameService: gameService,
	}
	if len(chatService) > 0 {
		h.ChatService = chatService[0]
	}
	return h
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

	currentGame, err := h.GameService.GetGame(ctx, gameID)
	if err != nil {
		slog.ErrorContext(ctx, "get game failed for websocket connect", "game_id", gameID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "game not found",
		})
		return
	}

	if claims.UserID != currentGame.WhitePlayerID && claims.UserID != currentGame.BlackPlayerID {
		slog.WarnContext(ctx, "unauthorized player websocket connect attempt", "game_id", gameID, "user_id", claims.UserID)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "unauthorized player for this game",
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

	moves, _ := h.GameService.GetMoves(ctx, gameID)

	go h.writePump(ctx, client)
	go h.readPump(ctx, client, room)

	h.sendClientGameState(ctx, client, currentGame, moves, "game_state")

	if room.Count() == 2 {
		h.sendGameStart(ctx, room, currentGame, moves)
	}
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

		var incomingMsg struct {
			Type    string `json:"type"`
			Move    string `json:"move,omitempty"`
			Message string `json:"message,omitempty"`
		}

		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			sendError(ctx, client, "invalid message format")
			continue
		}

		switch incomingMsg.Type {
		case "move":
			currentGame, err := h.GameService.MakeMove(
				ctx,
				client.GameID,
				client.UserID,
				incomingMsg.Move,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			h.broadcastGameState(ctx, room, currentGame)

		case "resign":
			currentGame, err := h.GameService.ResignGame(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			h.broadcastGameState(ctx, room, currentGame)

		case "draw", "offer_draw", "accept_draw":
			currentGame, err := h.GameService.DrawGame(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			h.broadcastGameState(ctx, room, currentGame)

		case "chat", "chat_message":
			if h.ChatService == nil {
				sendError(ctx, client, "chat service not available")
				continue
			}
			chatMsg, err := h.ChatService.SendMessage(
				ctx,
				client.GameID,
				client.UserID,
				incomingMsg.Message,
			)
			if err != nil {
				slog.WarnContext(ctx, "chat message failed", "game_id", client.GameID, "user_id", client.UserID, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			h.broadcastChatMessage(ctx, room, chatMsg)

		default:
			sendError(ctx, client, "unsupported message type")
			continue
		}
	}
}

func (h *Handler) broadcastGameState(ctx context.Context, room *Room, currentGame *game.Game) {
	moves, _ := h.GameService.GetMoves(ctx, currentGame.ID)

	response := game.GameStateMessage{
		Type:            "game_state",
		GameID:          currentGame.ID,
		Position:        currentGame.Position,
		Status:          currentGame.Status,
		Result:          currentGame.Result,
		EndReason:       currentGame.EndReason,
		TimeControlMode: currentGame.TimeControlMode,
		DailyMoveTimeMs: currentGame.DailyMoveTimeMs,
		InitialTimeMs:   currentGame.InitialTimeMs,
		IncrementMs:     currentGame.IncrementMs,
		WhiteTimeMs:     currentGame.WhiteTimeMs,
		BlackTimeMs:     currentGame.BlackTimeMs,
		CurrentTurn:     currentGame.CurrentTurn,
		TurnStartedAt:   currentGame.TurnStartedAt,
		Moves:           moves,
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(ctx, "marshal game state failed", "game_id", currentGame.ID, "error", err)
		return
	}

	room.Broadcast(data)
}

func (h *Handler) broadcastChatMessage(ctx context.Context, room *Room, msg *chat.ChatMessage) {
	wsMsg := chat.ChatWSMessage{
		Type:      "chat",
		ID:        msg.ID,
		GameID:    msg.GameID,
		UserID:    msg.UserID,
		Message:   msg.Message,
		CreatedAt: msg.CreatedAt,
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		slog.ErrorContext(ctx, "marshal chat message failed", "game_id", msg.GameID, "error", err)
		return
	}

	room.Broadcast(data)
}

func (h *Handler) BroadcastChatMessage(gameID string, msg *chat.ChatMessage) {
	if h.Hub == nil {
		return
	}
	h.Hub.Mutex.RLock()
	room, exists := h.Hub.Rooms[gameID]
	h.Hub.Mutex.RUnlock()
	if exists && room != nil {
		h.broadcastChatMessage(context.Background(), room, msg)
	}
}

func (h *Handler) sendClientGameState(
	ctx context.Context,
	client *Client,
	g *game.Game,
	moves []game.GameMove,
	messageType string,
) {
	if moves == nil {
		moves = []game.GameMove{}
	}

	response := game.GameStateMessage{
		Type:            messageType,
		GameID:          g.ID,
		Position:        g.Position,
		Status:          g.Status,
		Result:          g.Result,
		EndReason:       g.EndReason,
		TimeControlMode: g.TimeControlMode,
		DailyMoveTimeMs: g.DailyMoveTimeMs,
		InitialTimeMs:   g.InitialTimeMs,
		IncrementMs:     g.IncrementMs,
		WhiteTimeMs:     g.WhiteTimeMs,
		BlackTimeMs:     g.BlackTimeMs,
		CurrentTurn:     g.CurrentTurn,
		TurnStartedAt:   g.TurnStartedAt,
		Moves:           moves,
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(ctx, "marshal client game state failed", "game_id", g.ID, "error", err)
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}

func (h *Handler) sendGameStart(ctx context.Context, room *Room, g *game.Game, moves []game.GameMove) {
	if g == nil {
		var err error
		g, err = h.GameService.GetGame(ctx, room.GameID)
		if err != nil {
			slog.ErrorContext(ctx, "get game failed for game_start", "game_id", room.GameID, "error", err)
			return
		}
	}
	if moves == nil {
		moves, _ = h.GameService.GetMoves(ctx, room.GameID)
	}
	if moves == nil {
		moves = []game.GameMove{}
	}

	response := game.GameStateMessage{
		Type:            "game_start",
		GameID:          g.ID,
		Position:        g.Position,
		Status:          g.Status,
		Result:          g.Result,
		EndReason:       g.EndReason,
		TimeControlMode: g.TimeControlMode,
		DailyMoveTimeMs: g.DailyMoveTimeMs,
		InitialTimeMs:   g.InitialTimeMs,
		IncrementMs:     g.IncrementMs,
		WhiteTimeMs:     g.WhiteTimeMs,
		BlackTimeMs:     g.BlackTimeMs,
		CurrentTurn:     g.CurrentTurn,
		TurnStartedAt:   g.TurnStartedAt,
		Moves:           moves,
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
