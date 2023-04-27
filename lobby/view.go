package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func ShowServers(c *gin.Context) {

	var output []GameServer

	broadcast := func(key string, server *GameServer) bool {

		output = append(output, *server)

		return true
	}

	GAMESRV.Range(broadcast)

	c.IndentedJSON(http.StatusOK, output)
}

func UpsertServer(c *gin.Context) {

	gameserver := GameServer{}

	err1 := c.ShouldBindJSON(&gameserver)
	err2 := gameserver.CheckInput()

	err := errors.Join(err1, err2)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "VALIDATEERR - Invalid Json",
				"errors":  strings.Split(err.Error(), "\n")})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"success": true,
		"message": "Server correctly updated"})
}
