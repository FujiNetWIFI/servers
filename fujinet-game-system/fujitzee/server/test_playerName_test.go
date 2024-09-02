package main

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestPlayerPOV(t *testing.T) {
	_, players := createTestTable(2, 4)

	// 4 Players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Check all player's pov is of player 1
	for _, player := range players {
		state := c(player+"&pov=p1", apiState).(*GameState)
		if state.Players[0].id != "p1" {
			t.Fatal("Expect first player name to be p2:", state.Players[0].Name)
		}

		if !strings.HasPrefix(state.Prompt, "p") {
			t.Fatal("Expect prompt to be pN's turn:", state.Prompt)
		}
	}

	// Check player move time is >0
	state := c(players[0]+"&pov=p1", apiState).(*GameState)
	if state.MoveTime == 0 {
		t.Fatal("Expected p1's move time to be >0:", state.MoveTime)
	}

	// Move for player 1
	state = c(players[0]+"&pov=p1", apiScore, []gin.Param{{Key: "index", Value: "0"}}).(*GameState)
	if state.ActivePlayer != 1 {
		t.Fatal("Expected active player to be p2 (1):", state.ActivePlayer)
	}

	if state.MoveTime == 0 {
		t.Fatal("Expected p2's move time to be >0:", state.MoveTime)
	}

}
