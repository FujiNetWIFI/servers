package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

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

const ANTE = 1
const BRINGIN = 2
const LOW = 5
const HIGH = 10
const STARTING_PURSE = 200
const MOVE_TIME_GRACE_SECONDS = 4
const BOT_TIME_LIMIT = time.Second * time.Duration(3)
const PLAYER_TIME_LIMIT = time.Second * time.Duration(39)
const ENDGAME_TIME_LIMIT = time.Second * time.Duration(12)
const NEW_ROUND_FIRST_PLAYER_BUFFER = time.Second * time.Duration(1)

// Drop players who do not make a move in 5 minutes
const PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

const WAITING_MESSAGE = "Waiting for more players"

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BB": "POST",
	"BL": "BET", // BET LOW (e.g. 5 of 5/10, or 2 of 2/5 first round)
	"BH": "BET", // BET HIGH (e.g. 10)
	"CA": "CALL",
	"RL": "RAISE",
	"RH": "RAISE",
}

var botNames = []string{"Clyd", "Jim", "Kirk", "Hulk", "Fry", "Meg", "Grif", "AI"}

type validMove struct {
	Move string `json:"move"`
	Name string `json:"name"`
}

type card struct {
	value int
	suit  int
}

type Status int64

const (
	STATUS_WAITING Status = 0
	STATUS_PLAYING Status = 1
	STATUS_FOLDED  Status = 2
	STATUS_LEFT    Status = 3
)

type player struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Bet    int    `json:"bet"`
	Move   string `json:"move"`
	Purse  int    `json:"purse"`
	Hand   string `json:"hand"`

	// Internal
	isBot    bool
	cards    []card
	lastPing time.Time
}

type gameState struct {
	// External (JSON)
	LastResult   string      `json:"lastResult"`
	Round        int         `json:"round"`
	Pot          int         `json:"pot"`
	ActivePlayer int         `json:"activePlayer"`
	MoveTime     int         `json:"moveTime"`
	Viewing      int         `json:"viewing"`
	ValidMoves   []validMove `json:"validMoves"`
	Players      []player    `json:"players"`

	// Internal
	deck         []card
	deckIndex    int
	currentBet   int
	gameOver     bool
	clientPlayer int
	table        string
	wonByFolds   bool
	isMockGame   bool
	moveExpires  time.Time
	serverName   string
}

// Used to send a list of available tables
type gameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

func initializeGameServer() {

	// Append BOT to botNames array
	for i := 0; i < len(botNames); i++ {
		botNames[i] = botNames[i] + " BOT"
	}
}

func createGameState(playerCount int, isMockGame bool) *gameState {

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
	state.isMockGame = isMockGame

	// Force between 2 and 8 players during mock games
	if isMockGame {
		playerCount = int(math.Min(math.Max(2, float64(playerCount)), 8))
	}

	// Pre-populate player pool with bots
	for i := 0; i < playerCount; i++ {
		state.addPlayer(botNames[i], true)
	}

	if playerCount < 2 {
		state.LastResult = WAITING_MESSAGE
	}

	log.Print("Created GameState")
	return &state
}

func (state *gameState) updateMockPlayerCount(playerCount int) {
	if playerCount <= len(state.Players) || playerCount > 8 {
		return
	}

	// Add bot players that are waiting to play the next game
	delta := playerCount - len(state.Players)
	for i := 0; i < delta; i++ {
		state.addPlayer(botNames[len(state.Players)], true)
	}
}

func (state *gameState) newRound() {

	// Drop any players that left last round
	state.dropInactivePlayers(true)

	// Check if multiple players are still playing
	if state.Round > 0 {
		playersLeft := 0
		for _, player := range state.Players {
			if player.Status == STATUS_PLAYING {
				playersLeft++
			}
		}

		if playersLeft < 2 {
			state.endGame(false)
			return
		}
	} else {
		if len(state.Players) < 2 {
			return
		}
	}

	state.Round++

	// Clear pot at start so players can anti
	if state.Round == 1 {
		state.Pot = 0
		state.gameOver = false
	}

	// Reset players for this round
	for i := 0; i < len(state.Players); i++ {

		// Get pointer to player
		player := &state.Players[i]

		if state.Round > 1 {
			// If not the first round, add any bets into the pot
			state.Pot += player.Bet
		} else {

			// First round of a new game

			// A bot will leave if it has under 25 chips, another will take their place
			if player.isBot && player.Purse < 25 {
				player.Purse = STARTING_PURSE
				for j := 0; j < len(botNames); j++ {
					botNameUsed := false
					for k := 0; k < len(state.Players); k++ {
						if strings.EqualFold(botNames[j], state.Players[k].Name) {
							botNameUsed = true
							break
						}
					}
					if !botNameUsed {
						player.Name = botNames[j]
						break
					}
				}
			}

			// Reset player status and take the ANTI
			if player.Purse > 2 {
				player.Status = STATUS_PLAYING
				player.Purse -= ANTE
				state.Pot += ANTE
			} else {
				// Player doesn't have enough money to play
				player.Status = STATUS_WAITING
			}
			player.cards = []card{}
		}

		// Reset player's last move/bet for this round
		player.Move = ""
		player.Bet = 0
	}

	state.currentBet = 0

	// First round of a new game? Shuffle the cards and deal an extra card
	if state.Round == 1 {

		// Shuffle the deck 7 times :)
		for shuffle := 0; shuffle < 7; shuffle++ {
			rand.Shuffle(len(state.deck), func(i, j int) { state.deck[i], state.deck[j] = state.deck[j], state.deck[i] })
		}
		state.deckIndex = 0
		state.dealCards()
		if state.LastResult == WAITING_MESSAGE {
			state.LastResult = ""
		}
	}

	state.dealCards()
	state.ActivePlayer = state.getPlayerWithBestVisibleHand(state.Round > 1)
	state.resetPlayerTimer(true)
}

func (state *gameState) getPlayerWithBestVisibleHand(highHand bool) int {

	ranks := [][]int{}

	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		if player.Status == STATUS_PLAYING {
			rank := getRank(player.cards[1:len(player.cards)])

			// Add player number to start of rank to hold on to when sorting
			rank = append([]int{i}, rank...)
			ranks = append(ranks, rank)
		}
	}

	// Sort the ranks by value first, the breaking tie by suit
	sort.SliceStable(ranks, func(i, j int) bool {
		for k := 1; k < 9; k++ {
			if ranks[i][k] != ranks[j][k] {
				return ranks[i][k] < ranks[j][k]
			}
		}
		return false
	})

	// Return player with highest (or lowest) hand
	result := 0
	if highHand {
		result = ranks[0][0]
	} else {
		result = ranks[len(ranks)-1][0]
	}

	// If something goes amiss, just select the first player
	if result < 0 {
		result = 0
	}
	return result
}

func (state *gameState) dealCards() {
	for i, player := range state.Players {
		if player.Status == STATUS_PLAYING {
			player.cards = append(player.cards, state.deck[state.deckIndex])
			state.Players[i] = player
			state.deckIndex++
		}
	}
}

func (state *gameState) addPlayer(playerName string, isBot bool) {

	newPlayer := player{
		Name:   playerName,
		Status: 0,
		Purse:  STARTING_PURSE,
		cards:  []card{},
		isBot:  isBot,
	}

	state.Players = append(state.Players, newPlayer)
}

func (state *gameState) setClientPlayerByName(playerName string) {
	// If no player name was passed, simply return. This is an anonymous viewer.
	if len(playerName) == 0 {
		state.clientPlayer = -1
		return
	}
	state.clientPlayer = slices.IndexFunc(state.Players, func(p player) bool { return strings.EqualFold(p.Name, playerName) })

	// Add new player if there is room
	if state.clientPlayer < 0 && len(state.Players) < 8 {
		state.addPlayer(playerName, false)
		state.clientPlayer = len(state.Players) - 1
		state.updateLobby()
	}

	// Extra logic if a player is requesting
	if state.clientPlayer > 0 {

		// In case a player returns while they are still in the "LEFT" status (before the current game ended), add them back in as waiting
		if state.Players[state.clientPlayer].Status == STATUS_LEFT {
			state.Players[state.clientPlayer].Status = STATUS_WAITING
		}
	}
}

func (state *gameState) endGame(abortGame bool) {
	// The next request for /state will start a new game

	// Hand rank details
	// Rank: SF, 4K, FH, F, S, 3K, 2P, 1P, HC

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 5

	remainingPlayers := []int{}
	pockets := [][]cardrank.Card{}

	for index, player := range state.Players {
		state.Pot += player.Bet
		if !abortGame && player.Status == STATUS_PLAYING {
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

	if pivot == 0 {
		// If nobody won, the game was aborted. Display the waiting message if this
		// server does not contains bots.
		humanAvailSlots, _ := state.getHumanPlayerCountInfo()
		if humanAvailSlots == 8 {
			state.LastResult = WAITING_MESSAGE
			state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		} else {
			state.moveExpires = time.Now()
		}
		return
	}

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
		result = strings.ReplaceAll(result, "kickers", "kicker")
	} else {
		state.wonByFolds = true
		result += " won by default"
	}
	state.LastResult = result

	state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)

	log.Println(result)
}

// Emulates simplified player/logic for 5 card stud
func (state *gameState) runGameLogic() {
	state.playerPing()

	// We can't play a game until there are at least 2 players
	if len(state.Players) < 2 {
		// Reset the round to 0 so the client knows there is no active game being run
		state.Round = 0
		state.Pot = 0
		state.ActivePlayer = -1
		return
	}

	// Very first call of state? Initialize first round but do not play for any BOTs
	if state.Round == 0 {
		state.newRound()
		return
	}

	//isHumanPlayer := state.ActivePlayer == state.clientPlayer

	if state.gameOver {

		// Create a new game if the end game delay is past
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false)
			state.Round = 0
			state.Pot = 0
			state.gameOver = false
			state.newRound()
		}
		return
	}

	// Check if only one player is left
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING {
			playersLeft++
		}
	}

	// If only one player is left, just end the game now
	if playersLeft == 1 {
		state.endGame(false)
		return
	}

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table
	if state.ActivePlayer > -1 {
		if (state.currentBet > 0 && state.Players[state.ActivePlayer].Bet == state.currentBet) ||
			(state.currentBet == 0 && state.Players[state.ActivePlayer].Move != "") {
			if state.Round == 4 {
				state.endGame(false)
			} else {
				state.newRound()
			}
			return
		}
	}

	// If a real game, return if the move timer has not expired
	if !state.isMockGame {
		// Check timer if no active player, or the active player hasn't already left
		if state.ActivePlayer == -1 || state.Players[state.ActivePlayer].Status != STATUS_LEFT {
			moveTimeRemaining := int(time.Until(state.moveExpires).Seconds())
			if moveTimeRemaining > 0 {
				return
			}
		}
	} else {
		// If in a mock game, return if the client is the active player
		if !state.Players[state.ActivePlayer].isBot {
			return
		}
	}

	// If there is no active player, we are done
	if state.ActivePlayer < 0 {
		return
	}

	// Edge case - player leaves when it is their move - skip over them
	if state.Players[state.ActivePlayer].Status == STATUS_LEFT {
		state.nextValidPlayer()
		return
	}

	// Force a move for this player or BOT if they are in the game and have not folded
	if state.Players[state.ActivePlayer].Status == STATUS_PLAYING {
		cards := state.Players[state.ActivePlayer].cards
		moves := state.getValidMoves()

		// Default to FOLD
		choice := 0

		// Never fold if CHECK is an option. This applies to forced player moves as well as bots
		if len(moves) > 1 && moves[1].Move == "CH" {
			choice = 1
		}

		// If this is a bot, pick the best move using some simple logic (sometimes random)
		if state.Players[state.ActivePlayer].isBot {

			// Potential TODO: If on round 5 and check is not an option, fold if there is a visible hand that beats the bot's hand.
			//if len(cards) == 5 && len(moves) > 1 && moves[1].Move == "CH" {}

			// Hardly ever fold early if a BOT has an jack or higher.
			if state.Round < 3 && len(moves) > 1 && rand.Intn(3) > 0 && slices.ContainsFunc(cards, func(c card) bool { return c.value > 10 }) {
				choice = 1
			}

			// Likely don't fold if BOT has a pair or better
			rank := getRank(cards)
			if rank[0] < 300 && rand.Intn(20) > 0 {
				choice = 1
			}

			// Don't fold if BOT has a 2 pair or better
			if rank[0] < 200 {
				choice = 1
			}

			// Raise the bet if three of a kind or better
			if len(moves) > 2 && rank[0] < 312 && state.currentBet < LOW {
				choice = 2
			} else if len(moves) > 2 && state.getPlayerWithBestVisibleHand(true) == state.ActivePlayer && state.currentBet < HIGH && (rank[0] < 306) {
				choice = len(moves) - 1
			} else {

				// Consider bet/call/raise most of the time
				if len(moves) > 1 && rand.Intn(3) > 0 && (len(cards) > 2 ||
					cards[0].value == cards[1].value ||
					math.Abs(float64(cards[1].value-cards[0].value)) < 3 ||
					cards[0].value > 8 ||
					cards[1].value > 5) {

					// Avoid endless raises
					if state.currentBet >= 20 || rand.Intn(3) > 0 {
						choice = 1
					} else {
						choice = rand.Intn(len(moves)-1) + 1
					}

				}
			}
		}

		// Bounds check - clamp the move to the end of the array if a higher move is desired.
		// This may occur if a bot wants to call, but cannot, due to limited funds.
		if choice > len(moves)-1 {
			choice = len(moves) - 1
		}

		move := moves[choice]

		state.performMove(move.Move, true)
	}

}

// Drop players that left or have not pinged within the expected timeout
func (state *gameState) dropInactivePlayers(inMiddleOfGame bool) {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	players := []player{}

	for _, player := range state.Players {
		if len(state.Players) > 0 && player.Status != STATUS_LEFT && (inMiddleOfGame || player.isBot || player.lastPing.Compare(cutoff) > 0) {
			players = append(players, player)
		}
	}

	// If one player is left, don't drop them within the round, let the normal game end take care of it
	if inMiddleOfGame && len(players) == 1 {
		return
	}

	// Update the client player index in case it changed due to players being dropped
	if len(players) > 0 {
		state.clientPlayer = slices.IndexFunc(players, func(p player) bool { return strings.EqualFold(p.Name, state.Players[state.clientPlayer].Name) })
	}

	// Store if players were dropped, before updating the state player array
	playersWereDropped := len(state.Players) != len(players)

	state.Players = players

	// If only one player is left, we are waiting for more
	if len(state.Players) < 2 {
		state.LastResult = WAITING_MESSAGE
	}

	// If any player state changed, update the lobby
	if playersWereDropped {
		state.updateLobby()
	}

}

func (state *gameState) clientLeave() {
	if state.clientPlayer < 0 {
		return
	}
	player := &state.Players[state.clientPlayer]

	player.Status = STATUS_LEFT
	player.Move = "LEFT"

	// Check if no human players are playing. If so, end the game
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING && !player.isBot {
			playersLeft++
		}
	}

	// If the last player dropped, stop the game and update the lobby
	if playersLeft == 0 {
		state.endGame(true)
		state.dropInactivePlayers(false)
		return
	}
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *gameState) playerPing() {
	state.Players[state.clientPlayer].lastPing = time.Now()
}

// Performs the requested move for the active player, and returns true if successful
func (state *gameState) performMove(move string, internalCall ...bool) bool {

	if len(internalCall) == 0 || !internalCall[0] {
		state.playerPing()
	}

	// Get pointer to player
	player := &state.Players[state.ActivePlayer]

	// Sanity check if player is still in the game. Unless there is a bug, they should never be active if their status is != PLAYING
	if player.Status != STATUS_PLAYING {
		return false
	}

	// Only perform move if it is a valid move for this player
	if !slices.ContainsFunc(state.getValidMoves(), func(m validMove) bool { return m.Move == move }) {
		return false
	}

	if move == "FO" { // FOLD
		player.Status = STATUS_FOLDED
	} else if move != "CH" { // Not Checking

		// Default raise to 0 (effectively a CALL)
		raise := 0

		if move == "BH" || move == "RH" {
			raise = HIGH
		} else if move == "BL" || move == "RL" {
			raise = LOW
			if state.currentBet == BRINGIN {
				// If betting LOW the very first time and the pot is BRINGIN
				// just make their bet enough to make the total bet LOW
				raise -= BRINGIN
			}
		} else if move == "BB" {
			raise = BRINGIN
		}

		// Place the bet
		delta := state.currentBet + raise - player.Bet
		state.currentBet += raise
		player.Bet += delta
		player.Purse -= delta
	}

	player.Move = moveLookup[move]
	state.nextValidPlayer()

	return true
}

func (state *gameState) resetPlayerTimer(newRound bool) {
	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isBot {
		timeLimit = BOT_TIME_LIMIT
	}

	if newRound {
		timeLimit += NEW_ROUND_FIRST_PLAYER_BUFFER
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *gameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

	// Skip over player if not in this game (joined late / folded)
	for state.Players[state.ActivePlayer].Status != STATUS_PLAYING {
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
	}
	state.resetPlayerTimer(false)
}

func (state *gameState) getValidMoves() []validMove {
	moves := []validMove{}

	// Any player after the bring-in player may fold
	if state.currentBet > 0 || state.Round > 1 {
		moves = append(moves, validMove{Move: "FO", Name: "Fold"})
	}

	player := state.Players[state.ActivePlayer]

	if state.currentBet < LOW {
		// First round, BET BRINGIN (2) or BET LOW
		if state.currentBet == 0 {
			if state.Round == 1 {
				moves = append(moves, validMove{Move: "BB", Name: fmt.Sprint("Post ", BRINGIN)})
			} else {
				moves = append(moves, validMove{Move: "CH", Name: "Check"})
			}
		} else if player.Purse >= state.currentBet-player.Bet {
			moves = append(moves, validMove{Move: "CA", Name: "Call"})
		}
		if state.Round < 3 && player.Purse >= LOW {
			moves = append(moves, validMove{Move: "BL", Name: fmt.Sprint("Bet ", LOW)})
		} else if state.Round > 2 && player.Purse >= HIGH {
			moves = append(moves, validMove{Move: "BH", Name: fmt.Sprint("Bet ", HIGH)})
		}
	} else {
		if player.Purse >= state.currentBet-player.Bet {
			moves = append(moves, validMove{Move: "CA", Name: "Call"})
		}
		if state.Players[state.ActivePlayer].Purse >= state.currentBet-player.Bet+LOW {
			moves = append(moves, validMove{Move: "RL", Name: fmt.Sprint("Raise ", LOW)})
		}
	}

	return moves
}

// Creates a copy of the state and modifies it to be from the
// perspective of this client (e.g. player array, visible cards)
func (state *gameState) createClientState() *gameState {

	stateCopy := *state

	setActivePlayer := false

	// Check if:
	// 1. The game is over,
	// 2. Only one player is left (waiting for another player to join)
	// 3. We are at the end of a round, where the active player has moved
	// This lets the client perform end of round/game tasks/animation
	if state.gameOver ||
		len(stateCopy.Players) < 2 ||
		(stateCopy.ActivePlayer > -1 && ((state.currentBet > 0 && state.Players[state.ActivePlayer].Bet == state.currentBet) ||
			(state.currentBet == 0 && state.Players[state.ActivePlayer].Move != ""))) {
		stateCopy.ActivePlayer = -1
		setActivePlayer = true
	}

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []player{}

	// When on observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set flag to let client know they are not actively part of the game
	start := state.clientPlayer
	if start < 0 {
		start = 0
		stateCopy.Viewing = 1
	} else {
		stateCopy.Viewing = 0
	}

	// Loop through each player and create the hand, starting at this player, so all clients see the same order regardless of starting player
	for i := start; i < start+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == stateCopy.ActivePlayer {
			setActivePlayer = true
			stateCopy.ActivePlayer = i - start
		}

		player := statePlayers[playerIndex]
		player.Hand = ""

		switch player.Status {
		case STATUS_PLAYING:
			// Loop through and build hand string, taking
			// care to not disclose the first card of a hand to other players
			for cardIndex, card := range player.cards {
				if cardIndex > 0 || playerIndex == state.clientPlayer || (state.Round == 5 && !state.wonByFolds) {
					player.Hand += valueLookup[card.value] + suitLookup[card.suit]
				} else {
					player.Hand += "??"
				}
			}
		case STATUS_FOLDED:
			player.Hand = "??"
		}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, player)
	}

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		stateCopy.ValidMoves = state.getValidMoves()
	}

	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	if stateCopy.ActivePlayer > -1 {
		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	if stateCopy.MoveTime < 0 {
		stateCopy.MoveTime = 0
	}

	return &stateCopy
}

func (state *gameState) updateLobby() {
	if state.isMockGame {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	// Send the total human slots / players to the Lobby
	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.table)
}

func (state *gameState) getHumanPlayerCountInfo() (int, int) {
	humanAvailSlots := 8
	humanPlayerCount := 0

	for _, player := range state.Players {
		if player.isBot {
			humanAvailSlots--
		} else if player.Status != STATUS_LEFT {
			humanPlayerCount++
		}
	}
	return humanAvailSlots, humanPlayerCount
}

// Ranks hand as an array of large to small values representing sets of 4 or less. Intended for 4 visible cards or simple AI
func getRank(cards []card) []int {
	rank := []int{}
	rankSuit := []int{}
	sets := map[int]int{}

	// Loop through hand once to create sets (cards of the same value)
	for i := 0; i < len(cards); i++ {
		sets[cards[i].value]++
	}

	// Loop through a second time to add the rank of each set (or single card)
	for i := 0; i < len(cards); i++ {
		val := cards[i].value
		set := sets[val]

		// Ranking highest value the lowest so ascending sort can be used
		rank = append(rank, 100*(5-set)-val)

		// Ranking with suit as a tie breaker
		rankSuit = append(rankSuit, 100*(5-set)-(val*4+cards[i].suit))

	}

	sort.Ints(rank)
	// Fill out empty 999s to make a 4 length to avoid bounds checks
	for len(rank) < 4 {
		rank = append(rank, 999)
	}
	sort.Ints(rankSuit)
	rank = append(rank, rankSuit...)
	for len(rank) < 8 {
		rank = append(rank, 999)
	}
	return rank
}
