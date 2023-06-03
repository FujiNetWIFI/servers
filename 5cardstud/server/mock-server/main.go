package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// Create real game tables
	initializeRealTables()

	router.Run(":" + port)
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
		move := strings.ToUpper(c.Param("move"))
		state.performMove(move)
		saveState(state)
	}

	c.JSON(http.StatusOK, state.createClientState())
}

// Steps forward in the emulated game and returns the updated state
func apiState(c *gin.Context) {
	playerCount, _ := strconv.Atoi(c.DefaultQuery("count", "0"))
	state := getState(c, playerCount)
	state.runGameLogic()
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
	player := c.Query("player")
	return getTableState(table, player, playerCount)
}

func getTableState(table string, playerName string, playerCount int) *gameState {
	table = strings.ToLower(table)
	value, ok := stateMap.Load(table)

	var state *gameState

	if ok {
		stateCopy := *value.(*gameState)
		state = &stateCopy

		// Update player count for table if changed
		if state.isMockGame && playerCount > 1 && playerCount < 9 && playerCount != len(state.Players) {
			if len(state.Players) > playerCount {
				state = createGameState(playerCount, true)
				state.table = table
			} else {
				state.updateMockPlayerCount(playerCount)
			}
		}
	} else {
		// Create a brand new game
		state = createGameState(playerCount, true)
		state.table = table
		updateLobby(state)
	}

	//player := c.Query("player")
	if state.isMockGame {
		state.clientPlayer = 0
	} else {
		state.setClientPlayerByName(playerName)
	}
	return state
}

func saveState(state *gameState) {
	stateMap.Store(state.table, state)
}

func updateLobby(state *gameState) {
	if state.isMockGame {
		return
	}
	sendStateToLobby(8, len(state.Players), true, state.serverName, "?table="+state.table)
}

func initializeRealTables() {
	createRealTable("The Basement (6 bots)", "basement", 6)
	time.Sleep(time.Second)

	createRealTable("The Garage (4 bots)", "garage", 4)
	time.Sleep(time.Second)

	createRealTable("The Den", "den", 0)
	time.Sleep(time.Second)
}

func createRealTable(serverName string, table string, botCount int) {
	state := createGameState(botCount, false)
	state.table = table
	state.serverName = serverName
	saveState(state)
	updateLobby(state)
}
