package main

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/goccy/go-json"
)

const LOBBY_ENDPOINT_UPSERT = "http://lobby.fujinet.online/server"

// TODO before release: the client URLs are not assigned yet
var DefaultGameServerDetails = GameServer{
	Appkey:    0x0C,
	Game:      "Blackjack",
	Region:    "us",
	Serverurl: "https://blackjack.carr-designs.com/",
	Clients:   []GameClient{},
}

var UpdateLobby bool

type GameServer struct {
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

	serverDetails := DefaultGameServerDetails
	serverDetails.Maxplayers = maxPlayers
	serverDetails.Curplayers = curPlayers
	serverDetails.Status = "offline"
	if isOnline {
		serverDetails.Status = "online"
	}
	serverDetails.Server = server
	serverDetails.Serverurl += instanceUrlSuffix

	jsonPayload, err := json.Marshal(serverDetails)
	if err != nil {
		log.Println("Error marshalling lobby details:", err)
		return
	}
	log.Printf("Updating Lobby: %s", jsonPayload)

	request, err := http.NewRequest("POST", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Println("Error creating lobby request:", err)
		return
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	response, err := http.DefaultClient.Do(request)
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
