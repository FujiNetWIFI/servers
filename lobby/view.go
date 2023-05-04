package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// send the game servers stored to the client minimised
func ShowServersMinimised(c *gin.Context) {

	if GAMESRV.Count() == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{
				"success": false, "message": "No servers available"})

		return

	}

	// Return minified server result for 8-Bit Lobby Clients
	platform := c.Query("platform")
	if len(platform) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false, "message": "You need to submit a platform"})

		return
	}

	var ServerSlice []GameServer
	servers := func(key string, server *GameServer) bool {
		ServerSlice = append(ServerSlice, *server)
		return true
	}
	GAMESRV.Range(servers)

	SortServerSlice(&ServerSlice)

	var ServerMinSlice []GameServerMin

	for _, server := range ServerSlice {
		if ServerMinimised, ok := server.Minimize(platform); ok {
			ServerMinSlice = append(ServerMinSlice, ServerMinimised)
		}
	}

	if len(ServerMinSlice) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{
				"success": false, "message": "No servers available for " + platform})

		return
	}

	c.JSON(http.StatusOK, ServerMinSlice)
}

// send the game servers stored to the client in full
func ShowServers(c *gin.Context) {

	if GAMESRV.Count() == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{
				"success": false, "message": "No servers available"})

		return

	}

	var ServerSlice []GameServer
	servers := func(key string, server *GameServer) bool {
		ServerSlice = append(ServerSlice, *server)
		return true
	}
	GAMESRV.Range(servers)

	SortServerSlice(&ServerSlice)

	c.IndentedJSON(http.StatusOK, ServerSlice)
}

// insert/update uploaded server to the database
func UpsertServer(c *gin.Context) {

	server := GameServer{}

	err1 := c.ShouldBindJSON(&server)
	err2 := server.CheckInput()

	err := errors.Join(err1, err2)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err.Error(), "\n")})
		return
	}

	server.LastPing = time.Now()

	GAMESRV.Store(server.Key(), &server)

	c.JSON(http.StatusCreated, gin.H{"success": true,
		"message": "Server correctly updated"})
}
