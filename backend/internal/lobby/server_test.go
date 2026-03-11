package lobby

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hub := NewHub()
	// Use io.Discard for logger: E2E tests don't assert on log output, and
	// io.Discard is goroutine-safe (avoids a race between server goroutines
	// logging after the test has returned and testing.T cleanup).
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httptest.NewServer(NewServer(hub, logger))
}

func Test_simpleServer(t *testing.T) {
	t.Run("join and leave room", func(t *testing.T) {
		t.Parallel()

		s := newTestServer(t)
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "internal server error")

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Alice","room":"join-leave-room"}`))
		_, bytes, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(bytes), `{"type":"joined","room":"join-leave-room"}`; got != want {
			t.Fatalf("got %v want %v", got, want)
		}

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"leave"}`))
		_, bytes, err = c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(bytes), `{"type":"left","room":"join-leave-room"}`; got != want {
			t.Fatalf("got %v want %v", got, want)
		}

		c.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("start and ready messages", func(t *testing.T) {
		t.Parallel()

		s := newTestServer(t)
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "internal server error")

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Alice","room":"start-ready-room"}`))
		_, _, _ = c.Read(ctx)

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"ready"}`))
		_, bytes, _ := c.Read(ctx)
		if got, want := string(bytes), `{"type":"user_ready","name":"Alice"}`; got != want {
			t.Errorf("got %v want %v", got, want)
		}

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"start"}`))
		_, bytes, _ = c.Read(ctx)
		if got, want := string(bytes), `{"type":"game_started"}`; got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("multi-client interactions", func(t *testing.T) {
		t.Parallel()

		s := newTestServer(t)
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c1, _, _ := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c1.Close(websocket.StatusInternalError, "")
		_ = c1.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Alice","room":"multi-client-room"}`))
		_, _, _ = c1.Read(ctx)

		c2, _, _ := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c2.Close(websocket.StatusInternalError, "")
		_ = c2.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Bob","room":"multi-client-room"}`))
		_, _, _ = c2.Read(ctx)

		// Alice receives user_joined for Bob
		_, bytes, _ := c1.Read(ctx)
		if got, want := string(bytes), `{"type":"user_joined","name":"Bob"}`; got != want {
			t.Errorf("Alice got %v want %v", got, want)
		}

		// Alice starts; both receive game_started
		_ = c1.Write(ctx, websocket.MessageText, []byte(`{"type":"start"}`))
		_, aliceBytes, _ := c1.Read(ctx)
		_, bobBytes, _ := c2.Read(ctx)

		if string(aliceBytes) != string(bobBytes) {
			t.Errorf("clients received different game_started messages: Alice=%s Bob=%s", aliceBytes, bobBytes)
		}
		if got, want := string(aliceBytes), `{"type":"game_started"}`; got != want {
			t.Errorf("got %v want %v", got, want)
		}

		// Bob leaves; Alice receives user_left
		_ = c2.Write(ctx, websocket.MessageText, []byte(`{"type":"leave"}`))
		_, _, _ = c2.Read(ctx)
		_, bytes, _ = c1.Read(ctx)
		if got, want := string(bytes), `{"type":"user_left","name":"Bob"}`; got != want {
			t.Errorf("Alice got %v want %v", got, want)
		}
	})
}
