package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// insert/update uploaded server to the database. It also covers delete
func UpsertServer(c *gin.Context) {

	// if we have X-HTTP-Method-Override we call for server deletion
	if c.GetHeader("X-HTTP-Method-Override") == "DELETE" {
		DeleteServer(c)

		return
	}

	server := GameServer{}

	err1 := c.ShouldBindJSON(&server)
	if err1 != nil && err1.Error() == "EOF" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  []string{"Submitted Json cannot be parsed"}})
		return
	}

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

// sends back the current server version + uptime
func ShowStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true,
		"version": STRINGVER,
		"uptime":  uptime(STARTEDON)})
}

// show documentation in html
func ShowMain(c *gin.Context) {
	c.Data(http.StatusOK, gin.MIMEHTML, DOCHTML)
}

// delete server from database. It doesn't check if it exists.
func DeleteServer(c *gin.Context) {

	server := GameServerDelete{}

	err1 := c.ShouldBindJSON(&server)
	if err1 != nil && err1.Error() == "EOF" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  []string{"Submitted Json cannot be parsed"}})
		return
	}

	if _, err2 := url.ParseRequestURI(server.Serverurl); err2 != nil {
		err1 = errors.Join(err1, fmt.Errorf("Key 'GameServer.ServerUrl' Error: Field validation has to be a valid url"))
	}

	if len(server.Serverurl) > 64 {
		err1 = errors.Join(err1, fmt.Errorf("Key 'GameServer.ServerUrl' Error: Field validation length must be 64 or less characters"))
	}

	if err1 != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err1.Error(), "\n")})
		return
	}

	GAMESRV.Delete(server.Key())

	c.JSON(http.StatusNoContent, gin.H{"success": true,
		"message": "Server correctly deleted"})
}
