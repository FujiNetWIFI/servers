package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// A sync.Map is used to save the state at the end of a request without needing synchronization
// If a request errors out in the middle, it will not save the state, avoiding an invalid,
// partially updated state.
// A mutex is used so a given table can only be accessed by a single request at a time
var stateMap sync.Map
var tables []GameTable = []GameTable{}
var runningHubs sync.Map

var tableMutex KeyedMutex

var debugMode bool
var disableLobby bool

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
	log.Printf("Starting %s ...", versionString())

	flag.BoolVar(&debugMode, "debug", false, "Enable debug logging for requests and responses")
	flag.BoolVar(&disableLobby, "disable-lobby", false, "Disable lobby communication")
	flag.Parse()

	// Set environment flags
	UpdateLobby = os.Getenv("GO_PROD") == "1" && !disableLobby

	if UpdateLobby {
		log.Printf("This instance will update the lobby at " + LOBBY_ENDPOINT_UPSERT)
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.Println("Lobby communication is disabled.")
	}

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listing on port %s", port)

	setupServer()
	router := setupRouter()

	log.Printf("WebSocket server starting on :" + port) // Client connects to 8080

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}

// setupServer initializes game server state and the standard tables
func setupServer() {
	initializeGameServer()
	initializeTables()
}

// setupRouter builds the Gin engine with all API routes. Extracted from main so
// integration tests can run the real router via httptest.
func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/view", apiView)

	router.GET("/state", apiState)
	router.POST("/state", apiState)

	router.GET("/move/:move", apiMove)
	router.POST("/move/:move", apiMove)

	router.GET("/leave", apiLeave)
	router.POST("/leave", apiLeave)

	router.GET("/tables", apiTables)
	router.GET("/version", apiVersion)
	router.GET("/updateLobby", apiUpdateLobby)

	router.GET("/ws", func(c *gin.Context) {
		serveWs(c.Writer, c.Request)
	})

	return router
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

// Executes a move for the client player, if that player is currently active
func apiMove(c *gin.Context) {

	state, unlock := getState(c)
	var client *clientState
	func() {
		defer unlock()

		if state != nil {
			// Access check - only move if the client is the active player
			if state.clientPlayer == state.ActivePlayer {
				move := strings.ToUpper(c.Param("move"))
				state.performMove(move)
				saveState(state)
			}
			client = state.createClientState()
		}
	}()

	serializeResults(c, client)
}

// Steps forward and returns the updated state
func apiState(c *gin.Context) {
	hash := c.Query("hash")
	state, unlock := getState(c)

	var client *clientState
	func() {
		defer unlock()

		if state != nil {
			if state.clientPlayer >= 0 {
				state.RunGameLogic()
				saveState(state)
			}
			client = state.createClientState()
		}
	}()

	// Check if passed in hash matches the state
	if client != nil && len(hash) > 0 && hash == client.Hash {
		serializeResults(c, "1")
		return
	}

	serializeResults(c, client)
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

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {

	state, unlock := getState(c)
	var client *clientState
	func() {
		defer unlock()

		if state != nil {
			client = state.createClientState()
		}
	}()

	serializeResults(c, client)
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
				tableOutput = append(tableOutput, table)
			}
		}
	}
	serializeResults(c, tableOutput)
}

// Returns the server version, e.g. to verify a deployment
func apiVersion(c *gin.Context) {
	serializeResults(c, versionString())
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
		// IMPORTANT: Work directly with the pointer from the map, do not make a copy.
		state = value.(*GameState)
		state.setClientPlayerByName(player)

		// Start the hub.run goroutine for this table if it's not already running.
		// Pass the authoritative state pointer so the hub modifies the correct object.
		if _, loaded := runningHubs.LoadOrStore(table, true); !loaded {
			go hub.run(state, saveState)
			log.Printf("Started game logic hub for table: %s", table)
		}
	}

	return state, unlock
}

func saveState(state *GameState) {
	stateMap.Store(state.TableId, state)
}

func initializeTables() {

	// Create the real servers (hard coded for now)
	createTable("The Basement", "basement", 0, true)
	createTable("The Den", "den", 0, true)
	createTable("AI Room - 2 bots", "ai2", 2, true)
	createTable("AI Room - 4 bots", "ai4", 4, true)
	createTable("AI Room - 6 bots", "ai6", 6, true)

	// For client developers, create hidden tables for each # of bots (for ease of testing with a specific # of players in the game)
	// These will not update the lobby

	for i := 1; i < 8; i++ {
		createTable(fmt.Sprintf("Dev Room - %d bots", i), fmt.Sprintf("dev%d", i), i, false)
	}

}

func createTable(serverName string, table string, botCount int, registerLobby bool) {
	state := createGameState(botCount, registerLobby)
	state.TableId = table
	stateMap.Store(table, state)
	state.serverName = serverName
	saveState(state)
	state.updateLobby()

	tables = append([]GameTable{{Table: table, Name: serverName}}, tables...)

	if UpdateLobby {
		time.Sleep(time.Millisecond * time.Duration(100))
	}
}
