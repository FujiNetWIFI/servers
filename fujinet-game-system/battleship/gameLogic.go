package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

// These can be set to 0 for testing scenarios, so are outside of const
var BOT_TIME_LIMIT = time.Second * 3
var START_WAIT_TIME = time.Second * 31
var START_WAIT_TIME_ALL_READY = time.Second * 6
var START_WAIT_TIME_ONE_PLAYER = time.Second * 3
var ENDGAME_TIME_LIMIT = time.Second * 10
var PLAYER_TIME_LIMIT = time.Second * 45
var PLAYER_PENALIZED_TIME_LIMIT = time.Second * 15
var NEW_STATUS_TIME_EXTRA = time.Second * 5
var PLAYER_TIME_LIMIT_SINGLE_PLAYER = time.Second * 250 // don't go over 255 as 8 bit clients expect to store in single byte


const (
	MAX_PLAYERS = 4
	MOVE_TIME_GRACE_SECONDS = 4

	// 10x10 grid. in future, may support different gamefield sizes
	FIELD_WIDTH			 = 10

	// Drop players who do not make a move in 5 minutes
	PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

	// Prompts
	PROMPT_WAITING_FOR_MORE_PLAYERS = "Waiting for players"
	PROMPT_WAITING_ON_READY         = "Waiting for everyone to ready up"
	PROMPT_STARTING_IN              = "Starting in "
	PROMPT_PLACE_SHIPS		        = "Place your five ships"
	PROMPT_WAITING_PLACEMENT        = "Waiting on others to place"
	PROMPT_GAME_ABORTED             = "The game was aborted early"

	// GameState Status
	STATUS_LOBBY          = 0
	STATUS_PLACE_SHIPS 	  = 1
	STATUS_GAMESTART      = 10
	STATUS_MISS           = 11
	STATUS_HIT            = 12
	STATUS_SUNK           = 13
	STATUS_GAMEOVER       = 99

	// Player status
	PLAYER_STATUS_PLAYING = 0
	PLAYER_STATUS_DEFEATED = 1
	PLAYER_STATUS_VIEWING = 2
	PLAYER_STATUS_READY = 3
	PLAYER_STATUS_PLACE_SHIPS = 10

	// Field values
	FIELD_HIT = 1
	FIELD_MISS = 2
	FIELD_SUNK = -99
	FIELD_SIZE			 = FIELD_WIDTH * FIELD_WIDTH
	
	// Ship placement direction
	DIR_RIGHT = 0
	DIR_DOWN = 1
)

var SHIP_SIZES = []int{5, 4, 3, 3, 2} 

var botNames = []string{"Clyd", "Meg", "Kirk"}

var wordChanges = [][]string{
	[]string{" is "," are "},
	[]string{" has "," have "},
	[]string{" reigns "," reign "},
	[]string{" takes "," take "},
}

var GAME_WON_MESSAGES = []string{
//   12345678901234567890123
	"is the ultimate victor!", 
	"is the naval champion!", 
	"has won the battle!", 
	"ruled the seas today!", 
	"put everyone to shame!",
	"is the battle master!",
	"is the best commander!",
	"battled to victory!",
	"reigns supreme!",
	"won a sweet victory!",
	"kicked serious butt!",
	"conquered all foes!",
	"is the sea legend!",
	"is the best admiral!",
	"is the battle hero!",
	"has triumphed over all!",
	"is the last standing!",
	"dominated the seas!",
	"outplayed everyone!",
	"is the top gun!",
	"is the fleet ace!",
	"is the sea wolf!",
	"is the ocean king!",
	"outsmarted everyone!",
	"absolutely annihilated!",
	"won all the glory!",
	"takes the crown!",
	"crushed everyone!",
}


// Used to send a list of available tables
type GameTable struct {
	Table      string
	Name       string
	CurPlayers int
	MaxPlayers int
}

func resetTestMode() {
	// Set certain timeouts to 0 to facilitate running tests quickly
	BOT_TIME_LIMIT = 0
	START_WAIT_TIME = 0
	START_WAIT_TIME_ALL_READY = 0
	START_WAIT_TIME_ONE_PLAYER = 0
	ENDGAME_TIME_LIMIT = 0
}

func createGameState(playerCount int) *GameState {

	state := GameState{}

	// Pre-populate player pool with bots
	for i := 0; i < playerCount; i++ {
		state.addPlayer(botNames[i] + " Bot", true)
	}

	// Initialize game in Lobby state
	state.resetGame()

	return &state
}

func (state *GameState) startGame() {

	// If there aren't enough players to play, abort the game
	if len(state.Players) < 2 {
		if state.Status > STATUS_LOBBY {
			state.endGame(true)
		}
		return
	}

	// If brand new game, clear the ready flags (first index of scores) and set all scores to -1 (unset)
	// Also set any players that are not ready to spectators/viewing
	if state.Status == STATUS_LOBBY {
		state.gameOver = false
		state.Status = STATUS_PLACE_SHIPS
		state.Prompt = PROMPT_WAITING_PLACEMENT
		players := []Player{}

		clientPlayerID := state.Players[state.clientPlayer].id

		totalPlaying := 0

		// Initialize players, adding the playing players to the front of the list
		for i := 0; i < len(state.Players); i++ {
			player := &state.Players[i]

			// This player is playing - initialize their gamefield
			if totalPlaying < MAX_PLAYERS && (player.status == PLAYER_STATUS_READY || player.isBot) {
				totalPlaying++
				player.status = PLAYER_STATUS_PLACE_SHIPS

				// Place random ships for bot
				if player.isBot {
					for {
						randomPlacement := []int{0,0,0,0,0}
						for j := 0; j < len(SHIP_SIZES); j++ {
							randomPlacement[j] = rand.Intn(FIELD_SIZE*2)
						}
						if state.placeShipsFor(randomPlacement, player) {
							break
						}
					}
				}

				players = append(players, *player)
			} else {
				// Set player to viewing
				player.status = PLAYER_STATUS_VIEWING
			}
		}

		// Now loop through and add the spectating players at the end of the list
		for _, player := range state.Players {
			if player.status == PLAYER_STATUS_VIEWING {
				players = append(players, player)
			}
		}

		// Update the players array in the state with the newly sorted list
		state.Players = players

		// Update the total number of players in this game
		state.playerCount = totalPlaying

		// As the client player may have shifted positions, re-set their ID
		state.setClientPlayerByID(clientPlayerID)
	}
}

func (state *GameState) addPlayer(playerID string, isBot bool) {
	isViewing := false
	newPlayer := Player{
		Name:        playerID,
		id:          playerID,
		isBot:       isBot,
		isPenalized: false,
	}

	if !isBot {
		// Determine if the player is viewing, or if a bot should drop when they join
		if state.Status != STATUS_LOBBY {
			// Game started - player is viewing
			newPlayer.status = PLAYER_STATUS_VIEWING
			isViewing = true
		}
	}

	// Find the index of the first bot player in the player list
	insertIndex := slices.IndexFunc(state.Players, func(p Player) bool { return p.isBot })

	// If a bot or viewer, add to end
	if isBot || isViewing || insertIndex < 0 {
		insertIndex = len(state.Players)
	}

	// If a bot, set to ready
	if isBot {
		newPlayer.status = PLAYER_STATUS_READY
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
		state.dropInactivePlayers(true)
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
		if state.Players[state.clientPlayer].status == PLAYER_STATUS_VIEWING {
			return true
		}
	} else {
		// If a new game and spots available, set this player as no longer viewing
		if state.Status == STATUS_LOBBY && state.Players[state.clientPlayer].status == PLAYER_STATUS_VIEWING && len(state.Players) < MAX_PLAYERS {
			state.Players[state.clientPlayer].status = PLAYER_STATUS_PLAYING
		}
	}
	return false
}

func (state *GameState) debugForceEnd(winner int) {
	for i, _ := range state.Players {
		if state.Players[i].status != PLAYER_STATUS_VIEWING {
			if i==winner {
				state.Players[i].ShipsLeft = []int{1,1,1,1,1}
			}else {
				state.Players[i].ShipsLeft = []int{0,0,0,0,0}
			}
		}
	}
	state.endGame(false)
}

func (state *GameState) endGame(abortGame bool) {
	// The next request for /state will start a new game once the timer has counted down

	// If the game hasn't started, no need to do anything.
	if state.Status == STATUS_LOBBY {
		return
	}

	state.gameOver = true
	state.ActivePlayer = -1
	state.Status = STATUS_GAMEOVER
	state.Prompt = PROMPT_GAME_ABORTED

	if !abortGame {

		// Build game result to send to lobby
		gameResult := GameResult{}
		gameResult.Players = []GamePlayer{}
		for _, player := range state.Players {
			if player.status != PLAYER_STATUS_VIEWING {

				gamePlayer := GamePlayer{}
				gamePlayer.Winner = slices.Contains(player.ShipsLeft,1)
				if (gamePlayer.Winner) {
					state.Prompt = fmt.Sprintf("%s " + GAME_WON_MESSAGES[rand.Intn(len(GAME_WON_MESSAGES))], player.Name)
				}
				gamePlayer.Name = player.Name
				gamePlayer.Type = PLAYER_TYPE_HUMAN
				if player.isBot {
					gamePlayer.Type = PLAYER_TYPE_BOT
				}
				gameResult.Players = append(gameResult.Players, gamePlayer)
			}
		}
		
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		state.updateLobbyWithGameResult(&gameResult)
		
		
	} else {

		// If there are human players left, wait a bit before starting the next game
		if slices.ContainsFunc(state.Players, func(p Player) bool { return p.status != PLAYER_STATUS_VIEWING && !p.isBot }) {
			state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		} else {
			// Otherwise, all the human players left, so reset the game right away
			state.resetGame()
		}
	}

	
}

// Adds/removes bots as space allows, up to the number of bots the server started with
func (state *GameState) refreshBots() {
	if state.Status != STATUS_LOBBY {
		return
	}
	botDropped := false

	// Store current client player ID, in case we need it to reset the id after bots drop
	clientPlayerID := ""
	if state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		clientPlayerID = state.Players[state.clientPlayer].id
	}

	// Remove bots if overcrowded
	for len(state.Players) > MAX_PLAYERS && slices.ContainsFunc(state.Players, func(p Player) bool { return p.isBot }) {
		// If the table is full, drop a bot when this player joins
		_, _, _, botCount := state.getPlayerCounts()

		if botCount > 0 {
			for i := len(state.Players) - 1; i >= 0; i-- {
				if state.Players[i].isBot {
					state.Players[i].status = PLAYER_STATUS_READY
					state.botBox = slices.Insert(state.botBox, len(state.botBox), state.Players[i])
					state.Players = append(state.Players[:i], state.Players[i+1:]...)
					botDropped = true
					break
				}
			}
		}
	}

	// Or if the table is not full, fill it back in with bots
	for len(state.Players) < MAX_PLAYERS && len(state.botBox) > 0 {
		state.Players = slices.Insert(state.Players, len(state.Players), state.botBox[len(state.botBox)-1])
		state.botBox = state.botBox[:len(state.botBox)-1]
	}

	// Update client player in case a bot drop affected their index
	if botDropped && state.clientPlayer > 0 && state.clientPlayer < len(state.Players) {
		state.setClientPlayerByID(clientPlayerID)
	}
}

func (state *GameState) resetGame() {
	state.Status = STATUS_LOBBY
	state.ActivePlayer = -1
	state.LastAttackPos = 0
	state.aiCheckboardBit = rand.Intn(2)
	state.refreshBots()

	// Set player to unready if human, ready if bot (for future)
	for i := 0; i < len(state.Players); i++ {
		state.Players[i].status = ifElse(state.Players[i].isBot, PLAYER_STATUS_READY, PLAYER_STATUS_PLAYING)
		state.Players[i].ShipsLeft = nil
		state.Players[i].Gamefield = nil
		state.Players[i].ships = nil
	    state.Players[i].knownSunkShipsField = nil
	}

		if len(state.Players) < 2 {
		state.Prompt = PROMPT_WAITING_FOR_MORE_PLAYERS
	} else {
		state.Prompt = PROMPT_WAITING_ON_READY
	}
}

// The heart of the game. Runs a single cycle of game logic
func (state *GameState) runGameLogic() {

	// Let the game know this player is active
	state.playerPing()

	// If still waiting to start), check if the game can start
	if state.Status == STATUS_LOBBY {

		// Check if ready wait time has expired and at least one non bot player exists and all players are ready
		canStartNow, totalHumansReady, totalHumansNotReady, _ := state.getPlayerCounts()

		if canStartNow {

			// Start timer if not already started
			// Reset the timer if spots are left and someone just joins or unreadies
			if !state.startedStartCountdown || (totalHumansReady < 6 && totalHumansNotReady > state.prevTotalHumansNotReady) {
				state.startedStartCountdown = true

				// If just a single player is starting the game, start even quicker
				if totalHumansReady == 1 && totalHumansNotReady == 0 {
					state.moveExpires = time.Now().Add(START_WAIT_TIME_ONE_PLAYER)
				} else {
					state.moveExpires = time.Now().Add(START_WAIT_TIME)
				}
			}

			state.prevTotalHumansNotReady = totalHumansNotReady
			waitTime := int(time.Until(state.moveExpires).Seconds())

			// If everyone has readied up, shorten a long wait time
			if waitTime > 6 && totalHumansNotReady == 0 {
				state.moveExpires = time.Now().Add(START_WAIT_TIME_ALL_READY)
				waitTime = int(time.Until(state.moveExpires).Seconds())
			}

			if waitTime < 1 {
				state.startGame()
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

	if (state.Status == STATUS_PLACE_SHIPS) {
		// Check if any players still need to place ships
		if slices.ContainsFunc(state.Players, func(p Player) bool { return p.status != PLAYER_STATUS_VIEWING && p.ships == nil}) { 
			return
		}
		
		// Finally, initialize the empty gamefields
		for i := 0; i < len(state.Players); i++ {
			player := &state.Players[i]
			if player.status == PLAYER_STATUS_PLAYING {
				player.Gamefield = make([]int, FIELD_SIZE)
				player.ShipsLeft = []int{1,1,1,1,1}

				player.knownSunkShipsField = make([]int, FIELD_SIZE)

			}
		}

		// All players have placed ships, start the game!
		state.Status = STATUS_GAMESTART
		state.ActivePlayer = -1
		state.Prompt = ""
		state.nextValidPlayer()
		return
	}



	// If the game is currently over and the end game delay is past, reset the game
	if state.gameOver {
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false)
			state.resetGame()
		}
		return
	}

	// If there is no active player, or there is still time for the active player to make a move, we are done
	if state.ActivePlayer < 0 || int(time.Until(state.moveExpires).Seconds()) > 0 {
		return
	}

	// Force an action for the active player or BOT
	player := &state.Players[state.ActivePlayer]

	if player.isBot {
		state.botMove()
	} else {
		state.forceHumanMove(player)
	}
}

func (state *GameState) forceHumanMove(player *Player) {
	// Human player did not attack in time, so penalize them so their next wait time is shorter to avoid stalling the game
	player.isPenalized = true

	// Instead of picking an attack, skip this player's turn and move to the next player.
	state.nextValidPlayer()
}

func setRank(field *[]int, pos int, rank int) {
	if (*field)[pos]>=0  && (*field)[pos] < rank {
			(*field)[pos] = rank
	}
}
func (state *GameState) botMove() {

	// The AI scans the fields of all enemy players, ranking the spots most likely to have a ship
	// It will prefer to attack a spot that is more likely to hit multiple player's ships
	// It does not target human players or know the actual positions of any ships, sunken or otherwise.
	// In other words, it does not cheat - it just plays better than random.

	// The main ranked field tracking positions to attack.
	attackFieldRank := make([]int, FIELD_SIZE)

	// Highest rank reached. Only attacks of this rank will be considered
	highestRank := 0

	// Logic: Scan each player's field:
	// 	1. Remove hits that represent possible sunken ships
	//  2. Find attack locations adjacent to (or in between) hits

	for id, player := range state.Players {
		// Skip current player or non playing players
		if id == state.ActivePlayer || player.status != PLAYER_STATUS_PLAYING {
			continue
		}
		
		// Copy the player's current field, so it can be modified to indicate potential attacks
		attackField := make([]int, FIELD_SIZE)

		// Since this will use numbers over 0 to rank attacks, inverse the current values.
		// Also, copy any known sunk positions
		for pos := 0; pos < FIELD_SIZE; pos++ {
			attackField[pos] = -player.Gamefield[pos]
			
			if player.knownSunkShipsField[pos] == FIELD_SUNK {
				attackField[pos] = FIELD_SUNK
			}
		}

		maxShipSizeLeft := -1


		// Now, for each hit position remaining, figure out most likely attack positions
		
		// The following rank values are set:
		// 1 - Single hit found with no adjacent hits. Unsure of direction
		// 2 - Empty space between two hits that could be a ship
		// 3 - Multiple hits in a row found, likely direction of ship

		// Search for horizontal opportunities
		foundSize := 0
		for y:=0; y<FIELD_WIDTH; y++ {
			for x:=0; x<FIELD_WIDTH; x++ {
				if attackField[y*FIELD_WIDTH+x] == -FIELD_HIT {
					foundSize++
					
					// Look ahead to determine if end of ship
					if x==FIELD_WIDTH-1 || attackField[y*FIELD_WIDTH+x+1]!= -FIELD_HIT {
						rank := 1

						// If length>1, boost the attack rank as it is more likely there is a ship in this direction
						if foundSize > 1 {
							rank = 3
						} 

						// Test left
						testX := x-foundSize
						if testX >=0 && attackField[y*FIELD_WIDTH+testX]>=0  {
							setRank(&attackField, y*FIELD_WIDTH+testX, rank)
							leftToCheck := maxShipSizeLeft-foundSize
							
							// If a single width, check if there is empty space between this 
							// up to the largest ship spaces away
							if leftToCheck>0 {
								for testI:=0; testI<leftToCheck; testI++ {

									// Stop at edge
									if testX<0 {
										break
									}

									if attackField[y*FIELD_WIDTH+testX] == -FIELD_HIT {
										// Found another hit! Set the in-between spaces to rank 2
										for x2:=testX+1; x2<=x-foundSize; x2++ {
											setRank(&attackField, y*FIELD_WIDTH+x2, 2)
										}
									}

									// Stop if hit any non empty space
									if attackField[y*FIELD_WIDTH+testX] <0 {
										break
									}

									testX--
								}
							}

						}

						// Test right
						testX = x+1
						if testX < FIELD_WIDTH && attackField[y*FIELD_WIDTH+testX] >= 0 {
							setRank(&attackField, y*FIELD_WIDTH+testX, rank)

							// If a single width, check if there is empty space between this 
							// up to the largest ship spaces away
							leftToCheck := maxShipSizeLeft-foundSize
							if leftToCheck>0 {
								for testI:=0; testI<leftToCheck; testI++ {

									// Stop at edge
									if testX>=FIELD_WIDTH {
										break
									}

									if attackField[y*FIELD_WIDTH+testX] == -FIELD_HIT {
										// Found another hit! Set the in-between spaces to rank 2
										for x2:=x+1; x2<testX; x2++ {
											setRank(&attackField, y*FIELD_WIDTH+x2, 2)
										}
									}

									// Stop if hit any non empty space
									if attackField[y*FIELD_WIDTH+testX] <0 {
										break
									}

									testX++
								}
							}
						}
					}
				} else {
					foundSize=0
				}
			}
		}
		
		// Search for vertical opportunities
		foundSize = 0
		for x:=0; x<FIELD_WIDTH; x++ {
			for y:=0; y<FIELD_WIDTH; y++ {
				if attackField[y*FIELD_WIDTH+x] == -FIELD_HIT {
					foundSize++
					
					// Look ahead to determine if end of ship
					if y==FIELD_WIDTH-1 || attackField[(y+1)*FIELD_WIDTH+x]!= -FIELD_HIT {
						rank := 1

						// If length>1, boost the attack rank as it is more likely there is a ship in this direction
						if foundSize > 1 {
							rank = 3
						} 

						// Test up
						testY := y-foundSize
						if testY >=0 && attackField[testY*FIELD_WIDTH+x]==0 {
							setRank(&attackField, testY*FIELD_WIDTH+x, rank)

							leftToCheck := maxShipSizeLeft-foundSize
							
							// If a single width, check if there is empty space between this 
							// up to the largest ship spaces away
							if leftToCheck>0 {
								for testI:=0; testI<leftToCheck; testI++ {

									// Stop at edge
									if testY<0 {
										break
									}

									if attackField[testY*FIELD_WIDTH+x] == -FIELD_HIT {
										// Found another hit! Set the in-between spaces to rank 2
										for y2:=testY+1; y2<=y-foundSize; y2++ {
											setRank(&attackField, y2*FIELD_WIDTH+x, 2)
										}
									}

									// Stop if hit any non empty space
									if attackField[testY*FIELD_WIDTH+x] <0 {
										break
									}

									testY--
								}
							}
						}

						// Test down
						testY = y+1	
						if testY < FIELD_WIDTH && attackField[testY*FIELD_WIDTH+x] == 0 {
							setRank(&attackField, testY*FIELD_WIDTH+x, rank)

							// If a single width, check if there is empty space between this 
							// up to the largest ship spaces away
							leftToCheck := maxShipSizeLeft-foundSize
							if leftToCheck>0 {
								for testI:=0; testI<leftToCheck; testI++ {

									// Stop at edge
									if testY>=FIELD_WIDTH {
										break
									}

									if attackField[testY*FIELD_WIDTH+x] == -FIELD_HIT {
										// Found another hit! Set the in-between spaces to rank 2
										for y2:=y+1; y2<testY; y2++ {
											setRank(&attackField, y2*FIELD_WIDTH+x, 2)
										}
									}

									// Stop if hit any non empty space
									if attackField[testY*FIELD_WIDTH+x] <0 {
										break
									}

									testY++
								}
							}
						}
					}
				} else {
					foundSize=0
				}
			}
		}

		// Now add to the overall attack field rank 
		for pos:=0; pos<FIELD_SIZE; pos++ {
			if attackField[pos] > 0 {
				attackFieldRank[pos] += attackField[pos]
				if attackFieldRank[pos] > highestRank {
					highestRank = attackFieldRank[pos]
				}
			}
		}
	}

	// Generate a rank of positions and sort by attack rank
	validPositions := []int{}
	if highestRank > 0 {
		for pos:=0; pos<FIELD_SIZE; pos++ {
			if attackFieldRank[pos] == highestRank {
				validPositions = append(validPositions, pos)
			}
		}
	}

	if len(validPositions) == 0 {

		// If no attack positions were found, resort to search logic
		for attempt :=0; attempt<2; attempt++ {

			// Two attempt:
			// attempt 0 - random checkerboard pattern
			// attempt 1 - any open spot

			validPositions = []int{}
			
			for pos := 0; pos < FIELD_SIZE; pos++ {
				for id, player := range state.Players {
					// If found an open spot for another player, add it
					if id != state.ActivePlayer && player.status == PLAYER_STATUS_PLAYING && player.Gamefield[pos] == 0 && 
					(attempt==1 || ( (pos+(pos/FIELD_WIDTH)%2)%2==state.aiCheckboardBit)) {
						validPositions = append(validPositions, pos)
						break
					}
				}
			}

			if len(validPositions) > 0 {
				break
			}
		}
	}

	if len(validPositions) == 0 {
		log.Println("Bot found no valid positions to attack!!!")
	}
	
	pos := validPositions[rand.Intn(len(validPositions))]
	state.attack(pos)
}

// Drop players that left or have not pinged within the expected timeout
func (state *GameState) dropInactivePlayers(dropForNewPlayer bool) {
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

	// Count active players
	for _, player := range state.Players {
		if player.isBot || player.lastPing.Compare(cutoff) > 0 {
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
		if !state.gameOver && state.Status >= STATUS_GAMESTART && state.ActivePlayer < 0 {
			// The player immediately after the leaving player now owns that index, so set activePlayer the the index before them
			// and call nextValidPlayer() to start their turn
			state.ActivePlayer = currentActivePlayer - 1
			state.nextValidPlayer()
		}
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
	player.lastPing = time.Now().Add(time.Minute * time.Duration(-10))

	// Check if no human players are playing. If so, end the game
	humanPlayersLeft := 0
	playersLeft := 0

	for index, player := range state.Players {
		if index != state.clientPlayer && player.status != PLAYER_STATUS_VIEWING {
			playersLeft++
			if !player.isBot {
				humanPlayersLeft++
			}
		}
	}

	state.dropInactivePlayers(false)
	
	// If there aren't enough players to play, abort the game
	if playersLeft < 2 || humanPlayersLeft == 0 {
		state.endGame(true)
	}
	
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
		if !player.isBot {
			if player.status == PLAYER_STATUS_READY {
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

	if state.Status == STATUS_LOBBY && len(state.Players) > 1 {

		_, totalHumansReady, _, _ := state.getPlayerCounts()

		// Toggle ready state for this player if there is space
		if state.Players[state.clientPlayer].status == PLAYER_STATUS_READY {
			state.Players[state.clientPlayer].status = PLAYER_STATUS_PLAYING
		} else if totalHumansReady < MAX_PLAYERS {
			state.Players[state.clientPlayer].status = PLAYER_STATUS_READY
		}
	}
}

// Place player's ships on the gamefield
func (state *GameState) placeShips(shipPositions []int) bool {
	player := &state.Players[state.clientPlayer]
	return state.placeShipsFor(shipPositions, player)
}

// Place player's ships on the gamefield
func (state *GameState) placeShipsFor(shipPositions []int, player *Player) bool {
	if state.Status != STATUS_PLACE_SHIPS ||  player.status != PLAYER_STATUS_PLACE_SHIPS {
		return false
	}
	
	// Reset ship details
	player.ships = []Ship{}
	
	// Place each ship
	for i, pos := range shipPositions {
		shipSize := SHIP_SIZES[i]
		dir := pos / (FIELD_SIZE)
		gridPos := pos % (FIELD_SIZE)

		// Ensure the ship is within bounds and doesn't overlap an existing ship
		x := gridPos % FIELD_WIDTH
		y := gridPos / FIELD_WIDTH

		// Abort if this ship overlaps with another ship
		for j := 0; j < shipSize; j++ {
			if slices.ContainsFunc(player.ships, func(s Ship) bool { return slices.Contains(s.GridPos, gridPos) }) {
				player.ships = nil
				return false
			}
			if dir == DIR_RIGHT {
				gridPos++
			} else {
				gridPos+= FIELD_WIDTH
			}
		}
		
		// Reset grid position
		gridPos = pos % (FIELD_SIZE)

		// If ship is placed outside of bounds, abort
		if dir==DIR_RIGHT && x+shipSize>FIELD_WIDTH || dir==DIR_DOWN && y+shipSize>FIELD_WIDTH {
			player.ships = nil
			return false
		}

		ship := Ship{
			GridPos: make([]int, shipSize),
			Pos:  gridPos,
			Dir:  dir,
		}

		// Mark ship positions on gamefield and ship detail grid
		for j := 0; j < shipSize; j++ {
			ship.GridPos[j] = gridPos

			if dir == DIR_RIGHT {
				gridPos++
			} else {
				gridPos+= FIELD_WIDTH
			}
		}

		// Track this ship
		player.ships = append(player.ships, ship)
	}
	player.status = PLAYER_STATUS_PLAYING
	return true
}

// Performs the requested score for the active player, and returns true if successful
func (state *GameState) attack(pos int) bool {
	
	// Check for valid position
	if (pos<0 || pos>=FIELD_SIZE) {
		return false
	}
	
	attackedPlayers := 0
	playersHit := 0

	// Default t miss. A hit will override, and a sunk will override a hit.
	status := STATUS_MISS

	// Count bots playing
	activeBots := 0

	for _, p := range state.Players {
		if p.isBot  && p.status == PLAYER_STATUS_PLAYING {
			activeBots++
		}
	}

	// Attack each player
	for index, _  := range state.Players {
		player := &state.Players[index]

		// Can't attack self, non playing players, or players that have already been attacked at that position
		if index == state.ActivePlayer ||  player.status != PLAYER_STATUS_PLAYING || player.Gamefield[pos] > 0 {
			continue
		}
		attackedPlayers++

		// Loop through the player's ship positions to see if a ship is hit
		hitShip := slices.IndexFunc(player.ships, func(s Ship) bool { return slices.Contains(s.GridPos, pos) })
		if hitShip >= 0 {
			// Hit!
			playersHit++
			player.Gamefield[pos] = FIELD_HIT
			if status != STATUS_SUNK {
				status = STATUS_HIT
			}

			// Check if ship is sunk (all positions of this ship are hit)
			if !slices.ContainsFunc(player.ships[hitShip].GridPos, func(p int) bool { return player.Gamefield[p] != FIELD_HIT }) {
				// Mark ship as sunk
				player.ShipsLeft[hitShip] = 0
				status = STATUS_SUNK

				// We know this position represents a sunk ship, so update known sunk ships field
				// We update the rest of the ship locations if it can be determined later
				player.knownSunkShipsField[pos] = FIELD_SUNK
				
				// Check if all ships are sunk - player defeated
				if !slices.Contains(player.ShipsLeft, 1) {
					player.status = PLAYER_STATUS_DEFEATED
					log.Println("Player defeated:", player.Name)
				} else {

					// Otherwise, check if total hits match the number required to sunk the current ships.
					// If so, this means every hit is accounted for and we KNOW (without cheating) all hits represent sunk ships.

					// Count total sizes of sunk ships
					totalSunkHits := 0
					for i, shipLeft := range player.ShipsLeft {
						if shipLeft == 0 {
							totalSunkHits += SHIP_SIZES[i]
						}
					}

					// Count total hits in this player's gamefield
					totalGamefieldHits := 0
					for pos:=0; pos<FIELD_SIZE; pos++ {
						if player.Gamefield[pos] == FIELD_HIT {
							totalGamefieldHits++
						}	
					}

					// If the number of hits match, update known sunk ships field
					if totalGamefieldHits == totalSunkHits {						
						log.Println("All hits marked as sunk for player:", player.Name)
						for pos:=0; pos<FIELD_SIZE; pos++ {
							if player.Gamefield[pos] == FIELD_HIT {
								player.knownSunkShipsField[pos] = FIELD_SUNK
							}
						}
					}
				}
			} 
		} else {
			// Miss
			player.Gamefield[pos] = FIELD_MISS
		}
	}

	// No players attacked? Let the player try a new spot
	if attackedPlayers == 0 {
		return false
	}


	state.Prompt = ""

	// Updating prompt based on attack result was getting too busy
	// Commenting out for now
	// switch (status) {
	// 	case STATUS_MISS:
	// 		state.Prompt = state.Players[state.ActivePlayer].Name + " missed. "
	// 	default:
	// 		if playersHit > 1 {
	// 			state.Prompt = fmt.Sprintf("%s hit %d! ", state.Players[state.ActivePlayer].Name, playersHit)
	// 		} else {
	// 			state.Prompt = fmt.Sprintf("%s hit! ", state.Players[state.ActivePlayer].Name)
	// 		}
			
	// }

	// Update state status
	state.Status = status
	state.LastAttackPos = pos

	// Move on to next player
	state.nextValidPlayer()
	
	return true
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
				timeLimit = timeLimit + NEW_STATUS_TIME_EXTRA
			}
		}
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	curActivePlayer := state.ActivePlayer

	// Move to next available player
	for {
		state.ActivePlayer = (state.ActivePlayer + 1 ) % len(state.Players)
		if state.ActivePlayer == curActivePlayer || state.Players[state.ActivePlayer].status == PLAYER_STATUS_PLAYING {
			break
		}
	}

	// Check if we looped back to the same player - meaning they are the only one left and WON!
	if state.ActivePlayer == curActivePlayer {
		state.endGame(false)
		return
	}

	state.resetPlayerTimer();
}

// Creates a copy of the state and modifies it to be from the
// perspective of this calling player.
func (state *GameState) createClientState() *GameState {
	stateCopy := *state

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []Player{}

	// start at the current player
	start := state.clientPlayer
	
	// When on observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set Viewing flag to let client know they are not actively part of the game
	if state.Players[state.clientPlayer].status == PLAYER_STATUS_VIEWING {
		start = 0
	} 

	// Set current player's status
	stateCopy.PlayerStatus = state.Players[state.clientPlayer].status

	currentActivePlayerID := ""
	if stateCopy.ActivePlayer > -1 {
		currentActivePlayerID = statePlayers[stateCopy.ActivePlayer].id
	}

	// Loop through each players to add relative to calling player
	for i := start; i < start+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Add this player to the copy of the state going out
		if statePlayers[playerIndex].status != PLAYER_STATUS_VIEWING {
			stateCopy.Players = append(stateCopy.Players, statePlayers[playerIndex])
		}
	}


	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	// If there is an active player
	if state.ActivePlayer > -1 {

		// Set the active player to the new index from the client centric players list
		stateCopy.ActivePlayer = slices.IndexFunc(stateCopy.Players, func(p Player) bool { return strings.EqualFold(p.id, currentActivePlayerID) })
		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	// Ensure move time is not negative
	if stateCopy.MoveTime < 0 {
		stateCopy.MoveTime = 0
	}

	// If this player has not yet placed their ships, update the prompt
	if stateCopy.clientPlayer == 0 && stateCopy.Status == STATUS_PLACE_SHIPS && stateCopy.PlayerStatus == PLAYER_STATUS_PLACE_SHIPS{
		stateCopy.Prompt = PROMPT_PLACE_SHIPS
	}

	// ** COMMENTED OUT FOR NOW - PROMPT REDUNDANT **
	// Add active player's turn to prompt.
	// if stateCopy.Status >= STATUS_GAMESTART && stateCopy.ActivePlayer >= 0 {
	// 	// if stateCopy.ActivePlayer == 0 {
	// 	// 	stateCopy.Prompt = stateCopy.Prompt + "Your turn"
	// 	// }
	// 	// } else {
	// 	// 	stateCopy.Prompt = stateCopy.Prompt + stateCopy.Players[stateCopy.ActivePlayer].Name + "'s turn"
	// 	// }
	// }

	// If sentence begins with current player's name, replace it with YOU and fix resulting grammar
	if strings.HasPrefix(stateCopy.Prompt, stateCopy.Players[0].Name) {
		stateCopy.Prompt = strings.Replace(stateCopy.Prompt, stateCopy.Players[0].Name+" ", "You ", 1)
		stateCopy.Prompt = makeSingular(stateCopy.Prompt)
	}

	if BOT_TIME_LIMIT == 0 {
		log.Println(stateCopy.Prompt)
	}

	return &stateCopy
}

// Loop through and make any word changes for grammar
func makeSingular(phrase string) string {
	for _, change := range wordChanges {
		if strings.Contains(phrase, change[0]) {
			phrase = strings.Replace(phrase, change[0], change[1], 1)
			break
		}
	}
	return phrase
}

func (state *GameState) updateLobby() {
	state.updateLobbyWithGameResult(nil)
}

func (state *GameState) updateLobbyWithGameResult(gameResult *GameResult) {
	if !state.registerLobby {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	// Send the total human slots / players to the Lobby
	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.table, gameResult)
}

// Return number of active human players in the table, for the lobby
func (state *GameState) getHumanPlayerCountInfo() (int, int) {

	// Since bots sub out for players, the available slots will always be
	// max players even if bots are present
	humanAvailSlots := MAX_PLAYERS
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if !player.isBot && player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}

	// If the game has started, there are no more human slots available
	if state.Status > STATUS_LOBBY && state.Status < STATUS_GAMEOVER {
		humanAvailSlots = humanPlayerCount
	}

	return humanAvailSlots, humanPlayerCount
}
