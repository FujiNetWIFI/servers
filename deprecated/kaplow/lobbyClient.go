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

func UpdateLobby(GameName string, maxPlayers int, curPlayers int, isOnline bool, Server string, Serverurl string) error {

	// Appkey/game are hard coded, but the others could be read from a config file

	toupdate := GameServer{
		Game:       GameName,
		Appkey:     1,
		Server:     Server,
		Region:     "us",
		Serverurl:  Serverurl,
		Status:     IfElse(isOnline, "online", "offline"),
		Maxplayers: maxPlayers,
		Curplayers: curPlayers,
		Clients:    []GameClient{{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/kapow.xex"}},
	}

	return contactLobby(toupdate, "POST")
}

func RemoveLobby(Serverurl string) error {

	todelete := GameServerDelete{
		Serverurl: Serverurl,
	}

	return contactLobby(todelete, "DELETE")
}

// update lobby server with the data and associated http_verb = ["POST", "DELETE"]
func contactLobby(data interface{}, http_verb string) error {

	jsonPayload, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		slog.Warn("updateLobby", "Unable to persist struct: ", data)
		return err
	}

	request, err := http.NewRequest(http_verb, LOBBY_ENDPOINT_UPSERT, bytes.NewBuffer(jsonPayload))
	if err != nil {
		slog.Warn("LobbyUpdate", "Unable to create new ", http_verb, " request to: ", LOBBY_ENDPOINT_UPSERT)
		return err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		slog.Warn("UpdateLobby", "Unable to ", http_verb, "request to: ", LOBBY_ENDPOINT_UPSERT)
		return err
	}
	defer response.Body.Close()

	slog.Debug("UpdateLobby", "Lobby response status: ", response.StatusCode)

	if response.StatusCode > 300 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Warn("UpdateLobby", "Unable to read Lobby response body: ", err)
			return err
		}

		slog.Debug("UpdateLobby", "Lobby response body: ", string(body))
	}

	return nil
}
