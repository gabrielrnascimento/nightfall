package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/coder/websocket"
	"github.com/gabrielrnascimento/nightfall/backend/internal/game"
	"go.opentelemetry.io/otel/attribute"
)

type Client struct {
	conn             *websocket.Conn
	send             chan []byte
	name             string
	room             string
	messagesReceived int64
	messagesSent     int64
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
			c.messagesSent++

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
		c.messagesReceived++

		if err := c.handleMessage(ctx, content); err != nil {
			return err
		}
	}
}

func (c *Client) handleMessage(ctx context.Context, content []byte) error {
	ctx, span := tracer.Start(ctx, "websocket.message")
	defer span.End()

	if c.name == "" {
		span.SetAttributes(attribute.String("message.type", "join"))
		return c.handleJoin(ctx, content)
	}

	var env Envelope
	if err := json.Unmarshal(content, &env); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	span.SetAttributes(attribute.String("message.type", env.Type))

	switch env.Type {
	case "join":
		return c.handleJoin(ctx, content)
	case "leave":
		return c.handleLeave(ctx, content)
	case "start":
		return c.handleStart(ctx, content)
	case "ready":
		return c.handleReady(ctx, content)
	default:
		return fmt.Errorf("unknown message type: %s", env.Type)
	}
}

func (c *Client) handleJoin(ctx context.Context, content []byte) error {
	ctx, span := tracer.Start(ctx, "websocket.message.join")
	defer span.End()

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

	joinedMsg := fmt.Sprintf(`{"type":"joined","room":"%s"}`, room.name)
	c.send <- []byte(joinedMsg)

	slog.InfoContext(ctx, "player joined", "name", c.name, "room", c.room)
	return nil
}

func (c *Client) handleLeave(ctx context.Context, content []byte) error {
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

	slog.InfoContext(ctx, "player left", "name", c.name, "room", c.room)
	c.room = ""
	return nil
}

func (c *Client) handleStart(ctx context.Context, content []byte) error {
	var msg StartMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return err
	}

	hub.mutex.RLock()
	room, exists := hub.rooms[c.room]
	if !exists {
		return fmt.Errorf("client is not in a room")
	}
	hub.mutex.RUnlock()

	room.mutex.RLock()
	var players []string
	for client := range room.clients {
		players = append(players, client.name)
	}
	room.mutex.RUnlock()

	game := game.NewGame(players)

	pRoles := game.Start()

	rolesJson, _ := json.Marshal(pRoles)
	startMessage := fmt.Sprintf(`{"type":"game_started","roles":%s}`, rolesJson)

	room.broadcast([]byte(startMessage), nil)

	slog.InfoContext(ctx, "game started", "room", c.room, "player_count", len(players))
	return nil
}

func (c *Client) handleReady(ctx context.Context, content []byte) error {
	var msg ReadyMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return err
	}

	hub.mutex.RLock()
	room, exists := hub.rooms[c.room]
	if !exists {
		return fmt.Errorf("client is not in a room")
	}
	hub.mutex.RUnlock()

	readyMsg := fmt.Sprintf(`{"type":"user_ready","name":"%s"}`, c.name)
	room.broadcast([]byte(readyMsg), nil)

	slog.InfoContext(ctx, "player ready", "name", c.name, "room", c.room)
	return nil
}
