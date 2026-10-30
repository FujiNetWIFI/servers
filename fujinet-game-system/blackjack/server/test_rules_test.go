package main

import (
	"slices"
	"testing"
)

func TestHandValue(t *testing.T) {
	cases := []struct {
		cards []string
		total int
		soft  bool
	}{
		{[]string{"AS", "KD"}, 21, true},
		{[]string{"AS", "AD"}, 12, true},
		{[]string{"AS", "AD", "9C"}, 21, true},
		{[]string{"AS", "6D", "TC"}, 17, false},
		{[]string{"AS", "AD", "AH", "AC"}, 14, true},
		{[]string{"KS", "QD", "2C"}, 22, false},
		{[]string{"5S", "6D"}, 11, false},
	}
	for _, c := range cases {
		state := &GameState{}
		rig(t, state, c.cards...)
		total, soft := handValue(state.testDeck)
		if total != c.total || soft != c.soft {
			t.Errorf("%v: got %d soft=%v, want %d soft=%v", c.cards, total, soft, c.total, c.soft)
		}
	}
}

func TestPlayerBlackjackPaysThreeToTwo(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "AS", "9H", "KD", "7C")
	bet(t, state, "B1")
	finishHand(t, state)

	p := player(t, state, "p1")
	expectEqual(t, "purse", p.Purse, 1015)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_BLACKJACK)
	expectEqual(t, "move", p.Move, "+15")
	expectEqual(t, "dealer cards", len(state.Dealer), 2)
}

func TestDealerBlackjackAceUp(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "AH", "9D", "KC")
	bet(t, state, "B2")

	expectEqual(t, "round right after the deal", state.Round, ROUND_OVER)
	p := player(t, state, "p1")
	expectEqual(t, "purse", p.Purse, 980)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_LOSE)
	expectEqual(t, "last result", state.LastResult, "Dealer has blackjack")
}

func TestDealerBlackjackTenUp(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "KH", "9D", "AC")
	bet(t, state, "B1")
	expectEqual(t, "round right after the deal", state.Round, ROUND_OVER)
	expectEqual(t, "purse", player(t, state, "p1").Purse, 990)
}

func TestBothBlackjackPush(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "AS", "AH", "KD", "QC")
	bet(t, state, "B1")
	p := player(t, state, "p1")
	expectEqual(t, "round", state.Round, ROUND_OVER)
	expectEqual(t, "purse", p.Purse, 1000)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_PUSH)
	expectEqual(t, "move", p.Move, "push")
}

func TestPeekWithoutBlackjackPlaysOn(t *testing.T) {
	for _, up := range []string{"KH", "AH"} {
		state := newTable(t, 1, 0)
		hole := "7C"
		if up == "AH" {
			hole = "6C" // soft 17 stands
		}
		rig(t, state, "TS", up, "8D", hole)
		bet(t, state, "B1")
		expectEqual(t, up+" round", state.Round, ROUND_PLAYING)
		move(t, state, "p1", MOVE_STAND)
		finishHand(t, state)
		expectEqual(t, up+" dealer cards", len(state.Dealer), 2)
		expectEqual(t, up+" purse", player(t, state, "p1").Purse, 1010)
	}
}

func TestPlayerBustDealerDoesNotDraw(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "6H", "6D", "5C", "KS")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_HIT)
	p := player(t, state, "p1")
	expectEqual(t, "move", p.Move, "bust")
	finishHand(t, state)
	expectEqual(t, "dealer cards", len(state.Dealer), 2)
	expectEqual(t, "purse", p.Purse, 990)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_LOSE)
}

func TestDealerBusts(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "6H", "7D", "TC", "KS")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)
	expectEqual(t, "dealer cards", len(state.Dealer), 3)
	expectEqual(t, "last result", state.LastResult, "Dealer busts with 26")
	expectEqual(t, "purse", player(t, state, "p1").Purse, 1010)
}

func TestDealerHitsSoftSixteenStandsSoftEighteen(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "5H", "9D", "AC", "2S", "KD")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)
	expectEqual(t, "dealer cards", len(state.Dealer), 3)
	expectEqual(t, "dealer total", state.createClientState().DealerTotal, 18)
	expectEqual(t, "purse", player(t, state, "p1").Purse, 1010)
}

func TestPush(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "TH", "9D", "9C")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)
	p := player(t, state, "p1")
	expectEqual(t, "purse", p.Purse, 1000)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_PUSH)
}

func TestDoubleWin(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "6S", "6H", "5D", "TC", "TD", "8S")
	bet(t, state, "B2")
	move(t, state, "p1", MOVE_DOUBLE)
	p := player(t, state, "p1")
	expectEqual(t, "bet after double", p.Hands[0].Bet, 40)
	expectEqual(t, "cards after double", len(p.Hands[0].Cards), 3)
	expectEqual(t, "flags", p.Hands[0].flags()&FLAG_DOUBLED, FLAG_DOUBLED)
	finishHand(t, state)
	expectEqual(t, "purse", p.Purse, 1040)
	expectEqual(t, "move", p.Move, "+40")
}

func TestDoubleLoss(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "5S", "9H", "4D", "8C", "2S")
	bet(t, state, "B2")
	move(t, state, "p1", MOVE_DOUBLE)
	finishHand(t, state)
	expectEqual(t, "purse", player(t, state, "p1").Purse, 960)
}

func TestSplitThenDoubleAfterSplit(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "8S", "6H", "8D", "TC", "3C", "TD", "9H", "7S")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_SPLIT)

	p := player(t, state, "p1")
	expectEqual(t, "hands", len(p.Hands), 2)
	expectEqual(t, "hand 0", cardsToString(p.Hands[0].Cards), "8S3C")
	expectEqual(t, "hand 1", cardsToString(p.Hands[1].Cards), "8DTD")
	expectEqual(t, "purse after split", p.Purse, 980)
	expectEqual(t, "active hand", state.ActiveHand, 0)
	if slices.Contains(validCodes(state), MOVE_SPLIT) || slices.Contains(validCodes(state), MOVE_SURRENDER) {
		t.Errorf("split hand offered %v", validCodes(state))
	}

	move(t, state, "p1", MOVE_DOUBLE)
	expectEqual(t, "active hand after double", state.ActiveHand, 1)
	move(t, state, "p1", MOVE_STAND)
	finishHand(t, state)

	expectEqual(t, "hand 0 result", p.Hands[0].Result, RESULT_WIN)
	expectEqual(t, "hand 1 result", p.Hands[1].Result, RESULT_WIN)
	expectEqual(t, "purse", p.Purse, 1030)
	expectEqual(t, "move", p.Move, "+30")
}

func TestSplitAcesOneCardEachAndNoBlackjack(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "AS", "9H", "AD", "7C", "KD", "5S", "TS")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_SPLIT)

	p := player(t, state, "p1")
	expectEqual(t, "round after split aces", state.Round, ROUND_DEALER)
	expectEqual(t, "hand 0 cards", len(p.Hands[0].Cards), 2)
	expectEqual(t, "hand 1 cards", len(p.Hands[1].Cards), 2)
	expectEqual(t, "hand 0 flags", p.Hands[0].flags()&FLAG_BLACKJACK, 0)
	finishHand(t, state)

	expectEqual(t, "hand 0 result", p.Hands[0].Result, RESULT_WIN)
	expectEqual(t, "hand 1 result", p.Hands[1].Result, RESULT_WIN)
	expectEqual(t, "purse", p.Purse, 1020)
}

func TestSplitTenValuesTwentyOneIsNotBlackjack(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "KS", "9H", "QD", "8C", "AS", "2D", "TC")
	bet(t, state, "B1")
	if !slices.Contains(validCodes(state), MOVE_SPLIT) {
		t.Fatalf("king and queen should split, valid %v", validCodes(state))
	}
	move(t, state, "p1", MOVE_SPLIT)
	p := player(t, state, "p1")
	expectEqual(t, "hand 0 done at 21", p.Hands[0].done, true)
	expectEqual(t, "active hand", state.ActiveHand, 1)
	move(t, state, "p1", MOVE_HIT) // 12 + 10
	finishHand(t, state)
	expectEqual(t, "hand 0 result", p.Hands[0].Result, RESULT_WIN)
	expectEqual(t, "hand 1 result", p.Hands[1].Result, RESULT_LOSE)
	expectEqual(t, "purse", p.Purse, 1000)
}

func TestSplitOneHandBusts(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "8S", "TC", "8D", "7H", "TS", "6D", "KD")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_SPLIT)
	move(t, state, "p1", MOVE_STAND)
	move(t, state, "p1", MOVE_HIT)
	finishHand(t, state)
	p := player(t, state, "p1")
	expectEqual(t, "hand 0", p.Hands[0].Result, RESULT_WIN)
	expectEqual(t, "hand 1", p.Hands[1].Result, RESULT_LOSE)
	expectEqual(t, "move", p.Move, "push")
	expectEqual(t, "purse", p.Purse, 1000)
}

func TestNoResplit(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "8S", "TC", "8D", "7H", "8H", "8C")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_SPLIT)
	if slices.Contains(validCodes(state), MOVE_SPLIT) {
		t.Errorf("resplit offered: %v", validCodes(state))
	}
}

func TestSurrender(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "TS", "TH", "6D", "8C")
	bet(t, state, "B2")
	move(t, state, "p1", MOVE_SURRENDER)
	finishHand(t, state)
	p := player(t, state, "p1")
	expectEqual(t, "purse", p.Purse, 990)
	expectEqual(t, "result", p.Hands[0].Result, RESULT_SURRENDER)
	expectEqual(t, "move", p.Move, "-10")
	expectEqual(t, "dealer cards", len(state.Dealer), 2)
}

func TestMoveAvailability(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "5S", "TH", "6D", "8C", "2C")
	bet(t, state, "B1")
	expectEqual(t, "two card moves", slices.Equal(validCodes(state), []string{MOVE_HIT, MOVE_STAND, MOVE_DOUBLE, MOVE_SURRENDER}), true)
	move(t, state, "p1", MOVE_HIT)
	expectEqual(t, "after a hit", slices.Equal(validCodes(state), []string{MOVE_HIT, MOVE_STAND}), true)

	// Short of chips to match the bet: no double or split
	state = newTable(t, 1, 0)
	player(t, state, "p1").Purse = 150
	rig(t, state, "8S", "TH", "8D", "7C")
	bet(t, state, "B4")
	expectEqual(t, "short stack", slices.Equal(validCodes(state), []string{MOVE_HIT, MOVE_STAND, MOVE_SURRENDER}), true)
}

func TestHitToTwentyOneStandsAutomatically(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "9S", "TH", "2D", "7C", "KC")
	bet(t, state, "B1")
	move(t, state, "p1", MOVE_HIT)
	expectEqual(t, "round", state.Round, ROUND_DEALER)
}

func TestEightCardHandStands(t *testing.T) {
	state := newTable(t, 1, 0)
	rig(t, state, "2C", "TS", "2D", "7C", "2H", "2S", "3C", "3D", "AC", "AD")
	bet(t, state, "B1")
	for i := 0; i < 6; i++ {
		move(t, state, "p1", MOVE_HIT)
	}
	p := player(t, state, "p1")
	expectEqual(t, "cards", len(p.Hands[0].Cards), MAX_HAND_CARDS)
	expectEqual(t, "done", p.Hands[0].done, true)
	expectEqual(t, "round", state.Round, ROUND_DEALER)
}

func TestSeveralPlayersSettleIndependently(t *testing.T) {
	state := newTable(t, 3, 0)
	// p1 20, p2 blackjack, p3 13 hits to bust; dealer 18
	rig(t, state, "KS", "AS", "9S", "9H", "QD", "KH", "4C", "9C", "TD")
	bet(t, state, "B1", "B2", "B3")
	move(t, state, "p1", MOVE_STAND)
	move(t, state, "p3", MOVE_HIT)
	finishHand(t, state)
	expectEqual(t, "p1", player(t, state, "p1").Purse, 1010)
	expectEqual(t, "p2", player(t, state, "p2").Purse, 1030)
	expectEqual(t, "p3", player(t, state, "p3").Purse, 950)
	expectEqual(t, "house", state.houseNet, 10)
}
