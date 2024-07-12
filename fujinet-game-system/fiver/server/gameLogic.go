package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure/v2"
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
	- 4th street+ - 10
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
const NEW_ROUND_FIRST_PLAYER_BUFFER = 0

// Drop players who do not make a move in 5 minutes
const PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

const PROMPT_WAITING_FOR_MORE_PLAYERS = "Waiting for more players"
const PROMPT_WAITING_ON_READY = "Ready up to play"

type Score int64

const (
	SCORE_ONES        Score = 0
	SCORE_UPPER_TOTAL Score = 6
	SCORE_UPPER_BONUS Score = 7
	SCORE_SET3        Score = 8
	SCORE_SET4        Score = 9
	SCORE_FULLHOUSE   Score = 10
	SCORE_SRUN        Score = 11
	SCORE_LRUN        Score = 12
	SCORE_CHANCE      Score = 13
	SCORE_FIVER       Score = 14
	SCORE_TOTAL       Score = 15
)

var botNames = []string{"Clyd", "Jim", "Kirk", "Hulk", "Fry", "Meg", "Grif"}

// Used to send a list of available tables
type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

func initializeGameServer() {

	// Append BOT to botNames array
	for i := 0; i < len(botNames); i++ {
		botNames[i] = "AI " + botNames[i]
	}
}

func createGameState(playerCount int) *GameState {

	state := GameState{}

	// Pre-populate player pool with bots
	for i := 0; i < playerCount; i++ {
		state.addPlayer(strconv.Itoa(i+1)+botNames[i], true)
	}

	// Initialize game in wait state
	state.resetGame()

	return &state
}

func (state *GameState) newRound() {

	// If brand new round, clear the ready flags (first index of scores) and set all scores to -1 (unset)
	if state.Round == 0 {

		for i := 0; i < len(state.Players); i++ {
			for j := 0; j < 16; j++ {
				state.Players[i].Scores[j] = -1
			}
		}
	}

	// Drop any players that left last round
	state.dropInactivePlayers(true, false)

	// Check if multiple players are still playing
	if len(state.Players) < 2 {
		if state.Round > 0 {
			state.endGame(false)
		}
		return
	}

	state.Round++
	state.ActivePlayer = -1
	state.nextValidPlayer()

}

func (state *GameState) addPlayer(playerName string, isBot bool) {

	newPlayer := Player{
		Name:      playerName,
		Scores:    make([]int, 16),
		isBot:     isBot,
		isLeaving: false,
	}

	// Create single digit alias for the player, defaulting to numbers, then letters if all letters in the name are claimed
	// A bot will always be referred to by a number
	aliasSourceName := playerName + "123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i := 0; i < len(aliasSourceName); i++ { //run a loop and iterate through each character
		alias := strings.ToUpper(string(aliasSourceName[i]))
		if alias != " " && !slices.ContainsFunc(state.Players, func(p Player) bool { return p.Alias == alias }) {
			newPlayer.Alias = alias
			break
		}
	}

	state.Players = append(state.Players, newPlayer)
}

func (state *GameState) setClientPlayerByName(playerName string) {
	// If no player name was passed, simply return. This is an anonymous viewer.
	if len(playerName) == 0 {
		state.clientPlayer = -1
		return
	}
	state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.Name, playerName) })

	// If a new player is joining, remove any old players that timed out to make space
	if state.clientPlayer < 0 {
		// Drop any players that left to make space
		state.dropInactivePlayers(false, true)
	}

	// Add new player if there is room
	if state.clientPlayer < 0 && len(state.Players) < 8 {
		state.addPlayer(playerName, false)
		state.clientPlayer = len(state.Players) - 1

		// Set the ping for this player so they are counted as active when updating the lobby
		state.playerPing()

		// Update the lobby with the new state (new player joined)
		state.updateLobby()
	}
}

func (state *GameState) endGame(abortGame bool) {
	// The next request for /state will start a new game once the timer has counted down

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 99

	winningPlayer := -1
	winningScore := 0

	for index, player := range state.Players {

		if !abortGame && !player.isLeaving && player.Scores[SCORE_TOTAL] > winningScore {
			winningPlayer = index
			winningScore = player.Scores[SCORE_TOTAL]
		}
	}

	if winningPlayer > 0 {
		state.Prompt = fmt.Sprintf("%s won with score %i", winningPlayer, winningScore)
		state.moveExpires = time.Now().Add(10)
	} else {
		state.resetGame()
	}

	log.Println(state.Prompt)
}

func (state *GameState) resetGame() {

	for i := 0; i < len(state.Players); i++ {
		state.Players[i].Scores = make([]int, 16)
		if state.Players[i].isBot {
			state.Players[i].Scores[0] = 1 // Ready
		}
	}

	state.Round = 0
	state.ActivePlayer = -1
	state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	state.moveExpires = time.Now().Add(0)
}

// The heart of teh game. Runs a single cycle of game logic
func (state *GameState) runGameLogic() {

	// Let the game know this player is active
	state.playerPing()

	// If still on round 0 (waiting to start), see if we can start the game
	if state.Round == 0 {

		// Check if ready wait time has expired and at least one non bot player exists and all players are ready
		if int(time.Until(state.moveExpires).Seconds()) < 0 &&
			slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot == false }) &&
			!slices.ContainsFunc(state.Players, func(p Player) bool { return p.Scores[0] == 0 }) {

			state.newRound()
		} else {
			if len(state.Players) > 1 {
				state.Prompt = PROMPT_WAITING_ON_READY
			} else {
				state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
			}
		}

		return
	}

	// If the game is currently over and the end game delay is past, reset the game
	if state.gameOver {
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false, false)
			state.resetGame()
		}
		return
	}

	// If there is no active player, there is nothing to do
	if state.ActivePlayer < 0 {
		return
	}

	// Force a move for this player or BOT if they are in the game and have not folded

	// cards := state.Players[state.ActivePlayer].cards
	// moves := state.getValidMoves()

	// // Default to FOLD
	// choice := 0

	// // Never fold if CHECK is an option. This applies to forced player moves as well as bots
	// if len(moves) > 1 && moves[1].Move == "CH" {
	// 	choice = 1
	// }

	// // If this is a bot, pick the best move using some simple logic (sometimes random)
	// if state.Players[state.ActivePlayer].isBot {

	// 	// Potential TODO: If on round 5 and check is not an option, fold if there is a visible hand that beats the bot's hand.
	// 	//if len(cards) == 5 && len(moves) > 1 && moves[1].Move == "CH" {}

	// 	// Hardly ever fold early if a BOT has an jack or higher.
	// 	if state.Round < 3 && len(moves) > 1 && rand.Intn(3) > 0 && slices.ContainsFunc(cards, func(c card) bool { return c.value > 10 }) {
	// 		choice = 1
	// 	}

	// 	// Likely don't fold if BOT has a pair or better
	// 	rank := getRank(cards)
	// 	if rank[0] < 300 && rand.Intn(20) > 0 {
	// 		choice = 1
	// 	}

	// 	// Don't fold if BOT has a 2 pair or better
	// 	if rank[0] < 200 {
	// 		choice = 1
	// 	}

	// 	// Raise the bet if three of a kind or better
	// 	if len(moves) > 2 && rank[0] < 312 && state.currentBet < LOW {
	// 		choice = 2
	// 	} else if len(moves) > 2 && state.getPlayerWithBestVisibleHand(true) == state.ActivePlayer && state.currentBet < HIGH && (rank[0] < 306) {
	// 		choice = len(moves) - 1
	// 	} else {

	// 		// Consider bet/call/raise most of the time
	// 		if len(moves) > 1 && rand.Intn(3) > 0 && (len(cards) > 2 ||
	// 			cards[0].value == cards[1].value ||
	// 			math.Abs(float64(cards[1].value-cards[0].value)) < 3 ||
	// 			cards[0].value > 8 ||
	// 			cards[1].value > 5) {

	// 			// Avoid endless raises
	// 			if state.currentBet >= 20 || rand.Intn(3) > 0 {
	// 				choice = 1
	// 			} else {
	// 				choice = rand.Intn(len(moves)-1) + 1
	// 			}

	// 		}
	// 	}
	// }

	// // Bounds check - clamp the move to the end of the array if a higher move is desired.
	// // This may occur if a bot wants to call, but cannot, due to limited funds.
	// if choice > len(moves)-1 {
	// 	choice = len(moves) - 1
	// }

	// move := moves[choice]

	// state.performMove(move.Move, true)

}

// Drop players that left or have not pinged within the expected timeout
func (state *GameState) dropInactivePlayers(inMiddleOfGame bool, dropForNewPlayer bool) {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	players := []Player{}
	currentPlayerName := ""
	if state.clientPlayer > -1 {
		currentPlayerName = state.Players[state.clientPlayer].Name
	}

	for _, player := range state.Players {
		if !player.isLeaving && (player.isBot || player.lastPing.Compare(cutoff) > 0) {
			players = append(players, player)
		}
	}

	// Store if players were dropped, before updating the state player array
	playersWereDropped := len(state.Players) != len(players)

	if playersWereDropped {
		state.Players = players
	}

	// If a new player is joining, don't bother updating anything else
	if dropForNewPlayer {
		return
	}

	// Update the client player index in case it changed due to players being dropped
	if len(players) > 0 {
		state.clientPlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.Name, currentPlayerName) })
	}

	// If only one player is left, we are waiting for more
	if len(state.Players) < 2 {
		state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	}

	// If any player state changed, update the lobby
	if playersWereDropped {
		state.updateLobby()
	}

}

func (state *GameState) clientLeave() {
	if state.clientPlayer < 0 {
		return
	}
	player := &state.Players[state.clientPlayer]

	player.isLeaving = true

	// Check if no human players are playing. If so, end the game
	playersLeft := 0
	for _, player := range state.Players {
		if !player.isLeaving {
			playersLeft++
		}
	}

	// If the last player dropped, stop the game and update the lobby
	if playersLeft == 0 {
		state.endGame(true)
		state.dropInactivePlayers(false, false)
		return
	}
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *GameState) playerPing() {
	state.Players[state.clientPlayer].lastPing = time.Now()
}

// Performs the requested move for the active player, and returns true if successful
func (state *GameState) performMove(move string, internalCall ...bool) bool {

	// if len(internalCall) == 0 || !internalCall[0] {
	// 	state.playerPing()
	// }

	// // Get pointer to player
	// player := &state.Players[state.ActivePlayer]

	// // Sanity check if player is still in the game. Unless there is a bug, they should never be active if their status is != PLAYING
	// if player.Status != STATUS_PLAYING {
	// 	return false
	// }

	// // Only perform move if it is a valid move for this player
	// if !slices.ContainsFunc(state.getValidMoves(), func(m validMove) bool { return m.Move == move }) {
	// 	return false
	// }

	// if move == "FO" { // FOLD
	// 	player.Status = STATUS_FOLDED
	// } else if move != "CH" { // Not Checking

	// 	// Default raise to 0 (effectively a CALL)
	// 	raise := 0

	// 	if move == "RA" {
	// 		raise = state.raiseAmount
	// 		state.raiseCount++
	// 	} else if move == "BH" {
	// 		raise = HIGH
	// 		state.raiseAmount = HIGH
	// 	} else if move == "BL" {
	// 		raise = LOW
	// 		state.raiseAmount = LOW

	// 		// If betting LOW the very first time and the pot is BRINGIN
	// 		// just make their bet enough to make the total bet LOW
	// 		if state.currentBet == BRINGIN {
	// 			raise -= BRINGIN
	// 		}
	// 	} else if move == "BB" {
	// 		raise = BRINGIN
	// 	}

	// 	// Place the bet
	// 	delta := state.currentBet + raise - player.Bet
	// 	state.currentBet += raise
	// 	player.Bet += delta
	// 	player.Purse -= delta
	// }

	// player.Move = moveLookup[move]
	// state.nextValidPlayer()

	return true
}

func (state *GameState) resetPlayerTimer() {
	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isBot {
		timeLimit = BOT_TIME_LIMIT
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer++

	// Check if we should start the next round.
	if state.ActivePlayer >= len(state.Players) {
		state.ActivePlayer = 0

		// If we reached the end of round 13, it's the end of the game!
		if state.Round == 13 {
			state.endGame(false)
			return
		} else {
			// otherwise we start a new round
			state.newRound()
		}
	}

	// Reset player timer and reset dice for the start of the player's turn
	state.Prompt = state.Players[state.ActivePlayer].Name + "'s turn"
	state.Dice = ""
	state.RollsLeft = 3
	state.rollDice("")
	state.resetPlayerTimer()

}

func (state *GameState) rollDice(keep string) {
	newRoll := ""

	// If dice to keep was specified, move the dice from the current roll
	// to the new roll
	for i := len(keep); i >= 0; i-- {
		var existingIndex = strings.IndexAny(state.Dice, string(keep[i]))
		if existingIndex > -1 {
			newRoll = newRoll + string(state.Dice[existingIndex])
			state.Dice = state.Dice[0:existingIndex] + state.Dice[existingIndex+1:]
		}
	}

	for len(newRoll) < 5 {
		newRoll = newRoll + string(rand.Intn(6)+1)
	}

	state.Dice = newRoll
	state.RollsLeft--

}

func (state *GameState) getValidScores() []int {

	scores := make([]int, 16)
	currentScores := state.Players[state.ActivePlayer].Scores

	// Sort the dice
	diceParts := strings.Split(state.Dice, "")
	sort.Strings(diceParts)

	// Sorted dice string
	dice := strings.Join(diceParts, "")

	// Build array of dice sets and dice total at the same time
	diceTotal := 0
	diceSets := []string{""}
	setIndex := 0
	for i, digit := range diceParts {
		value, _ := strconv.Atoi(digit)
		diceTotal += value
		if i == 0 || digit == diceParts[i-1] {
			diceSets[setIndex] += digit
		} else {
			setIndex++
			diceSets = append(diceSets, digit)
		}
	}

	// Now find the available dice combination and corresponding score the player may choose from for the current roll

	// Upper - Check numbers 1 to 6
	for num := 1; num <= 6; num++ {
		var setIndex = slices.IndexFunc(diceSets, func(set string) bool { return string(set[0]) == strconv.Itoa(num) })
		if currentScores[num-1] < 0 && setIndex > -1 {
			scores[num-1] = num * len(diceSets[setIndex])
		}
	}

	// Lower

	// Sets of 3 and 4
	for num := 3; num <= 4; num++ {
		if currentScores[5+num] < 0 && slices.ContainsFunc(diceSets, func(set string) bool { return len(set) >= num }) {
			scores[5+num] = diceTotal
		}
	}

	// Full house ( two sets, set of 2 and set of 3)
	if currentScores[SCORE_FULLHOUSE] < 0 && len(diceSets) == 2 && len(diceSets[0]) >= 2 {
		scores[SCORE_FULLHOUSE] = 25
	}

	// Small run (1234, 2345, 3456)
	if currentScores[SCORE_SRUN] < 0 && (dice == "1234" || dice == "2345" || dice == "3456") {
		scores[SCORE_SRUN] = 30
	}

	// Large run (12345, 23456)
	if currentScores[SCORE_LRUN] < 0 && (dice == "12345" || dice == "23456") {
		scores[SCORE_LRUN] = 40
	}

	// All five - fiver!
	if currentScores[SCORE_FIVER] < 0 && len(diceSets) == 1 {
		scores[SCORE_FIVER] = 50
	}

	// Chance
	if currentScores[SCORE_CHANCE] < 0 {
		scores[SCORE_CHANCE] = diceTotal
	}

	return scores
}

// Creates a copy of the state and modifies it to be from the
// perspective of this calling player
func (state *GameState) createClientState() *GameState {

	stateCopy := *state

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []Player{}

	// When on observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set Viewing flag to let client know they are not actively part of the game
	start := state.clientPlayer
	if start < 0 {
		start = 0
		stateCopy.Viewing = 1
	} else {
		stateCopy.Viewing = 0
	}

	// Loop through each players to add relative to calling player
	for i := start; i < start+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if playerIndex == stateCopy.ActivePlayer {
			stateCopy.ActivePlayer = i - start
		}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
	}

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		stateCopy.ValidScores = state.getValidScores()
	}

	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	if stateCopy.ActivePlayer > -1 {
		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	// No need to send move time if the calling player isn't the active player
	if stateCopy.MoveTime < 0 || stateCopy.ActivePlayer != 0 {
		stateCopy.MoveTime = 0
	}

	// Compute hash - this will be compared with an incoming hash. If the same, the entire state does not
	// need to be sent back. This speeds up checks for change in state
	stateCopy.hash = "0"
	hash, _ := hashstructure.Hash(stateCopy, hashstructure.FormatV2, nil)
	stateCopy.hash = fmt.Sprintf("%d", hash)

	return &stateCopy
}

func (state *GameState) updateLobby() {
	if !state.registerLobby {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	// Send the total human slots / players to the Lobby
	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.table)
}

// Return number of active human players in the table, for the lobby
func (state *GameState) getHumanPlayerCountInfo() (int, int) {
	humanAvailSlots := 8
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if player.isBot {
			humanAvailSlots--
		} else if player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}
	return humanAvailSlots, humanPlayerCount
}
