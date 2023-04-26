package main

import (
	"time"

	"github.com/google/uuid"
)

type GameServer struct {
	Gamename   string    `json:"gamename"`
	Servername string    `json:"server"` // IP or name
	Instance   string    `json:"instance"`
	Status     string    `json:"status"`
	Maxplayers int       `json:"maxplayers"`
	Curplayers int       `json:"curplayers"`
	Checked_at time.Time `json:"checkedat"`
}

func newServer(gamename, servername, instance, status string, maxplayers, curplayers int, checked_at time.Time) *GameServer {
	return &GameServer{
		Gamename:   gamename,
		Servername: servername,
		Instance:   instance,
		Status:     status,
		Maxplayers: maxplayers,
		Curplayers: curplayers,
		Checked_at: checked_at,
	}
}

func init_dummy_servers() {
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Thom's Corner", "Table A - Humans", "Online", 8, 4, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Eric's Mock Server", "Table A - Bots!", "Online", 8, 1, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("5 CARD STUD", "Eric's Backup Server", "Table C", "(Offline)", 0, 0, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("Battleship", "8bitBattleship.com", "Server A", "Online", 2, 1, time.Now()))
	GAMESRV.Store(uuid.New().String(), newServer("Battleship", "8bitBattleship.com", "Server B", "Online", 2, 0, time.Now()))
}
