package lobby

import "sync"

type HubStore interface {
	GetOrCreateRoom(name string) *Room
	Broadcast(room *Room, msg []byte, exclude *Client)
	RemoveClient(room *Room, c *Client)
}

type Room struct {
	name        string
	clients     map[*Client]struct{}
	gameStarted bool
	mutex       sync.RWMutex
}

func (r *Room) addClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client] = struct{}{}
}

func (r *Room) broadcast(message []byte, exclude *Client) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for client := range r.clients {
		if client != exclude {
			select {
			case client.send <- message:
			default:
			}
		}
	}
}

type Hub struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

func (h *Hub) GetOrCreateRoom(name string) *Room {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	room, exists := h.rooms[name]
	if !exists {
		room = &Room{
			name:    name,
			clients: make(map[*Client]struct{}),
		}
		h.rooms[name] = room
	}
	return room
}

func (h *Hub) Broadcast(room *Room, msg []byte, exclude *Client) {
	room.broadcast(msg, exclude)
}

func (h *Hub) RemoveClient(room *Room, c *Client) {
	room.mutex.Lock()
	delete(room.clients, c)
	empty := len(room.clients) == 0
	room.mutex.Unlock()

	if empty {
		h.mutex.Lock()
		delete(h.rooms, room.name)
		h.mutex.Unlock()
	}
}
