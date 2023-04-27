package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

func init_dummy_servers() {
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Thom's Corner (demo)", "Table A - Humans", "Online", 8, 4, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Eric's Mock Server (demo)", "Table A - Bots!", "Online", 8, 1, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Eric's Backup Server (demo)", "Table C", "(Offline)", 0, 0, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("Battleship", "8bitBattleship.com (demo)", "Server A", "Online", 2, 1, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("Battleship", "8bitBattleship.com (demo)", "Server B", "Online", 2, 0, time.Now()))
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
