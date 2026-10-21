package websocket

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	Rooms map[string]*Room

	Mutex sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetOrCreateRoom(gameID string) *Room {

	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	room, exists := h.Rooms[gameID]

	if !exists {
		room = NewRoom(gameID)
		h.Rooms[gameID] = room
	}

	return room
}

func (h *Hub) RemoveRoom(gameID string) {

	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	delete(h.Rooms, gameID)
}

func (h *Hub) GetRoom(gameID string) *Room {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	return h.Rooms[gameID]
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.Mutex.RLock()
	room, exists := h.Rooms[roomID]
	h.Mutex.RUnlock()

	if exists && room != nil {
		room.Broadcast(message)
	}
}

func (h *Hub) BroadcastJSON(roomID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h.BroadcastToRoom(roomID, data)
	return nil
}

func (h *Hub) DisconnectUser(userID int64) {
	h.Mutex.RLock()
	var clients []*Client
	for _, room := range h.Rooms {
		room.Mutex.RLock()
		if client, exists := room.Clients[userID]; exists && client != nil {
			clients = append(clients, client)
		}
		if client, exists := room.Spectators[userID]; exists && client != nil {
			clients = append(clients, client)
		}
		room.Mutex.RUnlock()
	}
	h.Mutex.RUnlock()

	for _, client := range clients {
		client.Close()
	}
}

func (h *Hub) CloseRoom(roomID string) {
	room := h.GetRoom(roomID)
	if room == nil {
		return
	}

	room.Mutex.RLock()
	var clients []*Client
	for _, client := range room.Clients {
		clients = append(clients, client)
	}
	room.Mutex.RUnlock()

	for _, client := range clients {
		client.Close()
	}
}

func (h *Hub) DisconnectUserFromRoom(roomID string, userID int64) {
	room := h.GetRoom(roomID)
	if room == nil {
		return
	}

	room.Mutex.RLock()
	client := room.Clients[userID]
	room.Mutex.RUnlock()

	if client != nil {
		client.Close()
	}
}
