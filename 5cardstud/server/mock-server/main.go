package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var stateMap sync.Map

func main() {
	router := gin.Default()

	router.GET("/view", apiView)

	router.GET("/state", apiState)
	router.POST("/state", apiState)

	router.GET("/move/:move", apiMove)
	router.POST("/move/:move", apiMove)

	router.Run("192.168.2.41:8080")
}

// Api Request steps
// 1. Get state
// 2. Game Logic
// 3. Save state
// 4. Return client centric state

// Executes a move for the client player, if that player is currently active
func apiMove(c *gin.Context) {

	state := getState(c)

	// Access check - only move if the client is the active player
	if state.clientPlayer == state.ActivePlayer {
		move := c.Param("move")
		state.performMove(move)
		saveState(state)
	}

	c.IndentedJSON(http.StatusOK, state.createClientState())
}

// Steps forward in the emulated game and returns the updated state
func apiState(c *gin.Context) {

	state := getState(c)
	state.emulateGame()
	saveState(state)

	c.IndentedJSON(http.StatusOK, state.createClientState())
}

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {

	state := getState(c)
	c.IndentedJSON(http.StatusOK, state.createClientState())
}

// Gets the current game state for the specified table and adds the player id of the client to it
func getState(c *gin.Context) *gameState {
	table := c.Query("table")
	if table == "" {
		table = "default"
	}
	value, ok := stateMap.Load(table)

	var state *gameState

	if ok {
		state = value.(*gameState)
	} else {
		state = initGameState()
		state.table = table
	}

	//player := c.Query("player")
	state.clientPlayer = 2
	return state
}

func saveState(state *gameState) {
	stateMap.Store(state.table, state)
}
