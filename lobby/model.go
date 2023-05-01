package main

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type GameServer struct {
	// Internally added properties
	Id       int
	LastPing time.Time `json:"lastping" binding:"omitempty" `

	// Properties being sent from Game Server
	Game       string       `json:"game" binding:"required,printascii"`
	Gametype   int          `json:"gametype" binding:"required,numeric"`
	Server     string       `json:"server" binding:"required,printascii"`
	Region     string       `json:"region" binding:"required,printascii"`
	Serverurl  string       `json:"serverurl" binding:"required"`
	Status     string       `json:"status" binding:"required,oneof=online offline"`
	Maxplayers int          `json:"maxplayers" binding:"required,numeric"`
	Curplayers int          `json:"curplayers" binding:"numeric"`
	Clients    []GameClient `json:"clients" binding:"required"`
}

type GameClient struct {
	Platform string `json:"platform" binding:"required,printascii`
	Url      string `json:"url" binding:"required`
}

// Minified Structure to send to 8-bit Lobby Client
type GameServerMin struct {
	Id         int    `json:"i"`
	Game       string `json:"g"`
	Gametype   int    `json:"t"`
	Serverurl  string `json:"u"`
	Client     string `json:"c"`
	Server     string `json:"s"`
	Region     string `json:"r"`
	Online     int    `json:"o"`
	Maxplayers int    `json:"m"`
	Curplayers int    `json:"p"`
	Pingage    int    `json:"a"`
}

func newServer(game string, gametype int, server, region, url, status string, maxplayers, curplayers int, LastPing time.Time, clients []GameClient) *GameServer {
	return &GameServer{
		Game:       game,
		Gametype:   gametype,
		Server:     server,
		Region:     region,
		Serverurl:  url,
		Status:     status,
		Maxplayers: maxplayers,
		Curplayers: curplayers,
		LastPing:   LastPing,
		Clients:    clients,
		Id:         int(atomic.AddInt32(&SERVER_ID_COUNTER, 1)),
	}
}

// we index by Serverurl because it's unique
func (s *GameServer) Key() string {
	return s.Serverurl
}

// create a order for sorting
func (s *GameServer) Order() string {
	return s.Status + "#" + s.LastPing.String()
}

func init_dummy_servers() int {

	var DummyServers = []*GameServer{
		/*
			newServer("5 Card Stud", 1, "Mock Server (Bots) Table 1", "us", "https://5card.carr-designs.com/?table=1&count=3", "online", 4, 3, time.Now(), []GameClient{{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/5card.xex"}}),
			newServer("5 Card Stud", 1, "Mock Server (Bots) Table 2", "au", "https://5card.carr-designs.com/?table=2&count=7", "online", 8, 7, time.Now(), []GameClient{{Platform: "atari", Url: "TNFS://ec.tnfs.io/atari/5card.xex"}}),
			newServer("Battleship", 1, "8-Bit BattleServer", "eu", "https://localhost#1", "offline", 0, 0, time.Now(), []GameClient{{Platform: "atari", Url: "TNFS://192.168.2.41/atari/bship.xex"}}),
			newServer("Connect 4", 1, "Thom's Corner", "us", "https://localhost#2", "online", 2, 1, time.Now(), []GameClient{{Platform: "atari", Url: "TNFS://192.168.2.41/atari/bship.xex"}}),
			newServer("Light Cycle", 1, "TRON Fan Group Server", "us", "https://localhost#3", "online", 2, 0, time.Now(), []GameClient{{Platform: "atari", Url: "TNFS://192.168.2.41/atari/bship.xex"}}),
		*/
	}

	for _, server := range DummyServers {
		GAMESRV.Store(server.Key(), server)
	}

	return GAMESRV.Count()
}

// Do additional checking
func (s *GameServer) CheckInput() (err error) {

	/* The most important thing here is to provide clear statements to the client caller
	   about what is wrong with the json. The default GO validator errors do not do this.

		 For instance, it will tell you the field failed "max length" validation, but not
		 tell you what the max actually is.

		 Maybe use a custom validator later for consistency between go validator and custom
		 validation below.
	*/

	if s.Curplayers < 0 {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Curplayers' Error:Field validation for 'Curplayers' cannot be negative (%d)", s.Curplayers))
	}

	if s.Maxplayers < 0 {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Maxplayers' Error:Field validation for 'Maxplayers' cannot be negative (%d)", s.Maxplayers))
	}

	if s.Curplayers > s.Maxplayers {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Curplayers' and 'GameServer.Maxplayers' Error:Field validation for 'Curplayers' (%d) cannot be bigger than 'Maxplayers' (%d)", s.Curplayers, s.Maxplayers))
	}

	if s.Gametype < 1 || s.Gametype > 255 {
		err = errors.Join(err, fmt.Errorf("'gametype' must be between 1 and 255"))
	}

	if len(s.Game) > 12 {
		err = errors.Join(err, fmt.Errorf("'game' must be 12 or less characters"))
	}

	if len(s.Server) > 32 {
		err = errors.Join(err, fmt.Errorf("'server' must be 12 or less characters"))
	}

	if len(s.Serverurl) > 64 {
		err = errors.Join(err, fmt.Errorf("'serverurl' must be 64 or less characters"))
	}

	for _, client := range s.Clients {
		if len(client.Url) > 64 {
			err = errors.Join(err, fmt.Errorf("clients.url must be 64 or less characters"))
		}
	}

	return err
}
