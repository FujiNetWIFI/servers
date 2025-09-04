package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestPlayerLeavesMidGameNotTheirTurn(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)
	c(p3, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)
	c(p3, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := c(p2, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!", state.ActivePlayer)
	}

	// Player 1 leaves the game
	c(p1, apiLeave)

	// Player 2's turn.
	state = c(p2, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1 LEFT!")
	}

}

func TestMiddlePlayerLeavesOnTheirTun(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)
	c(p3, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)
	c(p3, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := c(p2, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 2 leaves the game
	c(p2, apiLeave)

	// Player 3's turn.
	state = c(p3, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2 LEFT!")
	}

}

func TestLastPlayerLeavesOnTheirTun(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)
	c(p3, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)
	c(p3, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	c(p2, apiState)
	c(p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 3's turn.
	state := c(p3, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2's turn!")
	}

	// Player 3 leaves the game
	c(p3, apiLeave)

	// Player 1's turn.
	state = c(p1, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 1 expected to be active after P3 LEFT!")
	}

}

func TestMiddlePlayerLeavesOnTheirTurn(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)
	c(p3, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)
	c(p3, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := c(p2, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 2 leaves the game
	c(p2, apiLeave)

	// Player 3's turn.
	state = c(p3, apiState).(*GameState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2 LEFT!")
	}

}

func Test2PlayerGame2ndPlayerLeavesOnTheirTurn(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	c(p2, apiState)
	c(p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2 leaves the game
	c(p2, apiLeave)

	// Player 1 check
	state := c(p1, apiState).(*GameState)
	if state.Round != ROUND_GAMEOVER {
		t.Fatal("Player 1 expects the game to be over")
	}

}

func Test2PlayerGame1stPlayerLeavesOnTheirTurn(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	c(p2, apiState)
	c(p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 1 leaves
	c(p1, apiLeave)

	// Player 2 check
	state := c(p2, apiState).(*GameState)
	if state.Round != ROUND_GAMEOVER {
		t.Fatal("Player 2 expects the game to be over")
	}

}

func Test1PlayerAbortsGameSecondRejoins(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	c(p2, apiState)
	c(p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 1 leaves
	c(p1, apiLeave)

	// Player 2 gets state
	state := c(p2, apiState).(*GameState)

	// Check tables
	tables := c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers != 1 {
		t.Fatal("Table should show 1 player")
	}

	if state.Prompt != PROMPT_GAME_ABORTED {
		t.Fatal("Player 2 expects the aborted game message")
	}

	// Player 2 leaves
	c(p2, apiLeave)

	// Check tables
	tables = c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers != 0 {
		t.Fatal("Table should show 0 players")
	}

	// Player 2 joins
	state = c(p2, apiState).(*GameState)

	if state.Round != ROUND_LOBBY {
		t.Fatal("Player 2 expects to be in the lobby")
	}
}

func Test2PlayersLeave1JoinsBack(t *testing.T) {
	table, _ := createTestTable(0, 0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	c(p1, apiState)
	c(p2, apiState)

	// Ready up
	c(p1, apiReady)
	c(p2, apiReady)

	// Player 1's turn - score first value
	c(p1, apiState)
	c(p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	c(p2, apiState)
	c(p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Players leave
	c(p1, apiLeave)

	// Check tables
	tables := c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers != 1 {
		t.Fatal("Table should show 1 player")
	}

	c(p2, apiLeave)

	// Check tables
	tables = c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers != 0 {
		t.Fatal("Table should show 0 players")
	}

	// Player 1 joins
	state := c(p1, apiState).(*GameState)

	if state.Round != ROUND_LOBBY {
		t.Fatal("Player 1 expects to be in the lobby")
	}
}

func TestBotGamePlayerLeavesThenJoins(t *testing.T) {
	_, players := createTestTable(4, 1)

	p1 := players[0]

	// Join game
	c(p1, apiState)

	// Check tables
	tables := c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers != 1 {
		t.Fatal("Table should show 1 player")
	}

	// Ready up
	c(p1, apiReady)

	// Player 1 leaves
	c(p1, apiLeave)

	// Check tables
	tables = c(p1, apiTables).([]GameTable)
	if tables[0].CurPlayers > 0 {
		t.Fatal("Table should show 0 current players")
	}

	// Player 1's joins again
	state := c(p1, apiState).(*GameState)

	// At this point the game should be in the lobby state
	if state.Round != ROUND_LOBBY {
		t.Fatal("Player 1 expects to be in game waitiny lobby after leaving bot match")
	}

}

func TestBotGamePlayerReadyToggle(t *testing.T) {
	_, players := createTestTable(4, 1)

	// Set wait time longer than 0 so ready lasts multiple requests
	START_WAIT_TIME = time.Second * 10
	START_WAIT_TIME_ONE_PLAYER = time.Second * 10

	p1 := players[0]

	// Join game
	c(p1, apiState)

	// Ready up
	c(p1, apiReady)

	state := c(p1, apiState).(*GameState)
	if !strings.HasPrefix(state.Prompt, PROMPT_STARTING_IN) {
		t.Fatal("Expected starting countdown after player 1 readies")
	}

	// Toggle Ready up
	c(p1, apiReady)

	state = c(p1, apiState).(*GameState)
	if state.Prompt != PROMPT_WAITING_ON_READY {
		t.Fatal("Expected waiting prompt as player un-redied")
	}
}
