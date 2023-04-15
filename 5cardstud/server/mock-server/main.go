package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

var stateMap sync.Map

func main() {
	log.Print("starting server...")

	router := gin.Default()

	router.GET("/view", apiView)

	router.GET("/state", apiState)
	router.POST("/state", apiState)

	router.GET("/move/:move", apiMove)
	router.POST("/move/:move", apiMove)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	router.Run(":8080")
}

// Api Request steps
// 1. Get state
// 2. Game Logic
// 3. Save state
// 4. Return client centric state

// Executes a move for the client player, if that player is currently active
func apiMove(c *gin.Context) {

	state := getState(c, 0)

	// Access check - only move if the client is the active player
	if state.clientPlayer == state.ActivePlayer {
		move := c.Param("move")
		state.performMove(move)
		saveState(state)
	}

	c.JSON(http.StatusOK, state.createClientState())
}

// Steps forward in the emulated game and returns the updated state
func apiState(c *gin.Context) {
	playerCount, _ := strconv.Atoi(c.DefaultQuery("count", "0"))
	state := getState(c, playerCount)
	state.emulateGame()
	saveState(state)

	c.JSON(http.StatusOK, state.createClientState())
}

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {

	state := getState(c, 0)
	c.IndentedJSON(http.StatusOK, state.createClientState())
}

// Gets the current game state for the specified table and adds the player id of the client to it
func getState(c *gin.Context, playerCount int) *gameState {
	table := c.Query("table")
	if table == "" {
		table = "default"
	}
	value, ok := stateMap.Load(table)

	var state *gameState

	if ok {
		state = value.(*gameState)
		if playerCount > 1 && playerCount < 9 {
			if len(state.Players) > playerCount {
				state = createGameState(playerCount)
				state.table = table
			} else {
				state.updatePlayerCount(playerCount)
			}
		}
	} else {
		state = createGameState(playerCount)
		state.table = table
	}

	//player := c.Query("player")
	state.clientPlayer = 1
	return state
}

func saveState(state *gameState) {
	stateMap.Store(state.table, state)
}
