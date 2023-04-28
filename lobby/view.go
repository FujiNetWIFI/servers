package main

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// sent the game servers stored to the client
func ShowServers(c *gin.Context) {

	var output []GameServer

	servers := func(key string, server *GameServer) bool {

		output = append(output, *server)

		return true
	}

	GAMESRV.Range(servers)

	sort.SliceStable(output, func(i, j int) bool {
		return output[i].Order() > output[j].Order()
	})

	c.IndentedJSON(http.StatusOK, output)
}

// insert/update uploaded server to the database
func UpsertServer(c *gin.Context) {

	/* JSON expected is:
	    {
	        "gametype": 1,
	        "server": "Super Chess",
	        "region": "eu",
	        "serverurl": "chess.rogersm.net",
	        "status": "online",
	        "maxplayers": 2,
	        "curplayers": 1,
	        "lastping": "2023-04-28T11:43:32.242816+02:00"
	    }

		See check rules in model.go file.
	*/
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

	c.JSON(http.StatusAccepted, gin.H{"success": true,
		"message": "Server correctly updated"})
}
