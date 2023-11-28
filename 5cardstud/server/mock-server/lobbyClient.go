package main

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/goccy/go-json"
)

const (
	//LOBBY_ENDPOINT_UPSERT = "http://127.0.0.1:8080/server"
	//LOBBY_ENDPOINT_UPSERT = "http://lobby.rogersm.net:8080/server"
	LOBBY_ENDPOINT_UPSERT = "http://lobby.fujinet.online/server"
)

// Defaults for this game server
// Appkey/game are hard coded, but the others could be read from a config file
var DefaultGameServerDetails = GameServer{
	Appkey:    1,
	Game:      "5 Card Stud",
	Region:    "us",
	Serverurl: "https://5card.carr-designs.com/",
	Clients: []GameClient{
		{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/5card.xex"},
	},
}

var UpdateLobby bool

type GameServer struct {
	// Properties being sent from Game Server
	Game       string       `json:"game"`
	Appkey     int          `json:"appkey"`
	Server     string       `json:"server"`
	Region     string       `json:"region"`
	Serverurl  string       `json:"serverurl"`
	Status     string       `json:"status"`
	Maxplayers int          `json:"maxplayers"`
	Curplayers int          `json:"curplayers"`
	Clients    []GameClient `json:"clients"`
}

type GameClient struct {
	Platform string `json:"platform"`
	Url      string `json:"url"`
}

func sendStateToLobby(maxPlayers int, curPlayers int, isOnline bool, server string, instanceUrlSuffix string) {

	if !UpdateLobby {
		return
	}

	// Start with copy of default game server details
	serverDetails := DefaultGameServerDetails
	serverDetails.Maxplayers = maxPlayers
	serverDetails.Curplayers = curPlayers
	if isOnline {
		serverDetails.Status = "online"
	} else {
		serverDetails.Status = "offline"
	}

	serverDetails.Server = server
	serverDetails.Serverurl += instanceUrlSuffix

	jsonPayload, err := json.Marshal(serverDetails)
	if err != nil {
		panic(err)
	}
	log.Printf("Updating Lobby: %s", jsonPayload)

	request, err := http.NewRequest("POST", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		panic(err)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	log.Printf("Lobby Response: %s", response.Status)
	if response.StatusCode > 300 {
		body, _ := io.ReadAll(response.Body)
		log.Println("response Body:", string(body))
	}

}
