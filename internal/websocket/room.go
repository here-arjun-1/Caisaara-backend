package websocket

import (
	"sync"

	"github.com/here-arjun-1/Caisaara-backend/internal/metrics"
)

type Room struct {
	GameID string

	Clients map[int64]*Client

	GameStarted bool

	Mutex sync.RWMutex
}

func NewRoom(gameID string) *Room {
	return &Room{
		GameID:  gameID,
		Clients: make(map[int64]*Client),
	}
}

func (r *Room) AddClient(client *Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	_, alreadyThere := r.Clients[client.UserID]
	if !alreadyThere {
		metrics.WebsocketConnections.Inc()
	}

	r.Clients[client.UserID] = client
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
}
