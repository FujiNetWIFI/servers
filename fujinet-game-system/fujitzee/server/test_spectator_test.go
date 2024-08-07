package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestSpecAlwaysSeesPlayerNames(t *testing.T) {
	players, table := createFullTestTable()

	spec := "/?player=spec" + table

	// Join game in order
	for _, player := range players {
		gc[*GameState](player, apiState)
	}

	// Spec joins
	gc[*GameState](spec, apiState)

	// All players ready up
	for _, player := range players {
		gc[*GameState](player, apiReady)
	}

	// Get state
	state := gc[*GameState](spec, apiState)

	// Spec's state should show round = 1
	if state.Round != 1 {
		t.Fatal("Expect round to be 1 after all ready")
	}

	// Spec should not be in the player list
	if slices.ContainsFunc(state.Players, func(p Player) bool { return p.Name == "spec" }) {
		t.Fatal("Expect SPEC player to not be in the players list")
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for i, player := range players {

		state = gc[*GameState](player, apiState)
		if state.Prompt != PROMPT_YOUR_TURN {
			t.Fatal("Player %i expected to see YOUR TURN prompt", i)
		}

		if state.ActivePlayer != 0 {
			t.Fatal("Player %i expected to see activePlayer 0", i)
		}

		state = gc[*GameState](spec, apiState)
		if state.Prompt == PROMPT_YOUR_TURN {
			t.Fatal("Spectator expected to see NOT see YOUR TURN prompt")
		}

		// Score a move to go to next player
		gc[*GameState](player, apiScore, []gin.Param{{Key: "index", Value: "0"}})
	}

}
