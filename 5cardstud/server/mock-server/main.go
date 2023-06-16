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

// This started as a sync.Map but could revert back to a map since a keyed mutex is being used
// to restrict state reading/setting to one thread at a time
var stateMap sync.Map

var tableMutex KeyedMutex

type KeyedMutex struct {
	mutexes sync.Map // Zero value is empty and ready for use
}

func (m *KeyedMutex) Lock(key string) func() {
	key = strings.ToLower(key)
	value, _ := m.mutexes.LoadOrStore(key, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()
	return func() { mtx.Unlock() }
}

func main() {
	log.Print("starting server...")

	router := gin.Default()

	router.GET("/view", apiView)

	router.GET("/state", apiState)
	router.POST("/state", apiState)

	router.GET("/move/:move", apiMove)
	router.POST("/move/:move", apiMove)

	router.GET("/leave", apiLeave)
	router.POST("/leave", apiLeave)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Local dev mode - do not update live lobby
	localMode := os.Getenv("GO_LOCAL")

	UpdateLobby = localMode != "1"

	initializeGameServer()
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

	state, unlock := getState(c, 0)
	func() {
		defer unlock()

		// Access check - only move if the client is the active player
		if state.clientPlayer == state.ActivePlayer {
			move := strings.ToUpper(c.Param("move"))
			state.performMove(move)
			saveState(state)
		}
	}()

	c.JSON(http.StatusOK, state.createClientState())
}

// Steps forward in the emulated game and returns the updated state
func apiState(c *gin.Context) {
	playerCount, _ := strconv.Atoi(c.DefaultQuery("count", "0"))
	state, unlock := getState(c, playerCount)
	func() {
		defer unlock()
		if state.clientPlayer >= 0 {
			state.runGameLogic()
			saveState(state)
		}
	}()

	c.JSON(http.StatusOK, state.createClientState())
}

// Drop from the specified table
func apiLeave(c *gin.Context) {
	state, unlock := getState(c, 0)
	func() {
		defer unlock()

		if state.clientPlayer >= 0 {
			state.clientLeave()
			saveState(state)
		}
	}()
	c.JSON(http.StatusOK, "bye")
	//c.JSON(http.StatusOK, state.createClientState())
}

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {

	state, unlock := getState(c, 0)
	unlock()

	c.IndentedJSON(http.StatusOK, state.createClientState())
}

// Gets the current game state for the specified table and adds the player id of the client to it
func getState(c *gin.Context, playerCount int) (*gameState, func()) {
	table := c.Query("table")

	if table == "" {
		table = "default"
	}
	table = strings.ToLower(table)
	player := c.Query("player")

	// Lock by the table so to avoid multiple threads updating the same table state
	unlock := tableMutex.Lock(table)

	return getTableState(table, player, playerCount), unlock
}

func getTableState(table string, playerName string, playerCount int) *gameState {
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
		state.updateLobby()
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

func initializeRealTables() {

	// Create the real servers (hard coded for now)

	createRealTable("The Red Room (2 bots)", "red", 2)
	createRealTable("The Blue Room (6 bots)", "blue", 6)
	createRealTable("The Green Room (4 bots)", "green", 4)
	createRealTable("The Basement", "basement", 0)
	createRealTable("The Den", "den", 0)

}

func createRealTable(serverName string, table string, botCount int) {
	state := createGameState(botCount, false)
	state.table = table
	state.serverName = serverName
	saveState(state)
	state.updateLobby()

	if UpdateLobby {
		time.Sleep(time.Millisecond * time.Duration(100))
	}
}
