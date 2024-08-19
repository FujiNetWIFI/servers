package main

import (
	"time"
)

// For simplicity on the 8bit side (using switch statement), using a single character for each key.
// DOUBLE CHECK that letter isn't already in use on the object!
// Double characters are used for the list objects

// The json is then converted to a \0 separated list of key/values when &raw=1 is passed in
// for further optimization

type Player struct {
	Name   string `json:"n"`
	Alias  int    `json:"a"`
	Scores []int  `json:"s"`

	// Internal
	id          string
	isBot       bool
	lastPing    time.Time
	isLeaving   bool
	isPenalized bool
	isViewing   bool
}

type GameState struct {
	// External (JSON)
	Name         string   `json:"n"` // Name of server - sent on first connect
	Prompt       string   `json:"p"`
	Round        int      `json:"r"`
	RollsLeft    int      `json:"l"`
	ActivePlayer int      `json:"a"`
	MoveTime     int      `json:"m"`
	Viewing      int      `json:"v"`
	Dice         string   `json:"d"`
	KeepRoll     string   `json:"k"`
	ValidScores  []int    `json:"c"`
	Players      []Player `json:"pl"`

	// Internal
	gameOver              bool
	startedStartCountdown bool
	clientPlayer          int
	moveExpires           time.Time
	botBox                []Player // if players join to replace the bots, bots go here until a player leaves

	// Meta/Lobby related
	table         string
	serverName    string
	registerLobby bool

	hash string //   `json:"z"` // external later
}
