package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBotsNeverPlayWithoutHumans: a table of bots must not deal or play hands
// unless a human player is seated
func TestBotsNeverPlayWithoutHumans(t *testing.T) {
	useFastTimers(t)
	initializeGameServer()
	state := createGameState(4, false) // 4 bots, no humans, no test flag
	state.TableId = "no-humans"

	for i := 0; i < 200; i++ {
		state.RunGameLogic()
	}
	assert.Equal(t, 0, state.GamesPlayed, "bots must not start hands without a human")
	assert.Equal(t, 0, state.Round, "table stays parked")
	for _, p := range state.Players {
		assert.Equal(t, STARTING_PURSE, p.Purse, "no chips should move")
	}
}

// TestBotsStopWhenLastHumanLeaves: bots abort the current hand and stop dealing
// when the only human leaves; play resumes when a human returns
func TestBotsStopWhenLastHumanLeaves(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 3, 91)

	human := newSimClient(server.URL, tableId, "Solo", policyCallAny)
	stop := startClients(t, human)

	// Wait until a hand is underway with the human dealt in
	deadline := 0
	for inHand := false; !inHand; deadline++ {
		require.Less(t, deadline, 5000, "hand never started")
		withTable(tableId, func(state *GameState) { inHand = state.Round >= 1 && !state.gameOver })
		time.Sleep(3 * time.Millisecond)
	}

	// Human leaves; stop their polling first so they cannot rejoin
	stop()
	_, err := human.get(fmt.Sprintf("/leave?table=%s&player=Solo", tableId))
	require.NoError(t, err)

	// Simulate the hub ticker continuing to drive the table
	var gamesAfterLeave int
	withTable(tableId, func(state *GameState) {
		state.moveExpires = state.moveExpires.Add(-time.Hour) // expire any timer
		for i := 0; i < 500; i++ {
			state.RunGameLogic()
		}
		gamesAfterLeave = state.GamesPlayed
		assert.True(t, state.gameOver || state.Round == 0, "no live hand without humans")
		for _, p := range state.Players {
			if p.isBot {
				assert.NotEqual(t, STATUS_PLAYING, p.Status, "bots must not be mid-hand")
			}
		}
	})

	// Keep ticking: the game count must not advance
	withTable(tableId, func(state *GameState) {
		for i := 0; i < 500; i++ {
			state.RunGameLogic()
		}
		assert.Equal(t, gamesAfterLeave, state.GamesPlayed, "bots must not start new hands")
	})

	// A human returning brings the table back to life
	human2 := newSimClient(server.URL, tableId, "Back", policyCallAny)
	stop2 := startClients(t, human2)
	defer stop2()
	forceNextHand(tableId)
	resumed := false
	waited := time.Now().Add(30 * time.Second)
	for !resumed && time.Now().Before(waited) {
		withTable(tableId, func(state *GameState) {
			resumed = state.GamesPlayed > gamesAfterLeave && state.gameOver
		})
		time.Sleep(3 * time.Millisecond)
	}
	assert.True(t, resumed, "play resumes when a human is back")
}
