package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	gin.DefaultWriter = io.Discard
	initializeGameServer()
	os.Exit(m.Run())
}

// Bots, the dealer and the end of hand pause act instantly. Human timers are
// left alone so humans only time out when a test asks for it.
func useFastTimers(t *testing.T) {
	t.Helper()
	bot, dealer, endgame := BOT_TIME_LIMIT, DEALER_TIME_LIMIT, ENDGAME_TIME_LIMIT
	BOT_TIME_LIMIT, DEALER_TIME_LIMIT, ENDGAME_TIME_LIMIT = -time.Second, -time.Second, -time.Second
	t.Cleanup(func() { BOT_TIME_LIMIT, DEALER_TIME_LIMIT, ENDGAME_TIME_LIMIT = bot, dealer, endgame })
}

func expireHumanTimers(t *testing.T) {
	t.Helper()
	player, single := PLAYER_TIME_LIMIT, PLAYER_TIME_LIMIT_SINGLE_PLAYER
	PLAYER_TIME_LIMIT, PLAYER_TIME_LIMIT_SINGLE_PLAYER = -time.Second, -time.Second
	t.Cleanup(func() { PLAYER_TIME_LIMIT, PLAYER_TIME_LIMIT_SINGLE_PLAYER = player, single })
}

func newTable(t *testing.T, humans int, bots int) *GameState {
	t.Helper()
	useFastTimers(t)
	state := createGameState(bots, false)
	state.TableId = "test"
	state.rng = rand.New(rand.NewSource(1))
	state.fixedBotBet = 10
	for i := 1; i <= humans; i++ {
		state.setClientPlayerByName(fmt.Sprintf("p%d", i))
	}
	return state
}

func parseCard(t *testing.T, s string) card {
	t.Helper()
	rank := strings.Index("..23456789TJQKA", s[:1])
	suit := strings.Index("CDHS", s[1:])
	if len(s) != 2 || rank < 2 || suit < 0 {
		t.Fatalf("bad card %q", s)
	}
	return card{Rank: rank, Suit: suit}
}

// Cards in the order they are drawn: each bettor's first card, the dealer's
// up card, each bettor's second card, the dealer's hole card, then every
// later draw
func rig(t *testing.T, state *GameState, cards ...string) {
	t.Helper()
	state.testDeck = nil
	for _, s := range cards {
		state.testDeck = append(state.testDeck, parseCard(t, s))
	}
}

func player(t *testing.T, state *GameState, name string) *Player {
	t.Helper()
	i := slices.IndexFunc(state.Players, func(p Player) bool { return p.Name == name })
	if i < 0 {
		t.Fatalf("no player %q", name)
	}
	return &state.Players[i]
}

func move(t *testing.T, state *GameState, name string, m string) {
	t.Helper()
	if state.ActivePlayer < 0 || state.Players[state.ActivePlayer].Name != name {
		t.Fatalf("%s to play %s but active player is %d (round %d)", name, m, state.ActivePlayer, state.Round)
	}
	if !state.performMove(m) {
		t.Fatalf("%s: move %s rejected, valid %v", name, m, state.getValidMoves())
	}
}

// Opens betting and has each human, in seat order, make the given bet
func bet(t *testing.T, state *GameState, bets ...string) {
	t.Helper()
	state.RunGameLogic()
	if state.Round != ROUND_BETTING {
		t.Fatalf("expected betting, round %d", state.Round)
	}
	for i, b := range bets {
		move(t, state, fmt.Sprintf("p%d", i+1), b)
	}
}

func finishHand(t *testing.T, state *GameState) {
	t.Helper()
	for i := 0; i < 100 && state.Round != ROUND_OVER; i++ {
		state.RunGameLogic()
	}
	if state.Round != ROUND_OVER {
		t.Fatalf("hand never finished: round %d active %d", state.Round, state.ActivePlayer)
	}
}

func validCodes(state *GameState) []string {
	codes := []string{}
	for _, m := range state.getValidMoves() {
		codes = append(codes, m.Move)
	}
	return codes
}

func expectEqual[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}
