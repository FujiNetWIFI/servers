package main

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "0", "J", "Q", "K", "Q"}
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BL": "BET", // BET LOW (e.g. 5)
	"Bh": "BET", // BET HIGH (e.g. 10)
	"CA": "CALL",
	"RL": "RAISE",
	"Rh": "RAISE",
}

type validMove struct {
	move string
	name string
}

type card struct {
	value int
	suit  int
}

type player struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Bet    int    `json:"bet"`
	Move   string `json:"move"`
	Purse  int    `json:"purse"`
	Hand   string `json:"hand"`

	// Internal
	cards []card
}

type gameState struct {
	// External (JSON)
	LastResult     string   `json:"lastResult"`
	Round          int      `json:"round"`
	Pot            int      `json:"pot"`
	ActivePlayer   int      `json:"activePlayer"`
	StartingPlayer int      `json:"startingPlayer"`
	ValidMoves     []string `json:"validMoves"`
	Players        []player `json:"players"`

	// Internal
	deck       []card
	deckIndex  int
	frame      int
	currentBid int
}

// Forcing player index right now
var playerIndex = 2

var state = gameState{}

func main() {
	router := gin.Default()
	router.GET("/state", getGameState)
	router.GET("/move/:move", sendMove)
	router.POST("/move/:move", sendMove)

	initGameState()
	router.Run("192.168.2.41:8080")
}

func initGameState() {
	deck := []card{}

	// Create deck of 52 cards
	for suit := 0; suit < 4; suit++ {
		for value := 2; value < 15; value++ {
			card := card{value: value, suit: suit}
			deck = append(deck, card)
		}
	}

	state.deck = deck
	state.frame = 0

	state.Round = 0
	state.StartingPlayer = -1
	state.Players = []player{
		{Name: "Thom Bot", Purse: 150},
		{Name: "ChatGTP", Purse: 150},
		{Name: "Player", Purse: 150},
		{Name: "Mozz Bot", Purse: 150},
	}
	newRound()
}

func newRound() {
	state.Round++

	// Reset players for this round
	for i := 0; i < len(state.Players); i++ {
		player := state.Players[i]

		if state.Round > 1 {
			// If not the first round, add any bets into the pot
			state.Pot += player.Bet
		} else {
			// First round of a new game? Reset player state
			player.Status = 1
			player.cards = []card{}
		}

		// Reset player's last move/bet for this round
		player.Move = ""
		player.Bet = 0

		state.Players[i] = player
	}

	state.currentBid = 0

	// First round of a new game? Shuffle the cards and deal an extra card
	if state.Round == 1 {

		// Shuffle the deck 7 times :)
		for shuffle := 0; shuffle < 7; shuffle++ {
			rand.Shuffle(len(state.deck), func(i, j int) { state.deck[i], state.deck[j] = state.deck[j], state.deck[i] })
		}
		state.deckIndex = 0
		state.Pot = 0
		state.StartingPlayer = (state.StartingPlayer + 1) % len(state.Players)
		dealCards()
	}

	dealCards()
}

func dealCards() {
	for i, player := range state.Players {
		player.cards = append(player.cards, state.deck[state.deckIndex])
		state.Players[i] = player
		state.deckIndex++
	}
}

// Emulates player/logic on a simplified 5 card stud server
func emulateGame(checkForRoundOnly bool) {

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table
	if (state.currentBid > 0 && state.Players[state.ActivePlayer].Bet == state.currentBid) ||
		(state.currentBid == 0 && state.Players[state.ActivePlayer].Move != "") {
		newRound()
		return
	}

	if checkForRoundOnly {
		return
	}

	// Peform a move for this player if they are in the game and have not folded
	if state.Players[state.ActivePlayer].Status == 1 {
		moves := getValidMoves()

		choice := 0
		if len(moves) > 1 {
			// Choose something other than FOLD (always the first option)
			// Avoid endless raises - Only raise up to 10
			if state.currentBid >= 10 || rand.Intn(2) == 1 {
				choice = 1
			} else {
				choice = rand.Intn(len(moves)-1) + 1
			}

		}
		move := moves[choice]

		performMove(state.ActivePlayer, move.move)
	}

}

// Performs the requested move, checking if allowed, and returns true if successful
func performMove(playerNum int, move string) bool {
	// Sanity check - make sure the active player is attempting the move
	if playerNum != state.ActivePlayer {
		return false
	}

	player := state.Players[state.ActivePlayer]

	// Sanity Check 2 - Player not allowed to move. In theory they would never be active
	if player.Status != 1 {
		return false
	}

	// Default as move was not completed
	completedMove := false

	switch move {
	case "FO":
		player.Status = 2
		completedMove = true
	case "CH":
		if state.currentBid == 0 {
			completedMove = true
		}
	case "BL":
		fallthrough
	case "RL":
		delta := state.currentBid + 5 - player.Bet
		if (state.currentBid == 0 || move == "RL") && player.Purse >= delta {
			state.currentBid += 5
			player.Bet += delta
			player.Purse -= delta
			completedMove = true
		}
	case "CA":
		delta := state.currentBid - player.Bet
		if player.Purse >= delta {
			player.Bet += delta
			player.Purse -= delta
			completedMove = true
		}
	}

	if completedMove {
		player.Move = moveLookup[move]
		state.Players[state.ActivePlayer] = player

		// Move to next player
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

		// Skip over player if not in this game (joined late / folded)
		for state.Players[state.ActivePlayer].Status != 1 {
			state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
		}
	}

	return completedMove
}

func getValidMoves() []validMove {
	moves := []validMove{
		{move: "FO", name: "Fold"},
	}

	player := state.Players[state.ActivePlayer]

	if state.currentBid == 0 {
		moves = append(moves, validMove{move: "CH", name: "Check"})

		if player.Purse >= 5 {
			moves = append(moves, validMove{move: "BL", name: "Bid 5"})
		}
	} else {
		if player.Purse >= state.currentBid-player.Bet {
			moves = append(moves, validMove{move: "CA", name: "Call"})
		}
		if state.Players[state.ActivePlayer].Purse >= state.currentBid-player.Bet+5 {
			moves = append(moves, validMove{move: "RL", name: "Raise 5"})
		}
	}

	return moves
}

func sendMove(c *gin.Context) {
	move := c.Param("move")

	//playerIndex := c.Query("player")

	performMove(playerIndex, move)
	returnGameState(c, playerIndex)

	//player := c.Query("player")
}

func getGameState(c *gin.Context) {

	//playerIndex := c.Query("player")

	emulateGame(state.ActivePlayer == playerIndex)
	returnGameState(c, playerIndex)
}

func returnGameState(c *gin.Context, thisPlayer int) {

	// Create a copy of the state for this player only
	stateCopy := state
	setActivePlayer := false

	// Check if we are at the end of the round, if so, no player is active, it is end of the round delay
	if (state.currentBid > 0 && state.Players[state.ActivePlayer].Bet == state.currentBid) ||
		(state.currentBid == 0 && state.Players[state.ActivePlayer].Move != "") {
		stateCopy.ActivePlayer = -1
		setActivePlayer = true
	}

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []player{}

	// Loop through each player and create the hand, starting at this player, so all clients see the same order regardless of starting player
	for i := thisPlayer; i < thisPlayer+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == stateCopy.ActivePlayer {
			setActivePlayer = true
			stateCopy.ActivePlayer = i - thisPlayer
		}

		player := statePlayers[playerIndex]
		player.Hand = ""

		// Loop through and build hand string, taking
		// care to not disclose the first card of a hand to other players
		for cardIndex, card := range player.cards {
			if cardIndex > 0 || playerIndex == thisPlayer {
				player.Hand += valueLookup[card.value] + suitLookup[card.suit]
			} else {
				player.Hand += "??"
			}
		}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, player)
	}

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		moves := []string{}
		for _, move := range getValidMoves() {
			moves = append(moves, fmt.Sprint(move.move, " ", move.name))
		}
		stateCopy.ValidMoves = moves
	}

	c.IndentedJSON(http.StatusOK, stateCopy)
}
