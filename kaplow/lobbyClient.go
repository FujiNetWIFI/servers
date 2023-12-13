package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

const (
	LOBBY_ENDPOINT_UPSERT = "http://fujinet.online:8080/server"
)

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

// Defaults for this game server
// Appkey/game are hard coded, but the others could be read from a config file
var SERVERDETAILS = GameServer{
	Appkey:    1,
	Game:      "Kapow!",
	Region:    "us",
	Serverurl: "https://5card.carr-designs.com/",
	Clients: []GameClient{
		{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/kapow.xex"},
	},
}

func UpdateLobby(maxPlayers int, curPlayers int, isOnline bool, Server string, Serverurl string) {

	SERVERDETAILS.Maxplayers = maxPlayers
	SERVERDETAILS.Curplayers = curPlayers
	if isOnline {
		SERVERDETAILS.Status = "online"
	} else {
		SERVERDETAILS.Status = "offline"
	}

	SERVERDETAILS.Server = Server
	SERVERDETAILS.Serverurl = Serverurl

	jsonPayload, err := json.MarshalIndent(SERVERDETAILS, "", "\t")
	if err != nil {
		slog.Warn("LobbyUpdate", "Unable to persist GAMESERVERDETAILS: ", SERVERDETAILS)
		return
	}

	request, err := http.NewRequest("POST", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		slog.Warn("LobbyUpdate", "Unable to create new request to: ", LOBBY_ENDPOINT_UPSERT)
		return
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		slog.Warn("LobbyUpdate", "Unable to POST request to: ", LOBBY_ENDPOINT_UPSERT)
		return
	}
	defer response.Body.Close()

	slog.Debug("LobbyUpdate", "Lobby response status: ", response.StatusCode)

	if response.StatusCode > 300 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Warn("LobbyUpdate", "Unable to read Lobby response body: ", err)
			return
		}

		slog.Debug("LobbyUpdate", "Lobby response body: ", string(body))
	}

}
