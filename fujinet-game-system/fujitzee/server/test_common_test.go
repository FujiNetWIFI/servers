package main

import (
	"fmt"
	"net/http/httptest"
	"os"
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
	os.Exit(m.Run())
}

//////////////////////////////////////////////////////////////////////////////////////////
// Test Helper Functions
//////////////////////////////////////////////////////////////////////////////////////////

// Go Call (gc) - used to call api* functions directly, specifying the return type
// Greatly reduces extra code around calling different functions
func gc[K any](path string, f func(*gin.Context), opt_params ...[]gin.Param) K {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", path, nil)
	if len(opt_params) > 0 {
		c.Params = opt_params[0]
	}
	f(c)
	r, _ := c.Get("testResult")
	return r.(K)
}

// Helper function - creates a uniquely named table with the specified number of bots
func createTestTable(bots int) string {
	tableIndex++
	table := fmt.Sprintf("t%d", tableIndex)
	createTable(table, table, bots, true)
	return "&table=" + table
}

// Helper function - create a table full of players and return the player query array
func createFullTestTable() ([]string, string) {
	table := createTestTable(0)

	players := make([]string, 6)
	for i := 0; i < 6; i++ {
		players[i] = fmt.Sprintf("/?player=p%d", i+1) + table
	}
	return players, table
}
