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

const MAX_PLAYERS = 6
const MOVE_TIME_GRACE_SECONDS = 4
const BOT_TIME_LIMIT = time.Second * time.Duration(2)
const PLAYER_TIME_LIMIT = time.Second * time.Duration(45) // 45
const PLAYER_PENALIZED_TIME_LIMIT = time.Second * time.Duration(7)
const ENDGAME_TIME_LIMIT = time.Second * time.Duration(5)
const START_TIME_LIMIT = time.Second * time.Duration(5) // 11
const NEW_ROUND_FIRST_PLAYER_BUFFER = 0

// Drop players who do not make a move in 5 minutes
const PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

const PROMPT_WAITING_FOR_MORE_PLAYERS = "Waiting for players"
const PROMPT_WAITING_ON_READY = "Waiting for everyone to ready up."
const PROMPT_STARTING_IN = "Starting in "

const (
	SCORE_ONES        = 0
	SCORE_UPPER_TOTAL = 6
	SCORE_UPPER_BONUS = 7

	SCORE_SET3      = 8
	SCORE_SET4      = 9
	SCORE_FULLHOUSE = 10
	SCORE_SRUN      = 11
	SCORE_LRUN      = 12
	SCORE_CHANCE    = 13
	SCORE_FUJZEE    = 14

	SCORE_TOTAL = 15
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
		state.gameOver = false
		for i := 0; i < len(state.Players); i++ {
			state.Players[i].Scores = make([]int, 16)
			for j := 0; j < 16; j++ {
				state.Players[i].Scores[j] = -1
			}
		}
	}

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
		Name:        playerName,
		Scores:      make([]int, 1),
		isBot:       isBot,
		isLeaving:   false,
		isPenalized: false,
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

	// Add to end of human players but before bot players
	insertIndex := slices.IndexFunc(state.Players, func(p Player) bool { return p.isBot })
	if isBot || insertIndex < 0 {
		insertIndex = len(state.Players)
	}

	state.Players = slices.Insert(state.Players, insertIndex, newPlayer)
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

	// Add new player if the game hasn't started yet and spots are available
	if state.clientPlayer < 0 && state.Round == 0 && len(state.Players) < MAX_PLAYERS {
		state.addPlayer(playerName, false)
		state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.Name, playerName) })

		// Set the ping for this player so they are counted as active when updating the lobby
		state.playerPing()

		// Update the lobby with the new state (new player joined)
		state.updateLobby()
	}
}

func (state *GameState) endGame(abortGame bool) {
	// The next request for /state will start a new game once the timer has counted down

	// If the game hasn't started, no need to do anything.
	if state.Round == 0 {
		return
	}

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 99
	state.RollsLeft = 0

	winningPlayer := -1
	winningScore := 0

	if !abortGame {
		for index, player := range state.Players {

			// Calculate the player's final score
			score := player.Scores[SCORE_UPPER_TOTAL] + player.Scores[SCORE_UPPER_BONUS]
			for i := SCORE_SET3; i < SCORE_TOTAL; i++ {
				score += player.Scores[i]
			}
			player.Scores[SCORE_TOTAL] = score

			if !player.isLeaving && score > winningScore {
				winningPlayer = index
				winningScore = score
			}
		}
	}

	if winningPlayer >= 0 {
		nameIndex := 0
		if state.Players[winningPlayer].isBot {
			nameIndex = 1
		}
		state.Prompt = fmt.Sprintf("%s won with a score of %d!", state.Players[winningPlayer].Name[nameIndex:], winningScore)
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
	} else {
		state.Prompt = "The game was aborted early"
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		//state.resetGame()
	}

	log.Println(state.Prompt)
}

func (state *GameState) resetGame() {

	for i := 0; i < len(state.Players); i++ {
		state.Players[i].Scores = make([]int, 1)
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

	// If still on round 0 (waiting to start), check if the game can start
	if state.Round == 0 {

		// Check if ready wait time has expired and at least one non bot player exists and all players are ready
		if slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot == false }) &&
			!slices.ContainsFunc(state.Players, func(p Player) bool { return p.Scores[0] == 0 }) {
			waitTime := int(time.Until(state.moveExpires).Seconds())
			if waitTime < 1 {
				state.newRound()
			} else {
				state.Prompt = PROMPT_STARTING_IN + strconv.Itoa(waitTime)
			}
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

	// If there is no active player, or currently waiting on a move, exit
	if state.ActivePlayer < 0 || int(time.Until(state.moveExpires).Seconds()) > 0 {
		return
	}

	// Force an action for this player or BOT if they are in the game and have not folded
	player := &state.Players[state.ActivePlayer]

	validScores, diceSets, sortedDice := state.getValidScores()

	if !player.isBot {
		// Human player did not respond in time. If they haven't rolled at all, penalize them
		// so they have a shorter period the next round. Once they roll, they are out of penalty
		if state.RollsLeft == 2 {
			player.isPenalized = true
		} else {
			player.isPenalized = false
		}

		// If human, score the next available score location, even if it scores zero.
		nextValidIndex := slices.IndexFunc(validScores, func(score int) bool { return score > -1 })
		state.scoreRoll(nextValidIndex)

	} else {

		// If not on the final roll, see if the bot should re-roll
		if state.RollsLeft > 0 {

			// If a small run, attempt to get large run if not yet scored
			if validScores[SCORE_SRUN] > 0 && validScores[SCORE_LRUN] == 0 {

				// Compact diceparts to just get unique digits - for easy run detection
				diceParts := strings.Split(sortedDice, "")
				diceDistinct := strings.Join(slices.Compact(diceParts), "")

				for _, keep := range []string{"1234", "2345", "3456"} {
					if diceDistinct == keep {
						state.rollDiceKeeping(keep)
						return
					}
				}
			}

			// Otherise, just try to preserve the largest helpful set, unless a full house was found
			if validScores[SCORE_FULLHOUSE] <= 0 && len(diceSets) > 1 {

				// Sort dice sets in descending order by larget set
				sort.Slice(diceSets, func(i, j int) bool {
					return len(diceSets[i]) > len(diceSets[j])
				})

				// Prefer to keep the largest set for an unfilled upper spot
				selectedSet := slices.IndexFunc(diceSets, func(set string) bool {
					val, _ := strconv.Atoi(string(set[0]))
					return validScores[val-1] > 0
				})

				if selectedSet < 0 {
					selectedSet = 0
				}

				// Roll dice, keeping the first (largest) set
				state.rollDiceKeeping(diceSets[selectedSet])
				return
			}

		}

		// Out of rolls - simply fill in highest scoring spot (not the brightest bot)
		bestIndex := -1
		bestScore := -1
		for index, score := range validScores {
			if score > bestScore && index != SCORE_CHANCE {
				bestIndex = index
				bestScore = score
			}
		}

		// Score chance if the best score is 0
		if bestScore < 1 && validScores[SCORE_CHANCE] > 0 {
			bestIndex = SCORE_CHANCE
		}

		// Override with full house if found
		if validScores[SCORE_FULLHOUSE] > 0 {
			bestIndex = SCORE_FULLHOUSE
		}

		state.scoreRoll(bestIndex)
	}
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
		if !player.isLeaving && !player.isBot {
			playersLeft++
		}
	}

	// If the last player dropped, stop the game and update the lobby
	if playersLeft == 0 {
		state.endGame(true)
	}
	state.dropInactivePlayers(false, false)
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *GameState) playerPing() {
	state.Players[state.clientPlayer].lastPing = time.Now()

	// An active player won't be penalized for now
	state.Players[state.clientPlayer].isPenalized = false
}

// Toggle ready state if waiting to start game
func (state *GameState) toggleReady() {

	if state.Round == 0 && len(state.Players) > 1 {
		// Toggle ready state for this player
		state.Players[state.clientPlayer].Scores[0] = (state.Players[state.clientPlayer].Scores[0] + 1) % 2

		// If all players have readied, start the countdown timer
		if slices.ContainsFunc(state.Players, func(p Player) bool { return !p.isBot }) &&
			!slices.ContainsFunc(state.Players, func(p Player) bool { return p.Scores[0] == 0 }) {
			state.moveExpires = time.Now().Add(START_TIME_LIMIT)
		}

		// Update prompt
		state.runGameLogic()
	}
}

// Performs the requested score for the active player, and returns true if successful
func (state *GameState) scoreRoll(index int, internalCall ...bool) bool {
	validScores, _, _ := state.getValidScores()

	// Check if a valid score index was chosen
	if index < len(validScores) && validScores[index] > -1 {

		player := &state.Players[state.ActivePlayer]

		// Score the current roll
		player.Scores[index] = validScores[index]

		// Recalculate the upper total + bonus if changed
		if index < SCORE_UPPER_TOTAL {
			score := 0
			filledIn := 0
			for i := SCORE_ONES; i < SCORE_UPPER_TOTAL; i++ {
				if player.Scores[i] > -1 {
					score += player.Scores[i]
					filledIn++
				}
			}

			player.Scores[SCORE_UPPER_TOTAL] = score
			if score >= 64 {
				player.Scores[SCORE_UPPER_BONUS] = 35
			} else if filledIn == 6 {
				player.Scores[SCORE_UPPER_BONUS] = 0
			}
		}

		// Move on to next player
		state.nextValidPlayer()
		return true
	}

	return false
}

func (state *GameState) resetPlayerTimer() {
	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isPenalized {
		timeLimit = PLAYER_PENALIZED_TIME_LIMIT
	}

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
	nameIndex := 0
	if state.Players[state.ActivePlayer].isBot {
		nameIndex = 1
	}
	state.Prompt = state.Players[state.ActivePlayer].Name[nameIndex:] + "'s turn"
	state.Dice = ""
	state.RollsLeft = 3
	state.rollDice("11111")
}

// Expects a string of 5 dice indexes, either 0 or 1: 0=keep, 1=roll
// For example, consider a roll "31363". To keep all the 3's and roll the 1 and 6, pass "01010"
func (state *GameState) rollDice(keepRoll string) {

	// Only roll when available, and 5 dice are passed
	if state.RollsLeft == 0 || len(keepRoll) != 5 {
		return
	}

	// Store keepRoll in the state for other players to follow along
	state.KeepRoll = keepRoll

	// Build the outcome of the new roll
	newRoll := ""

	// Preserve kept dice, rolling new dice
	for i := 0; i < 5; i++ {
		if keepRoll[i] == '1' {
			newRoll = newRoll + strconv.Itoa(rand.Intn(6)+1)
		} else {
			newRoll = newRoll + state.Dice[i:i+1]
		}
	}

	// Assign the new roll to state
	state.Dice = newRoll

	state.RollsLeft--

	state.resetPlayerTimer()

}

// Convenience function for Bot AI.
func (state *GameState) rollDiceKeeping(keepList string) {
	keepRoll := ""

	for i := 0; i < 5; i++ {
		keepThisDie := false

		// Loop through each die in the keep list to see if it applies
		for j := 0; j < len(keepList); j++ {
			if state.Dice[i] == keepList[j] {
				// If keeping, remove from the keep list so
				keepList = keepList[:j] + keepList[j+1:]
				keepThisDie = true
				break
			}
		}

		if keepThisDie {
			keepRoll += "0"
		} else {
			keepRoll += "1"
		}
	}

	state.rollDice(keepRoll)

}

func (state *GameState) getValidScores() ([]int, []string, string) {

	scores := make([]int, 15)
	currentScores := state.Players[state.ActivePlayer].Scores

	// Block out any rows that can't be scored a zero
	for i := 0; i < 15; i++ {
		if currentScores[i] >= 0 || i == SCORE_UPPER_TOTAL || i == SCORE_UPPER_BONUS {
			scores[i] = -1
		}
	}

	// Split the dice string into an array of dice
	diceParts := strings.Split(state.Dice, "")

	// Sort the dice for convenience
	sort.Strings(diceParts)
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

	// Get sorted list of unique digits - for easy run detection
	diceDistinct := strings.Join(slices.Compact(diceParts), "")

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

	// Full house ( two sets, each at least 2 - effecively a set of 2 and set of 3)
	if currentScores[SCORE_FULLHOUSE] < 0 && len(diceSets) == 2 && len(diceSets[0]) >= 2 && len(diceSets[1]) >= 2 {
		scores[SCORE_FULLHOUSE] = 25
	}

	// Small run (1234, 2345, 3456)
	if currentScores[SCORE_SRUN] < 0 && (strings.Contains(diceDistinct, "1234") || strings.Contains(diceDistinct, "2345") || strings.Contains(diceDistinct, "3456")) {
		scores[SCORE_SRUN] = 30
	}

	// Large run (12345, 23456)
	if currentScores[SCORE_LRUN] < 0 && (dice == "12345" || dice == "23456") {
		scores[SCORE_LRUN] = 40
	}

	// Chance
	if currentScores[SCORE_CHANCE] < 0 {
		scores[SCORE_CHANCE] = diceTotal
	}

	// All five - Fujzee!
	if currentScores[SCORE_FUJZEE] < 0 && len(diceSets) == 1 {
		scores[SCORE_FUJZEE] = 50
	}

	return scores, diceSets, dice
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

		// If the round is 0, only return the first score (ready or not)
		//if stateCopy.Round == 0 {
		//	statePlayers[playerIndex].Scores = statePlayers[playerIndex].Scores[0:1]
		//}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])

	}

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		stateCopy.ValidScores, _, _ = state.getValidScores()

		// Personalize prompt
		stateCopy.Prompt = "Your turn"
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
	humanAvailSlots := MAX_PLAYERS
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
