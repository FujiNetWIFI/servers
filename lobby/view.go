package main

import (
	"net/http"

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
