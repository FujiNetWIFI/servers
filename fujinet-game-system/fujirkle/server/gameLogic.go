package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"golang.org/x/exp/slices"
)

// These can be set to 0 for testing scenarios, so are outside of const
var BOT_TIME_LIMIT = time.Second * 3
var START_WAIT_TIME = time.Second * 31
var START_WAIT_TIME_ALL_READY = time.Second * 6
var START_WAIT_TIME_ONE_PLAYER = time.Second * 3
var ENDGAME_TIME_LIMIT = time.Second * 5
var FUJIRKLE_SHOW_TIME = time.Second * 3
var PLAYER_TIME_LIMIT = time.Second * 45
var PLAYER_PENALIZED_TIME_LIMIT = time.Second * 15
var NEW_ROUND_TIME_EXTRA = time.Second * 5
var PLAYER_TIME_LIMIT_SINGLE_PLAYER = time.Second * 255 // don't go over 255 as 8 bit clients expect to store in single byte

const (
	MAX_PLAYERS             = 6
	MOVE_TIME_GRACE_SECONDS = 4

	// Wire status bits, so a client can react without inferring from the prompt
	STATUS_FUJIRKLE = 1

	PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

	PROMPT_WAITING_FOR_MORE_PLAYERS = "Waiting for players"
	PROMPT_WAITING_ON_READY         = "Waiting for everyone to ready up"
	PROMPT_STARTING_IN              = "Starting in "
	PROMPT_YOUR_TURN                = "Your turn"
	PROMPT_GAME_ABORTED             = "The game was aborted early"
	PROMPT_FUJIRKLE                 = "FUJIRKLE! No score"
	PROMPT_HOT_DICE                 = "HOT DICE! Roll all six again"

	ROUND_LOBBY    = 0
	ROUND_GAMEOVER = 99

	READY_VIEWING = -2
	READY_UNSET   = 0
	READY_YES     = 1

	MOVE_ROLL = 1
	MOVE_BANK = 2
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
	BOT_TIME_LIMIT = 0
	START_WAIT_TIME = 0
	START_WAIT_TIME_ALL_READY = 0
	START_WAIT_TIME_ONE_PLAYER = 0
	ENDGAME_TIME_LIMIT = 0
	FUJIRKLE_SHOW_TIME = 0
	NEW_ROUND_TIME_EXTRA = 0
}

func initializeGameServer() {
	for i := 0; i < len(botNames); i++ {
		botNames[i] = "AI " + botNames[i]
	}
}

func createGameState(playerCount int) *GameState {
	state := GameState{}

	for i := 0; i < playerCount; i++ {
		state.addPlayer(strconv.Itoa(i+1)+botNames[i], true)
	}

	state.resetGame()

	return &state
}

func (state *GameState) addPlayer(playerID string, isBot bool) {
	isViewing := false
	newPlayer := Player{
		Name:      playerID,
		id:        playerID,
		isBot:     isBot,
		isLeaving: false,
		isViewing: false,
		Alias:     0,
		isSmart:   isBot && len(state.Players)%2 == 0,
	}

	if !isBot {
		if state.Round != ROUND_LOBBY {
			newPlayer.Ready = READY_VIEWING
			newPlayer.isViewing = true
			isViewing = true
		}

		playerName := playerID

		aliasSourceName := strings.ToUpper(playerName + "ZYXWUV")
		for i := 0; i < len(aliasSourceName); i++ {
			if string(aliasSourceName[i]) != " " && !slices.ContainsFunc(state.Players, func(p Player) bool { return strings.ToUpper(p.Name)[p.Alias] == aliasSourceName[i] }) {
				newPlayer.Alias = i
				break
			}
		}

		if newPlayer.Alias >= len(playerName) {
			if len(playerName) > 6 {
				playerName = playerName[:6]
			}
			playerName += " " + string(aliasSourceName[newPlayer.Alias])
			newPlayer.Alias = len(playerName) - 1
			newPlayer.Name = playerName
		}
	}

	insertIndex := slices.IndexFunc(state.Players, func(p Player) bool { return p.isBot })

	if isBot || isViewing || insertIndex < 0 {
		insertIndex = len(state.Players)
	}

	if isBot {
		newPlayer.Ready = READY_YES
	}

	state.Players = slices.Insert(state.Players, insertIndex, newPlayer)
	state.refreshBots()
}

func (state *GameState) setClientPlayerByID(playerID string) bool {
	if len(playerID) == 0 {
		state.clientPlayer = -1
		return false
	}
	state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, playerID) })

	if state.clientPlayer < 0 {
		state.dropInactivePlayers(true)
	}

	if state.clientPlayer < 0 {
		state.addPlayer(playerID, false)
		state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, playerID) })

		state.playerPing()
		state.updateLobby()

		if state.Players[state.clientPlayer].isViewing {
			return true
		}
	} else {
		if state.Round == ROUND_LOBBY && state.Players[state.clientPlayer].isViewing && len(state.Players) < MAX_PLAYERS {
			state.Players[state.clientPlayer].isViewing = false
		}
	}
	return false
}

func (state *GameState) resetGame() {
	state.Round = ROUND_LOBBY
	state.ActivePlayer = -1
	state.moveExpires = time.Now()
	state.startedStartCountdown = false
	state.finalRound = false
	state.finalRoundTrigger = ""
	state.fujirkled = false
	state.TurnScore = 0
	state.Dice = ""
	state.KeptDice = ""
	state.KeepRoll = ""

	state.refreshBots()

	for i := 0; i < len(state.Players); i++ {
		state.Players[i].Score = 0
		state.Players[i].Ready = READY_UNSET
		if state.Players[i].isBot {
			state.Players[i].Ready = READY_YES
		}
		state.Players[i].isViewing = false
	}

	if len(state.Players) < 2 {
		state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	} else {
		state.Prompt = PROMPT_WAITING_ON_READY
	}
}

func (state *GameState) newRound() {
	if len(state.Players) < 2 {
		if state.Round > ROUND_LOBBY {
			state.endGame(true)
		}
		return
	}

	state.Round++

	// Brand new game - lock in who is playing and who is spectating
	if state.Round == 1 {
		state.gameOver = false

		players := []Player{}
		clientPlayerID := ""
		if state.clientPlayer >= 0 && state.clientPlayer < len(state.Players) {
			clientPlayerID = state.Players[state.clientPlayer].id
		}

		totalPlaying := 0

		for i := 0; i < len(state.Players); i++ {
			player := &state.Players[i]

			if totalPlaying < MAX_PLAYERS && (player.Ready == READY_YES || player.isBot) {
				totalPlaying++
				player.isViewing = false
				player.Score = 0
				players = append(players, *player)
			} else {
				player.isViewing = true
				player.Ready = READY_VIEWING
			}
		}

		for _, player := range state.Players {
			if player.isViewing {
				players = append(players, player)
			}
		}

		state.Players = players
		state.setClientPlayerByID(clientPlayerID)
	}

	state.ActivePlayer = -1
	state.nextValidPlayer()
}

func (state *GameState) endGame(abortGame bool) {
	if state.Round == ROUND_LOBBY {
		return
	}

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = ROUND_GAMEOVER
	state.fujirkled = false
	state.TurnScore = 0

	winningPlayer := -1
	winningScore := 0

	if !abortGame {
		for index, player := range state.Players {
			if !player.isViewing && !player.isLeaving && player.Score > winningScore {
				winningPlayer = index
				winningScore = player.Score
			}
		}
	}

	winners := []string{}
	gameResult := GameResult{}
	gameResult.Players = []GamePlayer{}

	if winningPlayer >= 0 {
		for _, player := range state.Players {
			gamePlayer := GamePlayer{}

			if !player.isLeaving && !player.isViewing && player.Score == winningScore {
				gamePlayer.Winner = true
				nameIndex := 0
				if player.isBot {
					nameIndex = 1
				}
				winners = append(winners, player.Name[nameIndex:])
			}

			if !player.isViewing {
				gamePlayer.Name = player.Name
				gamePlayer.Type = PLAYER_TYPE_HUMAN
				if player.isBot {
					gamePlayer.Type = PLAYER_TYPE_BOT
				}
				gameResult.Players = append(gameResult.Players, gamePlayer)
			}
		}

		if len(winners) == 1 {
			state.Prompt = fmt.Sprintf("%s won with a score of %d", winners[0], winningScore)
		} else if len(winners) == 2 {
			state.Prompt = fmt.Sprintf("%s and %s tied for %d!", winners[0], winners[1], winningScore)
		} else {
			state.Prompt = fmt.Sprintf("%d players tied for %d! what luck!", len(winners), winningScore)
		}
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)

		state.updateLobbyWithGameResult(&gameResult)
	} else {
		if slices.ContainsFunc(state.Players, func(p Player) bool { return !p.isLeaving && !p.isViewing && !p.isBot }) {
			state.Prompt = PROMPT_GAME_ABORTED
			state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		} else {
			state.resetGame()
		}
	}

	log.Println(state.Prompt)
}

// Adds/removes bots as space allows, up to the number of bots the server started with
func (state *GameState) refreshBots() {
	if state.Round != ROUND_LOBBY {
		return
	}

	botDropped := false

	clientPlayerID := ""
	if state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		clientPlayerID = state.Players[state.clientPlayer].id
	}

	for len(state.Players) > MAX_PLAYERS && slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		for i := len(state.Players) - 1; i >= 0; i-- {
			if state.Players[i].isBot {
				state.botBox = slices.Insert(state.botBox, len(state.botBox), state.Players[i])
				state.Players = append(state.Players[:i], state.Players[i+1:]...)
				botDropped = true
				break
			}
		}
	}

	for len(state.Players) < MAX_PLAYERS && len(state.botBox) > 0 {
		state.Players = slices.Insert(state.Players, len(state.Players), state.botBox[len(state.botBox)-1])
		state.botBox = state.botBox[:len(state.botBox)-1]
	}

	if botDropped && state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		state.setClientPlayerByID(clientPlayerID)
	}
}

// The heart of the game. Runs a single cycle of game logic
func (state *GameState) runGameLogic() {
	state.playerPing()

	if state.Round == ROUND_LOBBY {
		canStartNow, totalHumansReady, totalHumansNotReady, _ := state.getPlayerCounts()

		if canStartNow {
			if !state.startedStartCountdown || (totalHumansReady < MAX_PLAYERS && totalHumansNotReady > state.prevTotalHumansNotReady) {
				state.startedStartCountdown = true

				if totalHumansReady == 1 && totalHumansNotReady == 0 {
					state.moveExpires = time.Now().Add(START_WAIT_TIME_ONE_PLAYER)
				} else {
					state.moveExpires = time.Now().Add(START_WAIT_TIME)
				}
			}

			state.prevTotalHumansNotReady = totalHumansNotReady
			waitTime := int(time.Until(state.moveExpires).Seconds())

			if waitTime > 6 && (totalHumansReady > 5 || totalHumansNotReady == 0) {
				state.moveExpires = time.Now().Add(START_WAIT_TIME_ALL_READY)
				waitTime = int(time.Until(state.moveExpires).Seconds())
			}

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

	if state.gameOver {
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false)
			state.resetGame()
		}
		return
	}

	if state.ActivePlayer < 0 || int(time.Until(state.moveExpires).Seconds()) > 0 {
		return
	}

	// The fujirkled roll has been on screen long enough - move on. Nothing was
	// banked, so clear the pick, or clients replay it as though it had been.
	if state.fujirkled {
		state.KeepRoll = ""
		state.nextValidPlayer()
		return
	}

	player := &state.Players[state.ActivePlayer]

	if player.isBot {
		state.botMove()
	} else {
		state.forceHumanMove(player)
	}
}

// Bot names carry a leading digit for ordering that players should not see
func (state *GameState) setTurnPrompt() {
	nameIndex := 0
	if state.Players[state.ActivePlayer].isBot {
		nameIndex = 1
	}
	state.Prompt = state.Players[state.ActivePlayer].Name[nameIndex:] + "'s turn"
}

// Begin the active player's turn with a fresh pool of six dice
func (state *GameState) startTurn() {
	state.TurnScore = 0
	state.KeptDice = ""
	state.fujirkled = false

	// KeepRoll deliberately survives the turn change. Banking commits the pick and
	// starts the next turn in one call, so clearing it here destroyed the mask
	// before any client could poll for it. commitSelection overwrites it on the
	// next pick.

	state.setTurnPrompt()

	state.rollPool(NUM_DICE)
}

// Set by tests to make rolls deterministic; nil in normal operation
var forceRoll func(count int) []int

// Roll count fresh dice into the pool and detect a fujirkle
func (state *GameState) rollPool(count int) {
	dice := make([]int, count)
	if forceRoll != nil {
		dice = forceRoll(count)
	} else {
		for i := 0; i < count; i++ {
			dice[i] = rand.Intn(6) + 1
		}
	}
	state.Dice = diceToString(dice)

	if !rollHasScore(dice) {
		state.fujirkled = true
		state.TurnScore = 0
		state.Prompt = PROMPT_FUJIRKLE
		state.moveExpires = time.Now().Add(FUJIRKLE_SHOW_TIME)
		return
	}

	state.resetPlayerTimer()
}

// Commit the masked dice to the turn score. Returns false if the selection is
// not a legal set-aside, in which case the state is left untouched.
func (state *GameState) commitSelection(mask string) bool {
	if state.fujirkled || len(state.Dice) == 0 {
		return false
	}

	dice := parseDice(state.Dice)
	kept, pool, ok := applyKeepMask(dice, mask)
	if !ok {
		return false
	}

	score, ok := scoreSelection(kept)
	if !ok {
		return false
	}

	state.TurnScore += score
	state.KeptDice += diceToString(kept)
	state.KeepRoll = mask
	state.Dice = diceToString(pool)

	return true
}

// Set aside the masked dice and roll whatever is left. Emptying the pool earns
// hot dice - a fresh set of six.
func (state *GameState) rollAgain(mask string) bool {
	if !state.commitSelection(mask) {
		return false
	}

	count := len(state.Dice)
	if count == 0 {
		count = NUM_DICE
		state.KeptDice = ""
		state.Prompt = PROMPT_HOT_DICE
	} else {
		// The banner belongs to the roll that earned it, and no later one
		state.setTurnPrompt()
	}

	state.rollPool(count)
	return true
}

// Set aside the masked dice and bank the turn score
func (state *GameState) bankScore(mask string) bool {
	if !state.commitSelection(mask) {
		return false
	}

	player := &state.Players[state.ActivePlayer]
	player.Score += state.TurnScore

	if player.Score >= TARGET_SCORE && !state.finalRound {
		state.finalRound = true
		state.finalRoundTrigger = player.id
	}

	state.nextValidPlayer()
	return true
}

func (state *GameState) forceHumanMove(player *Player) {
	// Out of time - keep the best dice available and bank rather than risk it
	dice := parseDice(state.Dice)
	mask, score := bestSelection(dice)

	if score == 0 {
		state.nextValidPlayer()
		return
	}

	player.isPenalized = true
	state.bankScore(maskToString(mask))
}

func (state *GameState) botMove() {
	dice := parseDice(state.Dice)
	mask, score := bestSelection(dice)

	if score == 0 {
		state.nextValidPlayer()
		return
	}

	used := 0
	for _, k := range mask {
		if k {
			used++
		}
	}
	remaining := len(dice) - used

	player := &state.Players[state.ActivePlayer]
	projected := state.TurnScore + score
	maskStr := maskToString(mask)

	// Winning right now always beats pushing further
	if player.Score+projected >= TARGET_SCORE {
		state.bankScore(maskStr)
		return
	}

	// Someone already crossed the line - this is the last turn, so chase the
	// leader rather than bank a losing score
	if state.finalRound {
		if player.Score+projected > state.leadingScore(player.id) {
			state.bankScore(maskStr)
		} else {
			state.rollAgain(maskStr)
		}
		return
	}

	// Hot dice are free - a fresh six can rarely fujirkle
	if remaining == 0 {
		state.rollAgain(maskStr)
		return
	}

	if projected < state.botBankThreshold(remaining, player.isSmart) {
		state.rollAgain(maskStr)
		return
	}

	state.bankScore(maskStr)
}

// Point at which a bot stops pushing, by dice left to roll. Roughly tracks the
// odds of fujirkling with that many dice.
func (state *GameState) botBankThreshold(remaining int, isSmart bool) int {
	if !isSmart {
		return 300
	}

	switch remaining {
	case 1:
		return 300
	case 2:
		return 450
	case 3:
		return 700
	case 4:
		return 1100
	case 5:
		return 2000
	}
	return 0
}

func (state *GameState) leadingScore(excludeID string) int {
	best := 0
	for _, player := range state.Players {
		if !player.isViewing && !player.isLeaving && !strings.EqualFold(player.id, excludeID) && player.Score > best {
			best = player.Score
		}
	}
	return best
}

func maskToString(mask []bool) string {
	buf := make([]byte, len(mask))
	for i, k := range mask {
		if k {
			buf[i] = '1'
		} else {
			buf[i] = '0'
		}
	}
	return string(buf)
}

// Drop players that left or have not pinged within the expected timeout
func (state *GameState) dropInactivePlayers(dropForNewPlayer bool) {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	players := []Player{}

	currentActivePlayer := state.ActivePlayer

	currentPlayerID := ""
	if state.clientPlayer > -1 && state.clientPlayer < len(state.Players) {
		currentPlayerID = state.Players[state.clientPlayer].id
	}

	activePlayerID := ""
	if state.ActivePlayer > -1 && state.ActivePlayer < len(state.Players) {
		activePlayerID = state.Players[state.ActivePlayer].id
	}

	for _, player := range state.Players {
		if !player.isLeaving && (player.isBot || player.lastPing.Compare(cutoff) > 0) {
			players = append(players, player)
		}
	}

	playersWereDropped := len(state.Players) != len(players)

	if playersWereDropped {
		state.Players = players
		state.refreshBots()
	}

	if dropForNewPlayer {
		return
	}

	if len(players) > 0 {
		state.clientPlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.id, currentPlayerID) })
		state.ActivePlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.id, activePlayerID) })

		if !state.gameOver && state.Round > ROUND_LOBBY && state.ActivePlayer < 0 {
			state.ActivePlayer = currentActivePlayer - 1
			state.nextValidPlayer()
		}
	}

	if len(state.Players) < 2 && state.Round < ROUND_GAMEOVER {
		state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	}

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

	if playersLeft < 2 || humanPlayersLeft == 0 {
		state.endGame(true)
	}
	state.dropInactivePlayers(false)
}

func (state *GameState) playerPing() {
	if state.clientPlayer >= 0 && state.clientPlayer < len(state.Players) {
		state.Players[state.clientPlayer].lastPing = time.Now()
		state.Players[state.clientPlayer].isPenalized = false
	}
}

// Returns true if enough players (including bots) readied to start, followed by # of players ready, not ready, bots
func (state *GameState) getPlayerCounts() (bool, int, int, int) {
	canStart := false
	totalHumansReady := 0
	totalHumansNotReady := 0
	totalBots := 0

	for _, player := range state.Players {
		if !player.isBot && !player.isLeaving {
			if player.Ready == READY_YES {
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

func (state *GameState) toggleReady() {
	if state.Round == ROUND_LOBBY && len(state.Players) > 1 {
		_, totalHumansReady, _, _ := state.getPlayerCounts()

		if state.Players[state.clientPlayer].Ready == READY_YES {
			state.Players[state.clientPlayer].Ready = READY_UNSET
		} else if totalHumansReady < MAX_PLAYERS {
			state.Players[state.clientPlayer].Ready = READY_YES
		}
	}
}

func (state *GameState) resetPlayerTimer() {
	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isPenalized {
		timeLimit = PLAYER_PENALIZED_TIME_LIMIT
	}

	if state.Players[state.ActivePlayer].isBot {
		timeLimit = BOT_TIME_LIMIT
	} else {
		_, humanCount := state.getHumanPlayerCountInfo()
		if humanCount == 1 {
			timeLimit = PLAYER_TIME_LIMIT_SINGLE_PLAYER
		} else if state.ActivePlayer == 0 {
			timeLimit = timeLimit + NEW_ROUND_TIME_EXTRA
		}
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	// The player who crossed the target gets the turn back only to end the game
	if state.finalRound && state.ActivePlayer >= 0 {
		next := state.peekNextPlayer()
		if next >= 0 && strings.EqualFold(state.Players[next].id, state.finalRoundTrigger) {
			state.endGame(false)
			return
		}
	}

	state.ActivePlayer++

	for state.ActivePlayer < len(state.Players) && state.Players[state.ActivePlayer].isViewing {
		state.ActivePlayer++
	}

	if state.ActivePlayer >= len(state.Players) {
		state.ActivePlayer = 0
		state.newRound()
		if state.gameOver || state.Round == ROUND_LOBBY {
			return
		}
		return
	}

	state.startTurn()
}

// Index of the player who would act next, skipping spectators
func (state *GameState) peekNextPlayer() int {
	index := state.ActivePlayer + 1

	for {
		if index >= len(state.Players) {
			index = 0
		}
		if index == state.ActivePlayer {
			return -1
		}
		if !state.Players[index].isViewing {
			return index
		}
		index++
	}
}

func (state *GameState) selectableDice() string {
	return selectableMask(parseDice(state.Dice))
}

func (state *GameState) createClientState(pov string) *GameState {
	stateCopy := *state

	stateCopy.Name = state.serverName

	statePlayers := stateCopy.Players
	stateCopy.Players = []Player{}

	start := slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.id, pov) })

	if start == -1 {
		start = state.clientPlayer
	}

	if state.clientPlayer < 0 || state.Players[state.clientPlayer].isViewing {
		start = 0
		stateCopy.Viewing = 1
	} else {
		stateCopy.Viewing = 0
	}

	currentActivePlayerID := ""
	if stateCopy.ActivePlayer > -1 {
		currentActivePlayerID = statePlayers[stateCopy.ActivePlayer].id
	}

	for isViewing := 0; isViewing < 2; isViewing++ {
		for i := start; i < start+len(statePlayers); i++ {
			playerIndex := i % len(statePlayers)

			if isViewing == 0 && !statePlayers[playerIndex].isViewing {
				stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
			}

			if isViewing == 1 && statePlayers[playerIndex].isViewing {
				stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
			}
		}
	}

	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	stateCopy.Status = 0
	if state.fujirkled {
		stateCopy.Status |= STATUS_FUJIRKLE
	}

	stateCopy.ValidMoves = 0

	if state.ActivePlayer > -1 {
		stateCopy.ActivePlayer = slices.IndexFunc(stateCopy.Players, func(p Player) bool { return strings.EqualFold(p.id, currentActivePlayerID) })

		if stateCopy.Viewing == 0 && state.ActivePlayer == state.clientPlayer && !state.fujirkled {
			stateCopy.ValidMoves = MOVE_ROLL | MOVE_BANK

			if len(pov) == 0 {
				stateCopy.Prompt = PROMPT_YOUR_TURN
			}
		}

		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	stateCopy.KeptDice = state.KeptDice
	stateCopy.Selectable = state.selectableDice()

	if stateCopy.MoveTime < 0 {
		stateCopy.MoveTime = 0
	}

	stateCopy.hash = "0"
	hash, _ := hashstructure.Hash(stateCopy, hashstructure.FormatV2, nil)
	stateCopy.hash = fmt.Sprintf("%d", hash)

	return &stateCopy
}

func (state *GameState) updateLobby() {
	state.updateLobbyWithGameResult(nil)
}

func (state *GameState) updateLobbyWithGameResult(gameResult *GameResult) {
	if !state.registerLobby {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.table, gameResult)
}

func (state *GameState) getHumanPlayerCountInfo() (int, int) {
	humanAvailSlots := MAX_PLAYERS
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if !player.isBot && !player.isLeaving && player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}

	if state.Round > ROUND_LOBBY && state.Round < ROUND_GAMEOVER {
		humanAvailSlots = humanPlayerCount
	}

	return humanAvailSlots, humanPlayerCount
}
