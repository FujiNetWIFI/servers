package main

import (
	"slices"
	"testing"
	"time"
)

func expireTimer(state *GameState) {
	state.moveExpires = time.Time{}
}

func TestBettingTimeoutSitsOut(t *testing.T) {
	state := newTable(t, 1, 1)
	rig(t, state, "TS", "9H", "8D", "7C", "KD")
	state.RunGameLogic()
	expectEqual(t, "active", state.Players[state.ActivePlayer].Name, "p1")
	expireTimer(state)
	state.RunGameLogic()

	p := player(t, state, "p1")
	expectEqual(t, "status", p.Status, STATUS_WAITING)
	expectEqual(t, "move", p.Move, "sit out")
	expectEqual(t, "hands", len(p.Hands), 0)
	expectEqual(t, "round", state.Round, ROUND_PLAYING)
	expectEqual(t, "bot in", player(t, state, "Clyd BOT").Status, STATUS_PLAYING)
	finishHand(t, state)
	expectEqual(t, "purse untouched", p.Purse, 1000)
}

func TestNobodyBets(t *testing.T) {
	state := newTable(t, 2, 0)
	state.RunGameLogic()
	expireTimer(state)
	state.RunGameLogic()
	expireTimer(state)
	state.RunGameLogic()
	expectEqual(t, "round", state.Round, ROUND_OVER)
	expectEqual(t, "result", state.LastResult, "No bets this hand")
	state.RunGameLogic()
	expectEqual(t, "betting reopens", state.Round, ROUND_BETTING)
}

func TestPlayTimeoutStands(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "9H", "8D", "7C", "KD")
	bet(t, state, "B1")
	state.RunGameLogic()
	expectEqual(t, "still waiting on the player", state.Round, ROUND_PLAYING)
	expireTimer(state)
	state.RunGameLogic()
	p := player(t, state, "p1")
	expectEqual(t, "move", p.Move, "stand")
	finishHand(t, state)
	expectEqual(t, "purse", p.Purse, 1010)
}

func TestRebetAndBetLimits(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "9H", "8D", "7C")
	bet(t, state, "B2")
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)

	state.RunGameLogic()
	moves := state.getValidMoves()
	expectEqual(t, "first move", moves[0], validMove{MOVE_REBET, "Same 20"})
	expectEqual(t, "count", len(moves), 5)

	player(t, state, "p1").Purse = 30
	expectEqual(t, "short purse", slices.Equal(validCodes(state), []string{MOVE_REBET, "B1", "B2"}), true)
	move(t, state, "p1", MOVE_REBET)
	expectEqual(t, "rebet amount", player(t, state, "p1").Hands[0].Bet, 20)
}

func TestBrokePlayerRebuys(t *testing.T) {
	state := newTable(t, 1, 0)
	player(t, state, "p1").Purse = 5
	state.RunGameLogic()
	expectEqual(t, "purse", player(t, state, "p1").Purse, STARTING_PURSE)
	expectEqual(t, "rebuys", state.rebuys, 995)
}

func TestJoinMidHandWaitsForNextHand(t *testing.T) {
	state := newTable(t, 1, 1)
	rig(t, state, "TS", "9S", "9H", "8D", "TC", "7C")
	bet(t, state, "B1")
	state.setClientPlayerByName("p2")
	p2 := player(t, state, "p2")
	expectEqual(t, "status", p2.Status, STATUS_WAITING)
	expectEqual(t, "hands", len(p2.Hands), 0)
	expectEqual(t, "view", state.createClientState().Viewing, 0)

	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)
	state.RunGameLogic()
	expectEqual(t, "humans seated first", state.Players[1].Name, "p2")
	move(t, state, "p1", "B1")
	expectEqual(t, "p2 asked to bet", state.Players[state.ActivePlayer].Name, "p2")
}

func TestFullTableQueuesThenParksABot(t *testing.T) {
	state := newTable(t, 1, 4)
	rig(t, state, "TS", "9S", "9S", "9S", "9S", "9H", "8D", "TC", "TC", "TC", "TC", "7C")
	bet(t, state, "B1")

	state.setClientPlayerByName("p2")
	expectEqual(t, "seats", len(state.Players), MAX_PLAYERS)
	expectEqual(t, "queued", len(state.joinQueue), 1)
	expectEqual(t, "viewing", state.createClientState().Viewing, 1)
	_, humans := state.getHumanPlayerCountInfo()
	expectEqual(t, "lobby count includes the queue", humans, 2)

	state.setClientPlayerByName("p1")
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)
	state.RunGameLogic()

	expectEqual(t, "seats", len(state.Players), MAX_PLAYERS)
	expectEqual(t, "queue", len(state.joinQueue), 0)
	expectEqual(t, "parked", len(state.botBox), 1)
	expectEqual(t, "seat 0", state.Players[0].Name, "p1")
	expectEqual(t, "seat 1", state.Players[1].Name, "p2")

	state.setClientPlayerByName("p2")
	state.clientLeave()
	expireHumanTimers(t)
	move(t, state, "p1", "B1")
	finishHand(t, state)
	state.RunGameLogic()
	expectEqual(t, "bot back", len(state.botBox), 0)
	expectEqual(t, "seats", len(state.Players), MAX_PLAYERS)
	if slices.ContainsFunc(state.Players, func(p Player) bool { return p.Name == "p2" }) {
		t.Errorf("p2 still seated")
	}
}

func TestLastHumanLeavingRefundsAndStops(t *testing.T) {
	state := newTable(t, 1, 2)
	rig(t, state, "TS", "9S", "9S", "9H", "8D", "TC", "TC", "7C")
	bet(t, state, "B3")
	expectEqual(t, "round", state.Round, ROUND_PLAYING)

	state.setClientPlayerByName("p1")
	state.clientLeave()
	expectEqual(t, "round", state.Round, ROUND_WAITING)
	for _, p := range state.Players {
		expectEqual(t, p.Name+" purse", p.Purse, 1000)
	}
	expectEqual(t, "leaver dropped", len(state.Players), 2)

	for i := 0; i < 5; i++ {
		state.RunGameLogic()
	}
	expectEqual(t, "bots do not play alone", state.Round, ROUND_WAITING)
}

func TestBotsNeedAHuman(t *testing.T) {
	state := newTable(t, 0, 2)
	state.RunGameLogic()
	expectEqual(t, "round", state.Round, ROUND_WAITING)
}

func TestActivePlayerLeavingIsSkipped(t *testing.T) {
	state := newTable(t, 2, 0)
	rig(t, state, "TS", "TD", "9H", "8D", "8C", "7C", "KD")
	bet(t, state, "B1", "B1")
	state.setClientPlayerByName("p1")
	state.clientLeave()
	state.RunGameLogic()
	expectEqual(t, "active", state.Players[state.ActivePlayer].Name, "p2")
	move(t, state, "p2", MOVE_STAND)
	finishHand(t, state)
	expectEqual(t, "p2 purse", player(t, state, "p2").Purse, 1010)
}

func TestReturningMidHandRejoins(t *testing.T) {
	state := newTable(t, 2, 0)
	rig(t, state, "TS", "TD", "9H", "8D", "8C", "7C")
	bet(t, state, "B1", "B1")
	state.setClientPlayerByName("p1")
	state.clientLeave()
	state.setClientPlayerByName("p1")
	expectEqual(t, "status", player(t, state, "p1").Status, STATUS_PLAYING)
	move(t, state, "p1", MOVE_STAND)
}

func TestStalePlayerDropped(t *testing.T) {
	state := newTable(t, 2, 0)
	player(t, state, "p2").lastPing = time.Now().Add(2 * PLAYER_PING_TIMEOUT)
	state.RunGameLogic()
	expectEqual(t, "players", len(state.Players), 1)
}

func TestReshuffleAtCutCard(t *testing.T) {
	state := newTable(t, 1, 0)
	bet(t, state, "B1")
	shoe := state.shoe
	expectEqual(t, "full shoe", len(shoe), DECKS*52)

	state.abortHand()
	state.shoeIndex = RESHUFFLE_AT - 1
	bet(t, state, "B1")
	expectEqual(t, "same shoe below the cut card", &state.shoe[0], &shoe[0])

	state.abortHand()
	state.shoeIndex = RESHUFFLE_AT
	bet(t, state, "B1")
	if &state.shoe[0] == &shoe[0] {
		t.Errorf("no reshuffle at the cut card")
	}
	expectEqual(t, "cards dealt from the new shoe", state.shoeIndex, 4)
}

func TestShoeComposition(t *testing.T) {
	state := newTable(t, 0, 0)
	state.newShoe()
	counts := map[card]int{}
	for _, c := range state.shoe {
		counts[c]++
	}
	expectEqual(t, "distinct cards", len(counts), 52)
	for c, n := range counts {
		if n != DECKS {
			t.Errorf("%v appears %d times", c, n)
		}
	}
}

func TestClientView(t *testing.T) {
	state := newTable(t, 2, 1)
	rig(t, state, "TS", "AS", "9S", "9H", "8D", "2C", "KC", "7C", "KD")
	bet(t, state, "B1", "B2")

	state.setClientPlayerByName("p2")
	cs := state.createClientState()
	expectEqual(t, "client first", cs.Players[0].Name, "p2")
	expectEqual(t, "p1 relative seat", cs.Players[2].Name, "p1")
	expectEqual(t, "active is p1", cs.ActivePlayer, 2)
	expectEqual(t, "no moves off turn", len(cs.ValidMoves), 0)
	expectEqual(t, "no timer off turn", cs.MoveTime, 0)
	expectEqual(t, "hole card hidden", cs.Dealer, "9H??")
	expectEqual(t, "up card total only", cs.DealerTotal, 9)
	expectEqual(t, "p2 hand", cs.Players[0].Hands[0].Cards, "AS2C")
	expectEqual(t, "p2 total", cs.Players[0].Hands[0].Total, 13)
	expectEqual(t, "p2 soft", cs.Players[0].Hands[0].Flags&FLAG_SOFT, FLAG_SOFT)

	state.setClientPlayerByName("p1")
	cs = state.createClientState()
	expectEqual(t, "active is me", cs.ActivePlayer, 0)
	expectEqual(t, "my moves", len(cs.ValidMoves), 4)
	expectEqual(t, "two humans timer", cs.MoveTime, int(PLAYER_TIME_LIMIT.Seconds())-MOVE_TIME_GRACE_SECONDS-1)

	move(t, state, "p1", MOVE_STAND)
	move(t, state, "p2", MOVE_STAND)
	finishHand(t, state)
	cs = state.createClientState()
	expectEqual(t, "hole revealed", cs.Dealer, "9H7CKD")
	expectEqual(t, "dealer total", cs.DealerTotal, 16+10)
	expectEqual(t, "dealer bust flag", cs.DealerFlags&FLAG_BUST, FLAG_BUST)

	state.setClientPlayerByName("")
	expectEqual(t, "anonymous viewer", state.createClientState().Viewing, 1)
}

func TestSinglePlayerTimer(t *testing.T) {
	state := newTable(t, 1, 2)
	state.setClientPlayerByName("p1")
	state.RunGameLogic()
	cs := state.createClientState()
	if cs.MoveTime < 249 || cs.MoveTime > 251 {
		t.Errorf("solo move time %d, want about 251", cs.MoveTime)
	}
}
