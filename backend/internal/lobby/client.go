package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	conn             *websocket.Conn
	send             chan []byte
	name             string
	room             string
	currentRoom      *Room
	hub              HubStore
	logger           *slog.Logger
	messagesReceived atomic.Int64
	messagesSent     atomic.Int64
}

func (c *Client) run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.readPump(ctx)
	})
	eg.Go(func() error {
		c.writePump(ctx)
		return nil
	})

	return eg.Wait()
}

func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusAbnormalClosure, "channel closed")
				return
			}

			if err := c.conn.Write(ctx, websocket.MessageText, message); err != nil {
				return
			}
			c.messagesSent.Add(1)

		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) readPump(ctx context.Context) error {
	defer func() {
		if c.currentRoom != nil {
			c.hub.RemoveClient(c.currentRoom, c)
			msg, _ := json.Marshal(UserLeftMsg{Type: "user_left", Name: c.name})
			c.hub.Broadcast(c.currentRoom, msg, nil)
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
		c.messagesReceived.Add(1)

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
		return c.dispatchJoin(ctx, content)
	}

	var env Envelope
	if err := json.Unmarshal(content, &env); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	span.SetAttributes(attribute.String("message.type", env.Type))

	switch env.Type {
	case "join":
		return c.dispatchJoin(ctx, content)
	case "leave":
		return c.dispatchLeave(ctx)
	case "start":
		return c.dispatchStart(ctx)
	case "ready":
		return c.dispatchReady(ctx)
	default:
		return fmt.Errorf("unknown message type: %s", env.Type)
	}
}

func (c *Client) dispatchJoin(ctx context.Context, content []byte) error {
	ctx, span := tracer.Start(ctx, "websocket.message.join")
	defer span.End()

	var joinMsg JoinMessage
	if err := json.Unmarshal(content, &joinMsg); err != nil {
		return err
	}

	userJoinedBytes, joinedBytes, err := handleJoin(c, c.hub, joinMsg.Name, joinMsg.Room)
	if err != nil {
		return err
	}

	c.hub.Broadcast(c.currentRoom, userJoinedBytes, c)
	c.send <- joinedBytes

	c.logger.InfoContext(ctx, "player joined", "name", c.name, "room", c.room)
	return nil
}

func (c *Client) dispatchLeave(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "websocket.message.leave")
	defer span.End()

	leftBytes, userLeftBytes, err := handleLeave(c, c.hub)
	if err != nil {
		return err
	}

	c.hub.Broadcast(c.currentRoom, userLeftBytes, nil)
	c.send <- leftBytes

	c.logger.InfoContext(ctx, "player left", "name", c.name, "room", c.room)
	c.currentRoom = nil
	c.room = ""
	return nil
}

func (c *Client) dispatchStart(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "websocket.message.start")
	defer span.End()

	gameStartedBytes, players, roles, err := handleStart(c, c.hub)
	if err != nil {
		return err
	}

	c.hub.Broadcast(c.currentRoom, gameStartedBytes, nil)

	roleAttrs := make([]attribute.KeyValue, 0, len(roles))
	for role, player := range roles {
		roleAttrs = append(roleAttrs, attribute.String("game.role."+role.String(), player))
	}
	span.SetAttributes(append([]attribute.KeyValue{attribute.Int("game.player_count", len(players))}, roleAttrs...)...)

	c.logger.InfoContext(ctx, "game started", "room", c.room, "player_count", len(players), "roles", roles)
	return nil
}

func (c *Client) dispatchReady(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "websocket.message.ready")
	defer span.End()

	userReadyBytes, err := handleReady(c, c.hub)
	if err != nil {
		return err
	}

	c.hub.Broadcast(c.currentRoom, userReadyBytes, nil)
	c.logger.InfoContext(ctx, "player ready", "name", c.name, "room", c.room)
	return nil
}
