package main

import "testing"

// Every claim the client's help screens make, asserted against the real rules.
// If a rule changes, this fails and the help text has to be updated with it.
//
// The help screens say:
//   roll six dice and set aside at least one die that scores
//   roll on to build up your turn score, or bank it to keep it for good
//   roll nothing that scores and you lose everything you held that turn
//   set aside all six and you earn hot dice - roll all six again
//   first to 10000 starts the final round
//   each 1 - 100        each 5 - 50
//   three of a kind - face value x 100, except three 1s which score 1000
//   four of a kind  - double the three
//   five of a kind  - x4      six of a kind - x8
//   straight 1 to 6 - 1500    three pairs   - 1500

func TestHelpTextSingleDice(t *testing.T) {
	if s, _ := scoreSelection([]int{1}); s != 100 {
		t.Fatalf("help says each 1 scores 100, got %d", s)
	}
	if s, _ := scoreSelection([]int{5}); s != 50 {
		t.Fatalf("help says each 5 scores 50, got %d", s)
	}
	// "each" - they stack
	if s, _ := scoreSelection([]int{1, 1}); s != 200 {
		t.Fatalf("two 1s should be 200, got %d", s)
	}
	if s, _ := scoreSelection([]int{5, 5}); s != 100 {
		t.Fatalf("two 5s should be 100, got %d", s)
	}
}

func TestHelpTextThreeOfAKind(t *testing.T) {
	// face value x 100
	for face := 2; face <= 6; face++ {
		want := face * 100
		got, _ := scoreSelection([]int{face, face, face})
		if got != want {
			t.Fatalf("three %ds: help says %d, got %d", face, want, got)
		}
	}
	// except three 1s
	if s, _ := scoreSelection([]int{1, 1, 1}); s != 1000 {
		t.Fatalf("help says three 1s score 1000, got %d", s)
	}
}

func TestHelpTextFourFiveSixOfAKind(t *testing.T) {
	for face := 1; face <= 6; face++ {
		three, _ := scoreSelection([]int{face, face, face})

		four, _ := scoreSelection([]int{face, face, face, face})
		if four != three*2 {
			t.Fatalf("four %ds: help says double the three (%d), got %d", face, three*2, four)
		}

		five, _ := scoreSelection([]int{face, face, face, face, face})
		if five != three*4 {
			t.Fatalf("five %ds: help says x4 (%d), got %d", face, three*4, five)
		}

		six, _ := scoreSelection([]int{face, face, face, face, face, face})
		if six != three*8 {
			t.Fatalf("six %ds: help says x8 (%d), got %d", face, three*8, six)
		}
	}
}

func TestHelpTextSpecials(t *testing.T) {
	if s, _ := scoreSelection([]int{1, 2, 3, 4, 5, 6}); s != 1500 {
		t.Fatalf("help says a straight 1 to 6 scores 1500, got %d", s)
	}
	if s, _ := scoreSelection([]int{2, 2, 3, 3, 4, 4}); s != 1500 {
		t.Fatalf("help says three pairs score 1500, got %d", s)
	}
}

func TestHelpTextTargetIs10000(t *testing.T) {
	if TARGET_SCORE != 10000 {
		t.Fatalf("help says first to 10000, but TARGET_SCORE is %d", TARGET_SCORE)
	}
}

// "set aside at least one die that scores" - a selection with any non scoring
// die in it must be refused outright
func TestHelpTextMustSetAsideOnlyScoringDice(t *testing.T) {
	if _, ok := scoreSelection([]int{2}); ok {
		t.Fatal("a lone 2 is not a scoring die, but was accepted")
	}
	if _, ok := scoreSelection([]int{1, 2}); ok {
		t.Fatal("keeping a 1 and a stray 2 should be refused")
	}
	if _, ok := scoreSelection([]int{}); ok {
		t.Fatal("keeping nothing should be refused")
	}
}

// "set aside any that score - at least one each roll". You are not forced to
// take every scoring die: keeping fewer leaves more dice in the pool to
// re-roll, which is a real choice, so a scoring subset must be legal.
func TestHelpTextMayKeepAScoringSubset(t *testing.T) {
	// One 1 out of three
	if s, ok := scoreSelection([]int{1}); !ok || s != 100 {
		t.Fatalf("keeping a single 1 out of several must be legal, got %d ok=%v", s, ok)
	}
	// Two 1s out of three
	if s, ok := scoreSelection([]int{1, 1}); !ok || s != 200 {
		t.Fatalf("keeping two 1s must be legal, got %d ok=%v", s, ok)
	}
	// A 5 while leaving 1s on the table
	if s, ok := scoreSelection([]int{5}); !ok || s != 50 {
		t.Fatalf("keeping just a 5 must be legal, got %d ok=%v", s, ok)
	}

	// And the server must actually accept the partial mask over the wire
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4)
	queueRoll(1, 1, 1, 1, 1)
	startGame(t, table, players)

	callRoll(players[0], "100000") // keep only the first of three 1s

	state := rawState(table)
	if state.TurnScore != 100 {
		t.Fatalf("keeping one 1 should score 100, got %d", state.TurnScore)
	}
	if len(state.Dice) != 5 {
		t.Fatalf("the other five dice should have been re-rolled, got %d", len(state.Dice))
	}
}

// "roll nothing that scores and you lose everything you held that turn"
func TestHelpTextNoScoreLosesTheTurnScore(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4) // opening roll
	queueRoll(2, 3, 4, 6, 2)    // five dice, nothing scores
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)

	// Bank 100 worth of dice into the turn score, then push and lose it
	callRoll(players[0], "100000")

	state := rawState(table)
	if !state.fujirkled {
		t.Fatal("a roll with no scoring dice should end the turn")
	}
	if state.TurnScore != 0 {
		t.Fatalf("help says you lose everything held that turn, but TurnScore is %d", state.TurnScore)
	}

	c(players[0], apiState)
	if rawState(table).Players[0].Score != 0 {
		t.Fatal("nothing should have been banked")
	}
}

// "set aside all six and you earn hot dice - roll all six again"
func TestHelpTextHotDiceGivesSixFreshDice(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 2, 3, 4, 5, 6) // a straight uses all six
	queueRoll(2, 2, 2, 3, 3, 3)

	startGame(t, table, players)

	callRoll(players[0], "111111")

	state := rawState(table)
	if len(state.Dice) != NUM_DICE {
		t.Fatalf("help says you roll all six again, got %d dice", len(state.Dice))
	}
	if state.TurnScore != 1500 {
		t.Fatalf("hot dice should keep the turn score, got %d", state.TurnScore)
	}
	if state.ActivePlayer != 0 {
		t.Fatal("hot dice should not end the turn")
	}
}

// "first to 10000 starts the final round" - and everyone else gets one last turn
func TestHelpTextFinalRoundGivesEveryoneOneLastTurn(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4)
	queueRoll(1, 1, 1, 2, 3, 4)
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)
	rawState(table).Players[0].Score = TARGET_SCORE - 1000

	callBank(players[0], "111000")

	state := rawState(table)
	if state.Players[0].Score < TARGET_SCORE {
		t.Fatalf("expected to cross the target, got %d", state.Players[0].Score)
	}
	if !state.finalRound {
		t.Fatal("crossing 10000 should start the final round")
	}
	if state.gameOver {
		t.Fatal("the other player is still owed a turn")
	}

	callBank(players[1], "111000")

	if !rawState(table).gameOver {
		t.Fatal("the game should end once play returns to the player who crossed")
	}
}
