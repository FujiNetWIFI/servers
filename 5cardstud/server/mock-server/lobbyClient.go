package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/goccy/go-json"
)

const (
	LOBBY_ENDPOINT_UPSERT = "http://127.0.0.1:8080/server"
)

var DefaultGameServerDetails = GameServer{
	Gametype:  1,
	Game:      "5 Card Stud",
	Server:    "Mock Server (Bots)",
	Region:    "us",
	Serverurl: "https://5card.carr-designs.com/",
	Clients: []GameClient{
		GameClient{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/5card.xex"},
	},
}

type GameServer struct {
	// Properties being sent from Game Server
	Game       string       `json:"game" binding:"required,printascii"`
	Gametype   int          `json:"gametype" binding:"required,numeric"`
	Server     string       `json:"server" binding:"required,printascii"`
	Region     string       `json:"region" binding:"required,printascii"`
	Serverurl  string       `json:"serverurl" binding:"required"`
	Status     string       `json:"status" binding:"required,oneof=online offline"`
	Maxplayers int          `json:"maxplayers" binding:"required,numeric"`
	Curplayers int          `json:"curplayers" binding:"required,numeric"`
	Clients    []GameClient `json:"clients" binding:"required"`
}

type GameClient struct {
	Platform string `json:"platform" binding:"required,printascii`
	Url      string `json:"url" binding:"required`
}

func sendStateToLobby(maxPlayers int, curPlayers int, isOnline bool, instanceServerSuffix string, instanceUrlSuffix string) {

	// Start with copy of default game server details
	serverDetails := DefaultGameServerDetails
	serverDetails.Maxplayers = maxPlayers
	serverDetails.Curplayers = curPlayers
	if isOnline {
		serverDetails.Status = "online"
	} else {
		serverDetails.Status = "offline"
	}

	serverDetails.Serverurl += instanceUrlSuffix
	serverDetails.Server += instanceServerSuffix

	jsonPayload, err := json.Marshal(serverDetails)
	if err != nil {
		panic(err)
	}
	log.Printf("Updating Lobby: %s", jsonPayload)

	request, err := http.NewRequest("POST", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	log.Printf("Lobby Response: %s", response.Status)
	if response.StatusCode > 300 {
		body, _ := ioutil.ReadAll(response.Body)
		log.Println("response Body:", string(body))
	}

}
