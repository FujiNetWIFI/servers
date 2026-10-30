package main

import "testing"

func TestBasicStrategy(t *testing.T) {
	cases := []struct {
		hand string
		up   string
		want string
	}{
		{"TS 6D", "TH", MOVE_SURRENDER},
		{"TS 6D", "AH", MOVE_SURRENDER},
		{"TS 6D", "8H", MOVE_HIT},
		{"TS 5D", "TH", MOVE_SURRENDER},
		{"TS 5D", "9H", MOVE_HIT},
		{"8S 8D", "TH", MOVE_SPLIT},
		{"AS AD", "AH", MOVE_SPLIT},
		{"9S 9D", "7H", MOVE_STAND},
		{"9S 9D", "8H", MOVE_SPLIT},
		{"TS KD", "6H", MOVE_STAND},
		{"5S 5D", "9H", MOVE_DOUBLE},
		{"4S 4D", "5H", MOVE_SPLIT},
		{"4S 4D", "4H", MOVE_HIT},
		{"AS 7D", "2H", MOVE_STAND},
		{"AS 7D", "6H", MOVE_DOUBLE},
		{"AS 7D", "9H", MOVE_HIT},
		{"AS 6D", "3H", MOVE_DOUBLE},
		{"AS 2D", "4H", MOVE_HIT},
		{"AS 2D", "5H", MOVE_DOUBLE},
		{"AS 8D", "6H", MOVE_STAND},
		{"6S 5D", "AH", MOVE_HIT},
		{"6S 5D", "TH", MOVE_DOUBLE},
		{"5S 4D", "2H", MOVE_HIT},
		{"5S 4D", "3H", MOVE_DOUBLE},
		{"TS 2D", "3H", MOVE_HIT},
		{"TS 2D", "4H", MOVE_STAND},
		{"TS 3D", "2H", MOVE_STAND},
		{"TS 3D", "7H", MOVE_HIT},
	}
	for _, c := range cases {
		state := &GameState{}
		rig(t, state, append(splitCards(c.hand), c.up)...)
		h := hand{Cards: state.testDeck[:2]}
		got := basicStrategy(h, state.testDeck[2], true, true, true)
		if got != c.want {
			t.Errorf("%s vs %s: got %s, want %s", c.hand, c.up, got, c.want)
		}
	}

	// Doubling or surrendering not allowed: fall back
	state := &GameState{}
	rig(t, state, "AS", "7D", "6H", "TS", "6D", "TH")
	expectEqual(t, "soft 18 no double", basicStrategy(hand{Cards: state.testDeck[:2]}, state.testDeck[2], false, false, false), MOVE_STAND)
	expectEqual(t, "16 no surrender", basicStrategy(hand{Cards: state.testDeck[3:5]}, state.testDeck[5], false, false, false), MOVE_HIT)
}

func splitCards(s string) []string {
	return []string{s[:2], s[3:]}
}
