package main

import (
	"testing"

	"github.com/gin-gonic/gin"
)

//////////////////////////////////////////////////////////////////////////////////////////
// TESTS
//////////////////////////////////////////////////////////////////////////////////////////

func TestPlayerLeavesMidGameNotTheirTurn(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)
	gc[*GameState](p3, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)
	gc[*GameState](p3, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 1 leaves the game
	gc[string](p1, apiLeave)

	// Player 2's turn.
	state = gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1 LEFT!")
	}

}

func TestMiddlePlayerLeavesOnTheirTun(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)
	gc[*GameState](p3, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)
	gc[*GameState](p3, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 2 leaves the game
	gc[string](p2, apiLeave)

	// Player 3's turn.
	state = gc[*GameState](p3, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2 LEFT!")
	}

}

func TestLastPlayerLeavesOnTheirTun(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)
	gc[*GameState](p3, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)
	gc[*GameState](p3, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	gc[*GameState](p2, apiState)
	gc[*GameState](p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 3's turn.
	state := gc[*GameState](p3, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2's turn!")
	}

	// Player 3 leaves the game
	gc[string](p3, apiLeave)

	// Player 1's turn.
	state = gc[*GameState](p1, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 1 expected to be active after P3 LEFT!")
	}

}

func TestMiddlePlayerLeavesOnTheirTurn(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table
	p3 := "/?player=p3" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)
	gc[*GameState](p3, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)
	gc[*GameState](p3, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state := gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 2 leaves the game
	gc[string](p2, apiLeave)

	// Player 3's turn.
	state = gc[*GameState](p3, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 3 expected to be active after P2 LEFT!")
	}

}

func Test2PlayerGame2ndPlayerLeavesOnTheirTurn(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	gc[*GameState](p2, apiState)
	gc[*GameState](p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2 leaves the game
	gc[string](p2, apiLeave)

	// Player 1 check
	state := gc[*GameState](p1, apiState)
	if state.Round != ROUND_GAMEOVER {
		t.Fatal("Player 1 expects the game to be over")
	}

}

func Test2PlayerGame1stPlayerLeavesOnTheirTurn(t *testing.T) {
	table := createTestTable(0)

	p1 := "/?player=p1" + table
	p2 := "/?player=p2" + table

	// Join game in order
	gc[*GameState](p1, apiState)
	gc[*GameState](p2, apiState)

	// Ready up
	gc[*GameState](p1, apiReady)
	gc[*GameState](p2, apiReady)

	// Player 1's turn - score first value
	gc[*GameState](p1, apiState)
	gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn - score first value
	gc[*GameState](p2, apiState)
	gc[*GameState](p2, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 1 leaves
	gc[string](p1, apiLeave)

	// Player 2 check
	state := gc[*GameState](p2, apiState)
	if state.Round != ROUND_GAMEOVER {
		t.Fatal("Player 2 expects the game to be over")
	}

}
