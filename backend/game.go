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
	Escort:    "Wscort",
	Sadboy:    "Sad Boy",
	Citizen:   "Citizen",
}

var allGameRoles = []GameRole{
	Detective,
	Assassin,
	Angel,
	Escort,
	Sadboy,
	Citizen,
}

func (gr GameRole) String() string {
	return gameRoles[gr]
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
	selected := allGameRoles[g.rng.IntN(len(allGameRoles))]
	return selected
}

func (g *Game) assignRoles() PlayerRoles {
	playerRoles := make(PlayerRoles)
	for _, player := range g.players {
		role := g.assignRole(playerRoles)
		playerRoles[role] = player
	}
	return playerRoles
}
