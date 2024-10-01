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

// These can be set to 0 for testing scenarios, so are outside of const
var BOT_TIME_LIMIT = time.Second * 3
var START_WAIT_TIME = time.Second * 5
var START_WAIT_TIME_EXTRA = time.Second * 10
var ENDGAME_TIME_LIMIT = time.Second * 8
var PLAYER_TIME_LIMIT = time.Second * 45
var PLAYER_PENALIZED_TIME_LIMIT = time.Second * 15
var NEW_ROUND_TIME_EXTRA = time.Second * 5
var PLAYER_TIME_LIMIT_SINGLE_PLAYER = time.Second * 255 // don't go over 255 as 8 bit clients expect to store in single byte

const (
	MAX_PLAYERS             = 6
	MOVE_TIME_GRACE_SECONDS = 4

	// Drop players who do not make a move in 5 minutes
	PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

	PROMPT_WAITING_FOR_MORE_PLAYERS = "Waiting for players"
	PROMPT_WAITING_ON_READY         = "Waiting for everyone to ready up."
	PROMPT_STARTING_IN              = "Starting in "
	PROMPT_YOUR_TURN                = "Your turn"
	PROMPT_GAME_ABORTED             = "The game was aborted early"

	// Special round values
	ROUND_LOBBY    = 0
	ROUND_FINAL    = 13
	ROUND_GAMEOVER = 99

	// Special score values
	SCORE_VIEWING = -2
	SCORE_UNSET   = -1
	SCORE_READY   = 1
	SCORE_UNREADY = 0

	// Score index for notable score types
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

var botNames = []string{"Clyd", "Meg", "Kirk", "Jim"}

// Used to send a list of available tables
type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

func resetTestMode() {
	// Set certain timeouts to 0 to facilitate running tests quickly
	BOT_TIME_LIMIT = 0
	START_WAIT_TIME = 0
	START_WAIT_TIME_EXTRA = 0
	ENDGAME_TIME_LIMIT = 0
	NEW_ROUND_TIME_EXTRA = 0
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

	// If there aren't enough players to play, abort the game
	if len(state.Players) < 2 {
		if state.Round > ROUND_LOBBY {
			state.endGame(true)
		}
		return
	}

	state.Round++

	// If brand new game, clear the ready flags (first index of scores) and set all scores to -1 (unset)
	// Also set any players that are not ready to spectators/viewing
	if state.Round == 1 {
		state.gameOver = false

		players := []Player{}

		clientPlayerID := state.Players[state.clientPlayer].id

		// Initialize players, adding the playing players to the front of the list
		for i := 0; i < len(state.Players); i++ {
			player := &state.Players[i]

			// This player is playing - initialize their scores
			if player.Scores[0] == SCORE_READY {
				player.Scores = make([]int, 16)
				for j := 0; j < 16; j++ {
					player.Scores[j] = SCORE_UNSET
				}
				// Append player
				players = append(players, *player)
			} else {
				// Set player to viewing
				player.isViewing = true
				player.Scores[0] = SCORE_VIEWING
			}
		}

		// Now loop through and add the spectating players at the end of the list
		for _, player := range state.Players {
			if player.isViewing {
				players = append(players, player)
			}
		}

		// Update the players array in the state with the newly sorted list
		state.Players = players

		// As the client player may have shifted positions, re-set their ID
		state.setClientPlayerByID(clientPlayerID)
	}

	state.ActivePlayer = -1
	state.nextValidPlayer()
}

func (state *GameState) addPlayer(playerID string, isBot bool) {
	isViewing := false
	newPlayer := Player{
		Name:        playerID,
		id:          playerID,
		Scores:      make([]int, 1),
		isBot:       isBot,
		isLeaving:   false,
		isPenalized: false,
		isViewing:   false,
		Alias:       0,
	}

	if !isBot {

		// Determine if the player is viewing, or if a bot should drop when they join
		if state.Round != ROUND_LOBBY {
			// Game started - player is viewing
			newPlayer.Scores[0] = SCORE_VIEWING
			newPlayer.isViewing = true
			isViewing = true
		}

		// Determine unique single character alias for human players, defaulting to the first letter of their name
		// A bot will always be referred to by the first character (a number)
		playerName := playerID

		// Find an appropriate index
		aliasSourceName := strings.ToUpper(playerName + "ZYXWUV")
		for i := 0; i < len(aliasSourceName); i++ { //run a loop and iterate through each character
			if string(aliasSourceName[i]) != " " && !slices.ContainsFunc(state.Players, func(p Player) bool { return strings.ToUpper(p.Name)[p.Alias] == aliasSourceName[i] }) {
				newPlayer.Alias = i
				break
			}
		}

		// If one of the appended letters was found, add that to the player's name after a space
		if newPlayer.Alias >= len(playerName) {
			if len(playerName) > 6 {
				playerName = playerName[:6]
			}
			playerName += " " + string(aliasSourceName[newPlayer.Alias])
			newPlayer.Alias = len(playerName) - 1
			newPlayer.Name = playerName
		}
	}

	// Add to end of human players but before bot players
	insertIndex := slices.IndexFunc(state.Players, func(p Player) bool { return p.isBot })

	// If a bot or viewer, add to end
	if isBot || isViewing || insertIndex < 0 {
		insertIndex = len(state.Players)
	}

	// If a bot, set to ready
	if isBot {
		newPlayer.Scores[0] = SCORE_READY
	}

	state.Players = slices.Insert(state.Players, insertIndex, newPlayer)
	state.refreshBots()
}

func (state *GameState) setClientPlayerByID(playerID string) bool {
	// If no player name was passed, simply return. This is an anonymous viewer.
	if len(playerID) == 0 {
		state.clientPlayer = -1
		return false
	}
	state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, playerID) })

	// If a new player is joining, remove any old players that timed out to make space
	if state.clientPlayer < 0 {
		// Drop any players that left to make space
		state.dropInactivePlayers(false, true)
	}

	// Add player to game
	if state.clientPlayer < 0 {
		state.addPlayer(playerID, false)
		state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, playerID) })

		// Set the ping for this player so they are counted as active when updating the lobby
		state.playerPing()

		// Update the lobby with the new state (new player joined)
		state.updateLobby()

		// If spectator, save state now since it won't be saved later for perf reasons
		if state.Players[state.clientPlayer].isViewing {
			return true
		}
	} else {
		// If a new game and spots available, set this player as no longer viewing
		if state.Round == ROUND_LOBBY && state.Players[state.clientPlayer].isViewing && len(state.Players) < MAX_PLAYERS {
			state.Players[state.clientPlayer].isViewing = false
		}
	}
	return false
}

func (state *GameState) endGame(abortGame bool) {
	// The next request for /state will start a new game once the timer has counted down

	// If the game hasn't started, no need to do anything.
	if state.Round == ROUND_LOBBY {
		return
	}

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = ROUND_GAMEOVER
	state.RollsLeft = 0

	winningPlayer := -1
	winningScore := 0

	if !abortGame {
		for index, player := range state.Players {
			if !player.isViewing && len(player.Scores) > SCORE_TOTAL {

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
	}

	winners := []string{}

	if winningPlayer >= 0 {
		// First gather names of winners (in case of the rare tie!)
		for _, player := range state.Players {
			if !player.isLeaving && !player.isViewing && player.Scores[SCORE_TOTAL] == winningScore {

				nameIndex := 0
				if state.Players[winningPlayer].isBot {
					nameIndex = 1
				}
				winners = append(winners, player.Name[nameIndex:])
			}
		}
		if len(winners) == 1 {
			state.Prompt = fmt.Sprintf("%s won with a score of %d!", winners[0], winningScore)
		} else if len(winners) == 2 {
			state.Prompt = fmt.Sprintf("%s and %s tied for %d!", winners[0], winners[1], winningScore)
		} else {
			state.Prompt = fmt.Sprintf("%d players tied for %d! what luck!", len(winners), winningScore)
		}
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
	} else {

		// If there are human players left, show the abort message so the winner can still view their scoreboard
		if slices.ContainsFunc(state.Players, func(p Player) bool { return !p.isLeaving && !p.isViewing && !p.isBot }) {
			state.Prompt = PROMPT_GAME_ABORTED
			state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		} else {
			// Otherwise, all the human players left, so reset the game right away
			state.resetGame()
		}
	}

	log.Println(state.Prompt)
}

// Adds/removes bots as space allows, up to the number of botsthe server started with
func (state *GameState) refreshBots() {
	if state.Round != ROUND_LOBBY {
		return
	}

	botDropped := false

	clientPlayerID := ""
	if state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		clientPlayerID = state.Players[state.clientPlayer].id
	}

	// Remove bots if overcrowded
	for len(state.Players) > 6 && slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		// If the table is full, drop a bot when this player joins
		_, _, _, botCount := state.getPlayerCounts()

		if botCount > 0 {
			for i := len(state.Players) - 1; i >= 0; i-- {
				if state.Players[i].isBot {
					state.botBox = slices.Insert(state.botBox, len(state.botBox), state.Players[i])
					state.Players = append(state.Players[:i], state.Players[i+1:]...)
					botDropped = true
					break
				}
			}
		}
	}

	// Or if the table is not full, fill it back in with bots
	for len(state.Players) < 6 && len(state.botBox) > 0 {
		state.Players = slices.Insert(state.Players, len(state.Players), state.botBox[len(state.botBox)-1])
		state.botBox = state.botBox[:len(state.botBox)-1]
	}

	// Update client player in case a bot drop affected their index
	if botDropped && state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		state.setClientPlayerByID(clientPlayerID)
	}
}

func (state *GameState) resetGame() {
	state.Round = ROUND_LOBBY
	state.ActivePlayer = -1
	state.moveExpires = time.Now()
	state.startedStartCountdown = false

	state.refreshBots()

	for i := 0; i < len(state.Players); i++ {
		// Default to single score holding ready or not (defaulting to 0 - unready)
		state.Players[i].Scores = make([]int, 1)

		// Bots are always ready to play!
		if state.Players[i].isBot {
			state.Players[i].Scores[0] = SCORE_READY
		}

		// Clear if player was viewing or not, since we are at lobby
		state.Players[i].isViewing = false
	}

	if len(state.Players) < 2 {
		state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	} else {
		state.Prompt = PROMPT_WAITING_ON_READY
	}

}

func (state *GameState) debugSkipToEnd(winners int) {
	p1 := BOT_TIME_LIMIT
	p2 := PLAYER_TIME_LIMIT
	p3 := PLAYER_PENALIZED_TIME_LIMIT
	p4 := NEW_ROUND_TIME_EXTRA
	p5 := PLAYER_TIME_LIMIT_SINGLE_PLAYER

	BOT_TIME_LIMIT = 0
	PLAYER_TIME_LIMIT = 0
	PLAYER_PENALIZED_TIME_LIMIT = 0
	NEW_ROUND_TIME_EXTRA = 0
	PLAYER_TIME_LIMIT_SINGLE_PLAYER = 0
	state.moveExpires = time.Now()

	prevRound := 0
	for !state.gameOver {
		state.runGameLogic()
		if state.Round == 13 && state.Round != prevRound {
			// Start of final round - give winners high score
			for i := 0; i < winners; i++ {
				for j := 0; j < SCORE_CHANCE; j++ {
					state.Players[i].Scores[j] = 0
				}
				state.Players[i].Scores[SCORE_CHANCE] = 500
			}
		}
		prevRound = state.Round

	}

	BOT_TIME_LIMIT = p1
	PLAYER_TIME_LIMIT = p2
	PLAYER_PENALIZED_TIME_LIMIT = p3
	NEW_ROUND_TIME_EXTRA = p4
	PLAYER_TIME_LIMIT_SINGLE_PLAYER = p5

}

// The heart of teh game. Runs a single cycle of game logic
func (state *GameState) runGameLogic() {

	// Let the game know this player is active
	state.playerPing()

	// If still on round 0 (waiting to start), check if the game can start
	if state.Round == ROUND_LOBBY {

		// Check if ready wait time has expired and at least one non bot player exists and all players are ready
		canStartNow, totalHumansReady, totalHumansNotReady, _ := state.getPlayerCounts()

		if canStartNow {

			if !state.startedStartCountdown {
				// Give a little extra start time if not everyone has readied up
				if totalHumansReady < 6 && totalHumansNotReady > 0 {
					state.moveExpires = time.Now().Add(START_WAIT_TIME_EXTRA)
				} else {
					// Everyone has readied up, so start sooner
					state.moveExpires = time.Now().Add(START_WAIT_TIME)
				}
				state.startedStartCountdown = true
			}

			waitTime := int(time.Until(state.moveExpires).Seconds())
			if waitTime < 1 {
				state.newRound()
			} else {
				state.Prompt = PROMPT_STARTING_IN + strconv.Itoa(waitTime)
			}
		} else {

			state.startedStartCountdown = false
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

	// Force an action for the active player or BOT if their time is up
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
		nextValidIndex := slices.IndexFunc(validScores, func(score int) bool { return score > SCORE_UNSET })
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

	// Track client player name and active player in case leaving shifts them
	currentActivePlayer := state.ActivePlayer

	currentPlayerID := ""
	if state.clientPlayer > -1 {
		currentPlayerID = state.Players[state.clientPlayer].id
	}

	activePlayerID := ""
	if state.ActivePlayer > -1 {
		activePlayerID = state.Players[state.ActivePlayer].id
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
		state.refreshBots()
	}

	// If a new player is joining, don't bother updating anything else
	if dropForNewPlayer {
		return
	}

	// Update the client player index in case it changed due to players being dropped
	if len(players) > 0 {
		state.clientPlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.id, currentPlayerID) })
		state.ActivePlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.id, activePlayerID) })

		// Check if the active player is the one who left, in which case, we need to start the turn of the next player in line
		if !state.gameOver && state.Round > 0 && state.ActivePlayer < 0 {
			// The player immediately after the leaving player now owns that index, so set activePlayer the the index before them
			// and call nextValidPlayer() to start their turn
			state.ActivePlayer = currentActivePlayer - 1
			state.nextValidPlayer()
		}
	}

	// If only one player is left, we are waiting for more
	if len(state.Players) < 2 && state.Round < ROUND_GAMEOVER {
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
	humanPlayersLeft := 0
	playersLeft := 0

	for _, player := range state.Players {
		if !player.isLeaving && !player.isViewing {
			playersLeft++
			if !player.isBot {
				humanPlayersLeft++
			}
		}
	}

	// If there aren't enough players to play, abort the game
	if playersLeft < 2 || humanPlayersLeft == 0 {
		state.endGame(true)
	}
	state.dropInactivePlayers(false, false)
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *GameState) playerPing() {

	// Only set ping if this player has an id
	if state.clientPlayer >= 0 {
		state.Players[state.clientPlayer].lastPing = time.Now()

		// An active player won't be penalized for now
		state.Players[state.clientPlayer].isPenalized = false
	}
}

// Returns true if enough players (including bots) readied to start, followed by # of players ready, not ready
func (state *GameState) getPlayerCounts() (bool, int, int, int) {
	canStart := false
	totalHumansReady := 0
	totalHumansNotReady := 0
	totalBots := 0

	for _, player := range state.Players {
		if !player.isBot && !player.isLeaving {
			if player.Scores[0] == SCORE_READY {
				totalHumansReady++
			} else {
				totalHumansNotReady++
			}
		} else if player.isBot {
			totalBots++
		}
	}

	if totalHumansReady > 1 || (totalHumansReady > 0 && totalBots > 0) {
		canStart = true
	}

	return canStart, totalHumansReady, totalHumansNotReady, totalBots

}

// Toggle ready state if waiting to start game
func (state *GameState) toggleReady() {

	if state.Round == ROUND_LOBBY && len(state.Players) > 1 {

		_, totalHumansReady, _, _ := state.getPlayerCounts()

		// Toggle ready state for this player if there is space
		if state.Players[state.clientPlayer].Scores[0] == SCORE_READY {
			state.Players[state.clientPlayer].Scores[0] = SCORE_UNREADY
		} else if totalHumansReady < 6 {
			state.Players[state.clientPlayer].Scores[0] = SCORE_READY
		}
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
				if player.Scores[i] > SCORE_UNSET {
					score += player.Scores[i]
					filledIn++
				}
			}

			player.Scores[SCORE_UPPER_TOTAL] = score
			if score >= 63 {
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
	} else {

		// If this is a single player against bots, relax the timeouts
		// as long as nobody is waiting to play
		_, humanCount := state.getHumanPlayerCountInfo()
		if humanCount == 1 {
			timeLimit = PLAYER_TIME_LIMIT_SINGLE_PLAYER
		} else {

			// If this is the first player of a new round, add some extra time
			// for client to animate the new round
			if state.ActivePlayer == 0 {
				timeLimit = timeLimit + NEW_ROUND_TIME_EXTRA
			}
		}
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer++

	// Skip over any viewers (spectators)
	for state.ActivePlayer < len(state.Players) && state.Players[state.ActivePlayer].isViewing {
		state.ActivePlayer++
	}

	// Check if we should start the next round.
	if state.ActivePlayer >= len(state.Players) {
		state.ActivePlayer = 0

		// If we reached the end of the final round, it's the end of the game!
		if state.Round == ROUND_FINAL {
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
	//if !UpdateLobby && state.RollsLeft == state.Round {
	//	newRoll = "55555"
	//}

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
			scores[i] = SCORE_UNSET
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
// perspective of this calling player.
// Alternatively, another player name may be passed in *pov*
// to see it from that player's perspective. Used when multiple
// people play from the same client and the order of players
// should not shift from one player to the next
func (state *GameState) createClientState(pov string) *GameState {
	stateCopy := *state

	// Set the server name - in the future this will not always be set based on state hash
	stateCopy.Name = state.serverName

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []Player{}

	// start at the passed in POV
	start := slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, pov) })

	// Default to the current player
	if start == -1 {
		start = state.clientPlayer
	}

	// When on observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set Viewing flag to let client know they are not actively part of the game
	if state.Players[state.clientPlayer].isViewing {
		start = 0
		stateCopy.Viewing = 1
	} else {
		stateCopy.Viewing = 0
	}

	currentActivePlayerID := ""
	if stateCopy.ActivePlayer > -1 {
		currentActivePlayerID = statePlayers[stateCopy.ActivePlayer].id
	}

	// Loop twice, first to add players, second to add viewers at the end
	for isViewing := 0; isViewing < 2; isViewing++ {
		// Loop through each players to add relative to calling player
		for i := start; i < start+len(statePlayers); i++ {

			// Wrap around to beginning of playar array when needed
			playerIndex := i % len(statePlayers)

			// Add this player to the copy of the state going out
			if isViewing == 0 && !statePlayers[playerIndex].isViewing {
				stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
			}

			// Add the viewers
			if isViewing == 1 && statePlayers[playerIndex].isViewing {
				stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
			}
		}
	}

	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	// If there is an active player
	if state.ActivePlayer > -1 {

		// Set the active player to the new index from the client centric players list
		stateCopy.ActivePlayer = slices.IndexFunc(stateCopy.Players, func(p Player) bool { return strings.EqualFold(p.id, currentActivePlayerID) })

		// Include the valid moves for the active player
		if stateCopy.Viewing == 0 {
			stateCopy.ValidScores, _, _ = state.getValidScores()

			// Personalize prompt if not viewing from a pob
			if len(pov) == 0 && state.ActivePlayer == state.clientPlayer {
				stateCopy.Prompt = PROMPT_YOUR_TURN
			}
		}

		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	// Ensure move time is not negative
	if stateCopy.MoveTime < 0 {
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

	// Since bots sub out for players, the available slots will always be
	// max players even if bots are present
	humanAvailSlots := MAX_PLAYERS
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if !player.isBot && !player.isLeaving && player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}

	// If the game has started, there are no more human slots available
	if state.Round > ROUND_LOBBY && state.Round < ROUND_GAMEOVER {
		humanAvailSlots = humanPlayerCount
	}

	return humanAvailSlots, humanPlayerCount
}
