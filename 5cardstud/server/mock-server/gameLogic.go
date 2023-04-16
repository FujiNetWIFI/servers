package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"

	"github.com/cardrank/cardrank"
	"golang.org/x/exp/slices"
)

/*
5 Card Stud Rules below to serve as guideline.

The logic to support below is not all implemented, and will be done as time allows.

Rules -  Assume Limit betting: Anti 1, Bringin 2,  Low 5, High 10
Suit Rank (for comparing first to act): S,H,D,C

Winning hands - tied hands split the pot, remainder is discarded

1. All players anti (e.g.) 1
2. First round
  - Player with lowest card goes first, with a mandatory bring in of 2. Option to make full bet (5)
	- Play moves Clockwise
	- Subsequent player can call 2 (assuming no full bet yet) or full bet 5
	- Subsequent Raises are inrecements of the highest bet (5 first round, or of the highest bet in later rounds)
	- Raises capped at 3 (e.g. max 20 = 5 + 3*5 round 1)
3. Remaining rounds
	- Player with highest ranked visible hand goes first
	- 3rd Street - 5, or if a pair is showing: 10, so max is 5*4 20 or 10*4 40
	- 4th street - 10
*/

const ANTI = 1
const BRINGIN = 2
const LOW = 5
const HIGH = 10
const STARTING_PURSE = 200

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BL": "BET", // BET LOW (e.g. 5 of 5/10, or 2 of 2/5 first round)
	"BH": "BET", // BET HIGH (e.g. 10)
	"CA": "CALL",
	"RL": "RAISE",
	"RH": "RAISE",
}

var playerPool = []player{
	{Name: "Thom"},
	{Name: "You"},
	{Name: "Norman"},
	{Name: "Mozzwald"},
	{Name: "TCowboy"},
	{Name: "Andy"},
	{Name: "Decipher"},
	{Name: "Eric"},
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
	LastResult   string   `json:"lastResult"`
	Round        int      `json:"round"`
	Pot          int      `json:"pot"`
	ActivePlayer int      `json:"activePlayer"`
	ValidMoves   []string `json:"validMoves"`
	Players      []player `json:"players"`

	// Internal
	deck         []card
	deckIndex    int
	currentBid   int
	gameOver     bool
	clientPlayer int
	table        string
	wonByFolds   bool
}

func createGameState(playerCount int) *gameState {
	deck := []card{}

	// Create deck of 52 cards
	for suit := 0; suit < 4; suit++ {
		for value := 2; value < 15; value++ {
			card := card{value: value, suit: suit}
			deck = append(deck, card)
		}
	}

	state := gameState{}
	state.deck = deck
	state.Round = 0
	state.ActivePlayer = -1

	// Force between 2 and 8 players
	playerCount = int(math.Min(math.Max(2, float64(playerCount)), 8))

	state.Players = playerPool[0:playerCount]
	for i := 0; i < playerCount; i++ {
		state.Players[i].Purse = STARTING_PURSE
	}

	log.Print("Created GameState")
	return &state
}

func (state *gameState) updatePlayerCount(playerCount int) {
	if playerCount <= len(state.Players) || playerCount > 8 {
		return
	}

	delta := playerCount - len(state.Players)
	for i := 0; i < delta; i++ {
		// Create a player that is waiting to play the next game
		player := playerPool[len(state.Players)]
		player.Status = 0
		player.Purse = STARTING_PURSE
		player.cards = []card{}
		state.Players = append(state.Players, player)
	}
}

func (state *gameState) newRound() {
	state.Round++

	// Reset players for this round
	for i := 0; i < len(state.Players); i++ {

		// Get pointer to player
		player := &state.Players[i]

		if state.Round > 1 {
			// If not the first round, add any bets into the pot
			state.Pot += player.Bet
		} else {

			// First round of a new game? Reset player status and take the ANTI
			if player.Purse > 5 {
				player.Status = 1
				player.Purse -= ANTI
			} else {
				player.Status = 0
			}
			player.cards = []card{}
		}

		// Reset player's last move/bet for this round
		player.Move = ""
		player.Bet = 0
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
		state.dealCards()
	}

	// Advance to the next player - TODO - pick player with best showing hand
	state.nextValidPlayer()

	state.dealCards()
}

func (state *gameState) dealCards() {
	for i, player := range state.Players {
		if player.Status == 1 {
			player.cards = append(player.cards, state.deck[state.deckIndex])
			state.Players[i] = player
			state.deckIndex++
		}
	}
}

func (state *gameState) endGame() {
	// A real server would compare hands to see who won and give the pot to the winner.
	// For now, we just set gameOver so play can start over
	// The next request for /state will start a new game

	// Hand rank details (for future)
	// Rank: SF, 4K, FH, F, S, 3K, 2P, 1P, HC

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 5

	remainingPlayers := []int{}
	pockets := [][]cardrank.Card{}

	for index, player := range state.Players {
		state.Pot += player.Bet
		if player.Status == 1 {
			remainingPlayers = append(remainingPlayers, index)
			hand := ""
			// Loop through and build hand string
			for _, card := range player.cards {
				hand += valueLookup[card.value] + suitLookup[card.suit]
			}
			pockets = append(pockets, cardrank.Must(hand))
		}
	}

	evs := cardrank.StudFive.EvalPockets(pockets, nil)
	order, pivot := cardrank.Order(evs, false)

	// Int divide, so "house" takes remainder
	perPlayerWinnings := state.Pot / pivot

	result := ""

	for i := 0; i < pivot; i++ {
		player := &state.Players[remainingPlayers[order[i]]]

		// Award winnings to player's purse
		player.Purse += int(perPlayerWinnings)

		// Add player's name to result
		if result != "" {
			result += " and "
		}
		result += player.Name
	}

	if len(remainingPlayers) > 1 {
		state.wonByFolds = false
		result += strings.Join(strings.Split(strings.Split(fmt.Sprintf(" won with %s", evs[order[0]]), " [")[0], ",")[0:2], ",")
	} else {
		state.wonByFolds = true
		result += " won by default"
	}
	state.LastResult = result
	log.Println(result)
}

// Emulates simplified player/logic for 5 card stud
func (state *gameState) emulateGame() {

	// Very first call of state? Initialize first round but do not play for any BOTs.
	if state.Round == 0 {
		state.newRound()
		return
	}

	checkForRoundOnly := state.ActivePlayer == state.clientPlayer
	if state.gameOver {
		state.Round = 0
		state.gameOver = false
		state.newRound()
	}

	// Check if only one player is left
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == 1 {
			playersLeft++
		}
	}

	if playersLeft == 1 {
		state.endGame()
		return
	}

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table
	if (state.currentBid > 0 && state.Players[state.ActivePlayer].Bet == state.currentBid) ||
		(state.currentBid == 0 && state.Players[state.ActivePlayer].Move != "") {
		if state.Round == 4 {
			state.endGame()
		} else {
			state.newRound()
		}
		return
	}

	if checkForRoundOnly {
		return
	}

	// Peform a move for this BOT if they are in the game and have not folded
	if state.Players[state.ActivePlayer].Status == 1 {
		moves := state.getValidMoves()

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

			// Avoid endless raises - BOTs only raise up to HIGH
			if state.currentBid >= HIGH || rand.Intn(3) > 0 {
				choice = 1
			} else {
				choice = rand.Intn(len(moves)-1) + 1
			}

		}

		move := moves[choice]

		state.performMove(move.move)
	}

}

// Performs the requested move for the active player, and returns true if successful
func (state *gameState) performMove(move string) bool {

	// Get pointer to player
	player := &state.Players[state.ActivePlayer]

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
		delta := state.currentBid + LOW - player.Bet
		if (state.currentBid == 0 || move == "RL") && player.Purse >= delta {
			state.currentBid += LOW
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
		state.nextValidPlayer()
	}

	return completedMove
}

func (state *gameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

	// Skip over player if not in this game (joined late / folded)
	for state.Players[state.ActivePlayer].Status != 1 {
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
	}
}

func (state *gameState) getValidMoves() []validMove {
	moves := []validMove{
		{move: "FO", name: "Fold"},
	}

	player := state.Players[state.ActivePlayer]

	if state.currentBid == 0 {
		moves = append(moves, validMove{move: "CH", name: "Check"})

		if player.Purse >= LOW {
			moves = append(moves, validMove{move: "BL", name: fmt.Sprint("Bet ", LOW)})
		}
	} else {
		if player.Purse >= state.currentBid-player.Bet {
			moves = append(moves, validMove{move: "CA", name: "Call"})
		}
		if state.Players[state.ActivePlayer].Purse >= state.currentBid-player.Bet+LOW {
			moves = append(moves, validMove{move: "RL", name: fmt.Sprint("Raise ", LOW)})
		}
	}

	return moves
}

// Creates a copy of the state and modifies it to be from the
// perspective of this client (e.g. player array, visible cards)
func (state *gameState) createClientState() gameState {

	stateCopy := *state

	setActivePlayer := false

	// Check if we are at the end of a round, or the game. If so, no player is active. This lets the client perform
	// end of round/game tasks/animation
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
	for i := state.clientPlayer; i < state.clientPlayer+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == stateCopy.ActivePlayer {
			setActivePlayer = true
			stateCopy.ActivePlayer = i - state.clientPlayer
		}

		player := statePlayers[playerIndex]
		player.Hand = ""

		switch player.Status {
		case 1:
			// Loop through and build hand string, taking
			// care to not disclose the first card of a hand to other players
			for cardIndex, card := range player.cards {
				if cardIndex > 0 || playerIndex == state.clientPlayer || (state.gameOver && !state.wonByFolds) {
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
		for _, move := range state.getValidMoves() {
			moves = append(moves, fmt.Sprint(move.move, " ", move.name))
		}
		stateCopy.ValidMoves = moves
	}

	return stateCopy
}
