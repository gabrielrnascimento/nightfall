package lobby

import (
	"testing"
)

func TestRoom_AddClient(t *testing.T) {
	hub := NewHub()
	room := hub.GetOrCreateRoom("test-room")

	c := &Client{send: make(chan []byte, 1)}
	room.addClient(c)

	room.mutex.RLock()
	_, found := room.clients[c]
	room.mutex.RUnlock()

	if !found {
		t.Fatal("expected client to be in room after addClient")
	}
}

func TestRoom_RemoveClient_Cleanup(t *testing.T) {
	t.Run("room deleted from hub when last client leaves", func(t *testing.T) {
		hub := NewHub()
		room := hub.GetOrCreateRoom("cleanup-room")

		c := &Client{send: make(chan []byte, 1)}
		room.addClient(c)

		hub.RemoveClient(room, c)

		hub.mutex.RLock()
		_, exists := hub.rooms["cleanup-room"]
		hub.mutex.RUnlock()

		if exists {
			t.Fatal("expected room to be deleted from hub when last client leaves")
		}
	})

	t.Run("room retained when other clients remain", func(t *testing.T) {
		hub := NewHub()
		room := hub.GetOrCreateRoom("active-room")

		c1 := &Client{send: make(chan []byte, 1)}
		c2 := &Client{send: make(chan []byte, 1)}
		room.addClient(c1)
		room.addClient(c2)

		hub.RemoveClient(room, c1)

		hub.mutex.RLock()
		_, exists := hub.rooms["active-room"]
		hub.mutex.RUnlock()

		if !exists {
			t.Fatal("expected room to remain in hub when other clients are present")
		}
	})
}

func TestRoom_Broadcast_Excludes(t *testing.T) {
	t.Run("excluded client does not receive message", func(t *testing.T) {
		hub := NewHub()
		room := hub.GetOrCreateRoom("broadcast-room")

		sender := &Client{send: make(chan []byte, 1)}
		receiver := &Client{send: make(chan []byte, 1)}
		room.addClient(sender)
		room.addClient(receiver)

		hub.Broadcast(room, []byte(`{"type":"test"}`), sender)

		select {
		case msg := <-receiver.send:
			if string(msg) != `{"type":"test"}` {
				t.Fatalf("receiver got wrong message: %s", msg)
			}
		default:
			t.Fatal("expected receiver to get message")
		}

		select {
		case <-sender.send:
			t.Fatal("excluded sender should not receive the broadcast")
		default:
			// correct: sender's channel is empty
		}
	})

	t.Run("nil exclude broadcasts to all clients", func(t *testing.T) {
		hub := NewHub()
		room := hub.GetOrCreateRoom("all-room")

		c1 := &Client{send: make(chan []byte, 1)}
		c2 := &Client{send: make(chan []byte, 1)}
		room.addClient(c1)
		room.addClient(c2)

		hub.Broadcast(room, []byte(`{"type":"all"}`), nil)

		for i, c := range []*Client{c1, c2} {
			select {
			case <-c.send:
				// received, as expected
			default:
				t.Fatalf("client %d did not receive broadcast", i+1)
			}
		}
	})
}
