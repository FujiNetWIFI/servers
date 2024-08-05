package main

import "time"

// For simplicity on the 8bit side (using switch statement), using a single character for each key.
// DOUBLE CHECK that letter isn't already in use on the object!
// Double characters are used for the list objects

type Player struct {
	Name   string `json:"n"`
	Alias  string `json:"a"`
	Scores []int  `json:"s"`

	// Internal
	isBot       bool
	lastPing    time.Time
	isLeaving   bool
	isPenalized bool
}

type GameState struct {
	// External (JSON)
	Prompt       string   `json:"p"`
	Round        int      `json:"r"`
	RollsLeft    int      `json:"l"`
	ActivePlayer int      `json:"a"`
	MoveTime     int      `json:"m"`
	Viewing      int      `json:"v"`
	Dice         string   `json:"d"`
	KeepRoll     string   `json:"k"`
	Players      []Player `json:"pl"`
	ValidScores  []int    `json:"c"`

	// Internal
	gameOver     bool
	clientPlayer int
	moveExpires  time.Time

	// Meta/Lobby related
	table         string
	serverName    string
	registerLobby bool

	hash string //   `json:"z"` // external later
}
