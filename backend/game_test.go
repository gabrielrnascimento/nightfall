package main

import (
	"math/rand/v2"
	"testing"
)

func newGame(seed uint64) *Game {
	rng := rand.New(rand.NewPCG(seed, seed))
	return &Game{rng}
}

func Test_Game(t *testing.T) {
	t.Run("should return a valid role", func(t *testing.T) {
		game := newGame(42)

		got := game.assignRole()

		if _, ok := gameRoles[got]; !ok {
			t.Errorf("got invalid role: %v", got)
		}
	})

	t.Run("should be deterministic", func(t *testing.T) {
		seed := uint64(12345)
		game1 := newGame(seed)
		game2 := newGame(seed)

		for range 100 {
			if game1.assignRole() != game2.assignRole() {
				t.Error("game role assignment is not deterministic")
			}
		}
	})

	t.Run("should return all possible roles at least once", func(t *testing.T) {
		game := newGame(42)

		results := make(map[GameRole]int)
		for range 100 {
			role := game.assignRole()
			results[role]++
		}

		for role := range gameRoles {
			if results[role] == 0 {
				t.Errorf("%v role never assigned", role)
			}
		}
	})

}
