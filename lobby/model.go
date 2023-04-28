package main

import (
	"errors"
	"fmt"
	"time"
)

type GameServer struct {
	Gamename   string    `json:"gamename" binding:"required,printascii"`
	Servername string    `json:"server" binding:"required,hostname_rfc1123"`
	Instance   string    `json:"instance" binding:"required,printascii"`
	Status     string    `json:"status" binding:"required,oneof=online offline"`
	Maxplayers int       `json:"maxplayers" binding:"required,numeric"`
	Curplayers int       `json:"curplayers" binding:"required,numeric"`
	LastPing   time.Time `json:"lastping" binding:"omitempty" `
}

func newServer(gamename, servername, instance, status string, maxplayers, curplayers int, lastPing time.Time) *GameServer {
	return &GameServer{
		Gamename:   gamename,
		Servername: servername,
		Instance:   instance,
		Status:     status,
		Maxplayers: maxplayers,
		Curplayers: curplayers,
		LastPing:   lastPing,
	}
}

// servername + "#" + Instance
func (s *GameServer) Key() string {
	return s.Servername + "#" + s.Instance
}

// create a order for sorting
func (s *GameServer) Order() string {
	return s.Status + "#" + s.LastPing.String()
}

func init_dummy_servers() int {

	var DummyServers = []*GameServer{
		newServer("5 CARD STUD (demo)", "thomcorner.com", "Table A - Humans", "online", 8, 4, time.Now()),
		newServer("5 CARD STUD (demo)", "erichomeserver.com", "Table A - Bots!", "online", 8, 1, time.Now()),
		newServer("5 CARD STUD (demo)", "erichomeserver.com", "Table C", "offline", 0, 0, time.Now()),
		newServer("Battleship (demo)", "8bitBattleship.com", "Server A", "online", 2, 1, time.Now()),
		newServer("Battleship (demo)", "8bitBattleship.com", "Server B", "online", 2, 0, time.Now()),
	}

	for _, server := range DummyServers {
		GAMESRV.Store(server.Key(), server)
	}

	return GAMESRV.Count()
}

// Do additional checking
func (s *GameServer) CheckInput() (err error) {

	// Key: 'GameServer.Gamename' Error:Field validation for 'Gamename' failed on the 'printascii' tag",
	if s.Curplayers < 0 {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Curplayers' Error:Field validation for 'Curplayers' cannot be negative (%d)", s.Curplayers))
	}

	if s.Maxplayers < 0 {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Maxplayers' Error:Field validation for 'Maxplayers' cannot be negative (%d)", s.Maxplayers))
	}

	if s.Curplayers > s.Maxplayers {
		err = errors.Join(err, fmt.Errorf("Key: 'GameServer.Curplayers and GameServer.Maxplayers' Error:Field validation for 'Curplayers' (%d) cannot be bigger than 'Maxplayers' (%d)", s.Curplayers, s.Maxplayers))
	}

	return err
}
