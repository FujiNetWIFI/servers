package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func gc[K comparable](path string, f func(*gin.Context), opt_params ...[]gin.Param) K {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", path, nil)
	if len(opt_params) > 0 {
		c.Params = opt_params[0]
	}
	f(c)
	r, _ := c.Get("result")
	return r.(K)
}

func TestPlayerLeavesMidGame(t *testing.T) {
	setTestMode()

	createTable("t1", "t1", 0, false)
	p1 := "/?player=p1&table=t1"
	p2 := "/?player=p2&table=t1"
	p3 := "/?player=p3&table=t1"

	// Join game in order
	state := gc[*GameState](p1, apiState)
	state = gc[*GameState](p2, apiState)
	state = gc[*GameState](p3, apiState)

	// Ready up
	state = gc[*GameState](p1, apiReady)
	state = gc[*GameState](p2, apiReady)
	state = gc[*GameState](p3, apiReady)

	// Player 1's turn - score first value
	state = gc[*GameState](p1, apiState)
	state = gc[*GameState](p1, apiScore, []gin.Param{{Key: "index", Value: "1"}})

	// Player 2's turn.
	state = gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's turn!")
	}

	// Player 1 leaves the game
	gc[string](p1, apiLeave)

	// Player 2's turn.
	state = gc[*GameState](p2, apiState)
	if state.ActivePlayer != 0 {
		t.Fatal("Player 2 expected to be active after P1's LEFT!")
	}

}
