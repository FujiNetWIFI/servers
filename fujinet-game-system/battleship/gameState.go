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
	LastAttackPos int
	Prompt	   	 string
	Players      []Player

	// Internal
	startedStartCountdown   bool
	prevTotalHumansNotReady int
	gameOver     bool
	clientPlayer int
	moveExpires  time.Time
	botBox                  []Player // if players join to replace the bots, bots go here until a player leaves
	
	lastSuccessfulAttackPos int

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
