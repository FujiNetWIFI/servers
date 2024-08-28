package main

import (
	"strings"
	"testing"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestAllHumanPlayersSameName(t *testing.T) {
	table, _ := createTestTable(0, 0)
	names := [...]string{"aaaaaaaa", "abaaaaaa", "aabaaaaa", "aaabaaaa", "aaaabaaa", "aaaaabaa"}
	newNames := [...]string{"aaaaaaaa", "abaaaaaa", "aabaaa z", "aaabaa y", "aaaaba x", "aaaaab w"}

	players := make([]string, 6)
	for i := 0; i < 6; i++ {
		players[i] = "/?player=" + names[i] + table
	}

	// Each player joines in turn
	for i := 0; i < 6; i++ {
		state := c(players[i], apiState).(*GameState)

		// Spec's state should show round = 1
		if !strings.EqualFold(state.Players[0].Name, newNames[i]) {
			t.Fatal("Expect player", i+1, "name", state.Players[0].Name, "to be "+newNames[i])
		}
	}

}

func TestGameEndMessageWith1Winner(t *testing.T) {
	_, players := createTestTable(0, 4)

	// 4 Players join game to max out server with bots
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Start game with state call of first player
	c(players[0], apiState)

	// Play out game to with single winner
	state := c(players[0]+"&skipToEnd=1", apiState).(*GameState)

	// Expect game to be overxw
	if !strings.HasPrefix(state.Prompt, "p1 won") {
		t.Fatal("Expect game over message to match:", state.Prompt)
	}
}

func TestGameEndMessageWith2WayTie(t *testing.T) {
	_, players := createTestTable(0, 4)

	// 4 Players join game to max out server with bots
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Start game with state call of first player
	c(players[0], apiState)

	// Play out game to with single winner
	state := c(players[0]+"&skipToEnd=2", apiState).(*GameState)

	// Expect game to be overxw
	if !strings.HasPrefix(state.Prompt, "p1 and p2 tied") {
		t.Fatal("Expect game over message to match:", state.Prompt)
	}
}

func TestGameEndMessageWith3WayTie(t *testing.T) {
	_, players := createTestTable(0, 4)

	// 4 Players join game to max out server with bots
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Start game with state call of first player
	c(players[0], apiState)

	// Play out game to with single winner
	state := c(players[0]+"&skipToEnd=3", apiState).(*GameState)

	// Expect game to be overxw
	if !strings.HasPrefix(state.Prompt, "3 players tied") {
		t.Fatal("Expect game over message to match:", state.Prompt)
	}
}
