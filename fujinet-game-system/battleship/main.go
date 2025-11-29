package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// A sync.Map is used to save the state at the end of a request without needing synchronization
// If a request errors out in the middle, it will not save the state, avoiding an invalid,
// partially updated state
// A mutex is used so a given table can only be accessed by a single request at a time
var stateMap sync.Map
var tables []GameTable = []GameTable{}

var tableMutex KeyedMutex

type KeyedMutex struct {
	mutexes sync.Map
}

func (m *KeyedMutex) Lock(key string) func() {
	key = strings.ToLower(key)
	value, _ := m.mutexes.LoadOrStore(key, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()
	return func() { mtx.Unlock() }
}

var PRODUCTION_MODE bool

func main() {
	log.Print("Starting server...")

	PRODUCTION_MODE = os.Getenv("GO_PROD") == "1"

	// Set environment flags
	if PRODUCTION_MODE {
		gin.SetMode(gin.ReleaseMode)
		UpdateLobby = true
	} else {
		LobbyEndpoint = LOBBY_QA_ENDPOINT_UPSERT

	}

	log.Println("This instance will update the lobby at", LobbyEndpoint)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listing on port %s", port)

	router := gin.Default()

	route := func(path string, handler gin.HandlerFunc) {
		router.GET(path, handler)
		router.POST(path, handler)
	}

	router.GET("/view", apiView)
	router.GET("/tables", apiTables)

	route("/state", apiState)
	route("/ready", apiReady)
	route("/place/:ships", apiPlace)
	route("/attack/:pos", apiAttack)
	route("/leave", apiLeave)

	if PRODUCTION_MODE {
		route("/updateLobby", apiUpdateLobby)
	} else {
		route("/debugEndGame/:winner", apiDebugEndGame)
	}
	

	initializeTables()

	router.Run(":" + port)
}

// Api Request steps
// 1. Get state
// 2. Game Logic
// 3. Save state
// 4. Return client centric state

// request pattern
// 1. get state (locks the state)
//   A. Start a function that updates table state
//   B. Defer unlocking the state until the current "state updating" function is complete
//   C. If state is not nil, perform logic
// 2. Serialize and return results

// Score the current roll at the requested index for the client player, if that player is currently active
func apiPlace(c *gin.Context) {

	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			state.playerPing()
			ships := strings.Split(c.Param("ships"), ",")

			if (len(ships) == 5) {
				shipPositions := []int{}
				for _, ship := range ships {
					shipInt, _ := strconv.Atoi(ship)
					shipPositions = append(shipPositions, shipInt)
				}
				state.placeShips(shipPositions)
				saveState(state)
			}
			state = state.createClientState()
		
		}
	}()

	serializeResults(c, state)
}

// Score the current roll at the requested index for the client player, if that player is currently active
func apiAttack(c *gin.Context) {

	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			// Access check - only move if the client is the active player
			if state.clientPlayer == state.ActivePlayer {
				pos, _ := strconv.Atoi(c.Param("pos"))
				state.playerPing()
				state.attack(pos)
				saveState(state)
				state = state.createClientState()
			}
		}
	}()

	serializeResults(c, state)
}

// Toggle if player is ready to start
func apiReady(c *gin.Context) {

	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			// Access check - only move if the client is a valid player
			if state.clientPlayer >= 0 {
				state.toggleReady()
				saveState(state)
				state = state.createClientState()
			}
		}
	}()

	serializeResults(c, state)
}

// Steps forward and returns the updated state
func apiState(c *gin.Context) {
	state, unlock := getState(c)

	func() {
		defer unlock()

		if state != nil {
			state.runGameLogic()
			saveState(state)

			state = state.createClientState()
		}
	}()
	serializeResults(c, state)
}

// Drop from the specified table
func apiLeave(c *gin.Context) {
	state, unlock := getState(c)

	func() {
		defer unlock()

		if state != nil {
			if state.clientPlayer >= 0 {
				state.clientLeave()
				state.updateLobby()
				saveState(state)
			}
		}
	}()
	serializeResults(c, "bye")
}
//apiEndGame
func apiDebugEndGame(c *gin.Context) {
	state, unlock := getState(c)

	func() {
		defer unlock()

		if state != nil {
			if state.clientPlayer >= 0 {
				winner,_ := strconv.Atoi(c.Param("winner"))
				state.debugForceEnd(winner)
				saveState(state)
			}
		}
	}()
	serializeResults(c, "bye")
}

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {

	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			state = state.createClientState()
		}
	}()

	serializeResults(c, state)
}

// Returns a list of real tables with player/slots for the client
// If passing "dev=1", will return developer testing tables instead of the live tables
func apiTables(c *gin.Context) {
	returnDevTables := c.Query("dev") == "1"

	tableOutput := []GameTable{}
	for _, table := range tables {
		value, ok := stateMap.Load(table.Table)
		if ok {
			state := value.(*GameState)
			if (returnDevTables && !state.registerLobby) || (!returnDevTables && state.registerLobby) {
				humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()
				table.CurPlayers = humanPlayerCount
				table.MaxPlayers = humanPlayerSlots
				if table.CurPlayers > table.MaxPlayers {
					table.CurPlayers = table.MaxPlayers
				}
				tableOutput = append(tableOutput, table)
			}
		}
	}
	serializeResults(c, tableOutput)
}

// Forces an update of all tables to the lobby - useful for adhoc use if the Lobby restarts or loses info
func apiUpdateLobby(c *gin.Context) {
	for _, table := range tables {
		value, ok := stateMap.Load(table.Table)
		if ok {
			state := value.(*GameState)
			state.updateLobby()
		}
	}

	serializeResults(c, "Lobby Updated")
}

// Gets the current game state for the specified table and adds the player id of the client to it
func getState(c *gin.Context) (*GameState, func()) {
	table := c.Query("table")

	if table == "" {
		table = "default"
	}
	table = strings.ToLower(table)
	player := c.Query("player")

	// Lock by the table so to avoid multiple threads updating the same table state
	unlock := tableMutex.Lock(table)

	// Load state
	value, ok := stateMap.Load(table)

	var state *GameState

	if ok {
		stateCopy := *value.(*GameState)
		state = &stateCopy
		if state.setClientPlayerByID(player) {
			saveState(state)
		}
	}

	return state, unlock
}

func saveState(state *GameState) {
	stateMap.Store(state.table, state)
}

func initializeTables() {

	// Create the real servers (hard coded for now)
	createTable("AI - 1 on 1", "ai1", 1, true)
	createTable("AI - 2 Bots", "ai2", 2, true)
	createTable("AI - 3 Bots", "ai3", 3, true)
	createTable("Cape Fuji", "r1", 0, true)
	createTable("High Seas", "r2", 0, true)

	// For client developers, create one hidden test room
	// This will not update the lobby
	createTable("test", "test", 0, false)

}

func createTable(serverName string, table string, botCount int, registerLobby bool) {
	state := createGameState(botCount)
	state.table = table
	state.serverName = serverName
	state.registerLobby = registerLobby

	saveState(state)
	state.updateLobby()

	tables = append([]GameTable{{Table: table, Name: serverName}}, tables...)

	if UpdateLobby && registerLobby {
		time.Sleep(time.Millisecond * time.Duration(100))
	}
}
