package main

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// used for form checking and sending the data back to the client in /viewFull
type GameServer struct {
	// Properties being sent from Game Server
	Game       string       `json:"game" binding:"required,printascii"`
	Appkey     int          `json:"appkey" binding:"required,number"`
	Server     string       `json:"server" binding:"required,printascii"`
	Region     string       `json:"region" binding:"required,printascii"`
	Serverurl  string       `json:"serverurl" binding:"required"`
	Status     string       `json:"status" binding:"required,oneof=online offline"`
	Maxplayers int          `json:"maxplayers" binding:"required,number"`
	Curplayers int          `json:"curplayers" binding:"number"` // golang validator has issues with 0 values
	Clients    []GameClient `json:"clients" binding:"required"`
}

type GameServerSlice []GameServer

// used for form checking and sending the data back to the client
type GameClient struct {
	Platform string `json:"platform" binding:"required,printascii"`
	Url      string `json:"url" binding:"required"`
}

// used for fomr checking only
type GameServerDelete struct {
	Serverurl string `json:"serverurl" binding:"required"`
}

// used to retrieve the data from the database from GameServerClient view
type GameServerClient struct {
	Serverurl       string
	Game            string
	Appkey          int
	Server          string
	Region          string
	Status          string
	Maxplayers      int
	Curplayers      int
	Lastping        time.Time
	Client_platform string
	Client_url      string
}

type GameServerClientSlice []GameServerClient

// Minified Structure to send to 8-bit Lobby Client in /view
type GameServerMin struct {
	Game       string `json:"g"`
	AppKey     int    `json:"t"`
	Serverurl  string `json:"u"`
	Client     string `json:"c"`
	Server     string `json:"s"`
	Region     string `json:"r"`
	Online     int    `json:"o"`
	Maxplayers int    `json:"m"`
	Curplayers int    `json:"p"`
	Pingage    int    `json:"a"`
}

// minimize file to send to 8 bit client filtering by platform
func (s GameServerClient) Minimize() (minimised GameServerMin) {

	return GameServerMin{
		Game:       s.Game,
		AppKey:     s.Appkey,
		Serverurl:  s.Serverurl,
		Client:     s.Client_url,
		Server:     s.Server,
		Region:     s.Region,
		Online:     IfElse(s.Status == "online", 1, 0),
		Maxplayers: s.Maxplayers,
		Curplayers: s.Curplayers,
		Pingage:    int(time.Since(s.Lastping).Seconds()),
	}
}

// conver a flat GameServerClient to a nested GameServer
func (s GameServerClient) toGameServer() (gameserver GameServer) {
	return GameServer{
		Game:       s.Game,
		Appkey:     s.Appkey,
		Server:     s.Server,
		Region:     s.Region,
		Serverurl:  s.Serverurl,
		Status:     s.Status,
		Maxplayers: s.Maxplayers,
		Curplayers: s.Curplayers,
		Clients:    []GameClient{{Platform: s.Client_platform, Url: s.Client_url}},
	}
}

// transform a flat GameServerClientSlice to nested GameServerSlice
func (s GameServerClientSlice) toGameServerSlice() (gameservers GameServerSlice) {

	var prev GameServer
	i := -1

	for _, gsc := range s {

		current := gsc.toGameServer()

		if prev.Serverurl == current.Serverurl {
			gameservers[i].Clients = append(gameservers[i].Clients, current.Clients...)
		} else {
			i++
			gameservers = append(gameservers, current)
		}

		prev = current
	}

	// Sort the ranks by online people at the top
	sort.SliceStable(gameservers, func(i, j int) bool {
		return gameservers[i].Curplayers > gameservers[j].Curplayers
	})

	return gameservers
}

// Do additional checking
func (s *GameServer) CheckInput() (err error) {

	err = errors.Join(
		ErrorIf(s.Curplayers < 0, fmt.Errorf("key: 'GameServer.Curplayers' Error:Field validation for 'Curplayers' cannot be negative (%d)", s.Curplayers)),
		ErrorIf(s.Maxplayers < 0, fmt.Errorf("key: 'GameServer.Maxplayers' Error:Field validation for 'Maxplayers' cannot be negative (%d)", s.Maxplayers)),
		ErrorIf(s.Curplayers > s.Maxplayers, fmt.Errorf("key: 'GameServer.Curplayers' and 'GameServer.Maxplayers' Error:Field validation for 'Curplayers' (%d) cannot be bigger than 'Maxplayers' (%d)", s.Curplayers, s.Maxplayers)),
		ErrorIf(s.Appkey < 1 || s.Appkey > 255, fmt.Errorf("key: 'GameServer.Appkey' Error: Field validation length must be between 1 and 255")),
		ErrorIf(len(s.Game) < 6 || len(s.Game) > 20, fmt.Errorf("key: 'GameServer.Game' Error: Field validation length must be between 6 and 20 characters")),
		ErrorIf(len(s.Region) > 12, fmt.Errorf("key: 'GameServer.Region' Error: Field validation length must be 12 or less characters")),
		ErrorIf(len(s.Server) > 32, fmt.Errorf("key: 'GameServer.Server' Error: Field validation length must be 32 or less characters")),
		ErrorIf(IsValidURI(s.Serverurl), fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation has to be a valid url")),
		ErrorIf(len(s.Serverurl) > 64, fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation length must be 64 or less characters")),
	)

	for _, client := range s.Clients {

		err = errors.Join(err,
			ErrorIf(IsValidURI(client.Url), fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation has to be a valid url")),
			ErrorIf(len(client.Url) > 64, fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation length must be 64 or less characters")),
		)

	}

	return err
}
func (s *GameServerDelete) CheckInput() (err error) {

	err = errors.Join(
		ErrorIf(IsValidURI(s.Serverurl), fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation has to be a valid url")),
		ErrorIf(len(s.Serverurl) > 64, fmt.Errorf("key: 'GameServer.ServerUrl' Error: Field validation length must be 64 or less characters")),
	)

	return err
}
