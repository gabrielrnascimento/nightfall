package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func Test_simpleServer(t *testing.T) {
	t.Run("ping", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(simpleServer{
			logf: t.Logf,
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "the sky is falling")

		err = c.Write(ctx, websocket.MessageText, []byte("ping"))
		if err != nil {
			t.Fatal(err)
		}

		_, bytes, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got := string(bytes)
		want := "pong"

		if got != want {
			t.Fatalf("got %v want %v", got, want)
		}

		c.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("noResponse", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(simpleServer{
			logf: t.Logf,
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "the sky is falling")

		err = c.Write(ctx, websocket.MessageText, []byte("hello"))
		if err != nil {
			t.Fatal(err)
		}

		// Test that server doesn't respond to non-ping messages
		// Use a short timeout to verify no message is received
		readCtx, readCancel := context.WithTimeout(context.Background(), time.Second*1)
		defer readCancel()

		_, _, err = c.Read(readCtx)
		if err == nil {
			t.Fatal("expected timeout error, but received a message")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}

		c.Close(websocket.StatusNormalClosure, "")
	})
}
