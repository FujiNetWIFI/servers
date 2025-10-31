package main

import "time"

// Gamestate

type Player struct {
	Name      string
	Gamefield []int
	ShipsLeft []int

	// Internal
	id          string
	isBot       bool
	isReady     bool
	lastPing    time.Time
	isPenalized bool
	status   	int
	ships 		[]Ship
}

type GameState struct {
	Status       int
	ActivePlayer int
	PlayerStatus int
	MoveTime     int
	Prompt	   	 string
	Players      []Player

	// Internal
	startedStartCountdown   bool
	prevTotalHumansNotReady int
	gameOver     bool
	clientPlayer int
	moveExpires  time.Time
	

	// Meta/Lobby related
	table         string
	serverName    string
	registerLobby bool
}


type Ship struct {
	GridPos []int
	Pos  int
	Dir  int
}
