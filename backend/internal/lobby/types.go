package lobby

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
