package websocket

import (
	"sync"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/metrics"
)

type Room struct {
	GameID string

	Clients map[int64]*Client

	Spectators map[int64]*Client

	GameStarted bool

	Game *game.Game

	Moves []game.GameMove

	Mutex sync.RWMutex
}

func NewRoom(gameID string) *Room {
	return &Room{
		GameID:     gameID,
		Clients:    make(map[int64]*Client),
		Spectators: make(map[int64]*Client),
	}
}

func (r *Room) SetGame(g *game.Game) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Game = g
}

func (r *Room) GetGame() *game.Game {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.Game
}

func (r *Room) SetMoves(moves []game.GameMove) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Moves = append([]game.GameMove(nil), moves...)
}

func (r *Room) AddMove(move game.GameMove) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Moves = append(r.Moves, move)
}

func (r *Room) GetMoves() []game.GameMove {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	if r.Moves == nil {
		return nil
	}
	result := make([]game.GameMove, len(r.Moves))
	copy(result, r.Moves)
	return result
}

func (r *Room) MoveCount() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return len(r.Moves)
}

func (r *Room) AddClient(client *Client) {
	r.Mutex.Lock()
	oldClient, exists := r.Clients[client.UserID]
	if !exists {
		metrics.WebsocketConnections.Inc()
	}
	r.Clients[client.UserID] = client
	r.Mutex.Unlock()

	if exists && oldClient != nil && oldClient != client {
		oldClient.Close()
	}
}

func (r *Room) RemoveClient(userID int64, client ...*Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if len(client) > 0 && client[0] != nil {
		if current, ok := r.Clients[userID]; ok && current != client[0] {
			return
		}
	}

	_, exists := r.Clients[userID]
	if exists {
		metrics.WebsocketConnections.Dec()
	}

	delete(r.Clients, userID)
}

func (r *Room) AddSpectator(client *Client) {
	r.Mutex.Lock()
	oldClient, exists := r.Spectators[client.UserID]
	if !exists {
		metrics.WebsocketConnections.Inc()
	}
	r.Spectators[client.UserID] = client
	r.Mutex.Unlock()

	if exists && oldClient != nil && oldClient != client {
		oldClient.Close()
	}
}

func (r *Room) RemoveSpectator(userID int64, client *Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	current, exists := r.Spectators[userID]
	if !exists {
		return
	}

	if current != client {
		return
	}

	metrics.WebsocketConnections.Dec()
	delete(r.Spectators, userID)
}

func (r *Room) HasClient(userID int64) bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	_, exists := r.Clients[userID]
	return exists
}

func (r *Room) Count() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	return len(r.Clients)
}

func (r *Room) SpectatorCount() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	return len(r.Spectators)
}

func (r *Room) IsEmpty() bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	return len(r.Clients) == 0 && len(r.Spectators) == 0
}

func (r *Room) MarkGameStarted() bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.GameStarted {
		return false
	}
	r.GameStarted = true
	return true
}

func (r *Room) IsGameStarted() bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	return r.GameStarted
}

func (r *Room) Broadcast(message []byte) {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for _, client := range r.Clients {
		if client != nil {
			client.SafeSend(message)
		}
	}
	for _, client := range r.Spectators {
		if client != nil {
			client.SafeSend(message)
		}
	}
}

func (r *Room) BroadcastToPlayers(message []byte) {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	for _, client := range r.Clients {
		if client != nil {
			client.SafeSend(message)
		}
	}
}
