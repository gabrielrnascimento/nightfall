package lobby

import (
	"encoding/json"
	"fmt"

	"github.com/gabrielrnascimento/nightfall/backend/internal/game"
)

// handleJoin is called when a client sends a join message.
func handleJoin(c *Client, hub HubStore, name, roomName string) ([]byte, []byte, error) {
	if name == "" || roomName == "" {
		return nil, nil, fmt.Errorf("no name or room provided")
	}

	if c.currentRoom != nil {
		hub.RemoveClient(c.currentRoom, c)
	}

	room := hub.GetOrCreateRoom(roomName)

	c.name = name
	c.room = roomName
	c.currentRoom = room

	room.addClient(c)

	userJoinedBytes, _ := json.Marshal(UserJoinedMsg{Type: "user_joined", Name: name})
	joinedBytes, _ := json.Marshal(JoinedMsg{Type: "joined", Room: roomName})

	return userJoinedBytes, joinedBytes, nil
}

// handleLeave removes the client from their current room and returns
// the (leftMsg, userLeftMsg) bytes to send, or an error if not in a room.
func handleLeave(c *Client, hub HubStore) ([]byte, []byte, error) {
	if c.currentRoom == nil {
		return nil, nil, fmt.Errorf("not in a room")
	}

	room := c.currentRoom

	leftBytes, err := json.Marshal(LeftMsg{Type: "left", Room: room.name})
	if err != nil {
		return nil, nil, err
	}
	userLeftBytes, err := json.Marshal(UserLeftMsg{Type: "user_left", Name: c.name})
	if err != nil {
		return nil, nil, err
	}

	hub.RemoveClient(room, c)
	return leftBytes, userLeftBytes, nil
}

// handleStart marks the room's game as started and returns the game_started broadcast bytes.
// Returns an error if the client is not in a room or the game has already started.
func handleStart(c *Client, _ HubStore) ([]byte, []string, game.PlayerRoles, error) {
	if c.currentRoom == nil {
		return nil, nil, nil, fmt.Errorf("not in a room")
	}

	room := c.currentRoom
	room.mutex.Lock()
	if room.gameStarted {
		room.mutex.Unlock()
		return nil, nil, nil, fmt.Errorf("game already started")
	}
	var players []string
	for client := range room.clients {
		players = append(players, client.name)
	}
	room.gameStarted = true
	room.mutex.Unlock()

	g := game.NewGame(players)
	roles := g.Start()

	gameStartedBytes, err := json.Marshal(GameStartedMsg{Type: "game_started"})
	if err != nil {
		return nil, nil, nil, err
	}

	return gameStartedBytes, players, roles, nil
}

// handleReady broadcasts that a client is ready in their room.
// Returns an error if the client is not in a room.
func handleReady(c *Client, _ HubStore) ([]byte, error) {
	if c.currentRoom == nil {
		return nil, fmt.Errorf("not in a room")
	}

	readyBytes, err := json.Marshal(UserReadyMsg{Type: "user_ready", Name: c.name})
	if err != nil {
		return nil, err
	}

	return readyBytes, nil
}
