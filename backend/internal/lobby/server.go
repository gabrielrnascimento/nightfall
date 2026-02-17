package lobby

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

type Server struct {
	Logf func(f string, v ...any)
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
		},
	})
	if err != nil {
		s.Logf("%v", err)
		return
	}
	defer c.CloseNow()
	s.Logf("client connected from %v", r.RemoteAddr)

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
		s.Logf("client disconnected normally from %v", r.RemoteAddr)
	} else {
		s.Logf("client disconnected with error from %v %v", r.RemoteAddr, err)
	}
}
