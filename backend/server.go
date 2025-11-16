package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type Envelope struct {
	Type string `json:"type"`
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	name string
	room string
}

func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusAbnormalClosure, "channel closed")
				return
			}

			if err := c.conn.Write(ctx, websocket.MessageText, message); err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) readPump(ctx context.Context) error {
	defer func() {
		if c.room != "" {
			hub.mutex.RLock()
			room := hub.rooms[c.room]
			hub.mutex.RUnlock()
			if room != nil {
				room.removeClient(c)
				leaveMsg := fmt.Sprintf(`{"type":"user_left","name":"%s"}`, c.name)
				room.broadcast([]byte(leaveMsg), nil)
			}
		}
		close(c.send)
	}()

	for {
		_, content, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return err
		}

		if err := c.handleMessage(content); err != nil {
			return err
		}
	}
}

func (c *Client) handleMessage(content []byte) error {
	if c.name == "" {
		err := c.handleJoin(content)
		if err != nil {
			return err
		}
		return nil
	}

	var env Envelope
	if err := json.Unmarshal(content, &env); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	switch env.Type {
	case "join":
		return c.handleJoin(content)
	case "leave":
		return c.handleLeave(content)
	default:
		return fmt.Errorf("unknown message type: %s", env.Type)
	}
}

type JoinMessage struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Room string `json:"room"`
}

func (c *Client) handleJoin(content []byte) error {
	var joinMsg JoinMessage
	if err := json.Unmarshal(content, &joinMsg); err != nil {
		return err
	}

	if joinMsg.Type != "join" || joinMsg.Name == "" || joinMsg.Room == "" {
		return fmt.Errorf("invalid join message")
	}

	if c.room != "" {
		hub.mutex.RLock()
		room, exists := hub.rooms[c.room]
		hub.mutex.RUnlock()
		if exists {
			room.removeClient(c)
		}
	}

	c.name = joinMsg.Name
	c.room = joinMsg.Room

	hub.mutex.Lock()
	room, exists := hub.rooms[joinMsg.Room]
	if !exists {
		room = &Room{
			name:    joinMsg.Room,
			clients: make(map[*Client]bool),
		}
		hub.rooms[joinMsg.Room] = room
	}
	hub.mutex.Unlock()

	room.addClient(c)

	userJoinedMsg := fmt.Sprintf(`{"type":"user_joined","name":"%s"}`, c.name)
	room.broadcast([]byte(userJoinedMsg), c)

	joinedMsg := fmt.Sprintf(`{"type":"joined","room": "%s"}`, room.name)
	c.send <- []byte(joinedMsg)

	return nil
}

type LeaveMessage struct {
	Type string `json:"type"`
}

func (c *Client) handleLeave(content []byte) error {
	var msg LeaveMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return err
	}

	if c.room != "" {
		hub.mutex.RLock()
		room, exists := hub.rooms[c.room]
		hub.mutex.RUnlock()
		if exists {
			room.removeClient(c)
			userLeftMsg := fmt.Sprintf(`{"type":"user_left","name":"%s"}`, c.name)
			room.broadcast([]byte(userLeftMsg), nil)

			leftMsg := fmt.Sprintf(`{"type":"left","room":"%s"}`, room.name)
			c.send <- []byte(leftMsg)
		}
	}

	c.room = ""
	return nil
}

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

type simpleServer struct {
	logf func(f string, v ...any)
}

func (s simpleServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
		},
	})
	if err != nil {
		s.logf("%v", err)
		return
	}
	defer c.CloseNow()
	s.logf("client connected from %v", r.RemoteAddr)

	client := &Client{
		conn: c,
		send: make(chan []byte, 256),
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- client.readPump(ctx)
	}()

	go func() {
		client.writePump(ctx)
	}()

	err = <-errChan

	if err == nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		s.logf("client disconnected normally from %v", r.RemoteAddr)
	} else {
		s.logf("client disconnected with error from %v %v", r.RemoteAddr, err)
	}
}
