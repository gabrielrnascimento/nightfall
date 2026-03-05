package lobby

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func Test_simpleServer(t *testing.T) {
	t.Run("join and leave room", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(Server{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "internal server error")

		var joinMessage = `{"type": "join", "name": "Alice", "room": "general"}`
		err = c.Write(ctx, websocket.MessageText, []byte(joinMessage))
		if err != nil {
			t.Fatal(err)
		}

		_, bytes, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got := string(bytes)
		want := `{"type":"joined","room":"general"}`

		if got != want {
			t.Fatalf("got %v want %v", got, want)
		}

		var leaveMessage = `{"type": "leave"}`
		err = c.Write(ctx, websocket.MessageText, []byte(leaveMessage))
		if err != nil {
			t.Fatal(err)
		}

		_, bytes, err = c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got = string(bytes)
		want = `{"type":"left","room":"general"}`

		if got != want {
			t.Fatalf("got %v want %v", got, want)
		}

		c.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("start and ready messages", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(Server{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c, _, err := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close(websocket.StatusInternalError, "internal server error")

		joinMessage := `{"type": "join", "name": "Alice", "room": "games"}`
		_ = c.Write(ctx, websocket.MessageText, []byte(joinMessage))
		_, _, _ = c.Read(ctx)

		err = c.Write(ctx, websocket.MessageText, []byte(`{"type": "ready"}`))
		if err != nil {
			t.Fatal(err)
		}
		_, bytes, _ := c.Read(ctx)
		got := string(bytes)
		want := `{"type":"user_ready","name":"Alice"}`
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}

		err = c.Write(ctx, websocket.MessageText, []byte(`{"type": "start"}`))
		if err != nil {
			t.Fatal(err)
		}
		_, bytes, _ = c.Read(ctx)
		got = string(bytes)
		want = `{"type":"game_started","roles":{"Assassin":"Alice"}}`
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("multi-client interactions", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(Server{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		c1, _, _ := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c1.Close(websocket.StatusInternalError, "")
		_ = c1.Write(ctx, websocket.MessageText, []byte(`{"type": "join", "name": "Alice", "room": "party"}`))
		_, _, _ = c1.Read(ctx)

		c2, _, _ := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c2.Close(websocket.StatusInternalError, "")
		_ = c2.Write(ctx, websocket.MessageText, []byte(`{"type": "join", "name": "Bob", "room": "party"}`))
		_, _, _ = c2.Read(ctx)

		_, bytes, _ := c1.Read(ctx)
		got := string(bytes)
		want := `{"type":"user_joined","name":"Bob"}`
		if got != want {
			t.Errorf("Alice got %v want %v", got, want)
		}

		_ = c1.Write(ctx, websocket.MessageText, []byte(`{"type": "start"}`))
		_, bytes, _ = c1.Read(ctx)
		wantStart := `{"type":"game_started","roles":{"Assassin":"Alice","Detective":"Bob"}}`
		if string(bytes) != wantStart {
			t.Errorf("Alice didn't get game_started, got %s", string(bytes))
		}

		_, bytes, _ = c2.Read(ctx)
		if string(bytes) != wantStart {
			t.Errorf("Bob didn't get game_started, got %s", string(bytes))
		}

		_ = c2.Write(ctx, websocket.MessageText, []byte(`{"type": "leave"}`))
		_, _, _ = c2.Read(ctx)

		_, bytes, _ = c1.Read(ctx)
		got = string(bytes)
		want = `{"type":"user_left","name":"Bob"}`
		if got != want {
			t.Errorf("Alice got %v want %v", got, want)
		}
	})

	t.Run("error scenarios", func(t *testing.T) {
		t.Parallel()

		s := httptest.NewServer(Server{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		})
		defer s.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		c, _, _ := websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c.Close(websocket.StatusInternalError, "")

		_ = c.Write(ctx, websocket.MessageText, []byte(`{invalid json}`))
		_, _, err := c.Read(ctx)
		if err == nil {
			t.Error("expected error for invalid JSON but got none")
		}

		c, _, _ = websocket.Dial(ctx, s.URL, &websocket.DialOptions{})
		defer c.Close(websocket.StatusInternalError, "")

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type": "join", "name": "Alice", "room": "lobby"}`))
		_, _, _ = c.Read(ctx)

		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type": "ghost"}`))
		_, _, err = c.Read(ctx)
		if err == nil {
			t.Error("expected error for unknown message type but got none")
		}
	})
}
