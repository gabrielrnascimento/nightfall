package lobby

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("nightfall/lobby")

type Server struct {
	Logger *slog.Logger
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "websocket.session")
	defer span.End()

	start := time.Now()

	event := &SessionEvent{
		Service:    "nightfall-backend",
		Event:      "websocket.session",
		RemoteAddr: r.RemoteAddr,
	}

	defer func() {
		event.DurationMs = time.Since(start).Milliseconds()
		event.Emit(ctx, event.Event)
	}()

	span.SetAttributes(attribute.String("net.peer.addr", r.RemoteAddr))

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "websocket accept failed")
		s.Logger.ErrorContext(ctx, "websocket accept failed", "error", err, "remote_addr", r.RemoteAddr)
		event.Outcome = "error"
		event.Error = err.Error()
		return
	}
	defer c.CloseNow()
	s.Logger.InfoContext(ctx, "client connected", "remote_addr", r.RemoteAddr)

	client := &Client{
		conn: c,
		send: make(chan []byte, 256),
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- client.readPump(ctx)
	}()

	go func() {
		client.writePump(ctx)
	}()

	err = <-errChan

	if client.name != "" {
		event.Player = &PlayerContext{ID: client.name}
	}
	if client.room != "" {
		hub.mutex.RLock()
		room := hub.rooms[client.room]
		hub.mutex.RUnlock()
		if room != nil {
			room.mutex.RLock()
			event.Room = &RoomContext{
				ID:          room.name,
				PlayerCount: len(room.clients),
			}
			room.mutex.RUnlock()
		}
	}
	event.Stats = &SessionStats{
		MessagesReceived: client.messagesReceived.Load(),
		MessagesSent:     client.messagesSent.Load(),
	}

	if err == nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		span.SetStatus(codes.Ok, "")
		event.Outcome = "success"
		s.Logger.InfoContext(ctx, "client disconnected", "remote_addr", r.RemoteAddr)
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, "abnormal disconnect")
		event.Outcome = "error"
		event.Error = err.Error()
		s.Logger.ErrorContext(ctx, "client disconnected with error", "remote_addr", r.RemoteAddr, "error", err)
	}
}
