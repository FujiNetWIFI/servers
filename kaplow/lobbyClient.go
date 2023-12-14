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

type GameServerDelete struct {
	Serverurl string `json:"serverurl"`
}

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

func UpdateLobby(maxPlayers int, curPlayers int, isOnline bool, Server string, Serverurl string) bool {

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
		slog.Warn("UpdateLobby", "Unable to persist GAMESERVERDETAILS: ", SERVERDETAILS)
		return false
	}

	request, err := http.NewRequest("POST", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		slog.Warn("LobbyUpdate", "Unable to create new update request to: ", LOBBY_ENDPOINT_UPSERT)
		return false
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		slog.Warn("UpdateLobby", "Unable to POST request to: ", LOBBY_ENDPOINT_UPSERT)
		return false
	}
	defer response.Body.Close()

	slog.Debug("UpdateLobby", "Lobby response status: ", response.StatusCode)

	if response.StatusCode > 300 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Warn("UpdateLobby", "Unable to read Lobby response body: ", err)
			return false
		}

		slog.Debug("UpdateLobby", "Lobby response body: ", string(body))
	}

	return true
}

func RemoveLobby(Serverurl string) bool {

	todelete := GameServerDelete{
		Serverurl: Serverurl,
	}

	jsonPayload, err := json.MarshalIndent(todelete, "", "\t")
	if err != nil {
		slog.Warn("RemoveLobby", "Unable to persist GAMESERVERDETAILS: ", SERVERDETAILS)
		return false
	}

	request, err := http.NewRequest("DELETE", LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		slog.Warn("RemoveLobby", "Unable to create new delete request to: ", LOBBY_ENDPOINT_UPSERT)
		return false
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		slog.Warn("RemoveLobby", "Unable to DELETE request to: ", LOBBY_ENDPOINT_UPSERT)
		return false
	}
	defer response.Body.Close()

	slog.Debug("RemoveLobby", "Lobby response status: ", response.StatusCode)

	if response.StatusCode > 300 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Warn("RemoveLobby", "Unable to read Lobby response body: ", err)
			return false
		}

		slog.Debug("RemoveLobby", "Lobby response body: ", string(body))
	}

	return true
}
