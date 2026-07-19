package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// useFastTimers zeroes the game timers so tests run instantly. Restores originals
// on cleanup. Tests using this must not run in parallel (package globals).
func useFastTimers(t *testing.T) {
	t.Helper()
	origBot := BOT_TIME_LIMIT
	origPlayer := PLAYER_TIME_LIMIT
	origEndgame := ENDGAME_TIME_LIMIT
	origBuffer := NEW_ROUND_FIRST_PLAYER_BUFFER

	// Negative values put moveExpires in the past so every gate opens immediately
	BOT_TIME_LIMIT = -time.Second
	ENDGAME_TIME_LIMIT = -time.Second * 2
	NEW_ROUND_FIRST_PLAYER_BUFFER = 0

	t.Cleanup(func() {
		BOT_TIME_LIMIT = origBot
		PLAYER_TIME_LIMIT = origPlayer
		ENDGAME_TIME_LIMIT = origEndgame
		NEW_ROUND_FIRST_PLAYER_BUFFER = origBuffer
	})
}

// parseCard converts a string like "AS" (Ace of Spades) or "TD" (Ten of Diamonds)
// into the internal card struct. Rank 2..14 (A=14), Suit C=0 D=1 H=2 S=3.
func parseCard(t *testing.T, s string) card {
	t.Helper()
	if len(s) != 2 {
		t.Fatalf("invalid card string %q", s)
	}
	rank := -1
	switch s[0] {
	case 'T':
		rank = 10
	case 'J':
		rank = 11
	case 'Q':
		rank = 12
	case 'K':
		rank = 13
	case 'A':
		rank = 14
	default:
		if s[0] >= '2' && s[0] <= '9' {
			rank = int(s[0] - '0')
		}
	}
	suit := -1
	switch s[1] {
	case 'C':
		suit = 0
	case 'D':
		suit = 1
	case 'H':
		suit = 2
	case 'S':
		suit = 3
	}
	if rank < 0 || suit < 0 {
		t.Fatalf("invalid card string %q", s)
	}
	return card{Rank: rank, Suit: suit}
}

func parseCards(t *testing.T, strs ...string) []card {
	t.Helper()
	cards := make([]card, 0, len(strs))
	for _, s := range strs {
		cards = append(cards, parseCard(t, s))
	}
	return cards
}

// rigDeck installs a test deck so the next hand deals exactly the given hole cards
// and community cards. holeCards[i] is the 2-card hand for the i-th PLAYING seat
// (in seat order). Deal order matches dealHoleCards: one card per player per pass.
func rigDeck(t *testing.T, state *GameState, holeCards [][]string, community []string) {
	t.Helper()
	deck := []card{}
	used := map[card]bool{}
	add := func(s string) {
		c := parseCard(t, s)
		if used[c] {
			t.Fatalf("card %s used twice in rigged deck", s)
		}
		used[c] = true
		deck = append(deck, c)
	}

	// Hole cards are dealt one card at a time, one pass per card number
	for cardNum := 0; cardNum < 2; cardNum++ {
		for _, hand := range holeCards {
			if len(hand) != 2 {
				t.Fatalf("each rigged hand needs exactly 2 cards, got %v", hand)
			}
			add(hand[cardNum])
		}
	}
	for _, s := range community {
		add(s)
	}

	// Fill the rest of the deck with the unused cards so it is a full 52
	for suit := 0; suit < 4; suit++ {
		for rank := 2; rank < 15; rank++ {
			c := card{Rank: rank, Suit: suit}
			if !used[c] {
				deck = append(deck, c)
			}
		}
	}
	state.testDeck = deck
}

// handTrace records what happened while a hand was played out
type handTrace struct {
	phases     []string // RoundName transitions observed
	iterations int
}

// playHand drives RunGameLogic until the current hand ends (gameOver). The state
// must be mid-hand or ready to start one. Fails the test if the hand doesn't
// complete within maxIters iterations.
func playHand(t *testing.T, state *GameState, maxIters int) handTrace {
	t.Helper()
	trace := handTrace{}
	lastPhase := ""
	for i := 0; i < maxIters; i++ {
		if state.RoundName != lastPhase && state.RoundName != "" {
			trace.phases = append(trace.phases, state.RoundName)
			lastPhase = state.RoundName
		}
		if state.gameOver {
			trace.iterations = i
			return trace
		}
		state.RunGameLogic()
	}
	t.Fatalf("hand did not complete within %d iterations (Round=%d %s, ActivePlayer=%d, gameOver=%v)",
		maxIters, state.Round, state.RoundName, state.ActivePlayer, state.gameOver)
	return trace
}

// totalChips sums all chips in play: player purses + the pot. Chips move from
// Purse into Pot at bet time (Player.Bet is bookkeeping, not separate chips).
// After a hand ends the pot has been paid out but the Pot field is kept for
// display, so it must not be counted again.
func totalChips(state *GameState) int {
	total := 0
	if !state.gameOver {
		total = state.Pot
	}
	for _, p := range state.Players {
		total += p.Purse
	}
	return total
}

// newBotTable creates a test table with the given number of bots and a seeded RNG.
// Engine tests drive bot-only games, so the humans-required rule is lifted.
func newBotTable(botCount int, seed int64) *GameState {
	initializeGameServer()
	state := createGameState(botCount, false)
	state.TableId = fmt.Sprintf("test-%d-%d", botCount, seed)
	state.rng = rand.New(rand.NewSource(seed))
	state.allowBotGames = true
	return state
}
