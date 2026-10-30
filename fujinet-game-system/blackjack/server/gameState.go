package main

import (
	"math/rand"
	"time"
)

const (
	MAX_PLAYERS      = 5
	MAX_HAND_CARDS   = 8
	MAX_DEALER_CARDS = 10
	DECKS            = 6
	RESHUFFLE_AT     = DECKS * 52 * 3 / 4
	STARTING_PURSE   = 1000
	MIN_BET          = 10

	MOVE_TIME_GRACE_SECONDS = 4
	PLAYER_PING_TIMEOUT     = -5 * time.Minute
)

// Even amounts keep 3:2 and surrender payouts whole
var BET_SIZES = []int{10, 20, 50, 100}

// Vars, not consts, so tests can zero them
var BOT_TIME_LIMIT = 2 * time.Second
var DEALER_TIME_LIMIT = 1500 * time.Millisecond
var PLAYER_TIME_LIMIT = 39 * time.Second
var PLAYER_TIME_LIMIT_SINGLE_PLAYER = 255 * time.Second // 8 bit clients store this in a byte
var ENDGAME_TIME_LIMIT = 8 * time.Second

const WAITING_MESSAGE = "Waiting for players"

const (
	ROUND_WAITING = 0
	ROUND_BETTING = 1
	ROUND_PLAYING = 2
	ROUND_DEALER  = 3
	ROUND_OVER    = 5
)

const (
	STATUS_WAITING = 0
	STATUS_PLAYING = 1
	STATUS_LEFT    = 3
)

const (
	FLAG_SOFT        = 1
	FLAG_BLACKJACK   = 2
	FLAG_BUST        = 4
	FLAG_DOUBLED     = 8
	FLAG_SURRENDERED = 16
	FLAG_DONE        = 32
)

const (
	RESULT_NONE = iota
	RESULT_WIN
	RESULT_LOSE
	RESULT_PUSH
	RESULT_BLACKJACK
	RESULT_SURRENDER
)

const (
	MOVE_HIT       = "HI"
	MOVE_STAND     = "ST"
	MOVE_DOUBLE    = "DD"
	MOVE_SPLIT     = "SP"
	MOVE_SURRENDER = "SU"
	MOVE_REBET     = "RB"
)

var betMoves = []string{"B1", "B2", "B3", "B4"}

var botNames = []string{"Clyd", "Meg", "Kirk", "Jim"}

type card struct {
	Rank int // 2..14, ace = 14
	Suit int // 0..3 = C D H S
}

type hand struct {
	Cards       []card
	Bet         int
	Result      int
	split       bool
	doubled     bool
	surrendered bool
	done        bool
}

type Player struct {
	Name   string
	Status int
	Purse  int
	Move   string
	Hands  []hand

	isBot    bool
	lastPing time.Time
	lastBet  int
}

type GameState struct {
	TableId      string
	LastResult   string
	Round        int
	ActivePlayer int
	ActiveHand   int
	Dealer       []card
	Players      []Player

	holeRevealed  bool
	shoe          []card
	shoeIndex     int
	moveExpires   time.Time
	clientPlayer  int
	clientName    string
	serverName    string
	registerLobby bool
	botBox        []Player
	joinQueue     []Player
	rng           *rand.Rand

	// Tests only
	allowBotGames bool
	testDeck      []card
	fixedBotBet   int
	houseNet      int
	wagered       int
	rebuys        int
	handsPlayed   int
}

type validMove struct {
	Move string `json:"m"`
	Name string `json:"n"`
}

type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

type clientHand struct {
	Bet    int    `json:"b"`
	Total  int    `json:"t"`
	Flags  int    `json:"f"`
	Result int    `json:"r"`
	Cards  string `json:"c"`
}

type clientPlayer struct {
	Name   string       `json:"n"`
	Status int          `json:"s"`
	Purse  int          `json:"p"`
	Move   string       `json:"m"`
	Hands  []clientHand `json:"hs"`
}

type clientState struct {
	LastResult   string         `json:"l"`
	Round        int            `json:"r"`
	ActivePlayer int            `json:"a"`
	ActiveHand   int            `json:"ah"`
	MoveTime     int            `json:"m"`
	Viewing      int            `json:"v"`
	Dealer       string         `json:"d"`
	DealerTotal  int            `json:"dt"`
	DealerFlags  int            `json:"df"`
	ValidMoves   []validMove    `json:"vm"`
	Players      []clientPlayer `json:"pl"`
	Hash         string         `json:"z"`
}
