package lobby

import (
	"encoding/json"
	"testing"
)

type broadcastCall struct {
	room    *Room
	msg     []byte
	exclude *Client
}

type fakeHub struct {
	rooms      map[string]*Room
	broadcasts []broadcastCall
}

func (h *fakeHub) GetOrCreateRoom(name string) *Room {
	room, exists := h.rooms[name]
	if !exists {
		room = &Room{
			name:    name,
			clients: make(map[*Client]struct{}),
		}
		h.rooms[name] = room
	}
	return room
}

func (h *fakeHub) Broadcast(room *Room, msg []byte, exclude *Client) {
	h.broadcasts = append(h.broadcasts, broadcastCall{room: room, msg: msg, exclude: exclude})
}

func (h *fakeHub) RemoveClient(room *Room, c *Client) {
	delete(room.clients, c)
	if len(room.clients) == 0 {
		delete(h.rooms, room.name)
	}
}

// --- Tests ---

func TestHandleJoin(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Client)
		joinName string
		joinRoom string
		wantErr  bool
		check    func(t *testing.T, c *Client, hub *fakeHub, userJoined, joined []byte)
	}{
		{
			name:     "valid join",
			joinName: "Alice",
			joinRoom: "lobby",
			check: func(t *testing.T, c *Client, hub *fakeHub, userJoined, joined []byte) {
				if c.name != "Alice" {
					t.Errorf("c.name = %q, want %q", c.name, "Alice")
				}
				if c.room != "lobby" {
					t.Errorf("c.room = %q, want %q", c.room, "lobby")
				}
				if c.currentRoom == nil {
					t.Fatal("c.currentRoom should not be nil after join")
				}
				var msg UserJoinedMsg
				if err := json.Unmarshal(userJoined, &msg); err != nil {
					t.Fatalf("userJoined unmarshal: %v", err)
				}
				if msg.Type != "user_joined" || msg.Name != "Alice" {
					t.Errorf("unexpected userJoined: %+v", msg)
				}
				var joinedMsg JoinedMsg
				if err := json.Unmarshal(joined, &joinedMsg); err != nil {
					t.Fatalf("joined unmarshal: %v", err)
				}
				if joinedMsg.Type != "joined" || joinedMsg.Room != "lobby" {
					t.Errorf("unexpected joined: %+v", joinedMsg)
				}
			},
		},
		{
			name:     "empty name returns error",
			joinName: "",
			joinRoom: "lobby",
			wantErr:  true,
		},
		{
			name:     "empty room returns error",
			joinName: "Alice",
			joinRoom: "",
			wantErr:  true,
		},
		{
			name:     "re-join moves client to new room",
			joinName: "Alice",
			joinRoom: "new-room",
			setup: func(c *Client) {
				oldRoom := &Room{name: "old-room", clients: make(map[*Client]struct{})}
				oldRoom.clients[c] = struct{}{}
				c.room = "old-room"
				c.currentRoom = oldRoom
				c.name = "Alice"
			},
			check: func(t *testing.T, c *Client, hub *fakeHub, _, _ []byte) {
				if c.room != "new-room" {
					t.Errorf("c.room = %q, want %q", c.room, "new-room")
				}
				if _, stillInOld := hub.rooms["old-room"]; stillInOld {
					t.Error("old room should have been cleaned up")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := &fakeHub{rooms: make(map[string]*Room)}
			c := &Client{}
			if tt.setup != nil {
				tt.setup(c)
				// register old room in fakeHub so RemoveClient can clean it up
				if c.currentRoom != nil {
					hub.rooms[c.currentRoom.name] = c.currentRoom
				}
			}

			userJoined, joined, err := handleJoin(c, hub, tt.joinName, tt.joinRoom)
			if (err != nil) != tt.wantErr {
				t.Fatalf("handleJoin() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, c, hub, userJoined, joined)
			}
		})
	}
}

func TestHandleLeave(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Client, *fakeHub)
		wantErr bool
		check   func(t *testing.T, left, userLeft []byte)
	}{
		{
			name: "normal leave",
			setup: func(c *Client, hub *fakeHub) {
				room := hub.GetOrCreateRoom("lobby")
				c.name = "Alice"
				c.room = "lobby"
				c.currentRoom = room
				room.addClient(c)
			},
			check: func(t *testing.T, left, userLeft []byte) {
				var leftMsg LeftMsg
				if err := json.Unmarshal(left, &leftMsg); err != nil {
					t.Fatalf("left unmarshal: %v", err)
				}
				if leftMsg.Type != "left" || leftMsg.Room != "lobby" {
					t.Errorf("unexpected left: %+v", leftMsg)
				}
				var ulMsg UserLeftMsg
				if err := json.Unmarshal(userLeft, &ulMsg); err != nil {
					t.Fatalf("userLeft unmarshal: %v", err)
				}
				if ulMsg.Type != "user_left" || ulMsg.Name != "Alice" {
					t.Errorf("unexpected userLeft: %+v", ulMsg)
				}
			},
		},
		{
			name:    "leave when not in room returns error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := &fakeHub{rooms: make(map[string]*Room)}
			c := &Client{}
			if tt.setup != nil {
				tt.setup(c, hub)
			}

			left, userLeft, err := handleLeave(c, hub)
			if (err != nil) != tt.wantErr {
				t.Fatalf("handleLeave() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, left, userLeft)
			}
		})
	}
}

func TestHandleStart(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Client, *fakeHub)
		wantErr bool
		check   func(t *testing.T, gameStarted []byte, players []string)
	}{
		{
			name: "start game",
			setup: func(c *Client, hub *fakeHub) {
				room := hub.GetOrCreateRoom("lobby")
				c.name = "Alice"
				c.room = "lobby"
				c.currentRoom = room
				room.addClient(c)
			},
			check: func(t *testing.T, gameStarted []byte, players []string) {
				var msg GameStartedMsg
				if err := json.Unmarshal(gameStarted, &msg); err != nil {
					t.Fatalf("gameStarted unmarshal: %v", err)
				}
				if msg.Type != "game_started" {
					t.Errorf("unexpected type: %s", msg.Type)
				}
				if len(players) != 1 || players[0] != "Alice" {
					t.Errorf("unexpected players: %v", players)
				}
			},
		},
		{
			name: "start already-started game returns error",
			setup: func(c *Client, hub *fakeHub) {
				room := hub.GetOrCreateRoom("lobby")
				room.gameStarted = true
				c.name = "Alice"
				c.room = "lobby"
				c.currentRoom = room
				room.addClient(c)
			},
			wantErr: true,
		},
		{
			name:    "start before joining room returns error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := &fakeHub{rooms: make(map[string]*Room)}
			c := &Client{}
			if tt.setup != nil {
				tt.setup(c, hub)
			}

			gameStarted, players, err := handleStart(c, hub)
			if (err != nil) != tt.wantErr {
				t.Fatalf("handleStart() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, gameStarted, players)
			}
		})
	}
}

func TestHandleReady(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Client, *fakeHub)
		wantErr bool
		check   func(t *testing.T, userReady []byte)
	}{
		{
			name: "ready in room",
			setup: func(c *Client, hub *fakeHub) {
				room := hub.GetOrCreateRoom("lobby")
				c.name = "Alice"
				c.room = "lobby"
				c.currentRoom = room
				room.addClient(c)
			},
			check: func(t *testing.T, userReady []byte) {
				var msg UserReadyMsg
				if err := json.Unmarshal(userReady, &msg); err != nil {
					t.Fatalf("userReady unmarshal: %v", err)
				}
				if msg.Type != "user_ready" || msg.Name != "Alice" {
					t.Errorf("unexpected userReady: %+v", msg)
				}
			},
		},
		{
			name:    "ready before joining returns error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := &fakeHub{rooms: make(map[string]*Room)}
			c := &Client{}
			if tt.setup != nil {
				tt.setup(c, hub)
			}

			userReady, err := handleReady(c, hub)
			if (err != nil) != tt.wantErr {
				t.Fatalf("handleReady() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, userReady)
			}
		})
	}
}
