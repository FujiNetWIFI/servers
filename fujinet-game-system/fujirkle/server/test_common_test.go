package main

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

//////////////////////////////////////////////////////////////////////////////////////////
// Test Initialization
//////////////////////////////////////////////////////////////////////////////////////////

var tableIndex int = 0

func TestMain(m *testing.M) {
	isTestMode = true
	initializeGameServer()
	resetTestMode()
	os.Exit(m.Run())
}

//////////////////////////////////////////////////////////////////////////////////////////
// Test Helper Functions
//////////////////////////////////////////////////////////////////////////////////////////

// Call - used to call api* functions directly
func c(path string, f func(*gin.Context), opt_params ...[]gin.Param) any {
	ctx := createCall(path, opt_params)
	f(ctx)
	r, _ := ctx.Get("testResult")
	return r
}

func createCall(path string, opt_params [][]gin.Param) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", path, nil)
	if len(opt_params) > 0 {
		ctx.Params = opt_params[0]
	}
	return ctx
}

// Convenience wrappers for the two endpoints that carry a keep mask
func callRoll(path string, mask string) any {
	return c(path, apiRoll, []gin.Param{{Key: "keep", Value: mask}})
}

func callBank(path string, mask string) any {
	return c(path, apiBank, []gin.Param{{Key: "keep", Value: mask}})
}

// Creates a uniquely named table with the specified number of bots and human players
func createTestTable(bots int, humans int) (string, []string) {
	resetTestMode()
	clearForcedRolls()
	tableIndex++
	table := fmt.Sprintf("t%d", tableIndex)
	createTable(table, table, bots, true)

	table = "&table=" + table
	players := make([]string, humans)
	for i := 0; i < humans; i++ {
		players[i] = fmt.Sprintf("/?player=p%d", i+1) + table
	}

	return table, players
}

// The server side state, as opposed to the client centric copy the api returns
func rawState(tableSuffix string) *GameState {
	name := strings.TrimPrefix(tableSuffix, "&table=")
	value, ok := stateMap.Load(name)
	if !ok {
		return nil
	}
	return value.(*GameState)
}

//////////////////////////////////////////////////////////////////////////////////////////
// Deterministic dice
//////////////////////////////////////////////////////////////////////////////////////////

var queuedRolls [][]int

func clearForcedRolls() {
	queuedRolls = nil
	forceRoll = nil
}

// queueRoll adds one roll to the queue. Once queued, rolls are consumed in
// order; anything past the end of the queue rolls all 1s so a test never
// blocks on randomness it did not specify.
func queueRoll(dice ...int) {
	queuedRolls = append(queuedRolls, dice)
	forceRoll = func(count int) []int {
		out := make([]int, count)
		for i := range out {
			out[i] = 1
		}

		if len(queuedRolls) == 0 {
			return out
		}

		next := queuedRolls[0]
		queuedRolls = queuedRolls[1:]

		for i := 0; i < count && i < len(next); i++ {
			out[i] = next[i]
		}
		return out
	}
}

// Join and ready every supplied player, then step the table into its first turn
func startGame(t *testing.T, table string, players []string) *GameState {
	t.Helper()

	for _, player := range players {
		c(player, apiState)
	}
	for _, player := range players {
		c(player, apiReady)
	}
	c(players[0], apiState)

	state := rawState(table)
	if state.Round == ROUND_LOBBY {
		t.Fatal("expected the game to have started")
	}
	return state
}
