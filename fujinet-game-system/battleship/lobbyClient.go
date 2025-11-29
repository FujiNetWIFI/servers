package main

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/goccy/go-json"
)

const (
	//LOBBY_ENDPOINT_UPSERT = "http://127.0.0.1:8081/server"
	LOBBY_ENDPOINT_UPSERT    = "https://lobby.fujinet.online/server"
	LOBBY_QA_ENDPOINT_UPSERT = "https://qalobby.fujinet.online/server"

	LOBBY_CLIENT_APP_KEY = 0x05 // Registered at https://github.com/FujiNetWIFI/fujinet-firmware/wiki/SIO-Command-$DC-Open-App-Key#lobby-client-app-key-ids
)

// Defaults for this game server
// Appkey/game are hard coded, but the others could be read from a config file
var DefaultGameServerDetails = GameServer{
	Appkey:    LOBBY_CLIENT_APP_KEY,
	Game:      "Battleship",
	Region:    "us",
	Serverurl: "https://battleship.carr-designs.com/",
	Clients: []GameClient{
		{Platform: "atari", Url: "tnfs://ec.tnfs.io/atari/fbs.xex"},
		//{Platform: "apple2", Url: "tnfs://ec.tnfs.io/apple2/fbs.po"},
		{Platform: "coco", Url: "tnfs://ec.tnfs.io/coco/fbs.dsk"},
	},
}

var UpdateLobby bool
var LobbyEndpoint string = LOBBY_ENDPOINT_UPSERT

const PLAYER_TYPE_BOT = "bot"
const PLAYER_TYPE_HUMAN = "human"

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
	GameResult *GameResult  `json:"gameResult,omitempty"`
}

type GameClient struct {
	Platform string `json:"platform"`
	Url      string `json:"url"`
}

type GameResult struct {
	Players []GamePlayer `json:"players"`
}

type GamePlayer struct {
	Name   string `json:"name"`
	Winner bool   `json:"winner"`
	Type   string `json:"type"`
}

func sendStateToLobby(maxPlayers int, curPlayers int, isOnline bool, server string, instanceUrlSuffix string, gameResult *GameResult) {

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
	serverDetails.GameResult = gameResult

	jsonPayload, err := json.Marshal(serverDetails)

	if err != nil {
		panic(err)
	}
	log.Printf("Updating Lobby: %s", jsonPayload)

	request, err := http.NewRequest("POST", LobbyEndpoint, bytes.NewBuffer(jsonPayload))
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
