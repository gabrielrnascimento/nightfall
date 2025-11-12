package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

type simpleServer struct {
	logf func(f string, v ...any)
}

func (s simpleServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		s.logf("%v", err)
		return
	}
	defer c.CloseNow()
	s.logf("client connected from %v", r.RemoteAddr)

	ctx := r.Context()
	for {
		err = handleMessage(ctx, c)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			s.logf("client disconnected normally from %v", r.RemoteAddr)
			return
		}
		if err != nil {
			s.logf("client disconnected with error from %v", r.RemoteAddr)
			return
		}
	}
}

func handleMessage(ctx context.Context, c *websocket.Conn) error {
	typ, content, err := c.Read(ctx)
	if err != nil {
		return err
	}

	message := string(content)
	if message == "ping" {
		if err := c.Write(ctx, typ, []byte("pong")); err != nil {
			return err
		}
	}

	fmt.Printf("message received. type: %v - content: %v\n", typ, message)
	return nil
}
