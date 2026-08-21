package main

import (
	"strings"
	"testing"
	"time"
)

func TestGameStartsWithTwoPlayers(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 2, 3, 4, 6, 6)

	state := startGame(t, table, players)

	if state.Round != 1 {
		t.Fatalf("Round = %d, want 1", state.Round)
	}
	if state.ActivePlayer != 0 {
		t.Fatalf("ActivePlayer = %d, want 0", state.ActivePlayer)
	}
	if state.Dice != "123466" {
		t.Fatalf("Dice = %q, want %q", state.Dice, "123466")
	}
	if state.TurnScore != 0 {
		t.Fatalf("TurnScore = %d, want 0", state.TurnScore)
	}
	if state.fujirkled {
		t.Fatal("did not expect a fujirkle")
	}
}

func TestBankScoresAndPassesTurn(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4)
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)

	callBank(players[0], "111000")

	state := rawState(table)
	if state.Players[0].Score != 1000 {
		t.Fatalf("banked score = %d, want 1000", state.Players[0].Score)
	}
	if state.ActivePlayer != 1 {
		t.Fatalf("ActivePlayer = %d, want 1", state.ActivePlayer)
	}
	if state.TurnScore != 0 {
		t.Fatalf("TurnScore = %d, want 0 at start of new turn", state.TurnScore)
	}
}

func TestFujirkleLosesTurnScore(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4) // opening roll
	queueRoll(2, 3, 4, 6, 2)    // five dice, nothing scores
	queueRoll(1, 2, 3, 4, 6, 6) // next player's opening roll

	startGame(t, table, players)

	// Keep a single 1 for 100, then push and fujirkle
	callRoll(players[0], "100000")

	state := rawState(table)
	if !state.fujirkled {
		t.Fatal("expected a fujirkle")
	}
	if state.TurnScore != 0 {
		t.Fatalf("TurnScore = %d, want 0 after fujirkle", state.TurnScore)
	}
	if state.Prompt != PROMPT_FUJIRKLE {
		t.Fatalf("Prompt = %q, want %q", state.Prompt, PROMPT_FUJIRKLE)
	}

	// The fujirkled roll lingers until the timer expires, then play moves on
	c(players[0], apiState)

	state = rawState(table)
	if state.Players[0].Score != 0 {
		t.Fatalf("score = %d, want 0 - a fujirkle banks nothing", state.Players[0].Score)
	}
	if state.ActivePlayer != 1 {
		t.Fatalf("ActivePlayer = %d, want 1", state.ActivePlayer)
	}
}

func TestHotDiceRefillsPool(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 2, 3, 4, 5, 6) // straight, 1500, uses every die
	queueRoll(2, 2, 2, 3, 3, 3) // the fresh six

	startGame(t, table, players)

	callRoll(players[0], "111111")

	state := rawState(table)
	if state.TurnScore != 1500 {
		t.Fatalf("TurnScore = %d, want 1500", state.TurnScore)
	}
	if len(state.Dice) != NUM_DICE {
		t.Fatalf("pool = %q, want six fresh dice", state.Dice)
	}
	if state.Dice != "222333" {
		t.Fatalf("Dice = %q, want %q", state.Dice, "222333")
	}
	if state.Prompt != PROMPT_HOT_DICE {
		t.Fatalf("Prompt = %q, want %q", state.Prompt, PROMPT_HOT_DICE)
	}
	if state.ActivePlayer != 0 {
		t.Fatal("hot dice should not end the turn")
	}
}

func TestInvalidSelectionRejected(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)

	// The lone 2 is not a scoring die
	callRoll(players[0], "010000")

	state := rawState(table)
	if state.Dice != "123466" {
		t.Fatalf("Dice = %q, want the roll to be untouched", state.Dice)
	}
	if state.TurnScore != 0 {
		t.Fatalf("TurnScore = %d, want 0", state.TurnScore)
	}
	if state.ActivePlayer != 0 {
		t.Fatal("an illegal move must not pass the turn")
	}

	// A mask of the wrong length is rejected too
	callBank(players[0], "111")

	state = rawState(table)
	if state.Players[0].Score != 0 {
		t.Fatalf("score = %d, want 0", state.Players[0].Score)
	}
}

func TestSelectableMaskMarksDeadDice(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)

	state := c(players[0], apiState).(*GameState)

	if state.Selectable != "100000" {
		t.Fatalf("Selectable = %q, want %q", state.Selectable, "100000")
	}
	if state.ValidMoves != MOVE_ROLL|MOVE_BANK {
		t.Fatalf("ValidMoves = %d, want roll|bank", state.ValidMoves)
	}
}

func TestReachingTargetTriggersFinalRoundThenEnds(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4) // p1 opening
	queueRoll(1, 1, 1, 2, 3, 4) // p2 opening
	queueRoll(1, 2, 3, 4, 6, 6)

	startGame(t, table, players)

	rawState(table).Players[0].Score = TARGET_SCORE - 500

	callBank(players[0], "111000")

	state := rawState(table)
	if !state.finalRound {
		t.Fatal("crossing the target should start the final round")
	}
	if state.gameOver {
		t.Fatal("the other player still owes a turn")
	}
	if state.ActivePlayer != 1 {
		t.Fatalf("ActivePlayer = %d, want 1", state.ActivePlayer)
	}

	// p2 takes their last turn, which returns play to p1 and ends the game
	callBank(players[1], "111000")

	state = rawState(table)
	if !state.gameOver {
		t.Fatal("expected the game to be over")
	}
	if state.Round != ROUND_GAMEOVER {
		t.Fatalf("Round = %d, want %d", state.Round, ROUND_GAMEOVER)
	}
	if !strings.Contains(state.Prompt, "won with a score of") {
		t.Fatalf("Prompt = %q, want a winner announcement", state.Prompt)
	}
}

func TestOnlyActivePlayerCanMove(t *testing.T) {
	table, players := createTestTable(0, 2)
	queueRoll(1, 1, 1, 2, 3, 4)

	startGame(t, table, players)

	// p2 is not the active player
	callBank(players[1], "111000")

	state := rawState(table)
	if state.Players[1].Score != 0 {
		t.Fatalf("p2 score = %d, want 0", state.Players[1].Score)
	}
	if state.ActivePlayer != 0 {
		t.Fatal("turn should still belong to p1")
	}
}

func TestReadyToggle(t *testing.T) {
	table, players := createTestTable(2, 1)

	// Every wait timer has to be raised, otherwise the "everyone is ready"
	// shortcut collapses the countdown to zero and the game starts at once
	START_WAIT_TIME = time.Second * 10
	START_WAIT_TIME_ONE_PLAYER = time.Second * 10
	START_WAIT_TIME_ALL_READY = time.Second * 10

	p1 := players[0]

	c(p1, apiState)
	c(p1, apiReady)

	state := c(p1, apiState).(*GameState)
	if !strings.HasPrefix(state.Prompt, PROMPT_STARTING_IN) {
		t.Fatalf("Prompt = %q, want a countdown", state.Prompt)
	}

	c(p1, apiReady)

	state = c(p1, apiState).(*GameState)
	if state.Prompt != PROMPT_WAITING_ON_READY {
		t.Fatalf("Prompt = %q, want %q", state.Prompt, PROMPT_WAITING_ON_READY)
	}

	if rawState(table).Round != ROUND_LOBBY {
		t.Fatal("game should still be in the lobby")
	}
}

func TestBotPlaysCompleteGame(t *testing.T) {
	table, players := createTestTable(3, 1)
	clearForcedRolls()

	p1 := players[0]

	c(p1, apiState)
	c(p1, apiReady)

	// Bots move as soon as their (zeroed) timer expires, so repeatedly
	// stepping the state runs whole turns without any human input
	for i := 0; i < 20000; i++ {
		state := rawState(table)
		if state.gameOver {
			break
		}

		if state.Round > ROUND_LOBBY && state.ActivePlayer >= 0 &&
			!state.Players[state.ActivePlayer].isBot && !state.fujirkled {
			dice := parseDice(state.Dice)
			mask, score := bestSelection(dice)
			if score > 0 {
				callBank(p1, maskToString(mask))
				continue
			}
		}

		c(p1, apiState)
	}

	state := rawState(table)
	if !state.gameOver {
		t.Fatal("expected the bots to finish a game")
	}

	winner := 0
	for _, player := range state.Players {
		if player.Score > winner {
			winner = player.Score
		}
	}
	if winner < TARGET_SCORE {
		t.Fatalf("winning score = %d, want at least %d", winner, TARGET_SCORE)
	}
}

// The final round is not "one more round" - everyone else owes exactly one more
// turn and the game ends when play would return to whoever crossed the target.
// Triggered mid list here, so the count wraps a round boundary, as it does when
// a bot crosses the line.
func TestFinalRoundTriggeredMidListEndsOnReturnToTrigger(t *testing.T) {
	table, players := createTestTable(0, 5)

	// Installs the deterministic roller before the opening roll. Anything past
	// the queue rolls all 1s, so every turn opens with six 1s and a "111000"
	// keep banks exactly 1000.
	queueRoll(1, 1, 1, 1, 1, 1)
	startGame(t, table, players)

	// Walk play round to player 3 (index 2), who then crosses the target
	for i := 0; i < 2; i++ {
		callBank(players[i], "111000")
	}

	state := rawState(table)
	if state.ActivePlayer != 2 {
		t.Fatalf("ActivePlayer = %d, want 2", state.ActivePlayer)
	}

	state.Players[2].Score = TARGET_SCORE - 1000
	callBank(players[2], "111000")

	state = rawState(table)
	if !state.finalRound {
		t.Fatal("crossing the target should start the final round")
	}
	if state.gameOver {
		t.Fatal("four players still owe a turn - the game must not end here")
	}

	// Everyone else takes exactly one more turn: 3, 4, then a wrap to 0, 1
	for _, i := range []int{3, 4, 0, 1} {
		if rawState(table).ActivePlayer != i {
			t.Fatalf("ActivePlayer = %d, want %d", rawState(table).ActivePlayer, i)
		}
		if rawState(table).gameOver {
			t.Fatalf("game ended early, before player %d had their last turn", i)
		}
		callBank(players[i], "111000")
	}

	state = rawState(table)
	if !state.gameOver {
		t.Fatal("play returned to the trigger, so the game should be over")
	}
	if state.Round != ROUND_GAMEOVER {
		t.Fatalf("Round = %d, want %d", state.Round, ROUND_GAMEOVER)
	}
	if !strings.Contains(state.Prompt, "won with a score of") {
		t.Fatalf("Prompt = %q, want a winner announcement", state.Prompt)
	}
}
