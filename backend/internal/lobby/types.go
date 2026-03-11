package lobby

// Inbound message types (client → server)

type Envelope struct {
	Type string `json:"type"`
}

type JoinMessage struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Room string `json:"room"`
}

type LeaveMessage struct {
	Type string `json:"type"`
}

type StartMessage struct {
	Type string `json:"type"`
}

type ReadyMessage struct {
	Type string `json:"type"`
}

// Outbound message types (server → client)

type JoinedMsg struct {
	Type string `json:"type"`
	Room string `json:"room"`
}

type UserJoinedMsg struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type LeftMsg struct {
	Type string `json:"type"`
	Room string `json:"room"`
}

type UserLeftMsg struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type GameStartedMsg struct {
	Type string `json:"type"`
}

type UserReadyMsg struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
