package main

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
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
	currentBid int
	gameOver   bool
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
	state.Round = 0
	state.StartingPlayer = -1
	state.Players = []player{
		{Name: "Thom Bot", Purse: 500},
		{Name: "Chat GPT", Purse: 500},
		{Name: "Player", Purse: 500},
		{Name: "Mozz Bot", Purse: 500},
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

	// Set the starting player, but run logic to skip over them if they folded
	state.ActivePlayer = state.StartingPlayer - 1
	nextValidPlayer()

	dealCards()
}

func dealCards() {
	for i, player := range state.Players {
		if player.Status == 1 {
			player.cards = append(player.cards, state.deck[state.deckIndex])
			state.Players[i] = player
			state.deckIndex++
		}
	}
}

func endGame() {
	// A real server would compare hands to see who won and give the pot to the winner.
	// For now, we just set gameOver so play can start over
	// The next request for /state will start a new game
	state.gameOver = true
	state.LastResult = "LOOK WHO WON"
	state.Round = 5

	for _, player := range state.Players {
		state.Pot += player.Bet
	}
}

// Emulates simplified player/logic for 5 card stud
func emulateGame(checkForRoundOnly bool) {

	if state.gameOver {
		state.Round = 0
		state.gameOver = false
		newRound()
	}

	// Check if only one player is left
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == 1 {
			playersLeft++
		}
	}

	if playersLeft == 1 {
		endGame()
		return
	}

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table
	if (state.currentBid > 0 && state.Players[state.ActivePlayer].Bet == state.currentBid) ||
		(state.currentBid == 0 && state.Players[state.ActivePlayer].Move != "") {
		if state.Round == 4 {
			endGame()
		} else {
			newRound()
		}
		return
	}

	if checkForRoundOnly {
		return
	}

	// Peform a move for this BOT if they are in the game and have not folded
	if state.Players[state.ActivePlayer].Status == 1 {
		moves := getValidMoves()

		// Default to FOLD
		choice := 0

		// Never fold if CHECK is an option. These BOTs are smarter than the average bear.
		if len(moves) > 1 && moves[1].move == "CH" {
			choice = 1
		}

		// Never fold if a BOT has a jack or higher. Why not, right?
		if len(moves) > 1 && slices.IndexFunc(state.Players[state.ActivePlayer].cards, func(c card) bool { return c.value > 10 }) > -1 {
			choice = 1
		}

		// Most of the time, consider bet/call/raise
		if len(moves) > 1 && rand.Intn(3) > 0 {

			// Avoid endless raises - BOTs only raise up to 10
			if state.currentBid >= 10 || rand.Intn(3) > 0 {
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

	// Sanity check - only the the active player can perform a move
	if playerNum != state.ActivePlayer {
		return false
	}

	player := state.Players[state.ActivePlayer]

	// Sanity Check 2 - Player status should be 1 to move. In theory they should never be active if their status is != 1
	if player.Status != 1 {
		return false
	}

	// Default as move was not completed
	completedMove := false

	switch move {
	case "FO": // FOLD
		player.Status = 2
		completedMove = true
	case "CH": // CHECK
		if state.currentBid == 0 {
			completedMove = true
		}
	case "BL": // BET LOW (5)
		fallthrough
	case "RL": // RAISE LOW (5)
		delta := state.currentBid + 5 - player.Bet
		if (state.currentBid == 0 || move == "RL") && player.Purse >= delta {
			state.currentBid += 5
			player.Bet += delta
			player.Purse -= delta
			completedMove = true
		}
	case "CA": // CALL
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
		nextValidPlayer()

	}

	return completedMove
}

func nextValidPlayer() {
	// Move to next player
	state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

	// Skip over player if not in this game (joined late / folded)
	for state.Players[state.ActivePlayer].Status != 1 {
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
	}
}

func getValidMoves() []validMove {
	moves := []validMove{
		{move: "FO", name: "Fold"},
	}

	player := state.Players[state.ActivePlayer]

	if state.currentBid == 0 {
		moves = append(moves, validMove{move: "CH", name: "Check"})

		if player.Purse >= 5 {
			moves = append(moves, validMove{move: "BL", name: "Bet 5"})
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

	// Check if we are at the end of the game, round, if so, no player is active, it is end of the round delay
	if (state.gameOver ||
		state.currentBid > 0 && state.Players[state.ActivePlayer].Bet == state.currentBid) ||
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

		switch player.Status {
		case 1:
			// Loop through and build hand string, taking
			// care to not disclose the first card of a hand to other players
			for cardIndex, card := range player.cards {
				if cardIndex > 0 || playerIndex == thisPlayer || state.gameOver {
					player.Hand += valueLookup[card.value] + suitLookup[card.suit]
				} else {
					player.Hand += "??"
				}
			}
		case 2:
			player.Hand = "??"
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
