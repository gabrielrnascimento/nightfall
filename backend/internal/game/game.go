package game

import (
	"log/slog"
	"math/rand/v2"
)

type GameRole int

const (
	Detective GameRole = iota
	Assassin
	Angel
	Escort
	Sadboy
	Citizen
)

var gameRoles = map[GameRole]string{
	Detective: "Detective",
	Assassin:  "Assassin",
	Angel:     "Angel",
	Escort:    "Escort",
	Sadboy:    "Sad Boy",
	Citizen:   "Citizen",
}

func (gr GameRole) String() string {
	return gameRoles[gr]
}

func (gr GameRole) MarshalText() ([]byte, error) {
	return []byte(gr.String()), nil
}

type PlayerRoles map[GameRole]string

func (pr PlayerRoles) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(pr))
	for role, player := range pr {
		attrs = append(attrs, slog.String(role.String(), player))
	}
	return slog.GroupValue(attrs...)
}

type Game struct {
	Players []string
}

func NewGame(players []string) Game {
	return Game{Players: players}
}

func (g *Game) assignRole(playerRoles PlayerRoles) GameRole {
	assassinPlayer := playerRoles[Assassin]
	if assassinPlayer == "" {
		return Assassin
	}

	detectivePlayer := playerRoles[Detective]
	if detectivePlayer == "" {
		return Detective
	}

	angelPlayer := playerRoles[Angel]
	if angelPlayer == "" {
		return Angel
	}

	escortPlayer := playerRoles[Escort]
	if escortPlayer == "" {
		return Escort
	}

	sadBoyPlayer := playerRoles[Sadboy]
	if sadBoyPlayer == "" {
		return Sadboy
	}

	return Citizen
}

func (g *Game) Start() PlayerRoles {
	playerRoles := make(PlayerRoles)

	slice := g.Players

	rand.Shuffle(len(g.Players), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})

	for _, player := range g.Players {
		role := g.assignRole(playerRoles)
		playerRoles[role] = player
	}

	return playerRoles
}
