package main

import (
	"math/rand/v2"
	"testing"
)

var players = []string{"Alice", "Bob", "Charlie", "David"}
var playerRoles = make(PlayerRoles)

func newGame(seed uint64) *Game {
	rng := rand.New(rand.NewPCG(seed, seed))
	return &Game{rng, players}
}

func Test_Game(t *testing.T) {
	t.Run("should return a valid role", func(t *testing.T) {
		game := newGame(42)

		got := game.assignRole(playerRoles)

		if _, ok := gameRoles[got]; !ok {
			t.Errorf("got invalid role: %v", got)
		}
	})

	t.Run("should be deterministic", func(t *testing.T) {
		seed := uint64(12345)
		game1 := newGame(seed)
		game2 := newGame(seed)

		for range 100 {
			if game1.assignRole(playerRoles) != game2.assignRole(playerRoles) {
				t.Error("game role assignment is not deterministic")
			}
		}
	})

	t.Run("should always return Assassin role first", func(t *testing.T) {
		game := newGame(42)

		role := game.assignRole(playerRoles)

		if role != Assassin {
			t.Errorf("expected Assassin got %s", role)
		}
	})

	t.Run("should always return Detective if there is an Assassin", func(t *testing.T) {
		game := newGame(42)
		playerRoles[Assassin] = players[0]

		role := game.assignRole(playerRoles)

		if role != Detective {
			t.Errorf("expected Detective got %s", role)
		}
	})

}
