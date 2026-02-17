package lobby

import "sync"

type Room struct {
	name    string
	clients map[*Client]bool
	mutex   sync.RWMutex
}

func (r *Room) addClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client] = true
}

func (r *Room) removeClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.clients, client)
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

var hub = &Hub{
	rooms: make(map[string]*Room),
}
