package lobby

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("nightfall/lobby")

type Server struct {
	Logf func(f string, v ...any)
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "websocket.session")
	defer span.End()

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
		s.Logf("%v", err)
		return
	}
	defer c.CloseNow()
	s.Logf("client connected from %v", r.RemoteAddr)

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

	if err == nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		s.Logf("client disconnected normally from %v", r.RemoteAddr)
	} else {
		s.Logf("client disconnected with error from %v %v", r.RemoteAddr, err)
	}
}
