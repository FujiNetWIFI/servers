package main

import "log"

const ACE = 14

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}

func cardValue(c card) int {
	switch {
	case c.Rank == ACE:
		return 11
	case c.Rank > 10:
		return 10
	}
	return c.Rank
}

func handValue(cards []card) (total int, soft bool) {
	aces := 0
	for _, c := range cards {
		if c.Rank == ACE {
			aces++
			total++
		} else {
			total += cardValue(c)
		}
	}
	if aces > 0 && total+10 <= 21 {
		return total + 10, true
	}
	return total, false
}

func isBlackjack(cards []card, split bool) bool {
	total, _ := handValue(cards)
	return !split && len(cards) == 2 && total == 21
}

func cardsToString(cards []card) string {
	s := ""
	for _, c := range cards {
		s += valueLookup[c.Rank] + suitLookup[c.Suit]
	}
	return s
}

func (state *GameState) newShoe() {
	state.shoeIndex = 0
	if state.testDeck != nil {
		state.shoe = state.testDeck
		state.testDeck = nil
		return
	}
	state.shoe = make([]card, 0, DECKS*52)
	for d := 0; d < DECKS; d++ {
		for suit := 0; suit < 4; suit++ {
			for rank := 2; rank <= ACE; rank++ {
				state.shoe = append(state.shoe, card{Rank: rank, Suit: suit})
			}
		}
	}
	state.rng.Shuffle(len(state.shoe), func(i, j int) { state.shoe[i], state.shoe[j] = state.shoe[j], state.shoe[i] })
	log.Printf("TABLE %s: new shoe", state.TableId)
}

func (state *GameState) draw() card {
	if state.shoeIndex >= len(state.shoe) {
		state.newShoe()
	}
	c := state.shoe[state.shoeIndex]
	state.shoeIndex++
	return c
}

func (h *hand) flags() int {
	f := 0
	total, soft := handValue(h.Cards)
	if soft {
		f |= FLAG_SOFT
	}
	if isBlackjack(h.Cards, h.split) {
		f |= FLAG_BLACKJACK
	}
	if total > 21 {
		f |= FLAG_BUST
	}
	if h.doubled {
		f |= FLAG_DOUBLED
	}
	if h.surrendered {
		f |= FLAG_SURRENDERED
	}
	if h.done {
		f |= FLAG_DONE
	}
	return f
}
