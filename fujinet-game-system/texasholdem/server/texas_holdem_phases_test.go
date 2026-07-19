package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTexasHoldemSetup verifies a new hand is set up correctly by the real engine
func TestTexasHoldemSetup(t *testing.T) {
	t.Run("Blinds_And_Deal", func(t *testing.T) {
		state := newBotTable(4, 42)
		state.newRound()

		assert.Equal(t, 1, state.Round, "hand starts at round 1 (pre-flop)")
		assert.Equal(t, "Pre-flop", state.RoundName)
		assert.Equal(t, SB+BB, state.Pot, "pot holds the blinds")
		assert.Equal(t, BB, state.currentBet, "current bet is the big blind")
		assert.Empty(t, state.CommunityCards, "no community cards pre-flop")

		for i, p := range state.Players {
			assert.Len(t, p.Hand, 2, "player %d has 2 hole cards", i)
			assert.Equal(t, STATUS_PLAYING, p.Status)
		}

		// Exactly one SB and one BB were posted
		blinds := []int{}
		for _, p := range state.Players {
			if p.Move == "POST" {
				blinds = append(blinds, p.Bet)
			}
		}
		assert.ElementsMatch(t, []int{SB, BB}, blinds, "one small and one big blind posted")

		assert.Equal(t, 4*STARTING_PURSE, totalChips(state), "chips conserved after blinds")
	})

	t.Run("Positions_Three_Plus_Players", func(t *testing.T) {
		state := newBotTable(4, 42)
		state.newRound()

		n := len(state.Players)
		sb := (state.buttonPos + 1) % n
		bb := (state.buttonPos + 2) % n
		utg := (state.buttonPos + 3) % n

		assert.Equal(t, "POST", state.Players[sb].Move, "seat left of button posts SB")
		assert.Equal(t, SB, state.Players[sb].Bet)
		assert.Equal(t, "POST", state.Players[bb].Move, "next seat posts BB")
		assert.Equal(t, BB, state.Players[bb].Bet)
		assert.Equal(t, utg, state.ActivePlayer, "UTG (left of BB) acts first pre-flop")
	})
}

// TestHeadsUpBlindsAndOrder verifies heads-up rules: button posts SB and acts
// first pre-flop; the other player acts first post-flop
func TestHeadsUpBlindsAndOrder(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(2, 7)
	state.newRound()

	button := state.buttonPos
	other := (button + 1) % 2

	assert.Equal(t, "POST", state.Players[button].Move, "button posts the small blind heads-up")
	assert.Equal(t, SB, state.Players[button].Bet)
	assert.Equal(t, "POST", state.Players[other].Move, "non-button posts the big blind")
	assert.Equal(t, BB, state.Players[other].Bet)
	assert.Equal(t, button, state.ActivePlayer, "button acts first pre-flop heads-up")

	// Complete the pre-flop round: button calls, BB checks
	state.performMove("CA", true)
	assert.Equal(t, other, state.ActivePlayer, "BB gets the option after a call")
	state.performMove("CH", true)

	assert.True(t, state.isBettingRoundComplete())
	state.advanceStreet()
	assert.Equal(t, 2, state.Round, "advanced to the flop")
	assert.Len(t, state.CommunityCards, 3)
	assert.Equal(t, other, state.ActivePlayer, "non-button acts first post-flop heads-up")
}

// TestBettingRoundTermination verifies the BB option and raise re-opening logic
func TestBettingRoundTermination(t *testing.T) {
	useFastTimers(t)

	t.Run("BB_Gets_Option_After_Limps", func(t *testing.T) {
		state := newBotTable(4, 11)
		state.newRound()

		n := len(state.Players)
		bb := (state.buttonPos + 2) % n

		// Everyone limps around to the BB
		for state.ActivePlayer != bb {
			if state.currentBet > state.Players[state.ActivePlayer].Bet {
				assert.True(t, state.performMove("CA", true))
			} else {
				assert.True(t, state.performMove("CH", true))
			}
			assert.False(t, state.isBettingRoundComplete(), "round must not end before the BB has acted")
		}

		// BB checks the option; now the round is complete
		assert.True(t, state.performMove("CH", true))
		assert.True(t, state.isBettingRoundComplete(), "round ends after BB checks the option")
	})

	t.Run("Raise_Reopens_Action", func(t *testing.T) {
		state := newBotTable(3, 11)
		state.newRound()

		// UTG (= button seat with 3 players... first to act) calls
		first := state.ActivePlayer
		assert.True(t, state.performMove("CA", true))

		// SB raises
		sbSeat := state.ActivePlayer
		raise := findMove(state.getValidMoves(), "RL")
		assert.NotEmpty(t, raise, "SB must have a raise available")
		assert.True(t, state.performMove(raise, true))

		// The original caller must be required to act again
		assert.False(t, state.Players[first].actedThisRound, "a raise must reopen action for prior callers")
		assert.False(t, state.isBettingRoundComplete())
		_ = sbSeat
	})
}

// TestFoldWinsPot verifies a hand ends immediately when all but one player folds,
// with the pot awarded to the survivor and chips conserved
func TestFoldWinsPot(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(3, 5)
	state.newRound()

	total := totalChips(state)
	n := len(state.Players)
	bb := (state.buttonPos + 2) % n
	bbName := state.Players[bb].Name
	bbPurseBefore := state.Players[bb].Purse
	pot := state.Pot

	// Everyone folds to the big blind
	for state.countByStatus(STATUS_PLAYING) > 1 {
		assert.True(t, state.performMove("FO", true))
	}
	state.RunGameLogic()

	assert.True(t, state.gameOver, "hand ends when all but one fold")
	assert.True(t, state.wonByFolds)
	assert.Contains(t, state.Winner, bbName, "big blind wins by default")
	assert.Equal(t, bbPurseBefore+pot, state.Players[bb].Purse, "winner collects the pot")
	assert.Equal(t, total, totalChips(state), "chips conserved")
}
