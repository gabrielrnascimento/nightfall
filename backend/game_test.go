package main

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"testing"
)

var players = []string{"Alice", "Bob", "Charlie", "David", "Ellie", "Frank"}
var playerRoles = make(PlayerRoles)

func newGame(seed uint64) *Game {
	rng := rand.New(rand.NewPCG(seed, seed))
	return &Game{rng, players}
}

func Test_GameRole(t *testing.T) {
	t.Run("String() should return correct name", func(t *testing.T) {
		if Detective.String() != "Detective" {
			t.Errorf("want Detective, got %s", Detective.String())
		}
		if Sadboy.String() != "Sad Boy" {
			t.Errorf("want Sad Boy, got %s", Sadboy.String())
		}
	})

	t.Run("MarshalText() should return correct bytes", func(t *testing.T) {
		got, err := Assassin.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText failed: %v", err)
		}
		if string(got) != "Assassin" {
			t.Errorf("want Assassin, got %s", string(got))
		}
	})

	t.Run("should marshal as JSON map key correctly", func(t *testing.T) {
		roles := PlayerRoles{
			Assassin: "Alice",
		}
		data, err := json.Marshal(roles)
		if err != nil {
			t.Fatalf("JSON marshal failed: %v", err)
		}

		want := `{"Assassin":"Alice"}`
		if string(data) != want {
			t.Errorf("want %s, got %s", want, string(data))
		}
	})
}

func Test_Game_assignRole(t *testing.T) {
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

	t.Run("should always return Angel if there is a Detective", func(t *testing.T) {
		game := newGame(42)
		playerRoles[Assassin] = players[0]
		playerRoles[Detective] = players[1]

		role := game.assignRole(playerRoles)

		if role != Angel {
			t.Errorf("expected Angel got %s", role)
		}
	})

	t.Run("should always return Escort if there is an Angel", func(t *testing.T) {
		game := newGame(42)
		playerRoles[Assassin] = players[0]
		playerRoles[Detective] = players[1]
		playerRoles[Angel] = players[2]

		role := game.assignRole(playerRoles)

		if role != Escort {
			t.Errorf("expected Escort got %s", role)
		}
	})

	t.Run("should always return Sad Boy if there is an Escort", func(t *testing.T) {
		game := newGame(42)
		playerRoles[Assassin] = players[0]
		playerRoles[Detective] = players[1]
		playerRoles[Angel] = players[2]
		playerRoles[Escort] = players[3]

		role := game.assignRole(playerRoles)

		if role != Sadboy {
			t.Errorf("expected Sad Boy got %s", role)
		}
	})

	t.Run("should return Citizen if all other roles are filled", func(t *testing.T) {
		game := newGame(42)
		playerRoles[Assassin] = players[0]
		playerRoles[Detective] = players[1]
		playerRoles[Angel] = players[2]
		playerRoles[Escort] = players[3]
		playerRoles[Sadboy] = players[4]

		role := game.assignRole(playerRoles)

		if role != Citizen {
			t.Errorf("expected Citizen got %s", role)
		}
	})
}

func Test_Game_Start(t *testing.T) {
	t.Run("should assign roles to all players", func(t *testing.T) {
		game := newGame(42)

		got := game.Start()

		want := PlayerRoles{
			Assassin:  "Alice",
			Detective: "Bob",
			Angel:     "Charlie",
			Escort:    "David",
			Sadboy:    "Ellie",
			Citizen:   "Frank",
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
