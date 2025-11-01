package main

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

//////////////////////////////////////////////////////////////////////////////////////////
// COMMON TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestBotPlayerReadyToggle(t *testing.T) {
	_, players := createTestTable(1, 1)

	// Set wait time longer than 0 so ready lasts multiple requests
	START_WAIT_TIME = time.Second * 5
	START_WAIT_TIME_ONE_PLAYER = time.Second * 5

	p1 := players[0]

	// Join game
	c(p1, apiState)

	// Ready up
	c(p1, apiReady)

	state := c(p1, apiState).(*GameState)
	if !strings.HasPrefix(state.Prompt, PROMPT_STARTING_IN) {
		t.Fatal("Expected starting countdown after player 1 readies. Got Prompt:",state.Prompt)
	}

	// Toggle Ready up
	c(p1, apiReady)

	state = c(p1, apiState).(*GameState)
	if state.Prompt != PROMPT_WAITING_ON_READY {
		t.Fatal("Expected waiting prompt as player un-redied")
	}
}



func TestBotGameStartOnePlayer(t *testing.T) {
	_, players := createTestTable(1, 1)

	p1 := players[0]

	// Join game
	c(p1, apiState)

	// Ready up
	c(p1, apiReady)

	// Expect to place ships
	state := c(p1, apiState).(*GameState)
	if  (state.Status != STATUS_PLACE_SHIPS) {
		t.Fatal("Expected Status STATUS_PLACE_SHIPS. Got:",state.Status)
	}

	// Place ships
	// xxxxx
	// xxxx
	// xxx
	// xxx
	// xx
	c(p1, apiPlace,[]gin.Param{{Key: "ships", Value: "/api/place/0,10,20,30,40"}})

	state = c(p1, apiState).(*GameState)
	if  (state.Status != STATUS_GAMESTART) {
		t.Fatal("Expected Status STATUS_GAMESTART. Got:",state.Status)
	}
	if  (state.ActivePlayer != 0) {
		t.Fatal("Expected P1 to be activePlayer. Got:",state.ActivePlayer)
	}
}

func TestFullGameTwoPlayers(t *testing.T) {
	_, players := createTestTable(0, 2)

	p1 := players[0]
	p2 := players[1]

	// Join game
	c(p1, apiState)
	c(p2, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)

	// Expect to place ships
	state := c(p1, apiState).(*GameState)
	if  (state.Status != STATUS_PLACE_SHIPS) {
		t.Fatal("Expected Status STATUS_PLACE_SHIPS. Got:",state.Status)
	}

	// Both players place ships
	// xxxxx
	// xxxx
	// xxx
	// xxx
	// xx
	c(p1, apiPlace,[]gin.Param{{Key: "ships", Value: "0,10,20,30,40"}})
	c(p2, apiPlace,[]gin.Param{{Key: "ships", Value: "0,10,20,30,40"}})

	state = c(p1, apiState).(*GameState)
	if  (state.Status != STATUS_GAMESTART) {
		t.Fatal("Expected Status STATUS_GAMESTART. Got:",state.Status)
	}
	if  (state.ActivePlayer != 0) {
		t.Fatal("Expected P1 to be activePlayer. Got:",state.ActivePlayer)
	}
	if  (state.PlayerStatus != PLAYER_STATUS_PLAYING) {
		t.Fatal("Expected P1 PlayerStatus to be playing. Got:",state.PlayerStatus)
	}

	// Player 2 attacks out of turn. Expect game status to remain the same
	state = c(p2, apiAttack,[]gin.Param{{Key: "pos", Value: "99"}}).(*GameState)
	if (state.Status != STATUS_GAMESTART) {
		t.Fatal("Expected Status STATUS_GAMESTART. Got:",state.Status)
	}

	// P1 Attack and miss
	state = c(p1, apiAttack,[]gin.Param{{Key: "pos", Value: "99"}}).(*GameState)
	if (state.Status != STATUS_MISS) {
		t.Fatal("Expected Status STATUS_MISS. Got:",state.Status)
	}

	// P2 Attack and hit
	state = c(p2, apiAttack,[]gin.Param{{Key: "pos", Value: "40"}}).(*GameState)
	if (state.Status != STATUS_HIT) {
		t.Fatal("Expected Status STATUS_HIT. Got:",state.Status)
	}

	// P1 Attack same spot - should allow a second attack since this was already attacked
	state = c(p1, apiAttack,[]gin.Param{{Key: "pos", Value: "99"}}).(*GameState)
	if (state.ActivePlayer != 0) {
		t.Fatal("Expected ActivePlayer to stay at 0. Got:",state.ActivePlayer)
	}

	// P1 Attack new spot
	state = c(p1, apiAttack,[]gin.Param{{Key: "pos", Value: "98"}}).(*GameState)
	if (state.ActivePlayer != 1) {
		t.Fatal("Expected ActivePlayer to be 1. Got:",state.ActivePlayer)
	}

	// P2 Attack and sink P1's XX ship
	state = c(p2, apiAttack,[]gin.Param{{Key: "pos", Value: "41"}}).(*GameState)
	if (state.Status != STATUS_SUNK) {
		t.Fatal("Expected Status STATUS_SUNK. Got:",state.Status)
	}
	if (state.Players[1].ShipsLeft[4] != 0) {
		t.Fatal("Expected ShipsLeft[4] to be 0. Got:",state.Players[0].ShipsLeft[4])
	}

	// Now take turns attacking until P2 inevitably wins
	for i := 0; i < 33; i++ {
		c(p1, apiAttack,[]gin.Param{{Key: "pos", Value: strconv.Itoa(i)}})
		state =c(p2, apiAttack,[]gin.Param{{Key: "pos", Value: strconv.Itoa(i)}}).(*GameState)
	}

	// The game has finished!
	if (state.Status != STATUS_GAMEOVER) {
		t.Fatal("Expected Status STATUS_GAMEOVER. Got:",state.Status)
	}

}