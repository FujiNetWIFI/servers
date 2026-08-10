package main

import (
	"time"
)

// Single character json keys keep the 8 bit client parser small, matching the
// convention used by the other FujiNet game servers. Passing &raw=1 flattens
// this to a \0 separated key/value list.
//
// Unlike Fujitzee, a Fujirkle player holds a single banked total rather than a
// scorecard, so the ready flag is its own field instead of sharing Scores[0].

type Player struct {
	Name  string `json:"n"`
	Alias int    `json:"a"`
	Score int    `json:"s"`
	Ready int    `json:"y"`

	// Internal
	id          string
	isBot       bool
	lastPing    time.Time
	isLeaving   bool
	isPenalized bool
	isViewing   bool
	isSmart     bool
}

type GameState struct {
	// External (JSON)
	Name         string   `json:"n"`
	Prompt       string   `json:"p"`
	Round        int      `json:"r"`
	ActivePlayer int      `json:"a"`
	MoveTime     int      `json:"m"`
	Viewing      int      `json:"v"`
	Dice         string   `json:"d"`
	KeepRoll     string   `json:"k"`
	TurnScore    int      `json:"t"`
	KeptDice     string   `json:"e"`
	Selectable   string   `json:"x"`
	ValidMoves   int      `json:"c"`
	Players      []Player `json:"pl"`

	// Internal
	gameOver                bool
	startedStartCountdown   bool
	prevTotalHumansNotReady int
	clientPlayer            int
	moveExpires             time.Time
	botBox                  []Player

	// Set when the active player's roll has no scoring dice. The roll stays on
	// screen until the timer expires so everyone can see it.
	fujirkled bool

	// Set once a player reaches the target; the game ends when play returns
	// to them, giving everyone else one last turn.
	finalRound        bool
	finalRoundTrigger string

	// Meta/Lobby related
	table         string
	serverName    string
	registerLobby bool

	hash string
}
