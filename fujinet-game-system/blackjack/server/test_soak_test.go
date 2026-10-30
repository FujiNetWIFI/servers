package main

import (
	"testing"
)

// Chips must balance at every step, and basic strategy's edge must land near
// the known -0.4% to -0.5% for these rules
func TestBotSoak(t *testing.T) {
	rounds := 300000
	if testing.Short() {
		rounds = 10000
	}

	state := newTable(t, 0, len(botNames))
	state.allowBotGames = true
	start := 0
	for _, p := range state.Players {
		start += p.Purse
	}

	for state.handsPlayed < rounds {
		state.RunGameLogic()

		chips := state.houseNet - state.rebuys
		for _, p := range state.Players {
			chips += p.Purse
			if state.Round != ROUND_OVER {
				for _, h := range p.Hands {
					chips += h.Bet
				}
			}
		}
		if chips != start {
			t.Fatalf("hand %d round %d: chips %d, want %d", state.handsPlayed, state.Round, chips, start)
		}
	}

	edge := -float64(state.houseNet) / float64(state.wagered)
	t.Logf("%d rounds, %d wagered, player edge %.3f%%", rounds, state.wagered, edge*100)
	if !testing.Short() && (edge < -0.009 || edge > -0.001) {
		t.Errorf("player edge %.3f%% is outside -0.9%%..-0.1%%", edge*100)
	}
}
