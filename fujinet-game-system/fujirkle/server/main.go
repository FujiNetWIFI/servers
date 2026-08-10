package main

import (
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

	if PRODUCTION_MODE {
		gin.SetMode(gin.ReleaseMode)
		UpdateLobby = true
	} else {
		LobbyEndpoint = LOBBY_QA_ENDPOINT_UPSERT
	}

	log.Printf("This instance will update the lobby at " + LobbyEndpoint)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s", port)

	router := gin.Default()

	route := func(path string, handler gin.HandlerFunc) {
		router.GET(path, handler)
		router.POST(path, handler)
	}

	router.GET("/view", apiView)
	router.GET("/tables", apiTables)

	route("/state", apiState)
	route("/ready", apiReady)
	route("/roll/:keep", apiRoll)
	route("/bank/:keep", apiBank)
	route("/leave", apiLeave)

	route("/updateLobby", apiUpdateLobby)

	initializeGameServer()
	initializeTables()

	router.Run(":" + port)
}

// Set aside the masked dice and roll whatever is left
func apiRoll(c *gin.Context) {
	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			if state.clientPlayer == state.ActivePlayer {
				state.playerPing()
				state.rollAgain(c.Param("keep"))
				saveState(state)
				state = state.createClientState(c.Query("pov"))
			}
		}
	}()

	serializeResults(c, state)
}

// Set aside the masked dice and bank the turn score
func apiBank(c *gin.Context) {
	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			if state.clientPlayer == state.ActivePlayer {
				state.playerPing()
				state.bankScore(c.Param("keep"))
				saveState(state)
				state = state.createClientState(c.Query("pov"))
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
			if state.clientPlayer >= 0 {
				state.toggleReady()
				saveState(state)
				state = state.createClientState(c.Query("pov"))
			}
		}
	}()

	serializeResults(c, state)
}

// Steps forward and returns the updated state
func apiState(c *gin.Context) {
	hash := c.Query("hash")
	state, unlock := getState(c)

	func() {
		defer unlock()

		if state != nil {
			state.runGameLogic()
			saveState(state)

			state = state.createClientState(c.Query("pov"))
		}
	}()

	if state != nil && len(hash) > 0 && hash == state.hash {
		serializeResults(c, "1")
		return
	}

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

// Returns a view of the current state without causing it to change. For debugging side-by-side with a client
func apiView(c *gin.Context) {
	state, unlock := getState(c)
	func() {
		defer unlock()

		if state != nil {
			state = state.createClientState(c.Query("pov"))
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

	unlock := tableMutex.Lock(table)

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
	createTable("The Bar", "bar", 0, true)
	createTable("Kitchen Table", "kit", 0, true)
	createTable("AI Room - 2 bots", "ai2", 2, true)
	createTable("AI Room - 4 bots", "ai4", 4, true)

	for i := 1; i < 4; i++ {
		createTable(fmt.Sprintf("Dev Room - %d bots", i), fmt.Sprintf("dev%d", i), i, false)
	}
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
