package main

// Fujirkle scoring - standard rules with hot dice.
//
//   1s ............ 100 each     3 of a kind ..... face x 100 (three 1s = 1000)
//   5s ............. 50 each     4/5/6 of a kind . 3-kind x2 / x4 / x8
//   Straight 1-6 .. 1500         Three pairs ..... 1500

const (
	NUM_DICE     = 6
	TARGET_SCORE = 10000

	SCORE_STRAIGHT    = 1500
	SCORE_THREE_PAIRS = 1500
	SCORE_SINGLE_ONE  = 100
	SCORE_SINGLE_FIVE = 50
)

type diceCount [7]int

func countDice(dice []int) diceCount {
	var c diceCount
	for _, d := range dice {
		if d >= 1 && d <= 6 {
			c[d]++
		}
	}
	return c
}

func parseDice(s string) []int {
	dice := make([]int, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '1' && s[i] <= '6' {
			dice = append(dice, int(s[i]-'0'))
		}
	}
	return dice
}

func diceToString(dice []int) string {
	buf := make([]byte, len(dice))
	for i, d := range dice {
		buf[i] = byte('0' + d)
	}
	return string(buf)
}

func nOfAKindScore(face int, n int) int {
	if n < 3 {
		return 0
	}
	base := face * 100
	if face == 1 {
		base = 1000
	}
	switch n {
	case 3:
		return base
	case 4:
		return base * 2
	case 5:
		return base * 4
	case 6:
		return base * 8
	}
	return 0
}

func isStraight(c diceCount) bool {
	for f := 1; f <= 6; f++ {
		if c[f] != 1 {
			return false
		}
	}
	return true
}

func isThreePairs(c diceCount) bool {
	pairs := 0
	for f := 1; f <= 6; f++ {
		if c[f] == 2 {
			pairs++
		} else if c[f] != 0 {
			return false
		}
	}
	return pairs == 3
}

// scoreSelection returns the highest score obtainable using *every* supplied
// die. ok is false when at least one die cannot be part of a scoring combo,
// which is what makes an attempted set-aside illegal.
func scoreSelection(dice []int) (int, bool) {
	if len(dice) == 0 {
		return 0, false
	}
	c := countDice(dice)
	score := bestScoreAll(&c, len(dice))
	if score < 0 {
		return 0, false
	}
	return score, true
}

// Exhaustive search over every legal decomposition; -1 means the remaining
// dice cannot all be scored. Six dice keeps the search space trivial.
func bestScoreAll(c *diceCount, remaining int) int {
	if remaining == 0 {
		return 0
	}

	best := -1

	if remaining == NUM_DICE {
		if isStraight(*c) {
			return SCORE_STRAIGHT
		}
		if isThreePairs(*c) {
			best = SCORE_THREE_PAIRS
		}
	}

	for f := 1; f <= 6; f++ {
		for n := 3; n <= c[f]; n++ {
			c[f] -= n
			if sub := bestScoreAll(c, remaining-n); sub >= 0 {
				if v := nOfAKindScore(f, n) + sub; v > best {
					best = v
				}
			}
			c[f] += n
		}
	}

	if c[1] > 0 {
		c[1]--
		if sub := bestScoreAll(c, remaining-1); sub >= 0 && SCORE_SINGLE_ONE+sub > best {
			best = SCORE_SINGLE_ONE + sub
		}
		c[1]++
	}

	if c[5] > 0 {
		c[5]--
		if sub := bestScoreAll(c, remaining-1); sub >= 0 && SCORE_SINGLE_FIVE+sub > best {
			best = SCORE_SINGLE_FIVE + sub
		}
		c[5]++
	}

	return best
}

// rollHasScore reports whether any subset of the roll scores. When false the
// player has fujirkled and forfeits the turn score.
func rollHasScore(dice []int) bool {
	c := countDice(dice)
	if c[1] > 0 || c[5] > 0 {
		return true
	}
	for f := 1; f <= 6; f++ {
		if c[f] >= 3 {
			return true
		}
	}
	return isThreePairs(c)
}

// bestSelection returns the highest scoring subset of the roll. Ties are
// broken toward using fewer dice, leaving more in the pool to re-roll.
func bestSelection(dice []int) ([]bool, int) {
	n := len(dice)
	mask := make([]bool, n)
	bestScore := 0
	bestUsed := 0

	for m := 1; m < (1 << n); m++ {
		sub := make([]int, 0, n)
		for i := 0; i < n; i++ {
			if m&(1<<i) != 0 {
				sub = append(sub, dice[i])
			}
		}
		score, ok := scoreSelection(sub)
		if !ok {
			continue
		}
		if score > bestScore || (score == bestScore && len(sub) < bestUsed) {
			bestScore = score
			bestUsed = len(sub)
			for i := 0; i < n; i++ {
				mask[i] = m&(1<<i) != 0
			}
		}
	}

	return mask, bestScore
}

// selectableMask marks every die that can take part in some scoring
// combination, so a client can grey out the rest without knowing the rules.
// The marked dice are always a legal selection in their own right, which lets a
// client offer a single "keep everything that scores" action.
func selectableMask(dice []int) string {
	n := len(dice)
	if n == 0 {
		return ""
	}

	selectable := make([]bool, n)

	for m := 1; m < (1 << n); m++ {
		sub := make([]int, 0, n)
		for i := 0; i < n; i++ {
			if m&(1<<i) != 0 {
				sub = append(sub, dice[i])
			}
		}
		if _, ok := scoreSelection(sub); !ok {
			continue
		}
		for i := 0; i < n; i++ {
			if m&(1<<i) != 0 {
				selectable[i] = true
			}
		}
	}

	buf := make([]byte, n)
	for i, k := range selectable {
		if k {
			buf[i] = '1'
		} else {
			buf[i] = '0'
		}
	}
	return string(buf)
}

// applyKeepMask splits a roll by a client supplied mask. Unlike Fujitzee -
// where '1' means re-roll this die - a Fujirkle set-aside is permanent, so here
// '1' means keep/score this die and '0' leaves it in the pool.
func applyKeepMask(dice []int, mask string) (kept []int, pool []int, ok bool) {
	if len(mask) != len(dice) {
		return nil, nil, false
	}
	for i, d := range dice {
		switch mask[i] {
		case '1':
			kept = append(kept, d)
		case '0':
			pool = append(pool, d)
		default:
			return nil, nil, false
		}
	}
	return kept, pool, true
}
