package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestSpecAlwaysSeesPlayerNames(t *testing.T) {
	table, players := createTestTable(0, 6)

	spec := "/?player=spec" + table

	// Spec joins
	c(spec, apiState)

	// Other players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Get state
	state := c(spec, apiState).(*GameState)

	// Spec's state should show round = 1
	if state.Round != 1 {
		t.Fatal("Expect round to be 1 after all ready")
	}

	// Spec should be the last player as viewing
	player := state.Players[len(state.Players)-1]
	if player.Name != "spec" || !player.isViewing || player.Scores[0] != SCORE_VIEWING {
		t.Fatal("Expect SPEC player to be the last player, set to viewing, and score[0]=score_viewing")
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for i, player := range players {

		state = c(player, apiState).(*GameState)
		if state.Prompt != PROMPT_YOUR_TURN {
			t.Fatal("Player %i expected to see YOUR TURN prompt", i)
		}

		if state.ActivePlayer != 0 {
			t.Fatal("Player %i expected to see activePlayer 0", i)
		}

		state = c(spec, apiState).(*GameState)
		if state.Prompt == PROMPT_YOUR_TURN {
			t.Fatal("Spectator expected to see NOT see YOUR TURN prompt")
		}

		// Score a move to go to next player
		c(player, apiScore, []gin.Param{{Key: "index", Value: "0"}})
	}

}

func TestBotsLeavingForPlayersWithSpec(t *testing.T) {
	table, players := createTestTable(2, 6)

	spec := "/?player=spec" + table

	// Spec joins
	state := c(spec, apiState).(*GameState)

	// Expect 3 players (2 bots + spec)
	if len(state.Players) != 3 {
		t.Fatal("Expect 3 players (2 bots + spec)n")
	}

	// Other players join game
	for _, player := range players {
		state = c(player, apiState).(*GameState)
	}

	// There should be 7 players, all humans
	if len(state.Players) != 7 {
		t.Fatal("Expect 7 players after all humans join")
	}

	// There should be no bots left
	if slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		t.Fatal("Expect no bots left after all humans join")
	}

	// 2 players leave, leaving room for one bot to join back
	c(players[4], apiLeave)
	c(players[5], apiLeave)

	// 6th player should be a bot
	state = c(spec, apiState).(*GameState)

	if len(state.Players) != 6 {
		t.Fatal("Expect 6 players after 2 humans leave")
	}

	if !state.Players[5].isBot {
		t.Fatal("Expect 6th player to be a bot after 2 humans leave")
	}

	// 2 players join back
	c(players[4], apiState)
	state = c(players[5], apiState).(*GameState)

	// There should be 7 players, all humans
	if len(state.Players) != 7 {
		t.Fatal("Expect 7 players after 2 humans join back")
	}

	// There should be no bots left
	if slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		t.Fatal("Expect no bots left after 2 humans join back")
	}

	// All 6 players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Get state
	state = c(spec, apiState).(*GameState)

	// Spec's state should show round = 1
	if state.Round != 1 {
		t.Fatal("Expect round to be 1 after all ready")
	}

	// Spec should be the last player as viewing
	player := state.Players[len(state.Players)-1]
	if player.Name != "spec" || !player.isViewing || player.Scores[0] != SCORE_VIEWING {
		t.Fatal("Expect SPEC player to be the last player, set to viewing, and score[0]=score_viewing")
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for i, player := range players {

		state = c(player, apiState).(*GameState)
		if state.Prompt != PROMPT_YOUR_TURN {
			t.Fatal("Player", i, "expected to see YOUR TURN prompt")
		}

		if state.ActivePlayer != 0 {
			t.Fatal("Player", i, "expected to see activePlayer 0")
		}

		state = c(spec, apiState).(*GameState)
		if state.Prompt == PROMPT_YOUR_TURN {
			t.Fatal("Spectator expected to see NOT see YOUR TURN prompt")
		}

		// Score a move to go to next player
		c(player, apiScore, []gin.Param{{Key: "index", Value: "0"}})
	}

	// 2 players leave midgame
	c(players[4], apiLeave)
	c(players[5], apiLeave)

	// Get state from spec pov
	state = c(spec, apiState).(*GameState)

	// There should be no bots left
	if slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		t.Fatal("Expect no bots after 2 humans leave midgame")
	}

	// Spec should still be the last player and still viewing.
	player = state.Players[len(state.Players)-1]
	if player.Name != "spec" || !player.isViewing || player.Scores[0] != SCORE_VIEWING {
		t.Fatal("Expect SPEC player to be the last player, set to viewing, and score[0]=score_viewing")
	}

	// The remaining players except one leave, ending the game
	c(players[1], apiLeave)
	c(players[2], apiLeave)
	c(players[3], apiLeave)

	state = c(players[0], apiState).(*GameState)

	// Game should be aborted early
	if state.Prompt != PROMPT_GAME_ABORTED {
		t.Fatal("Player 1 expects the aborted game message")
	}

	// Wait until abort message goes away
	for true {
		time.Sleep(10 * time.Millisecond)
		state = c(players[0], apiState).(*GameState)

		// Game should be aborted early
		if state.Prompt != PROMPT_GAME_ABORTED {
			break
		}
	}

	// The game should be over and back in the lobby, with 2 human players (p1 and spec)
	player = state.Players[0]
	if player.Name != "p1" || player.isViewing || player.Scores[0] != SCORE_UNREADY {
		t.Fatal("Expect 1st player to be p1, unready, not viewing")
	}
	player = state.Players[1]
	if player.Name != "spec" || player.isViewing || player.Scores[0] != SCORE_UNREADY {
		t.Fatal("Expect 2nd player to be spec, unready, not viewing")
	}

	for i := 2; i < 4; i++ {
		player = state.Players[i]
		if !player.isBot || player.isViewing || player.Scores[0] != SCORE_READY {
			t.Fatal("Expect", i+1, "player to be bot, ready, not viewing")
		}
	}

}

func TestPlayersSeeSpecJoinMidGame(t *testing.T) {
	table, players := createTestTable(3, 3)

	spec := "/?player=spec" + table

	// players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Get state
	state := c(players[0], apiState).(*GameState)
	if state.Round != 1 {
		t.Fatal("Expected round to be 1 after all players ready up")
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for i, player := range players {

		state = c(player, apiState).(*GameState)
		if state.Prompt != PROMPT_YOUR_TURN {
			t.Fatal("Player %i expected to see YOUR TURN prompt", i)
		}

		if state.ActivePlayer != 0 {
			t.Fatal("Player %i expected to see activePlayer 0", i)
		}

		// Score a move to go to next player
		c(player, apiScore, []gin.Param{{Key: "index", Value: "0"}})
	}

	// Spec joins
	c(spec, apiState)

	// Get player 1 state
	state = c(players[0], apiState).(*GameState)

	// Spec's state should show round = 1
	if state.Players[len(state.Players)-1].Scores[0] != SCORE_VIEWING {
		t.Fatal("Expect last player to be the spectactor")
	}

}

func TestSpecWatchesEntireGame(t *testing.T) {
	table, players := createTestTable(0, 3)

	spec := "/?player=spec" + table

	// Spec joins
	c(spec, apiState)

	// Other players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Get state
	state := c(spec, apiState).(*GameState)

	// Spec's state should show round = 1
	if state.Round != 1 {
		t.Fatal("Expect round to be 1 after all ready")
	}

	// Spec should be the last player as viewing
	player := state.Players[len(state.Players)-1]
	if player.Name != "spec" || !player.isViewing || player.Scores[0] != SCORE_VIEWING {
		t.Fatal("Expect SPEC player to be the last player, set to viewing, and score[0]=score_viewing")
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for round := 1; round <= 13; round++ {
		for _, player := range players {

			state = c(player, apiState).(*GameState)

			if state.Round != round {
				t.Fatal("Expected to be in round", round, "instead of", state.Round)
			}

			// Score the first value>0 for this player, or 0 as a fallback
			scoreIndex := -1
			for i := 0; i < len(state.ValidScores); i++ {
				if state.ValidScores[i] >= 0 {
					scoreIndex = i
				}
				if state.ValidScores[i] > 0 {
					break
				}

			}
			c(player, apiScore, []gin.Param{{Key: "index", Value: strconv.Itoa(scoreIndex)}})
		}
	}
}

func TestEntireGameOnePlayerLeavesMidway(t *testing.T) {
	_, players := createTestTable(0, 3)

	// Other players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for round := 1; round <= 13; round++ {
		// Player leaves at start of round 6
		if round == 6 {
			c(players[0], apiLeave)
			players = players[1:]
		}
		for _, player := range players {

			state := c(player, apiState).(*GameState)

			if state.Round != round {
				t.Fatal("Expected to be in round", round, "instead of", state.Round)
			}

			// Score the first value>0 for this player, or 0 as a fallback
			scoreIndex := -1
			for i := 0; i < len(state.ValidScores); i++ {
				if state.ValidScores[i] >= 0 {
					scoreIndex = i
				}
				if state.ValidScores[i] > 0 {
					break
				}

			}
			c(player, apiScore, []gin.Param{{Key: "index", Value: strconv.Itoa(scoreIndex)}})
		}
	}
}

func TestEntireGameOnePlayerLeavesComesBack(t *testing.T) {
	_, players := createTestTable(0, 3)

	// Other players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Loop through each player's turn and confirm they see activePlayer 0 the spec does not see "Your turn", while the player does
	for round := 1; round <= 13; round++ {
		// Player leaves at start of round 6. When they come back it will be as a spec, so their score call will be ignored
		if round == 6 {
			c(players[0], apiLeave)
		}
		for _, player := range players {

			state := c(player, apiState).(*GameState)

			if state.Round != round {
				t.Fatal("Expected to be in round", round, "instead of", state.Round)
			}

			// Score the first value>0 for this player, or 0 as a fallback

			if state.Viewing == 0 {
				scoreIndex := -1
				for i := 0; i < len(state.ValidScores); i++ {
					if state.ValidScores[i] >= 0 {
						scoreIndex = i
					}
					if state.ValidScores[i] > 0 {
						break
					}

				}
				c(player, apiScore, []gin.Param{{Key: "index", Value: strconv.Itoa(scoreIndex)}})
			}
		}
	}
}

func TestSpecIsViewerOnGameStart(t *testing.T) {
	table, players := createTestTable(0, 6)

	spec := "/?player=spec" + table

	// Spec joins
	c(spec, apiState)

	// Other players join game
	for _, player := range players {
		c(player, apiState)
	}

	// All players ready up
	for _, player := range players {
		c(player, apiReady)
	}

	// Get state from spec
	state := c(spec, apiState).(*GameState)

	// Spec's state should show round = 1
	if state.Round != 1 {
		t.Fatal("Expect round to be 1 after all ready")
	}

	// Spec's state should show viewing = 1
	if state.Viewing != 1 {
		t.Fatal("Expect spec to be viewing")
	}

}
