package main

import (
	"strings"
	"testing"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestAllHumanPlayersSameName(t *testing.T) {
	table := createTestTable(0)
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
