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
	resetTestMode()
	os.Exit(m.Run())
}

//////////////////////////////////////////////////////////////////////////////////////////
// Test Helper Functions
//////////////////////////////////////////////////////////////////////////////////////////

// Call - used to call api* functions directly
// Greatly reduces extra code around calling different functions
func c(path string, f func(*gin.Context), opt_params ...[]gin.Param) any {
	c := createCall(path, opt_params)
	f(c)
	r, _ := c.Get("testResult")
	return r
}

// This just makes it easier to step into a call above with a single step over
func createCall(path string, opt_params [][]gin.Param) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", path, nil)
	if len(opt_params) > 0 {
		c.Params = opt_params[0]
	}
	return c
}

// Helper function - creates a uniquely named table with the specified number of bots and human players
func createTestTable(bots int, humans int) (string, []string) {
	resetTestMode()
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
