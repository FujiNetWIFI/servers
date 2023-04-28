package main

import (
	"errors"
	"fmt"
	"time"
)

type GameServer struct {
	Gametype   int       `json:"gametype" binding:"required,numeric"`
	Server     string    `json:"server" binding:"required,printascii"`
	Region     string    `json:"region" binding:"required,printascii"`
	Serverurl  string    `json:"serverurl" binding:"required,hostname_rfc1123"`
	Status     string    `json:"status" binding:"required,oneof=online offline"`
	Maxplayers int       `json:"maxplayers" binding:"required,numeric"`
	Curplayers int       `json:"curplayers" binding:"required,numeric"`
	LastPing   time.Time `json:"lastping" binding:"omitempty" `
}

func newServer(gametype int, server, region, url, status string, maxplayers, curplayers int, lastPing time.Time) *GameServer {
	return &GameServer{
		Gametype:   gametype,
		Server:     server,
		Region:     region,
		Serverurl:  url,
		Status:     status,
		Maxplayers: maxplayers,
		Curplayers: curplayers,
		LastPing:   lastPing,
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
		newServer(1, "5 CARD STUD (demo)", "us", "https://thomcorner.com", "online", 8, 4, time.Now()),
		newServer(1, "5 CARD STUD (demo)", "us", "http://erichomeserver.com", "online", 8, 1, time.Now()),
		newServer(1, "5 CARD STUD (demo)", "eu", "tcp://erichomeserver.com", "offline", 0, 0, time.Now()),
		newServer(1, "Battleship (demo)", "asia", "tcps://8bitBattleship.com", "online", 2, 1, time.Now()),
		newServer(1, "Battleship (demo)", "australia", "8bitBattleship.com", "online", 6, 1, time.Now()),
	}

	for _, server := range DummyServers {
		GAMESRV.Store(server.Key(), server)
	}

	return GAMESRV.Count()
}

// Do additional checking
func (s *GameServer) CheckInput() (err error) {

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
