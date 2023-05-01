package main

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

func ShowServersLobbyClient(c *gin.Context, sortedServerList []GameServer, platform string) {

	var serverList []GameServerMin

	for _, server := range sortedServerList {
		// We only return this server if a game client for this platform exists

		// Find the appropriate client for this platform
		for _, client := range server.Clients {
			if strings.EqualFold(client.Platform, platform) {

				// Create a copy of the server to change for this client's response
				serverMin := GameServerMin{
					Id:         server.Id,
					Game:       server.Game,
					Gametype:   server.Gametype,
					Serverurl:  server.Serverurl,
					Client:     client.Url,
					Server:     server.Server,
					Region:     server.Region,
					Maxplayers: server.Maxplayers,
					Curplayers: server.Curplayers,
					Pingage:    int(time.Since(server.LastPing).Seconds()),
				}

				if strings.EqualFold(server.Status, "online") {
					serverMin.Online = 1
				}

				serverList = append(serverList, serverMin)
				break
			}
		}

	}

	c.JSON(http.StatusOK, serverList)
}

// sent the game servers stored to the client
func ShowServers(c *gin.Context) {

	// Created sorted list of servers
	var output []GameServer
	servers := func(key string, server *GameServer) bool {
		output = append(output, *server)
		return true
	}
	GAMESRV.Range(servers)

	// output should be: online first, offline last. Inside each category, newer last ping goes first
	sort.SliceStable(output, func(i, j int) bool {
		return output[i].Order() > output[j].Order()
	})

	// Return minified server result for 8-Bit Lobby Clients
	platform := c.Query("platform")
	if len(platform) > 0 {
		ShowServersLobbyClient(c, output, platform)
	} else {
		c.IndentedJSON(http.StatusOK, output)
	}
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
	if server.Id == 0 {
		server.Id = int(atomic.AddInt32(&SERVER_ID_COUNTER, 1))
	}
	GAMESRV.Store(server.Key(), &server)

	c.JSON(http.StatusAccepted, gin.H{"success": true,
		"message": "Server correctly updated"})
}
