package main

import "slices"

// Basic strategy for 6 decks, dealer stands on soft 17, double after split,
// late surrender
func basicStrategy(h hand, up card, canDouble, canSplit, canSurrender bool) string {
	total, soft := handValue(h.Cards)
	u := cardValue(up)
	pair := len(h.Cards) == 2 && cardValue(h.Cards[0]) == cardValue(h.Cards[1])

	if canSurrender && !soft && !pair && (total == 16 && u >= 9 || total == 15 && u == 10) {
		return MOVE_SURRENDER
	}

	if canSplit && pair {
		switch cardValue(h.Cards[0]) {
		case 11, 8:
			return MOVE_SPLIT
		case 9:
			if u != 7 && u < 10 {
				return MOVE_SPLIT
			}
		case 7, 3, 2:
			if u <= 7 {
				return MOVE_SPLIT
			}
		case 6:
			if u <= 6 {
				return MOVE_SPLIT
			}
		case 4:
			if u == 5 || u == 6 {
				return MOVE_SPLIT
			}
		}
	}

	double := func(otherwise string) string {
		if canDouble {
			return MOVE_DOUBLE
		}
		return otherwise
	}

	if soft {
		switch {
		case total >= 19:
			return MOVE_STAND
		case total == 18:
			if u >= 3 && u <= 6 {
				return double(MOVE_STAND)
			}
			if u <= 8 {
				return MOVE_STAND
			}
		case total == 17:
			if u >= 3 && u <= 6 {
				return double(MOVE_HIT)
			}
		case total >= 15:
			if u >= 4 && u <= 6 {
				return double(MOVE_HIT)
			}
		default:
			if u == 5 || u == 6 {
				return double(MOVE_HIT)
			}
		}
		return MOVE_HIT
	}

	switch {
	case total >= 17:
		return MOVE_STAND
	case total >= 13:
		if u <= 6 {
			return MOVE_STAND
		}
	case total == 12:
		if u >= 4 && u <= 6 {
			return MOVE_STAND
		}
	case total == 11:
		if u != 11 {
			return double(MOVE_HIT)
		}
	case total == 10:
		if u <= 9 {
			return double(MOVE_HIT)
		}
	case total == 9:
		if u >= 3 && u <= 6 {
			return double(MOVE_HIT)
		}
	}
	return MOVE_HIT
}

func (state *GameState) botMove() string {
	moves := state.getValidMoves()
	can := func(m string) bool {
		return slices.ContainsFunc(moves, func(v validMove) bool { return v.Move == m })
	}
	h := state.Players[state.ActivePlayer].Hands[state.ActiveHand]
	return basicStrategy(h, state.Dealer[0], can(MOVE_DOUBLE), can(MOVE_SPLIT), can(MOVE_SURRENDER))
}

func (state *GameState) botBet() int {
	p := state.Players[state.ActivePlayer]
	if state.fixedBotBet > 0 {
		return min(state.fixedBotBet, p.Purse)
	}
	if p.lastBet > 0 && p.lastBet <= p.Purse && state.rng.Intn(2) == 0 {
		return p.lastBet
	}
	affordable := 0
	for _, b := range BET_SIZES[:3] {
		if b <= p.Purse {
			affordable++
		}
	}
	return BET_SIZES[state.rng.Intn(affordable)]
}
