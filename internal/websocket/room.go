package websocket

import "sync"

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

	r.Clients[client.UserID] = client
}

func (r *Room) RemoveClient(userID int64) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	delete(r.Clients, userID)
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

		select {
		case client.Send <- message:
		default:
		}
	}
}
