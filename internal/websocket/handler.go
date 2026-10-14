package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/invitation"
)

type InviteFinder interface {
	GetInviteByGameID(ctx context.Context, gameID string) (*invitation.Invite, error)
}

type PasswordVersionGetter interface {
	GetPasswordVersion(ctx context.Context, userID int64) (int, error)
}

type disconnectTimerEntry struct {
	timer          *time.Timer
	disconnectedAt time.Time
}

type Handler struct {
	Hub              *Hub
	GameService      game.GameService
	ChatService      chat.ChatService
	InviteFinder     InviteFinder
	UserRepo         PasswordVersionGetter
	disconnectTimers sync.Map
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

func (h *Handler) SetInviteFinder(finder InviteFinder) {
	h.InviteFinder = finder
}

func (h *Handler) SetUserRepo(repo PasswordVersionGetter) {
	h.UserRepo = repo
}

func (h *Handler) SetPasswordVersionGetter(getter PasswordVersionGetter) {
	h.UserRepo = getter
}

func (h *Handler) validatePasswordVersion(ctx context.Context, claims *token.AccessTokenClaims) bool {
	if h.UserRepo == nil || claims == nil || claims.IsGuest || claims.UserID <= 0 {
		return true
	}
	currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, claims.UserID)
	if err != nil {
		return false
	}
	return claims.PasswordVersion == currentVersion
}

func (h *Handler) validateActiveClient(ctx context.Context, client *Client) bool {
	if h.UserRepo == nil || client == nil || client.IsGuest || client.UserID <= 0 {
		return true
	}
	currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, client.UserID)
	if err != nil {
		return false
	}
	return client.PasswordVersion == currentVersion
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := context.WithoutCancel(c.Request.Context())
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

	if !h.validatePasswordVersion(ctx, claims) {
		slog.WarnContext(
			ctx,
			"websocket password version mismatch",
			"game_id", gameID,
			"user_id", claims.UserID,
		)

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or revoked token",
		})
		return
	}

	currentGame, err := h.GameService.GetGame(ctx, gameID)
	if err != nil {
		if h.InviteFinder != nil {
			if invite, inviteErr := h.InviteFinder.GetInviteByGameID(ctx, gameID); inviteErr == nil && invite != nil {
				if claims.UserID == invite.CreatorID {
					conn, upgradeErr := upgrader.Upgrade(c.Writer, c.Request, nil)
					if upgradeErr != nil {
						slog.ErrorContext(ctx, "websocket upgrade failed", "game_id", gameID, "error", upgradeErr)
						return
					}

					client := &Client{
						Conn:            conn,
						UserID:          claims.UserID,
						GameID:          gameID,
						PasswordVersion: claims.PasswordVersion,
						IsGuest:         claims.IsGuest,
						Send:            make(chan []byte, 256),
						done:            make(chan struct{}),
					}

					room := h.Hub.GetOrCreateRoom(gameID)
					room.AddClient(client)

					wsCtx := context.WithoutCancel(ctx)
					go h.writePump(wsCtx, client)
					go h.readPump(wsCtx, client, room)

					waitingMsg := map[string]interface{}{
						"type":    "waiting",
						"game_id": gameID,
					}
					if data, mErr := json.Marshal(waitingMsg); mErr == nil {
						client.SafeSend(data)
					}
					return
				}
			}
		}

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
		Conn:            conn,
		UserID:          claims.UserID,
		GameID:          gameID,
		PasswordVersion: claims.PasswordVersion,
		IsGuest:         claims.IsGuest,
		Send:            make(chan []byte, 256),
		done:            make(chan struct{}),
	}

	timerKey := gameID + ":" + strconv.FormatInt(claims.UserID, 10)
	if val, ok := h.disconnectTimers.LoadAndDelete(timerKey); ok {
		if entry, ok := val.(*disconnectTimerEntry); ok && entry.timer != nil {
			entry.timer.Stop()
		}
	}

	room := h.Hub.GetOrCreateRoom(gameID)
	room.AddClient(client)
	h.broadcastPlayerPresence(room, gameID, claims.UserID, true)

	moves, _ := h.GameService.GetMoves(ctx, gameID)

	wsCtx := context.WithoutCancel(ctx)
	go h.writePump(wsCtx, client)
	go h.readPump(wsCtx, client, room)

	h.sendClientGameState(wsCtx, client, currentGame, moves, "game_state")

	if room.Count() == 2 && len(moves) == 0 && room.MarkGameStarted() {
		h.sendGameStart(wsCtx, room, currentGame, moves)
	}
}

func (h *Handler) writePump(ctx context.Context, client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Close()
	}()

	for {
		select {
		case <-client.done:
			return
		case message, ok := <-client.Send:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

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
		case <-ticker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Handler) readPump(
	ctx context.Context,
	client *Client,
	room *Room,
) {
	defer func() {
		room.RemoveClient(client.UserID, client)
		client.Close()

		h.handlePlayerDisconnect(client.GameID, client.UserID, room)

		if room.Count() == 0 {
			h.Hub.RemoveRoom(room.GameID)
		}
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.WarnContext(ctx, "websocket read error", "user_id", client.UserID, "game_id", client.GameID, "error", err)
			}
			return
		}
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))

		if !h.validateActiveClient(ctx, client) {
			slog.WarnContext(ctx, "active websocket password version revoked", "user_id", client.UserID, "game_id", client.GameID)
			sendError(ctx, client, "invalid or revoked token")
			time.Sleep(10 * time.Millisecond)
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

		case "offer_draw", "draw_offer":
			currentGame, accepted, offererID, err := h.GameService.OfferDraw(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			if accepted {
				h.broadcastGameState(ctx, room, currentGame)
			} else {
				h.broadcastDrawOffer(ctx, room, client.GameID, offererID)
			}

		case "accept_draw", "draw_accept":
			currentGame, err := h.GameService.AcceptDraw(
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

		case "decline_draw", "reject_draw", "draw_decline", "draw_reject":
			_, err := h.GameService.DeclineDraw(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			h.broadcastDrawDeclined(ctx, room, client.GameID, client.UserID)

		case "draw":
			currentGame, accepted, offererID, err := h.GameService.OfferDraw(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
				sendError(ctx, client, err.Error())
				continue
			}
			if accepted {
				h.broadcastGameState(ctx, room, currentGame)
			} else {
				h.broadcastDrawOffer(ctx, room, client.GameID, offererID)
			}

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

		case "claim_timeout", "timeout", "claim_abandonment", "claim_disconnect", "claim_win":
			g, err := h.GameService.GetGame(ctx, client.GameID)
			if err != nil {
				sendError(ctx, client, err.Error())
				continue
			}

			opponentID := g.WhitePlayerID
			if client.UserID == g.WhitePlayerID {
				opponentID = g.BlackPlayerID
			}

			timerKey := client.GameID + ":" + strconv.FormatInt(opponentID, 10)
			var claimed bool
			if val, ok := h.disconnectTimers.Load(timerKey); ok {
				if entry, ok := val.(*disconnectTimerEntry); ok {
					if time.Since(entry.disconnectedAt) >= 2*time.Minute {
						if entry.timer != nil {
							entry.timer.Stop()
						}
						h.disconnectTimers.Delete(timerKey)
						currentGame, err := h.GameService.EndGameOnDisconnect(ctx, client.GameID, opponentID)
						if err != nil {
							sendError(ctx, client, err.Error())
							continue
						}
						h.broadcastGameState(ctx, room, currentGame)
						claimed = true
					}
				}
			}

			if !claimed {
				currentGame, err := h.GameService.ClaimTimeout(
					ctx,
					client.GameID,
				)
				if err != nil {
					slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", incomingMsg.Type, "error", err)
					sendError(ctx, client, err.Error())
					continue
				}
				h.broadcastGameState(ctx, room, currentGame)
			}

		default:
			sendError(ctx, client, "unsupported message type")
			continue
		}
	}
}

func (h *Handler) broadcastDrawOffer(ctx context.Context, room *Room, gameID string, offeredBy int64) {
	wsMsg := map[string]interface{}{
		"type":       "draw_offer",
		"game_id":    gameID,
		"offered_by": offeredBy,
	}
	data, err := json.Marshal(wsMsg)
	if err != nil {
		slog.ErrorContext(ctx, "marshal draw offer failed", "game_id", gameID, "error", err)
		return
	}
	room.Broadcast(data)
}

func (h *Handler) broadcastDrawDeclined(ctx context.Context, room *Room, gameID string, declinedBy int64) {
	wsMsg := map[string]interface{}{
		"type":        "draw_declined",
		"game_id":     gameID,
		"declined_by": declinedBy,
	}
	data, err := json.Marshal(wsMsg)
	if err != nil {
		slog.ErrorContext(ctx, "marshal draw declined failed", "game_id", gameID, "error", err)
		return
	}
	room.Broadcast(data)
}

func (h *Handler) broadcastGameState(ctx context.Context, room *Room, g *game.Game) {
	moves, _ := h.GameService.GetMoves(ctx, g.ID)
	whiteTime, blackTime, turnStartedAt := calculateLiveClocks(g)

	if (g.WhiteUsername == "" || g.BlackUsername == "") && h.GameService != nil {
		if fullGame, err := h.GameService.GetGame(ctx, g.ID); err == nil && fullGame != nil {
			g.WhiteUsername = fullGame.WhiteUsername
			g.BlackUsername = fullGame.BlackUsername
		}
	}

	if g.Status == game.StatusFinished && g.Rated && g.WhiteRatingBefore == nil && h.GameService != nil {
		if fullGame, err := h.GameService.GetGame(ctx, g.ID); err == nil && fullGame != nil {
			g.WhiteRatingBefore = fullGame.WhiteRatingBefore
			g.WhiteRatingAfter = fullGame.WhiteRatingAfter
			g.BlackRatingBefore = fullGame.BlackRatingBefore
			g.BlackRatingAfter = fullGame.BlackRatingAfter
			g.WhiteProvisional = fullGame.WhiteProvisional
			g.BlackProvisional = fullGame.BlackProvisional
		}
	}

	var lastMove string
	if len(moves) > 0 {
		lastMove = moves[len(moves)-1].Move
	}

	response := game.GameStateMessage{
		Type:            "game_state",
		GameID:          g.ID,
		WhitePlayerID:   g.WhitePlayerID,
		BlackPlayerID:   g.BlackPlayerID,
		WhiteUsername:   g.WhiteUsername,
		BlackUsername:   g.BlackUsername,
		Position:        g.Position,
		Status:          g.Status,
		Result:          g.Result,
		EndReason:       g.EndReason,
		Rated:           g.Rated,
		TimeControlMode: g.TimeControlMode,
		DailyMoveTimeMs: g.DailyMoveTimeMs,
		InitialTimeMs:   g.InitialTimeMs,
		IncrementMs:     g.IncrementMs,
		WhiteTimeMs:     whiteTime,
		BlackTimeMs:     blackTime,
		CurrentTurn:     g.CurrentTurn,
		TurnStartedAt:   turnStartedAt,
		DrawOfferedBy:   g.DrawOfferedBy,
		LastMove:        lastMove,
		Moves:           moves,
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for _, client := range room.Clients {
		msg := response
		if g.Status == game.StatusFinished && g.Rated && g.WhiteRatingBefore != nil && g.WhiteRatingAfter != nil && g.BlackRatingBefore != nil && g.BlackRatingAfter != nil {
			switch client.UserID {
			case g.WhitePlayerID:
				msg.RatingUpdate = &game.RatingUpdate{
					Mode:        g.TimeControlMode,
					Before:      *g.WhiteRatingBefore,
					After:       *g.WhiteRatingAfter,
					Change:      *g.WhiteRatingAfter - *g.WhiteRatingBefore,
					Provisional: g.WhiteProvisional != nil && *g.WhiteProvisional,
				}
			case g.BlackPlayerID:
				msg.RatingUpdate = &game.RatingUpdate{
					Mode:        g.TimeControlMode,
					Before:      *g.BlackRatingBefore,
					After:       *g.BlackRatingAfter,
					Change:      *g.BlackRatingAfter - *g.BlackRatingBefore,
					Provisional: g.BlackProvisional != nil && *g.BlackProvisional,
				}
			}
		}

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		select {
		case client.Send <- data:
		default:
		}
	}
}

func (h *Handler) broadcastChatMessage(ctx context.Context, room *Room, msg *chat.ChatMessage) {
	username := msg.Username
	if username == "" && h.GameService != nil {
		if g, err := h.GameService.GetGame(ctx, msg.GameID); err == nil && g != nil {
			switch msg.UserID {
			case g.WhitePlayerID:
				username = g.WhiteUsername
			case g.BlackPlayerID:
				username = g.BlackUsername
			}
		}
	}

	wsMsg := chat.ChatWSMessage{
		Type:      "chat",
		ID:        msg.ID,
		GameID:    msg.GameID,
		UserID:    msg.UserID,
		Username:  username,
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

func (h *Handler) BroadcastGameState(g *game.Game) {
	if h.Hub == nil || g == nil {
		return
	}
	h.Hub.Mutex.RLock()
	room, exists := h.Hub.Rooms[g.ID]
	h.Hub.Mutex.RUnlock()
	if exists && room != nil {
		h.broadcastGameState(context.Background(), room, g)
	}
}

func calculateLiveClocks(g *game.Game) (int64, int64, *time.Time) {
	if g == nil {
		return 0, 0, nil
	}
	if g.Status != game.StatusActive || g.TimeControlMode == game.ModeDaily {
		return g.WhiteTimeMs, g.BlackTimeMs, g.TurnStartedAt
	}

	now := time.Now()
	var turnStartedAt time.Time
	switch {
	case g.TurnStartedAt != nil:
		turnStartedAt = *g.TurnStartedAt
	case g.StartedAt != nil:
		turnStartedAt = *g.StartedAt
	default:
		turnStartedAt = now
	}

	elapsedMs := now.Sub(turnStartedAt).Milliseconds()
	if elapsedMs < 0 {
		elapsedMs = 0
	}

	whiteTimeMs := g.WhiteTimeMs
	blackTimeMs := g.BlackTimeMs

	isWhiteTurn := g.CurrentTurn == "w" || g.CurrentTurn == "White"
	if isWhiteTurn {
		whiteTimeMs -= elapsedMs
		if whiteTimeMs < 0 {
			whiteTimeMs = 0
		}
	} else {
		blackTimeMs -= elapsedMs
		if blackTimeMs < 0 {
			blackTimeMs = 0
		}
	}

	return whiteTimeMs, blackTimeMs, &now
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
	whiteTime, blackTime, turnStartedAt := calculateLiveClocks(g)

	if (g.WhiteUsername == "" || g.BlackUsername == "") && h.GameService != nil {
		if fullGame, err := h.GameService.GetGame(ctx, g.ID); err == nil && fullGame != nil {
			g.WhiteUsername = fullGame.WhiteUsername
			g.BlackUsername = fullGame.BlackUsername
		}
	}

	var lastMove string
	if len(moves) > 0 {
		lastMove = moves[len(moves)-1].Move
	}

	response := game.GameStateMessage{
		Type:            messageType,
		GameID:          g.ID,
		WhitePlayerID:   g.WhitePlayerID,
		BlackPlayerID:   g.BlackPlayerID,
		WhiteUsername:   g.WhiteUsername,
		BlackUsername:   g.BlackUsername,
		Position:        g.Position,
		Status:          g.Status,
		Result:          g.Result,
		EndReason:       g.EndReason,
		Rated:           g.Rated,
		TimeControlMode: g.TimeControlMode,
		DailyMoveTimeMs: g.DailyMoveTimeMs,
		InitialTimeMs:   g.InitialTimeMs,
		IncrementMs:     g.IncrementMs,
		WhiteTimeMs:     whiteTime,
		BlackTimeMs:     blackTime,
		CurrentTurn:     g.CurrentTurn,
		TurnStartedAt:   turnStartedAt,
		DrawOfferedBy:   g.DrawOfferedBy,
		LastMove:        lastMove,
		Moves:           moves,
	}

	if g.Status == game.StatusFinished && g.Rated && g.WhiteRatingBefore != nil && g.WhiteRatingAfter != nil && g.BlackRatingBefore != nil && g.BlackRatingAfter != nil {
		switch client.UserID {
		case g.WhitePlayerID:
			response.RatingUpdate = &game.RatingUpdate{
				Mode:        g.TimeControlMode,
				Before:      *g.WhiteRatingBefore,
				After:       *g.WhiteRatingAfter,
				Change:      *g.WhiteRatingAfter - *g.WhiteRatingBefore,
				Provisional: g.WhiteProvisional != nil && *g.WhiteProvisional,
			}
		case g.BlackPlayerID:
			response.RatingUpdate = &game.RatingUpdate{
				Mode:        g.TimeControlMode,
				Before:      *g.BlackRatingBefore,
				After:       *g.BlackRatingAfter,
				Change:      *g.BlackRatingAfter - *g.BlackRatingBefore,
				Provisional: g.BlackProvisional != nil && *g.BlackProvisional,
			}
		}
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.ErrorContext(ctx, "marshal client game state failed", "game_id", g.ID, "error", err)
		return
	}

	if client != nil {
		client.SafeSend(data)
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
	whiteTime, blackTime, turnStartedAt := calculateLiveClocks(g)

	if (g.WhiteUsername == "" || g.BlackUsername == "") && h.GameService != nil {
		if fullGame, err := h.GameService.GetGame(ctx, g.ID); err == nil && fullGame != nil {
			g.WhiteUsername = fullGame.WhiteUsername
			g.BlackUsername = fullGame.BlackUsername
		}
	}

	var lastMove string
	if len(moves) > 0 {
		lastMove = moves[len(moves)-1].Move
	}

	response := game.GameStateMessage{
		Type:            "game_start",
		GameID:          g.ID,
		WhitePlayerID:   g.WhitePlayerID,
		BlackPlayerID:   g.BlackPlayerID,
		WhiteUsername:   g.WhiteUsername,
		BlackUsername:   g.BlackUsername,
		Position:        g.Position,
		Status:          g.Status,
		Result:          g.Result,
		EndReason:       g.EndReason,
		TimeControlMode: g.TimeControlMode,
		DailyMoveTimeMs: g.DailyMoveTimeMs,
		InitialTimeMs:   g.InitialTimeMs,
		IncrementMs:     g.IncrementMs,
		WhiteTimeMs:     whiteTime,
		BlackTimeMs:     blackTime,
		CurrentTurn:     g.CurrentTurn,
		TurnStartedAt:   turnStartedAt,
		DrawOfferedBy:   g.DrawOfferedBy,
		LastMove:        lastMove,
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

	if client != nil && !client.SafeSend(data) {
		slog.WarnContext(ctx, "client send buffer full, dropped error message", "user_id", client.UserID)
	}
}

func (h *Handler) ConnectTournament(c *gin.Context) {
	ctx := c.Request.Context()
	tournamentID := c.Param("tournamentID")
	accessToken := c.Query("token")

	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	claims, err := token.ValidateToken(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	if !h.validatePasswordVersion(ctx, claims) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(ctx, "tournament websocket upgrade failed", "tournament_id", tournamentID, "error", err)
		return
	}

	roomKey := "tournament:" + tournamentID
	room := h.Hub.GetOrCreateRoom(roomKey)

	client := &Client{
		Conn:            conn,
		UserID:          claims.UserID,
		GameID:          roomKey,
		PasswordVersion: claims.PasswordVersion,
		IsGuest:         claims.IsGuest,
		Send:            make(chan []byte, 256),
		done:            make(chan struct{}),
	}

	room.AddClient(client)

	go func() {
		defer func() {
			room.RemoveClient(client.UserID, client)
			if room.Count() == 0 {
				h.Hub.RemoveRoom(roomKey)
			}
			client.Close()
		}()

		conn.SetReadLimit(maxMessageSize)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			if !h.validateActiveClient(ctx, client) {
				break
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			client.Close()
		}()

		for {
			select {
			case <-client.done:
				return
			case msg, ok := <-client.Send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()
}

func (h *Handler) BroadcastTournamentEvent(tournamentID int64, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	roomKey := "tournament:" + strconv.FormatInt(tournamentID, 10)
	h.Hub.BroadcastToRoom(roomKey, data)
}

func (h *Handler) broadcastPlayerPresence(room *Room, gameID string, userID int64, connected bool) {
	if room == nil {
		return
	}
	msgType := "player_disconnected"
	if connected {
		msgType = "player_reconnected"
	}
	wsMsg := map[string]interface{}{
		"type":    msgType,
		"game_id": gameID,
		"user_id": userID,
	}
	data, err := json.Marshal(wsMsg)
	if err != nil {
		return
	}
	room.Broadcast(data)
}

func (h *Handler) handlePlayerDisconnect(gameID string, userID int64, room *Room) {
	if room != nil && room.HasClient(userID) {
		return
	}

	ctx := context.Background()
	g, err := h.GameService.GetGame(ctx, gameID)
	if err != nil || g == nil || g.Status != game.StatusActive || g.TimeControlMode == game.ModeDaily {
		return
	}

	if userID != g.WhitePlayerID && userID != g.BlackPlayerID {
		return
	}

	if room != nil {
		h.broadcastPlayerPresence(room, gameID, userID, false)
	}

	timerKey := gameID + ":" + strconv.FormatInt(userID, 10)
	if val, ok := h.disconnectTimers.LoadAndDelete(timerKey); ok {
		if entry, ok := val.(*disconnectTimerEntry); ok && entry.timer != nil {
			entry.timer.Stop()
		}
	}

	entry := &disconnectTimerEntry{
		disconnectedAt: time.Now(),
	}
	entry.timer = time.AfterFunc(2*time.Minute, func() {
		h.handleDisconnectTimeout(gameID, userID)
	})
	h.disconnectTimers.Store(timerKey, entry)
}

func (h *Handler) handleDisconnectTimeout(gameID string, userID int64) {
	timerKey := gameID + ":" + strconv.FormatInt(userID, 10)
	h.disconnectTimers.Delete(timerKey)

	room := h.Hub.GetRoom(gameID)
	if room != nil && room.HasClient(userID) {
		return
	}

	ctx := context.Background()
	g, err := h.GameService.EndGameOnDisconnect(ctx, gameID, userID)
	if err != nil || g == nil {
		return
	}

	if room != nil {
		h.broadcastGameState(ctx, room, g)
	}
}

func (h *Handler) CheckActiveClients(ctx context.Context) {
	if h.UserRepo == nil || h.Hub == nil {
		return
	}

	h.Hub.Mutex.RLock()
	var clients []*Client
	for _, room := range h.Hub.Rooms {
		room.Mutex.RLock()
		for _, client := range room.Clients {
			if client != nil && client.UserID > 0 && !client.IsGuest {
				clients = append(clients, client)
			}
		}
		room.Mutex.RUnlock()
	}
	h.Hub.Mutex.RUnlock()

	for _, client := range clients {
		if !h.validateActiveClient(ctx, client) {
			sendError(ctx, client, "invalid or revoked token")
			go func(c *Client) {
				time.Sleep(20 * time.Millisecond)
				c.Close()
			}(client)
		}
	}
}

func (h *Handler) StartActiveAuthChecker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.CheckActiveClients(ctx)
			}
		}
	}()
}
