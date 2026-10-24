package websocket

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/here-arjun-1/Caisaara-backend/internal/tournament"
)

type InviteFinder interface {
	GetInviteByGameID(ctx context.Context, gameID string) (*invitation.Invite, error)
}

type WatchInviteService interface {
	GetWatchInvite(ctx context.Context, code string) (*invitation.WatchInvite, error)
	DeleteWatchInvite(ctx context.Context, gameID string) error
}

type PasswordVersionGetter interface {
	GetPasswordVersion(ctx context.Context, userID int64) (int, error)
}

type TournamentFinder interface {
	FindByID(ctx context.Context, id int64) (*tournament.Tournament, error)
}

type disconnectTimerEntry struct {
	timer          *time.Timer
	disconnectedAt time.Time
}

type Handler struct {
	Hub                *Hub
	GameService        game.GameService
	ChatService        chat.ChatService
	InviteFinder       InviteFinder
	WatchInviteService WatchInviteService
	UserRepo           PasswordVersionGetter
	TournamentFinder   TournamentFinder
	disconnectTimers   sync.Map
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

func (h *Handler) SetWatchInviteService(service WatchInviteService) {
	h.WatchInviteService = service
}

func (h *Handler) SetUserRepo(repo PasswordVersionGetter) {
	h.UserRepo = repo
}

func (h *Handler) SetPasswordVersionGetter(getter PasswordVersionGetter) {
	h.UserRepo = getter
}

func (h *Handler) SetTournamentFinder(finder TournamentFinder) {
	h.TournamentFinder = finder
}

func (h *Handler) SetTournamentService(finder TournamentFinder) {
	h.TournamentFinder = finder
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
	writeWait      = WriteWait
	pongWait       = PongWait
	pingPeriod     = PingPeriod
	maxMessageSize = MaxMessageSize

	maxSpectatorsPerGame = 50
)

var upgrader = Upgrader

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

					client := newClientFromClaims(conn, claims, gameID)
					room := h.Hub.GetOrCreateRoom(gameID)
					room.AddClient(client)

					h.startPumps(ctx, client, room)

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

	client := newClientFromClaims(conn, claims, gameID)

	timerKey := gameID + ":" + strconv.FormatInt(claims.UserID, 10)
	if val, ok := h.disconnectTimers.LoadAndDelete(timerKey); ok {
		if entry, ok := val.(*disconnectTimerEntry); ok && entry.timer != nil {
			entry.timer.Stop()
		}
	}

	room := h.Hub.GetOrCreateRoom(gameID)
	room.AddClient(client)
	h.broadcastPlayerPresence(room, gameID, claims.UserID, true)

	moves := room.GetMoves()
	if len(moves) == 0 {
		moves, _ = h.GameService.GetMoves(ctx, gameID)
		room.SetMoves(moves)
	}
	if freshGame, freshErr := h.GameService.GetGame(ctx, gameID); freshErr == nil && freshGame != nil {
		currentGame = freshGame
	}
	room.SetGame(currentGame)

	h.startPumps(ctx, client, room)

	h.sendClientGameState(ctx, client, currentGame, moves, "game_state")

	if room.Count() == 2 && len(moves) == 0 && room.MarkGameStarted() {
		h.sendGameStart(ctx, room, currentGame, moves)
	}
}

func (h *Handler) ConnectSpectator(c *gin.Context) {
	ctx := context.WithoutCancel(c.Request.Context())
	code := c.Param("code")
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

	if claims.IsGuest || claims.UserID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "login required to watch"})
		return
	}

	if h.WatchInviteService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "watching not available"})
		return
	}

	invite, err := h.WatchInviteService.GetWatchInvite(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "watch link not found or expired"})
		return
	}
	gameID := invite.GameID

	currentGame, err := h.GameService.GetGame(ctx, gameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
		return
	}

	if claims.UserID == currentGame.WhitePlayerID || claims.UserID == currentGame.BlackPlayerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you are playing this game"})
		return
	}

	if currentGame.Status != game.StatusActive {
		c.JSON(http.StatusGone, gin.H{"error": "game is already over"})
		return
	}

	room := h.Hub.GetOrCreateRoom(gameID)
	if room.SpectatorCount() >= maxSpectatorsPerGame {
		c.JSON(http.StatusForbidden, gin.H{"error": "too many spectators"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(ctx, "spectator websocket upgrade failed", "game_id", gameID, "error", err)
		if room.IsEmpty() {
			h.Hub.RemoveRoom(gameID)
		}
		return
	}

	client := newClientFromClaims(conn, claims, gameID)
	room.AddSpectator(client)

	go h.writePump(ctx, client)
	go h.spectatorReadPump(ctx, client, room)

	moves := room.GetMoves()
	if len(moves) == 0 {
		moves, _ = h.GameService.GetMoves(ctx, gameID)
		room.SetMoves(moves)
	}
	if room.GetGame() == nil {
		room.SetGame(currentGame)
	}
	h.sendClientGameState(ctx, client, currentGame, moves, "game_state")

	h.broadcastSpectatorCount(room)
}

func (h *Handler) spectatorReadPump(ctx context.Context, client *Client, room *Room) {
	defer func() {
		room.RemoveSpectator(client.UserID, client)
		client.Close()

		h.broadcastSpectatorCount(room)

		if room.IsEmpty() {
			h.Hub.RemoveRoom(room.GameID)
		}
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	limiter := newWSRateLimiter(5, time.Second)

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))

		if !limiter.allow() {
			sendError(ctx, client, "rate limit exceeded")
			continue
		}

		var incomingMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			sendError(ctx, client, "invalid message format")
			continue
		}

		if incomingMsg.Type == "leave" {
			return
		}
		sendError(ctx, client, "spectators can only watch")
	}
}

func (h *Handler) broadcastSpectatorCount(room *Room) {
	wsMsg := map[string]interface{}{
		"type":    "spectator_count",
		"game_id": room.GameID,
		"count":   room.SpectatorCount(),
	}
	data, err := json.Marshal(wsMsg)
	if err != nil {
		return
	}
	room.Broadcast(data)
}

func (h *Handler) endWatching(ctx context.Context, gameID string, spectators []*Client) {
	if h.WatchInviteService != nil {
		deleteCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_ = h.WatchInviteService.DeleteWatchInvite(deleteCtx, gameID)
		cancel()
	}

	time.Sleep(10 * time.Second)

	for _, client := range spectators {
		client.Close()
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

type wsRateLimiter = RateLimiter

func newWSRateLimiter(limit float64, window time.Duration) *wsRateLimiter {
	return NewRateLimiter(limit, window)
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

		if room.IsEmpty() {
			h.Hub.RemoveRoom(room.GameID)
		}
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	limiter := newWSRateLimiter(10, time.Second)

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.WarnContext(ctx, "websocket read error", "user_id", client.UserID, "game_id", client.GameID, "error", err)
			}
			return
		}
		_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))

		if !limiter.allow() {
			slog.WarnContext(ctx, "websocket rate limit exceeded", "user_id", client.UserID, "game_id", client.GameID)
			sendError(ctx, client, "rate limit exceeded")
			continue
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
			var cachedGame *game.Game
			var cachedMoves []game.GameMove
			if room != nil {
				cachedGame = room.GetGame()
				cachedMoves = room.GetMoves()
			}
			currentGame, err := h.GameService.MakeMove(
				ctx,
				client.GameID,
				client.UserID,
				incomingMsg.Move,
				cachedGame,
				cachedMoves,
			)
			if err != nil {
				h.handleActionError(ctx, client, incomingMsg.Type, err)
				continue
			}
			if room != nil {
				newMove := game.GameMove{
					GameID:        client.GameID,
					PlayerID:      strconv.FormatInt(client.UserID, 10),
					MoveNumber:    room.MoveCount() + 1,
					Move:          incomingMsg.Move,
					PositionAfter: currentGame.Position,
					CreatedAt:     time.Now(),
				}
				room.AddMove(newMove)
				room.SetGame(currentGame)
			}
			h.broadcastGameState(ctx, room, currentGame)

		case "resign":
			currentGame, err := h.GameService.ResignGame(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				h.handleActionError(ctx, client, incomingMsg.Type, err)
				continue
			}
			if room != nil {
				room.SetGame(currentGame)
			}

		case "offer_draw", "draw_offer", "draw":
			currentGame, accepted, offererID, err := h.GameService.OfferDraw(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				h.handleActionError(ctx, client, incomingMsg.Type, err)
				continue
			}
			if accepted {
				if room != nil {
					room.SetGame(currentGame)
				}
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
				h.handleActionError(ctx, client, incomingMsg.Type, err)
				continue
			}
			if room != nil {
				room.SetGame(currentGame)
			}

		case "decline_draw", "reject_draw", "draw_decline", "draw_reject":
			_, err := h.GameService.DeclineDraw(
				ctx,
				client.GameID,
				client.UserID,
			)
			if err != nil {
				h.handleActionError(ctx, client, incomingMsg.Type, err)
				continue
			}
			h.broadcastDrawDeclined(ctx, room, client.GameID, client.UserID)

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
						if room != nil {
							room.SetGame(currentGame)
						}
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
					h.handleActionError(ctx, client, incomingMsg.Type, err)
					continue
				}
				if room != nil {
					room.SetGame(currentGame)
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

func (h *Handler) broadcastGameState(ctx context.Context, room *Room, g *game.Game, moves ...[]game.GameMove) {
	var m []game.GameMove
	if len(moves) > 0 && moves[0] != nil {
		m = moves[0]
	} else if room != nil {
		m = room.GetMoves()
	}

	baseMsg := h.buildGameStateMessage(ctx, g, m, "game_state")

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for _, client := range room.Clients {
		msg := baseMsg
		attachRatingUpdate(&msg, g, client.UserID)

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		select {
		case client.Send <- data:
		default:
		}
	}

	spectatorData, err := json.Marshal(baseMsg)
	if err != nil {
		return
	}
	for _, client := range room.Spectators {
		if client != nil {
			client.SafeSend(spectatorData)
		}
	}

	if g.Status != game.StatusFinished {
		return
	}

	endedData, err := json.Marshal(map[string]interface{}{
		"type":    "watch_ended",
		"game_id": g.ID,
	})
	if err != nil {
		return
	}

	var spectators []*Client
	for _, client := range room.Spectators {
		if client != nil {
			client.SafeSend(endedData)
			spectators = append(spectators, client)
		}
	}

	go h.endWatching(context.WithoutCancel(ctx), g.ID, spectators)
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

	room.BroadcastToPlayers(data)
}

func (h *Handler) BroadcastChatMessage(gameID string, msg *chat.ChatMessage) {
	if h.Hub == nil {
		return
	}
	h.Hub.Mutex.RLock()
	room, exists := h.Hub.Rooms[gameID]
	h.Hub.Mutex.RUnlock()
	if exists && room != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.broadcastChatMessage(ctx, room, msg)
	}
}

func (h *Handler) BroadcastDirectMessage(receiverID int64, msg *chat.DirectMessage) {
	if h.Hub == nil || msg == nil {
		return
	}
	roomKey := "dm:" + strconv.FormatInt(receiverID, 10)
	payload := gin.H{
		"type":    "direct_message",
		"message": msg,
	}
	_ = h.Hub.BroadcastJSON(roomKey, payload)

	senderRoomKey := "dm:" + strconv.FormatInt(msg.SenderID, 10)
	if senderRoomKey != roomKey {
		_ = h.Hub.BroadcastJSON(senderRoomKey, payload)
	}
}

func (h *Handler) ConnectDirectMessages(c *gin.Context) {
	ctx := context.WithoutCancel(c.Request.Context())
	accessToken := c.Query("token")

	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "token required",
		})
		return
	}

	claims, err := token.ValidateToken(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	if claims.IsGuest || claims.UserID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "registered user required for messaging",
		})
		return
	}

	if !h.validatePasswordVersion(ctx, claims) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or revoked token",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(ctx, "direct messages websocket upgrade failed", "user_id", claims.UserID, "error", err)
		return
	}

	roomKey := "dm:" + strconv.FormatInt(claims.UserID, 10)
	client := newClientFromClaims(conn, claims, roomKey)

	room := h.Hub.GetOrCreateRoom(roomKey)
	room.AddClient(client)

	defer func() {
		room.RemoveClient(client.UserID, client)
		client.Close()
		if room.Count() == 0 {
			h.Hub.RemoveRoom(roomKey)
		}
	}()

	go h.writePump(ctx, client)

	_ = client.SendJSON(gin.H{
		"type":    "connected",
		"user_id": claims.UserID,
	})

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
		room.SetGame(g)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.broadcastGameState(ctx, room, g)
	}
}

func calculateLiveClocks(g *game.Game) (int64, int64, *time.Time) {
	if g == nil {
		return 0, 0, nil
	}
	if g.Status != game.StatusActive {
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
	if g.TimeControlMode == game.ModeDaily {
		dailyLimitMs := g.DailyMoveTimeMs
		if dailyLimitMs <= 0 {
			dailyLimitMs = 86400000
		}
		remaining := dailyLimitMs - elapsedMs
		if remaining < 0 {
			remaining = 0
		}
		if isWhiteTurn {
			return remaining, dailyLimitMs, &now
		}
		return dailyLimitMs, remaining, &now
	}

	if !game.ClockStarted(g) {
		return whiteTimeMs, blackTimeMs, &now
	}

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
	response := h.buildGameStateMessage(ctx, g, moves, messageType)
	if client != nil {
		attachRatingUpdate(&response, g, client.UserID)
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

	response := h.buildGameStateMessage(ctx, g, moves, "game_start")

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
	ctx := context.WithoutCancel(c.Request.Context())
	tournamentID := c.Param("tournamentID")
	id, err := strconv.ParseInt(tournamentID, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament id"})
		return
	}

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

	if h.TournamentFinder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tournament service unavailable"})
		return
	}

	_, err = h.TournamentFinder.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, tournament.ErrTournamentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tournament not found"})
			return
		}
		slog.ErrorContext(ctx, "tournament lookup failed", "tournament_id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(ctx, "tournament websocket upgrade failed", "tournament_id", tournamentID, "error", err)
		return
	}

	roomKey := "tournament:" + strconv.FormatInt(id, 10)
	room := h.Hub.GetOrCreateRoom(roomKey)

	client := newClientFromClaims(conn, claims, roomKey)

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

		limiter := newWSRateLimiter(10, time.Second)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			if !limiter.allow() {
				continue
			}
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := h.GameService.EndGameOnDisconnect(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "end game on disconnect failed", "game_id", gameID, "user_id", userID, "error", err)
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
		for _, client := range room.Spectators {
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

func newClientFromClaims(conn *websocket.Conn, claims *token.AccessTokenClaims, gameID string) *Client {
	var isGuest bool
	var version int
	var userID int64
	if claims != nil {
		isGuest = claims.IsGuest
		version = claims.PasswordVersion
		userID = claims.UserID
	}
	return NewClient(conn, userID, gameID, version, isGuest)
}

func (h *Handler) startPumps(ctx context.Context, client *Client, room *Room) {
	wsCtx := context.WithoutCancel(ctx)
	go h.writePump(wsCtx, client)
	go h.readPump(wsCtx, client, room)
}

func (h *Handler) handleActionError(ctx context.Context, client *Client, msgType string, err error) {
	slog.WarnContext(ctx, "websocket action failed", "game_id", client.GameID, "user_id", client.UserID, "type", msgType, "error", err)
	sendError(ctx, client, err.Error())
}

func (h *Handler) buildGameStateMessage(
	ctx context.Context,
	g *game.Game,
	moves []game.GameMove,
	messageType string,
) game.GameStateMessage {
	if moves == nil && h.GameService != nil {
		moves, _ = h.GameService.GetMoves(ctx, g.ID)
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

	return game.GameStateMessage{
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
}

func attachRatingUpdate(msg *game.GameStateMessage, g *game.Game, userID int64) {
	if g == nil || g.Status != game.StatusFinished || !g.Rated || g.WhiteRatingBefore == nil || g.WhiteRatingAfter == nil || g.BlackRatingBefore == nil || g.BlackRatingAfter == nil {
		return
	}
	switch userID {
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
