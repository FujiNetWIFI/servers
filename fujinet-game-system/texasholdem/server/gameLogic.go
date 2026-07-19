package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cardrank/cardrank"
	hashstructure "github.com/mitchellh/hashstructure/v2"
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

const (
	ANTE        = 1
	BRINGIN     = 2 // Not used in Texas Hold'em
	LOW         = 5 // Not used in Texas Hold'em
	HIGH        = 10 // Not used in Texas Hold'em
	SB          = 5 // Small Blind
	BB          = 10 // Big Blind
	STARTING_PURSE = 1000
)
const MOVE_TIME_GRACE_SECONDS = 4

// Timing limits are vars (not consts) so tests can zero them for fast, deterministic runs
var BOT_TIME_LIMIT = time.Second * time.Duration(3)
var PLAYER_TIME_LIMIT = time.Second * time.Duration(39)
var ENDGAME_TIME_LIMIT = time.Second * time.Duration(12)
var NEW_ROUND_FIRST_PLAYER_BUFFER = time.Second * time.Duration(1)

// Drop players who do not make a move in 5 minutes
const PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

const WAITING_MESSAGE = "Waiting for more players"

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
// moveLookup maps the 2-character move codes clients send to the friendly text
// shown as the player's last move
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BB": "POST",  // Blind posting (server-initiated)
	"BL": "BET",   // Bet low (1x big blind)
	"BH": "BET",   // Bet high (2x big blind)
	"CA": "CALL",
	"RL": "RAISE", // Minimum raise
	"RH": "RAISE", // Bigger raise
	"AI": "ALLIN",
}

var botNames = []string{"Clyd", "Jim", "Kirk", "Hulk", "Fry", "Meg", "Grif", "GPT"}

var botProfiles = map[string]BotProfile{
	"Clyd BOT":   {Name: "Tight-Aggressive", VPIP: 0.20, PFR: 0.80, BluffFrequency: 0.10},
	"Jim BOT":    {Name: "Loose-Passive", VPIP: 0.60, PFR: 0.10, BluffFrequency: 0.05},
	"Kirk BOT":   {Name: "The Rock", VPIP: 0.10, PFR: 0.50, BluffFrequency: 0.01},
	"Hulk BOT":   {Name: "The Maniac", VPIP: 0.80, PFR: 0.70, BluffFrequency: 0.40},
	"Fry BOT":    {Name: "Calling Station", VPIP: 0.70, PFR: 0.05, BluffFrequency: 0.02},
	"Meg BOT":    {Name: "Balanced", VPIP: 0.35, PFR: 0.60, BluffFrequency: 0.15},
	"Grif BOT":   {Name: "The Bluffer", VPIP: 0.40, PFR: 0.50, BluffFrequency: 0.50},
	"GPT BOT":    {Name: "GTO Pro", VPIP: 0.25, PFR: 0.75, BluffFrequency: 0.20},
}

// For simplicity on the 8bit side (using switch statement), using a single character for each key.
// DOUBLE CHECK that letter isn't already in use on the object!
// Double characters are used for the list objects (validMoves and players)

type validMove struct {
	Move string `json:"m"`
	Name string `json:"n"`
}

type card struct {
	Rank int `json:"Rank"`
	Suit int `json:"Suit"`
}

type Status int64

const (
	STATUS_WAITING Status = 0
	STATUS_PLAYING Status = 1
	STATUS_FOLDED  Status = 2
	STATUS_LEFT    Status = 3
	STATUS_ALL_IN  Status = 4
)

// BotProfile defines the playing style of a bot.
type BotProfile struct {
	Name           string
	VPIP           float64 // Voluntarily Puts In Pot: How often the bot chooses to play a hand (0-1).
	PFR            float64 // Pre-Flop Raise: How often the bot raises pre-flop when they do play (0-1).
	BluffFrequency float64 // How often the bot will try to bluff (0-1).
}

type Player struct {
	Name    string `json:"Name"`
	Status  Status `json:"Status"`
	Bet     int    `json:"Bet"`
	Move    string `json:"Move"`
	Purse   int    `json:"Purse"`
	Hand    []card `json:"Hand"`
	IsHuman bool   `json:"IsHuman"`

	// Internal
	isBot          bool
	lastPing       time.Time
	profile        BotProfile
	actedThisRound bool // Has this player voluntarily acted in the current betting round?
	totalBet       int  // Cumulative chips contributed this hand (for side pot calculation)
}

type GameState struct {
	GamesPlayed  int    `json:"gamesPlayed"`
	Hash         string `json:"z"` // state hash for client change detection via ?hash=
	TableId      string `json:"tableId"`
	// External (JSON)
	LastResult   string      `json:"LastResult"`
	Winner       string      `json:"Winner"`
	Round        int         `json:"Round"`
	RoundName    string      `json:"roundName"`
	Pot          int         `json:"Pot"`
	ActivePlayer int         `json:"ActivePlayer"`
	MoveTime     int         `json:"MoveTime"`
	Viewing      int         `json:"Viewing"`
	ValidMoves   []validMove `json:"ValidMoves"`
	Players      []Player    `json:"Players"`
	CommunityCards []card `json:"CommunityCards"`
	CurrentBet int `json:"currentBet"`

	// Internal
	// Deck must never be serialized to clients (it would leak upcoming cards),
	// and it is mutated under the table lock while serialization of client
	// copies happens outside it
	Deck          []card `json:"-" hash:"ignore"`
	deckIndex     int
	currentBet    int
	gameOver      bool
	clientPlayer  int
	wonByFolds    bool
	moveExpires   time.Time
	serverName    string
	raiseCount    int
	raiseAmount   int
	registerLobby bool
	allowBotGames bool // tests only: allow hands with zero human players

	buttonPos     int    // Seat index of the dealer button; rotates each hand
	lastRaiseSize int    // Size of the last bet/raise increment this round (min-raise tracking)
	rng           *rand.Rand
	testDeck      []card // When set (tests only), newRound uses this deck instead of shuffling
}

// Used to send a list of available tables
type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

var initServerOnce sync.Once

func initializeGameServer() {
	// Guarded by sync.Once: appending " BOT" twice would corrupt bot names
	initServerOnce.Do(func() {
		for i := 0; i < len(botNames); i++ {
			botNames[i] = botNames[i] + " BOT"
		}
	})
}

func (state *GameState) initializeDeck() {
	state.Deck = []card{}
	// Create deck of 52 cards
	for suit := 0; suit < 4; suit++ {
		for value := 2; value < 15; value++ {
			card := card{Rank: value, Suit: suit}
			state.Deck = append(state.Deck, card)
		}
	}
}

// random returns the state's RNG, lazily initializing it so states constructed
// without createGameState (e.g. in tests) still work
func (state *GameState) random() *rand.Rand {
	if state.rng == nil {
		state.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return state.rng
}

func (state *GameState) shuffleDeck() {
	// Tests may rig the deck for deterministic showdowns
	if state.testDeck != nil {
		state.Deck = make([]card, len(state.testDeck))
		copy(state.Deck, state.testDeck)
		state.deckIndex = 0
		return
	}

	// Shuffle the deck 7 times :)
	rng := state.random()
	for shuffle := 0; shuffle < 7; shuffle++ {
		rng.Shuffle(len(state.Deck), func(i, j int) { state.Deck[i], state.Deck[j] = state.Deck[j], state.Deck[i] })
	}
	state.deckIndex = 0
}

func (state *GameState) dealHoleCards() {
	// Deal 2 rounds of cards to each player, one card at a time.
	for cardNum := 0; cardNum < 2; cardNum++ {
		for i := range state.Players {
			player := &state.Players[i]
			if player.Status == STATUS_PLAYING {
				player.Hand = append(player.Hand, state.Deck[state.deckIndex])
				state.deckIndex++
			}
		}
	}
}

func createGameState(playerCount int, registerLobby bool) *GameState {

	state := GameState{}
	state.initializeDeck()
	state.Round = 0
	state.ActivePlayer = -1
	state.registerLobby = registerLobby
	state.CommunityCards = []card{}
	state.buttonPos = -1
	state.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Pre-populate player pool with bots
	for i := 0; i < playerCount; i++ {
		state.addPlayer(botNames[i], true)
	}

	if playerCount < 2 {
		state.LastResult = WAITING_MESSAGE
	}

	return &state
}

// nextSeatWith returns the next seat index strictly after "from" (wrapping) whose
// player status matches "status", or -1 if no such seat exists.
func (state *GameState) nextSeatWith(from int, status Status) int {
	n := len(state.Players)
	if n == 0 {
		return -1
	}
	for i := 1; i <= n; i++ {
		seat := ((from + i) % n)
		if seat < 0 {
			seat += n
		}
		if state.Players[seat].Status == status {
			return seat
		}
	}
	return -1
}

// postBlind posts up to "amount" of a blind for the given seat, going all-in if short
func (state *GameState) postBlind(seat int, amount int) int {
	player := &state.Players[seat]
	if amount > player.Purse {
		amount = player.Purse
	}
	player.Purse -= amount
	player.Bet += amount
	player.totalBet += amount
	state.Pot += amount
	// Just "POST" - the amount shows as the player's bet, and 8-bit clients
	// render moves in a 5-character field
	player.Move = "POST"
	if player.Purse == 0 {
		player.Status = STATUS_ALL_IN
	}
	return amount
}

// newRound starts a brand new hand: resets per-hand state, deals hole cards,
// rotates the dealer button, and posts blinds.
func (state *GameState) newRound() {

	// Drop any players that left last round
	state.dropInactivePlayers(true, false)

	if len(state.Players) < 2 {
		return
	}

	state.GamesPlayed++
	state.Round = 1
	state.Winner = ""
	state.RoundName = "Pre-flop"
	state.Pot = 0
	state.gameOver = false
	state.wonByFolds = false
	state.CommunityCards = []card{}
	state.currentBet = 0
	state.raiseCount = 0
	state.raiseAmount = 0
	state.lastRaiseSize = 0
	log.Printf("=== TEXAS HOLD'EM: Starting new hand (Game #%d) ===", state.GamesPlayed)

	// Reset players for the new hand
	playingCount := 0
	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]

		// A bot with under 25 chips leaves; another takes their place with a fresh purse
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

		player.Hand = []card{}
		player.Bet = 0
		player.totalBet = 0
		player.Move = ""
		player.actedThisRound = false

		// Deal in everyone who has chips and hasn't left
		if player.Status != STATUS_LEFT && player.Purse > 0 {
			player.Status = STATUS_PLAYING
			playingCount++
		} else if player.Status != STATUS_LEFT {
			player.Status = STATUS_WAITING
		}
	}

	if playingCount < 2 {
		state.Round = 0
		state.ActivePlayer = -1
		state.LastResult = WAITING_MESSAGE
		return
	}

	state.shuffleDeck()
	if state.LastResult == WAITING_MESSAGE {
		state.LastResult = ""
	}

	// Deal 2 hole cards to each playing player
	state.dealHoleCards()
	log.Printf("CARDS: Dealt 2 hole cards to each player")

	// Rotate the dealer button to the next playing seat
	state.buttonPos = state.nextSeatWith(state.buttonPos, STATUS_PLAYING)

	var smallBlindIndex, bigBlindIndex, firstToAct int
	if playingCount == 2 {
		// Heads-up: button posts the small blind and acts first pre-flop
		smallBlindIndex = state.buttonPos
		bigBlindIndex = state.nextSeatWith(smallBlindIndex, STATUS_PLAYING)
		firstToAct = smallBlindIndex
	} else {
		smallBlindIndex = state.nextSeatWith(state.buttonPos, STATUS_PLAYING)
		bigBlindIndex = state.nextSeatWith(smallBlindIndex, STATUS_PLAYING)
		firstToAct = state.nextSeatWith(bigBlindIndex, STATUS_PLAYING)
	}

	posted := state.postBlind(smallBlindIndex, SB)
	log.Printf("BLINDS: %s posts small blind $%d", state.Players[smallBlindIndex].Name, posted)
	posted = state.postBlind(bigBlindIndex, BB)
	log.Printf("BLINDS: %s posts big blind $%d", state.Players[bigBlindIndex].Name, posted)

	// The bet to match is the full big blind even if the BB posted short (all-in)
	state.currentBet = BB
	state.lastRaiseSize = BB

	// If a blind poster went all-in posting, first-to-act may need recomputing
	if state.Players[firstToAct].Status != STATUS_PLAYING {
		firstToAct = state.nextSeatWith(firstToAct, STATUS_PLAYING)
	}
	if firstToAct < 0 {
		// Everyone is all-in from the blinds; run out the board immediately
		state.ActivePlayer = -1
		state.runOutBoardAndShowdown()
		return
	}

	state.ActivePlayer = firstToAct
	log.Printf("BETTING: Pre-flop betting begins with %s", state.Players[state.ActivePlayer].Name)
	state.resetPlayerTimer(true)
}

func (state *GameState) getPlayerWithBestVisibleHand(highHand bool) int {

	ranks := [][]int{}

	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		if player.Status == STATUS_PLAYING || player.Status == STATUS_ALL_IN {
			rank := getRank(player.Hand, state.CommunityCards)

			// Add player number to start of rank to hold on to when sorting
			rank = append([]int{i}, rank...)
			ranks = append(ranks, rank)
		}
	}

	if len(ranks) == 0 {
		return 0
	}

	// Sort ascending by EvalRank: LOWER rank value is a BETTER hand (cardrank convention)
	sort.SliceStable(ranks, func(i, j int) bool {
		// ranks[i][0] is the player index, ranks[i][1] is the hand rank.
		return ranks[i][1] < ranks[j][1]
	})

	// After sorting, the player with the best hand is at the top of the list.
	// highHand=false can be used to find the worst hand (for lowball games, etc.)
	result := 0
	if highHand {
		result = ranks[0][0] // Best hand
	} else {
		result = ranks[len(ranks)-1][0] // Worst hand
	}

	if result < 0 {
		result = 0
	}
	return result
}

func (state *GameState) dealCommunityCards(count int) {
	for i := 0; i < count; i++ {
		state.CommunityCards = append(state.CommunityCards, state.Deck[state.deckIndex])
		state.deckIndex++
	}
}

func (state *GameState) resetPlayersForNewBettingRound() {
	state.currentBet = 0
	state.raiseCount = 0
	state.raiseAmount = 0
	state.lastRaiseSize = 0
	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		if player.Status == STATUS_PLAYING || player.Status == STATUS_ALL_IN {
			player.Bet = 0
			player.Move = ""
			player.actedThisRound = false
		}
	}

	// First to act post-flop is the first player still able to act after the button
	firstToAct := state.nextSeatWith(state.buttonPos, STATUS_PLAYING)
	if firstToAct < 0 {
		// Nobody can act (everyone all-in); board will be run out by the caller
		state.ActivePlayer = -1
		return
	}
	state.ActivePlayer = firstToAct
	state.resetPlayerTimer(false)
}

// countByStatus returns the number of players matching any of the given statuses
func (state *GameState) countByStatus(statuses ...Status) int {
	count := 0
	for _, player := range state.Players {
		for _, s := range statuses {
			if player.Status == s {
				count++
				break
			}
		}
	}
	return count
}

// runOutBoardAndShowdown deals any remaining community cards and goes straight to
// showdown. Used when all remaining players are all-in and no more betting is possible.
func (state *GameState) runOutBoardAndShowdown() {
	for len(state.CommunityCards) < 5 {
		if len(state.CommunityCards) == 0 {
			state.dealCommunityCards(3)
		} else {
			state.dealCommunityCards(1)
		}
	}
	state.Round = 4
	state.RoundName = "Showdown"
	log.Printf("PHASE: All players all-in - running out the board to showdown")
	state.endGame(false)
}

func (state *GameState) addPlayer(playerName string, isBot bool) {
	player := Player{
		Name:     playerName,
		Status:   STATUS_WAITING,
		Purse:    STARTING_PURSE,
		isBot:    isBot,
		lastPing: time.Now(),
	}
	if isBot {
		if profile, ok := botProfiles[playerName]; ok {
			player.profile = profile
		} else {
			// Default profile if not found
			player.profile = BotProfile{Name: "Default", VPIP: 0.3, PFR: 0.3, BluffFrequency: 0.1}
		}
	}
	state.Players = append(state.Players, player)
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

	// Extra logic if a player is requesting
	if state.clientPlayer >= 0 {

		// In case a player returns while they are still in the "LEFT" status (before the current game ended), add them back in as waiting
		if state.Players[state.clientPlayer].Status == STATUS_LEFT {
			state.Players[state.clientPlayer].Status = STATUS_WAITING
		}
	}
}

func (state *GameState) endGame(abortGame bool) {
	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 5 // Signifies end of game

	// Bets were already added to the pot when made (performMove/postBlind); just clear them
	for i := range state.Players {
		state.Players[i].Bet = 0
	}
	log.Printf("POT: Final pot size is $%d", state.Pot)

	// Find players still in the hand (including all-in players)
	playersInHandIndices := []int{}
	for i, p := range state.Players {
		if p.Status == STATUS_PLAYING || p.Status == STATUS_ALL_IN {
			playersInHandIndices = append(playersInHandIndices, i)
		}
	}

	var result string

	// Case 1: Only one player left, they win by default.
	if !abortGame && len(playersInHandIndices) == 1 {
		winnerIndex := playersInHandIndices[0]
		winner := &state.Players[winnerIndex]
		log.Printf("WINNER: %s wins by default (all others folded) - pot: $%d", winner.Name, state.Pot)
		winner.Purse += state.Pot
		result = fmt.Sprintf("%s won by default", winner.Name)
		state.wonByFolds = true
	} else if !abortGame && len(playersInHandIndices) > 1 {
		// Case 2: Showdown. Multiple players left.
		log.Printf("SHOWDOWN: %d players remain for showdown", len(playersInHandIndices))
		state.wonByFolds = false

		board := ""
		for _, card := range state.CommunityCards {
			board += valueLookup[card.Rank] + suitLookup[card.Suit]
		}
		boardCards := cardrank.Must(board)

		pockets := [][]cardrank.Card{}
		playerMap := []int{} // map index in pockets back to state.Players index
		for _, playerIndex := range playersInHandIndices {
			player := state.Players[playerIndex]
			hand := ""
			for _, card := range player.Hand {
				hand += valueLookup[card.Rank] + suitLookup[card.Suit]
			}
			pockets = append(pockets, cardrank.Must(hand))
			playerMap = append(playerMap, playerIndex)
		}

		// Use Texas Hold'em evaluator (lower HiRank = better hand)
		evs := cardrank.Holdem.EvalPockets(pockets, boardCards)
		order, pivot := cardrank.Order(evs, false)

		if pivot > 0 {
			// Distribute the pot in layers by contribution level so all-in players
			// only win the portion of the pot they contested (main pot + side pots)
			levels := []int{}
			for _, playerIndex := range playersInHandIndices {
				total := state.Players[playerIndex].totalBet
				if !slices.Contains(levels, total) {
					levels = append(levels, total)
				}
			}
			sort.Ints(levels)

			distributed := 0
			prev := 0
			var lastWinners []int
			for _, level := range levels {
				// Pot slice for this layer: every player (including folded) contributes
				// up to "level", minus what was counted in lower layers
				slice := 0
				for i := range state.Players {
					contrib := state.Players[i].totalBet
					if contrib > level {
						contrib = level
					}
					if contrib > prev {
						slice += contrib - prev
					}
				}
				prev = level
				if slice == 0 {
					continue
				}

				// Winners of this layer: best hand among contenders who contributed this much
				best := cardrank.Invalid
				winners := []int{}
				for pi, playerIndex := range playersInHandIndices {
					if state.Players[playerIndex].totalBet < level {
						continue
					}
					rank := evs[pi].HiRank
					if best == cardrank.Invalid || rank < best {
						best = rank
						winners = []int{playerIndex}
					} else if rank == best {
						winners = append(winners, playerIndex)
					}
				}
				if len(winners) == 0 {
					continue
				}

				share := slice / len(winners)
				remainder := slice % len(winners)
				for j, winnerIndex := range winners {
					state.Players[winnerIndex].Purse += share
					if j == 0 {
						// Odd chip(s) go to the first winner so the pot always balances
						state.Players[winnerIndex].Purse += remainder
					}
				}
				distributed += slice
				lastWinners = winners
				if len(levels) > 1 {
					log.Printf("POT: Layer up to $%d ($%d) won by %v", level, slice, winners)
				}
			}

			// Any chips above the highest contender level (e.g. from a player who bet
			// more and then folded) go to the winners of the top side pot
			leftover := state.Pot - distributed
			if leftover > 0 && len(lastWinners) > 0 {
				state.Players[lastWinners[0]].Purse += leftover
			}

			winnerNames := []string{}
			for i := 0; i < pivot; i++ {
				winnerNames = append(winnerNames, state.Players[playerMap[order[i]]].Name)
			}
			// Desc(false) = the high-hand description (Desc(true) is for lowball games)
			handDesc := evs[order[0]].Desc(false)
			log.Printf("WINNER: %s wins with %s - pot: $%d", strings.Join(winnerNames, " and "), handDesc, state.Pot)
			result = fmt.Sprintf("%s won with %s", strings.Join(winnerNames, " and "), handDesc)
			result = strings.Split(result, " [")[0] // Clean up description
		} else {
			result = "No winner could be determined in showdown."
		}

	} else {
		// Case 3: No players left in hand or game aborted.
		humanAvailSlots, _ := state.getHumanPlayerCountInfo()
		if humanAvailSlots == 8 {
			result = WAITING_MESSAGE
		} else {
			result = "Game ended. Ready for new hand."
		}
	}

	state.Winner = result
	state.LastResult = result
	log.Println(result)

	// Set timer for starting the next hand
	// Always set a delay before the next round starts to allow players to see the result.
	state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
}

// isBettingRoundComplete returns true when every player still able to act has
// voluntarily acted this round and matched the current bet. All-in players are done
// by definition. Also true when nobody can act (everyone all-in).
func (state *GameState) isBettingRoundComplete() bool {
	if state.gameOver || state.Round < 1 || state.Round > 4 {
		return false
	}
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING {
			if !player.actedThisRound || player.Bet < state.currentBet {
				return false
			}
		}
	}
	return true
}

// advanceStreet moves the hand to the next phase (flop/turn/river/showdown)
// after a betting round completes
func (state *GameState) advanceStreet() {
	// If fewer than 2 players remain in the hand, it's over
	if state.countByStatus(STATUS_PLAYING, STATUS_ALL_IN) < 2 {
		state.endGame(false)
		return
	}

	// If fewer than 2 players can still act (rest are all-in), no more betting is
	// possible; run out the remaining board and go to showdown
	if state.Round < 4 && state.countByStatus(STATUS_PLAYING) < 2 {
		state.runOutBoardAndShowdown()
		return
	}

	switch state.Round {
	case 1: // Pre-flop -> Flop
		log.Printf("=== PHASE TRANSITION: Pre-flop betting complete ===")
		state.dealCommunityCards(3)
		state.Round = 2
		state.RoundName = "Flop"
	case 2: // Flop -> Turn
		log.Printf("=== PHASE TRANSITION: Flop betting complete ===")
		state.dealCommunityCards(1)
		state.Round = 3
		state.RoundName = "Turn"
	case 3: // Turn -> River
		log.Printf("=== PHASE TRANSITION: Turn betting complete ===")
		state.dealCommunityCards(1)
		state.Round = 4
		state.RoundName = "River"
	default: // River -> Showdown
		log.Printf("=== PHASE TRANSITION: River betting complete ===")
		state.RoundName = "Showdown"
		log.Printf("PHASE: Showdown - Determining winner with %d community cards", len(state.CommunityCards))
		state.endGame(false)
		return
	}

	log.Printf("PHASE: %s - %d community cards on board", state.RoundName, len(state.CommunityCards))
	state.resetPlayersForNewBettingRound()
	if state.ActivePlayer < 0 {
		// Nobody left who can act; run the board out
		state.runOutBoardAndShowdown()
		return
	}
	log.Printf("BETTING: %s betting begins with %s", state.RoundName, state.Players[state.ActivePlayer].Name)
}

// countActiveHumans returns the number of human players seated at the table
// (any status except LEFT)
func (state *GameState) countActiveHumans() int {
	humans := 0
	for i := range state.Players {
		if !state.Players[i].isBot && state.Players[i].Status != STATUS_LEFT {
			humans++
		}
	}
	return humans
}

// RunGameLogic drives the Texas Hold'em hand forward: starts hands, forces moves for
// bots and timed-out humans, and advances streets when betting rounds complete.
func (state *GameState) RunGameLogic() {

	// Bots never play on their own: if no human (remote) player is seated, abort
	// any hand in progress and park the table until a human joins. Without this,
	// the per-table hub ticker would keep bot-only games running forever.
	// (allowBotGames is a test-only escape hatch.)
	if !state.allowBotGames && state.countActiveHumans() == 0 {
		// Fold anyone still marked as in a hand (also cleans up after an abort
		// triggered elsewhere, e.g. the last human's /leave)
		for i := range state.Players {
			p := &state.Players[i]
			if p.Status == STATUS_PLAYING || p.Status == STATUS_ALL_IN {
				p.Status = STATUS_FOLDED
				p.Move = "FOLD"
			}
		}
		if state.Round > 0 && !state.gameOver {
			log.Printf("TABLE %s: no human players left - all bots fold, game stops", state.TableId)
			state.endGame(true)
		}
		return
	}

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

	if state.gameOver {
		// Create a new game if the end game delay is past
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false, false)
			state.Round = 0
			state.Pot = 0
			state.gameOver = false
			state.newRound()
		}
		return
	}

	// If only one player is left in the hand, they win now
	if state.countByStatus(STATUS_PLAYING, STATUS_ALL_IN) == 1 {
		state.endGame(false)
		return
	}

	// A human move (via /move) may have completed the betting round; advance promptly
	// before the timer gate so play never stalls or double-acts a player
	if state.isBettingRoundComplete() {
		state.advanceStreet()
		return
	}

	// Return if the move timer has not expired
	// Check timer if no active player, or the active player hasn't already left
	if state.ActivePlayer == -1 || state.Players[state.ActivePlayer].Status != STATUS_LEFT {
		moveTimeRemaining := int(time.Until(state.moveExpires).Seconds())
		if moveTimeRemaining > 0 {
			return
		}
	}

	// If there is no active player, we are done
	if state.ActivePlayer < 0 {
		return
	}

	// Edge cases
	// - player leaves when it is their move - skip over them
	// - player's turn but they are waiting (out of this hand) or already all-in
	if state.Players[state.ActivePlayer].Status != STATUS_PLAYING {
		state.nextValidPlayer()
		return
	}

	// Force a move for the active player: bots pick a move, humans time out and fold
	move := ""
	if state.Players[state.ActivePlayer].isBot {
		move = state.getBotMove()
	} else {
		// Human player did not act in time; fold them
		moves := state.getValidMoves()
		if len(moves) > 0 {
			move = moves[0].Move // moves[0] is always FOLD
		}
	}

	if move != "" {
		log.Printf("MOVE: %s plays %s", state.Players[state.ActivePlayer].Name, move)
		state.performMove(move, true)
	}

	// Check for street advancement after the forced move
	if state.isBettingRoundComplete() {
		state.advanceStreet()
	}
}

// getStartingHandStrength evaluates the quality of a two-card Texas Hold'em starting hand.
// Returns a score from 0 to 10, where 10 is the best.
func getStartingHandStrength(cards []card) int {
	if len(cards) != 2 {
		return 0
	}
	c1 := cards[0]
	c2 := cards[1]

	// Ensure c1 is the higher rank card
	if c1.Rank < c2.Rank {
		c1, c2 = c2, c1
	}

	isPair := c1.Rank == c2.Rank
	isSuited := c1.Suit == c2.Suit
	gap := c1.Rank - c2.Rank

	// Pocket Pairs
	if isPair {
		if c1.Rank >= 12 { return 10 } // AA, KK, QQ
		if c1.Rank >= 9 { return 9 }  // JJ, TT, 99
		if c1.Rank >= 6 { return 8 }  // 88, 77, 66
		return 7 // 55, 44, 33, 22
	}

	// Suited High Cards
	if isSuited {
		if c1.Rank >= 13 && c2.Rank >= 11 { return 9 } // AKs, AQs, AJs, KQs
		if c1.Rank >= 13 { return 8 } // ATs, A9s...
		if c1.Rank >= 11 && c2.Rank >= 9 { return 7 } // KJs, KTs, QJs, QTs, JTs
	}

	// Unsuited High Cards
	if c1.Rank >= 13 && c2.Rank >= 12 { return 7 } // AKo, AQo
	if c1.Rank >= 13 && c2.Rank >= 10 { return 6 } // AJo, ATo
	if c1.Rank >= 12 && c2.Rank >= 11 { return 6 } // KQo, KJo

	// Suited Connectors
	if isSuited && gap == 1 {
		if c1.Rank >= 9 { return 6 } // T9s, 98s, 87s
		return 5 // 76s, 65s, 54s
	}

	// Suited Gappers
	if isSuited && gap > 1 && gap <= 3 {
		return 4
	}

	// Anything else
	return 1
}

// findMove returns the valid move with the given 2-character code ("CH", "CA",
// "BL", "RL", ...), or "" if not available
func findMove(moves []validMove, prefix string) string {
	for _, m := range moves {
		if m.Move == prefix || strings.HasPrefix(m.Move, prefix+" ") {
			return m.Move
		}
	}
	return ""
}

func (state *GameState) getBotMove() string {
	player := state.Players[state.ActivePlayer]
	profile := player.profile
	moves := state.getValidMoves()

	// If no moves are possible, do nothing.
	if len(moves) == 0 {
		return ""
	}

	check := findMove(moves, "CH")
	call := findMove(moves, "CA")

	// Prefer the small bet/min raise; aggressive profiles sometimes take the big one
	bet := findMove(moves, "BL")
	if big := findMove(moves, "BH"); bet == "" || (big != "" && state.random().Float64() < profile.PFR/2) {
		bet = big
	}
	raise := findMove(moves, "RL")
	if big := findMove(moves, "RH"); raise == "" || (big != "" && state.random().Float64() < profile.PFR/2) {
		raise = big
	}

	// Fallback preference when no strategic move is chosen: check, then call, then fold
	fallback := func() string {
		if check != "" {
			return check
		}
		if call != "" {
			return call
		}
		return "FO"
	}

	// Pre-flop strategy
	if state.Round == 1 {
		handStrength := getStartingHandStrength(player.Hand)

		// VPIP check: Decide if the hand is strong enough to play based on VPIP.
		// A lower VPIP means the bot is tighter and requires a stronger hand.
		requiredStrength := int(11 - (profile.VPIP * 10))
		if handStrength < requiredStrength {
			// If a bet is required to stay in, fold; otherwise check for free
			if state.currentBet > player.Bet {
				return "FO"
			}
			return fallback()
		}

		// PFR check: raise strong hands based on aggression
		if raise != "" && handStrength >= 8 && state.random().Float64() < profile.PFR {
			return raise
		}

		// Bluffing check
		if state.random().Float64() < profile.BluffFrequency {
			if raise != "" {
				return raise
			}
			if bet != "" {
				return bet
			}
		}

		return fallback()
	}

	// Post-flop strategy: evaluate made-hand strength with the community cards
	rank := cardrank.EvalRank(getRank(player.Hand, state.CommunityCards)[0])
	category := rank.Fixed()

	// Strong hands (two pair or better): bet/raise aggressively per profile
	if category <= cardrank.TwoPair {
		if raise != "" && state.random().Float64() < 0.5+profile.PFR/2 {
			return raise
		}
		if bet != "" {
			return bet
		}
		return fallback()
	}

	// Medium hands (one pair): mostly call/check, occasionally bet
	if category == cardrank.Pair {
		if bet != "" && state.random().Float64() < profile.PFR/2 {
			return bet
		}
		return fallback()
	}

	// Weak hands: check if free, bluff occasionally, otherwise fold to bets
	if check != "" {
		if bet != "" && state.random().Float64() < profile.BluffFrequency {
			return bet
		}
		return check
	}
	if state.random().Float64() < profile.BluffFrequency {
		if raise != "" {
			return raise
		}
		if call != "" {
			return call
		}
	}
	return "FO"
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
		if len(state.Players) > 0 && player.Status != STATUS_LEFT && (inMiddleOfGame || player.isBot || player.lastPing.Compare(cutoff) > 0) {
			players = append(players, player)
		}
	}

	// If one player is left, don't drop them within the round, let the normal game end take care of it
	if inMiddleOfGame && len(players) == 1 {
		return
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
		state.LastResult = WAITING_MESSAGE
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

	player.Status = STATUS_LEFT
	player.Move = "LEFT"

	// Check if no human players are playing. If so, end the game
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING && !player.isBot {
			playersLeft++
		}
	}

	// If the last human dropped mid-hand, stop the game and update the lobby
	if playersLeft == 0 && !state.gameOver {
		state.endGame(true)
		state.dropInactivePlayers(false, false)
		return
	}
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *GameState) playerPing() {
	if state.clientPlayer < 0 || state.clientPlayer >= len(state.Players) {
		return // Not in a valid player context to ping
	}
	state.Players[state.clientPlayer].lastPing = time.Now()
}

// Performs the requested move for the active player, and returns true if successful
func (state *GameState) performMove(move string, internalCall ...bool) bool {
	log.Printf("DEBUG: performMove called with move: %s for player %d\n", move, state.ActivePlayer)

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
	} else if move == "CH" { // CHECK
		// No chips move on a check
	} else { // CA, BL, BH, RL, RH, AI - all move chips into the pot
		// The chip amount is derived server-side from the move code, so clients
		// only ever send short codes (8-bit friendly)
		betAmount := state.moveChipAmount(move)
		if betAmount < 0 {
			return false
		}
		if betAmount > player.Purse {
			betAmount = player.Purse // Cannot bet more than available purse
		}

		player.Purse -= betAmount
		state.Pot += betAmount
		player.Bet += betAmount
		player.totalBet += betAmount

		// A bet/raise above the current bet reopens the action for everyone else
		if player.Bet > state.currentBet {
			raiseSize := player.Bet - state.currentBet
			state.currentBet = player.Bet
			state.lastRaiseSize = raiseSize
			state.raiseCount++
			for i := range state.Players {
				if i != state.ActivePlayer && state.Players[i].Status == STATUS_PLAYING {
					state.Players[i].actedThisRound = false
				}
			}
		}

		if player.Purse == 0 {
			player.Status = STATUS_ALL_IN
		}
	}

	player.actedThisRound = true

	// Assign the move string directly, or use lookup for simple moves
	if lookup, ok := moveLookup[move]; ok {
		player.Move = lookup
	} else {
		player.Move = move
	}
	state.nextValidPlayer()

	return true
}

func (state *GameState) resetPlayerTimer(newRound bool) {
	if state.ActivePlayer < 0 {
		return
	}

	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isBot {
		timeLimit = BOT_TIME_LIMIT
	}

	if newRound {
		timeLimit += NEW_ROUND_FIRST_PLAYER_BUFFER
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	// Move to the next player still able to act (skips folded/left/waiting/all-in).
	// Returns -1 via nextSeatWith if nobody can act - callers handle street advancement.
	state.ActivePlayer = state.nextSeatWith(state.ActivePlayer, STATUS_PLAYING)
	state.resetPlayerTimer(false)
}

// moveChipAmount returns how many chips the active player would put in for the
// given move code, or -1 if the code is not a chip-moving move. Amounts are
// computed server-side so clients only ever send short 2-character codes
// (easy to handle in an 8-bit client's switch statement):
//
//	FO fold | CH check | CA call | BL bet (1xBB) | BH bet (2xBB)
//	RL min-raise | RH bigger raise | AI all-in
func (state *GameState) moveChipAmount(move string) int {
	player := state.Players[state.ActivePlayer]
	callAmount := state.currentBet - player.Bet

	// Minimum raise increment is the size of the last bet/raise this round, or
	// the big blind if there has been none
	minRaiseIncrement := state.lastRaiseSize
	if minRaiseIncrement < BB {
		minRaiseIncrement = BB
	}

	switch move {
	case "CA":
		return callAmount
	case "BL":
		return BB
	case "BH":
		return 2 * BB
	case "RL":
		return callAmount + minRaiseIncrement
	case "RH":
		return callAmount + 2*minRaiseIncrement
	case "AI":
		return player.Purse
	}
	return -1
}

func (state *GameState) getValidMoves() []validMove {
	moves := []validMove{}

	player := state.Players[state.ActivePlayer]

	// Always allow fold
	moves = append(moves, validMove{Move: "FO", Name: "Fold"})

	// Calculate amount needed to call
	callAmount := state.currentBet - player.Bet

	if callAmount <= 0 {
		// Nothing to call: CHECK is free (includes the BB pre-flop option)
		moves = append(moves, validMove{Move: "CH", Name: "Check"})
	} else if player.Purse >= callAmount {
		// CALL is only offered if the player can cover it; otherwise AI below
		// serves as the call-for-less
		moves = append(moves, validMove{Move: "CA", Name: fmt.Sprintf("Call %d", callAmount)})
	}

	if state.currentBet == 0 {
		// No bet yet this round: offer the two bet sizes
		if player.Purse >= state.moveChipAmount("BL") {
			moves = append(moves, validMove{Move: "BL", Name: fmt.Sprintf("Bet %d", BB)})
		}
		if player.Purse >= state.moveChipAmount("BH") {
			moves = append(moves, validMove{Move: "BH", Name: fmt.Sprintf("Bet %d", 2*BB)})
		}
	} else {
		// A bet has been made: offer the two raise sizes
		if rl := state.moveChipAmount("RL"); player.Purse >= rl {
			moves = append(moves, validMove{Move: "RL", Name: fmt.Sprintf("Raise %d", rl)})
		}
		if rh := state.moveChipAmount("RH"); player.Purse >= rh {
			moves = append(moves, validMove{Move: "RH", Name: fmt.Sprintf("Raise %d", rh)})
		}
	}

	// Always allow all-in if the player has chips
	if player.Purse > 0 {
		moves = append(moves, validMove{Move: "AI", Name: fmt.Sprintf("All-in %d", player.Purse)})
	}

	return moves
}

// clientPlayer is the compact player representation sent to clients, following
// the original 8-bit client spec (single character, lower case json keys)
type clientPlayer struct {
	Name   string `json:"n"`
	Status Status `json:"s"`
	Bet    int    `json:"b"`
	Move   string `json:"m"`
	Purse  int    `json:"p"`
	Hand   string `json:"h"`
}

// clientState is the compact state sent to clients (original 8-bit client spec).
// Keys are single character, lower case, to make parsing easy on 8-bit clients;
// array keys are two characters. `c` (community cards) and `z` (state hash) are
// Texas Hold'em additions to the original 5 Card Stud spec.
type clientState struct {
	LastResult   string         `json:"l"`
	Round        int            `json:"r"`
	Pot          int            `json:"p"`
	ActivePlayer int            `json:"a"`
	MoveTime     int            `json:"m"`
	Viewing      int            `json:"v"`
	Community    string         `json:"c"`
	ValidMoves   []validMove    `json:"vm"`
	Players      []clientPlayer `json:"pl"`
	Hash         string         `json:"z"`
}

// cardsToString renders cards in the 2-characters-per-card wire format, e.g.
// "KSKH" = King of Spades + King of Hearts
func cardsToString(cards []card) string {
	s := ""
	for _, c := range cards {
		s += valueLookup[c.Rank] + suitLookup[c.Suit]
	}
	return s
}

// Creates the compact client-centric view of the state (player array rotated so
// the client is index 0, hole cards masked, valid moves only on the client's turn)
func (state *GameState) createClientState() *clientState {

	cs := &clientState{
		LastResult:   state.LastResult,
		Round:        state.Round,
		Pot:          state.Pot,
		ActivePlayer: state.ActivePlayer,
		Community:    cardsToString(state.CommunityCards),
	}

	setActivePlayer := false

	// Check if:
	// 1. The game is over,
	// 2. Only one player is left (waiting for another player to join)
	// 3. The betting round is complete (street about to advance)
	// This lets the client perform end of round/game tasks/animation
	if state.gameOver ||
		len(state.Players) < 2 ||
		state.isBettingRoundComplete() {
		cs.ActivePlayer = -1
		setActivePlayer = true
	}

	// When an observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set flag to let client know they are not actively part of the game
	start := state.clientPlayer
	if start < 0 {
		start = 0
		cs.Viewing = 1
	}

	// Hole cards of everyone still in the hand are revealed at showdown
	showdown := state.Round == 5 && !state.wonByFolds

	// Loop through each player, starting at this client's player, so all clients
	// see the players in the same order regardless of their own position
	for i := start; i < start+len(state.Players); i++ {

		// Wrap around to beginning of player array when needed
		playerIndex := i % len(state.Players)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == state.ActivePlayer {
			setActivePlayer = true
			cs.ActivePlayer = i - start
		}

		player := state.Players[playerIndex]
		cp := clientPlayer{
			Name:   player.Name,
			Status: player.Status,
			Bet:    player.Bet,
			Move:   player.Move,
			Purse:  player.Purse,
		}

		// Build the hand string: own cards (or everyone's at showdown) are visible,
		// other live hands are masked as "??" per card, a folded hand is just "??"
		switch player.Status {
		case STATUS_PLAYING, STATUS_ALL_IN:
			if playerIndex == state.clientPlayer || showdown {
				cp.Hand = cardsToString(player.Hand)
			} else {
				cp.Hand = strings.Repeat("??", len(player.Hand))
			}
		case STATUS_FOLDED:
			cp.Hand = "??"
		}

		cs.Players = append(cs.Players, cp)
	}

	// Determine valid moves for this player (if their turn)
	if cs.ActivePlayer == 0 {
		cs.ValidMoves = state.getValidMoves()
	}

	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	cs.MoveTime = int(time.Until(state.moveExpires).Seconds())

	if cs.ActivePlayer > -1 {
		cs.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	// No need to send move time if the calling player isn't the active player
	if cs.MoveTime < 0 || cs.ActivePlayer != 0 {
		cs.MoveTime = 0
	}

	// Compute hash - this will be compared with an incoming hash. If the same, the entire state does not
	// need to be sent back. This speeds up checks for change in state
	cs.Hash = "0"
	hash, _ := hashstructure.Hash(cs, hashstructure.FormatV2, nil)
	cs.Hash = fmt.Sprintf("%d", hash)

	return cs
}

func (state *GameState) updateLobby() {
	if !state.registerLobby {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	// Send the total human slots / players to the Lobby
	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.TableId)
}

// Return number of active human players in the table, for the lobby
func (state *GameState) getHumanPlayerCountInfo() (int, int) {
	humanAvailSlots := 8
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if player.isBot {
			humanAvailSlots--
		} else if player.Status != STATUS_LEFT && player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}
	return humanAvailSlots, humanPlayerCount
}

// getRank evaluates the best 5-card hand from 2 hole cards plus the community cards.
// Cards use the live deck convention: Rank 2..14 (14 = Ace), Suit 0..3.
// Returns the cardrank EvalRank as a single-element slice; LOWER is BETTER
// (1 = royal flush, 7462 = worst high card).
func getRank(holeCards []card, communityCards []card) []int {
	// The evaluator needs a full 5-card hand available (2 hole + 3+ community)
	if len(holeCards) < 2 || len(holeCards)+len(communityCards) < 5 {
		return []int{int(cardrank.Nothing)}
	}

	// Convert to cardrank string notation (same convention as endGame showdown)
	pocket := ""
	for _, c := range holeCards {
		pocket += valueLookup[c.Rank] + suitLookup[c.Suit]
	}
	board := ""
	for _, c := range communityCards {
		board += valueLookup[c.Rank] + suitLookup[c.Suit]
	}

	ev := cardrank.Holdem.Eval(cardrank.Must(pocket), cardrank.Must(board))
	return []int{int(ev.HiRank)}
}
