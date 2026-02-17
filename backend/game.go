package main

import "math/rand/v2"

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

type Game struct {
	rng     *rand.Rand
	players []string
}

func (g *Game) assignRole(playerRoles PlayerRoles) GameRole {
	assassinPlayer, _ := playerRoles[Assassin]
	if assassinPlayer == "" {
		return Assassin
	}

	detectivePlayer, _ := playerRoles[Detective]
	if detectivePlayer == "" {
		return Detective
	}

	angelPlayer, _ := playerRoles[Angel]
	if angelPlayer == "" {
		return Angel
	}

	escortPlayer, _ := playerRoles[Escort]
	if escortPlayer == "" {
		return Escort
	}

	sadBoyPlayer, _ := playerRoles[Sadboy]
	if sadBoyPlayer == "" {
		return Sadboy
	}

	return Citizen
}

func (g *Game) Start() PlayerRoles {
	playerRoles := make(PlayerRoles)

	for _, player := range g.players {
		role := g.assignRole(playerRoles)
		playerRoles[role] = player
	}

	return playerRoles
}
