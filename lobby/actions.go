package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// send the game servers stored to the client minimised
func ShowServersMinimised(c *gin.Context) {

	// Return minified server result for 8-Bit Lobby Clients
	platform := c.Query("platform")
	if len(platform) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false, "message": "You need to submit a platform"})

		return
	}

	// optional field. If appkey is empty, it becomes a None (-1)
	appkeyForm := c.Query("appkey")

	appkey := Atoi(appkeyForm, -1)

	ServerSliceClient, _ := txGameServerGetBy(platform, appkey)

	if len(ServerSliceClient) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{"success": false,
				"message": "No servers available for " + platform})

		return
	}

	var ServerMinSlice []GameServerMin

	for _, server := range ServerSliceClient {
		ServerMinSlice = append(ServerMinSlice, server.Minimize())
	}

	c.JSON(http.StatusOK, ServerMinSlice)
}

// send the game servers stored to the client in full
// TODO: sort the names, too confusing
func ShowServers(c *gin.Context) {

	GameServerClient, err := txGameServerGetAll()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			gin.H{"success": false,
				"message": "Database transaction issue",
				"errors":  []string{err.Error()}})

		return
	}

	if len(GameServerClient) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{"success": false, "message": "No servers available"})

		return

	}

	GameServerSlice := GameServerClient.toGameServerSlice()

	c.IndentedJSON(http.StatusOK, GameServerSlice)
}

// insert/update uploaded server to the database. It also covers delete
func UpsertServer(c *gin.Context) {

	server := GameServer{}

	err1 := c.ShouldBindJSON(&server)
	if err1 != nil && err1.Error() == "EOF" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  []string{"Submitted Json cannot be parsed"}})
		return
	}

	err2 := server.CheckInput()

	err := errors.Join(err1, err2)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err.Error(), "\n")})
		return
	}

	err = txGameServerUpsert(server)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			gin.H{"success": false,
				"message": "Database transaction issue",
				"errors":  []string{err.Error()}})

		return
	}

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

	err = txGameServerDelete(server.Serverurl)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false, "message": "Database transaction issue",
			"errors": []string{err.Error()}})

		return
	}

	c.JSON(http.StatusNoContent, gin.H{"success": true,
		"message": "Server correctly deleted"})
}
