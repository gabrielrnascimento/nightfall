package main

import "math/rand/v2"

// MARK: GameRole
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
	Detective: "detective",
	Assassin:  "assassin",
	Angel:     "angel",
	Escort:    "escort",
	Sadboy:    "sadboy",
	Citizen:   "citizen",
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

// MARK: Game
type Game struct {
	rng *rand.Rand
}

func (g *Game) assignRole() GameRole {
	return allGameRoles[g.rng.IntN(len(allGameRoles))]
}
