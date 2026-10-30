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

var stateMap sync.Map
var tables []GameTable
var tableMutex KeyedMutex // One request at a time per table

var debugMode bool
var disableLobby bool

type KeyedMutex struct {
	mutexes sync.Map
}

func (m *KeyedMutex) Lock(key string) func() {
	value, _ := m.mutexes.LoadOrStore(strings.ToLower(key), &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()
	return mtx.Unlock
}

func main() {
	log.Printf("Starting %s ...", versionString())

	flag.BoolVar(&debugMode, "debug", false, "Log requests and responses")
	flag.BoolVar(&disableLobby, "disable-lobby", false, "Disable lobby communication")
	flag.Parse()

	UpdateLobby = os.Getenv("GO_PROD") == "1" && !disableLobby
	if UpdateLobby {
		log.Printf("This instance will update the lobby at %s", LOBBY_ENDPOINT_UPSERT)
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.Println("Lobby communication is disabled.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)

	setupServer()
	if err := setupRouter().Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func setupServer() {
	initializeGameServer()
	initializeTables()
}

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
	return router
}

func apiMove(c *gin.Context) {
	state, unlock := getState(c)
	var client *clientState
	func() {
		defer unlock()
		if state != nil {
			if state.clientPlayer >= 0 && state.clientPlayer == state.ActivePlayer {
				state.playerPing()
				state.performMove(strings.ToUpper(c.Param("move")))
			}
			client = state.createClientState()
		}
	}()
	serializeResults(c, client)
}

func apiState(c *gin.Context) {
	hash := c.Query("hash")
	state, unlock := getState(c)
	var client *clientState
	func() {
		defer unlock()
		if state != nil {
			if state.clientName != "" {
				state.playerPing()
				state.RunGameLogic()
			}
			client = state.createClientState()
		}
	}()

	if client != nil && hash != "" && hash == client.Hash {
		serializeResults(c, "1")
		return
	}
	serializeResults(c, client)
}

func apiLeave(c *gin.Context) {
	state, unlock := getState(c)
	func() {
		defer unlock()
		if state != nil && state.clientName != "" {
			state.clientLeave()
			state.updateLobby()
		}
	}()
	serializeResults(c, "bye")
}

// Read only view, for debugging alongside a client
func apiView(c *gin.Context) {
	table := strings.ToLower(c.Query("table"))
	unlock := tableMutex.Lock(table)
	var client *clientState
	if value, ok := stateMap.Load(table); ok {
		state := value.(*GameState)
		saved := state.clientPlayer
		state.clientPlayer = -1
		client = state.createClientState()
		state.clientPlayer = saved
	}
	unlock()
	serializeResults(c, client)
}

// dev=1 lists the hidden developer tables instead
func apiTables(c *gin.Context) {
	dev := c.Query("dev") == "1"
	out := []GameTable{}
	for _, table := range tables {
		if value, ok := stateMap.Load(table.Table); ok {
			state := value.(*GameState)
			if dev != state.registerLobby {
				unlock := tableMutex.Lock(table.Table)
				table.MaxPlayers, table.CurPlayers = state.getHumanPlayerCountInfo()
				unlock()
				out = append(out, table)
			}
		}
	}
	serializeResults(c, out)
}

func apiVersion(c *gin.Context) {
	serializeResults(c, versionString())
}

func apiUpdateLobby(c *gin.Context) {
	for _, table := range tables {
		if value, ok := stateMap.Load(table.Table); ok {
			unlock := tableMutex.Lock(table.Table)
			value.(*GameState).updateLobby()
			unlock()
		}
	}
	serializeResults(c, "Lobby Updated")
}

func getState(c *gin.Context) (*GameState, func()) {
	table := strings.ToLower(c.Query("table"))
	if table == "" {
		table = "default"
	}
	unlock := tableMutex.Lock(table)
	value, ok := stateMap.Load(table)
	if !ok {
		return nil, unlock
	}
	state := value.(*GameState)
	state.setClientPlayerByName(c.Query("player"))
	return state, unlock
}

func initializeTables() {
	createTable("The Basement", "basement", 0, true)
	createTable("The Den", "den", 0, true)
	createTable("AI Room - 2 bots", "ai2", 2, true)
	createTable("AI Room - 4 bots", "ai4", 4, true)

	for i := 1; i < MAX_PLAYERS; i++ {
		createTable(fmt.Sprintf("Dev Room - %d bots", i), fmt.Sprintf("dev%d", i), i, false)
	}
}

func createTable(serverName string, table string, botCount int, registerLobby bool) {
	state := createGameState(botCount, registerLobby)
	state.TableId = table
	state.serverName = serverName
	stateMap.Store(table, state)
	state.updateLobby()

	tables = append(tables, GameTable{Table: table, Name: serverName})

	if UpdateLobby {
		time.Sleep(100 * time.Millisecond)
	}
}
