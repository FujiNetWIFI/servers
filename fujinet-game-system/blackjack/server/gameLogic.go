package main

import (
	"fmt"
	"log"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"

	hashstructure "github.com/mitchellh/hashstructure/v2"
)

var initServerOnce sync.Once

func initializeGameServer() {
	initServerOnce.Do(func() {
		for i := range botNames {
			botNames[i] += " BOT"
		}
	})
}

func createGameState(botCount int, registerLobby bool) *GameState {
	state := &GameState{
		ActivePlayer:  -1,
		clientPlayer:  -1,
		registerLobby: registerLobby,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
		LastResult:    WAITING_MESSAGE,
	}
	for i := 0; i < botCount; i++ {
		state.Players = append(state.Players, newPlayer(botNames[i], true))
	}
	return state
}

func newPlayer(name string, isBot bool) Player {
	return Player{Name: name, Purse: STARTING_PURSE, isBot: isBot, lastPing: time.Now()}
}

func (state *GameState) RunGameLogic() {
	if !state.allowBotGames && state.countHumans() == 0 {
		if state.Round != ROUND_WAITING {
			state.abortHand()
		}
		return
	}

	switch state.Round {
	case ROUND_WAITING:
		state.startBetting()
	case ROUND_BETTING:
		state.runBetting()
	case ROUND_PLAYING:
		state.runPlaying()
	case ROUND_DEALER:
		if state.timerExpired() {
			state.dealerStep()
		}
	case ROUND_OVER:
		if state.timerExpired() {
			state.startBetting()
		}
	}
}

func (state *GameState) startBetting() {
	state.refreshSeats()

	state.Dealer = nil
	state.holeRevealed = false
	state.ActivePlayer = -1
	state.ActiveHand = 0

	if len(state.Players) == 0 {
		state.Round = ROUND_WAITING
		state.LastResult = WAITING_MESSAGE
		return
	}

	for i := range state.Players {
		p := &state.Players[i]
		p.Hands = nil
		p.Move = ""
		if p.Purse < MIN_BET {
			state.rebuys += STARTING_PURSE - p.Purse
			p.Purse = STARTING_PURSE
		}
		if p.Status != STATUS_LEFT {
			p.Status = STATUS_WAITING
		}
	}

	state.Round = ROUND_BETTING
	state.LastResult = "Place your bets"
	state.nextBettor()
	state.runBetting()
}

func (state *GameState) nextBettor() {
	for i := state.ActivePlayer + 1; i < len(state.Players); i++ {
		if state.Players[i].Status == STATUS_WAITING {
			state.ActivePlayer = i
			state.resetPlayerTimer()
			return
		}
	}
	state.ActivePlayer = -1
}

func (state *GameState) runBetting() {
	for state.ActivePlayer >= 0 {
		p := &state.Players[state.ActivePlayer]
		if p.isBot && p.Status != STATUS_LEFT {
			state.placeBet(state.botBet())
		} else if p.Status != STATUS_LEFT {
			if !state.timerExpired() {
				return
			}
			p.Move = "sit out"
		}
		state.nextBettor()
	}
	state.dealHand()
}

func (state *GameState) placeBet(amount int) {
	p := &state.Players[state.ActivePlayer]
	p.Purse -= amount
	p.lastBet = amount
	p.Hands = []hand{{Bet: amount}}
	p.Status = STATUS_PLAYING
	p.Move = fmt.Sprintf("bet %d", amount)
	state.wagered += amount
}

func (state *GameState) dealHand() {
	bettors := []*Player{}
	for i := range state.Players {
		if state.Players[i].Status == STATUS_PLAYING {
			bettors = append(bettors, &state.Players[i])
		}
	}
	if len(bettors) == 0 {
		state.Round = ROUND_OVER
		state.LastResult = "No bets this hand"
		state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		return
	}

	if state.testDeck != nil || state.shoe == nil || state.shoeIndex >= RESHUFFLE_AT {
		state.newShoe()
	}

	for pass := 0; pass < 2; pass++ {
		for _, p := range bettors {
			p.Hands[0].Cards = append(p.Hands[0].Cards, state.draw())
		}
		state.Dealer = append(state.Dealer, state.draw())
	}

	for _, p := range bettors {
		if isBlackjack(p.Hands[0].Cards, false) {
			p.Hands[0].done = true
			p.Move = "bj"
		}
	}

	if cardValue(state.Dealer[0]) >= 10 && isBlackjack(state.Dealer, false) {
		state.holeRevealed = true
		state.settle()
		return
	}

	state.Round = ROUND_PLAYING
	state.LastResult = ""
	state.ActivePlayer = 0
	state.ActiveHand = 0
	state.advance()
}

// Moves to the next unfinished hand, or to the dealer when there is none
func (state *GameState) advance() {
	for i := max(state.ActivePlayer, 0); i < len(state.Players); i++ {
		p := &state.Players[i]
		if p.Status != STATUS_PLAYING {
			continue
		}
		for h := range p.Hands {
			if !p.Hands[h].done {
				state.ActivePlayer = i
				state.ActiveHand = h
				state.resetPlayerTimer()
				return
			}
		}
	}
	state.ActivePlayer = -1
	state.ActiveHand = 0
	state.Round = ROUND_DEALER
	state.moveExpires = time.Now().Add(DEALER_TIME_LIMIT)
}

func (state *GameState) runPlaying() {
	if state.ActivePlayer < 0 {
		state.advance()
		return
	}
	p := &state.Players[state.ActivePlayer]
	if p.Status != STATUS_PLAYING {
		state.advance()
		return
	}
	if !state.timerExpired() {
		return
	}
	move := MOVE_STAND
	if p.isBot {
		move = state.botMove()
	}
	state.performMove(move)
}

func (state *GameState) performMove(move string) bool {
	if state.ActivePlayer < 0 || state.Round != ROUND_BETTING && state.Round != ROUND_PLAYING {
		return false
	}
	if !slices.ContainsFunc(state.getValidMoves(), func(m validMove) bool { return m.Move == move }) {
		return false
	}
	p := &state.Players[state.ActivePlayer]

	if state.Round == ROUND_BETTING {
		state.placeBet(state.betAmount(move))
		state.nextBettor()
		state.runBetting()
		return true
	}

	h := &p.Hands[state.ActiveHand]
	switch move {
	case MOVE_HIT:
		h.Cards = append(h.Cards, state.draw())
		p.Move = "hit"
	case MOVE_STAND:
		h.done = true
		p.Move = "stand"
	case MOVE_DOUBLE:
		p.Purse -= h.Bet
		h.Bet *= 2
		h.doubled = true
		h.done = true
		h.Cards = append(h.Cards, state.draw())
		p.Move = "double"
	case MOVE_SPLIT:
		p.Purse -= h.Bet
		second := hand{Cards: []card{h.Cards[1]}, Bet: h.Bet, split: true}
		h.Cards = []card{h.Cards[0], state.draw()}
		h.split = true
		second.Cards = append(second.Cards, state.draw())
		p.Hands = append(p.Hands, second)
		if p.Hands[0].Cards[0].Rank == ACE {
			p.Hands[0].done = true
			p.Hands[1].done = true
		}
		p.Move = "split"
	case MOVE_SURRENDER:
		h.surrendered = true
		h.done = true
		p.Move = "surr"
	}

	for i := range p.Hands {
		h := &p.Hands[i]
		total, _ := handValue(h.Cards)
		if total > 21 {
			h.done = true
			if i == state.ActiveHand {
				p.Move = "bust"
			}
		} else if total == 21 || len(h.Cards) >= MAX_HAND_CARDS {
			h.done = true
		}
	}

	if p.Hands[state.ActiveHand].done {
		state.advance()
	} else {
		state.resetPlayerTimer()
	}
	return true
}

func (state *GameState) betAmount(move string) int {
	if move == MOVE_REBET {
		return state.Players[state.ActivePlayer].lastBet
	}
	return BET_SIZES[slices.Index(betMoves, move)]
}

func (state *GameState) getValidMoves() []validMove {
	moves := []validMove{}
	if state.ActivePlayer < 0 {
		return moves
	}
	p := state.Players[state.ActivePlayer]

	if state.Round == ROUND_BETTING {
		if p.lastBet > 0 && p.lastBet <= p.Purse {
			moves = append(moves, validMove{MOVE_REBET, fmt.Sprintf("Same %d", p.lastBet)})
		}
		for i, b := range BET_SIZES {
			if b <= p.Purse {
				moves = append(moves, validMove{betMoves[i], fmt.Sprintf("Bet %d", b)})
			}
		}
		return moves
	}

	if state.Round != ROUND_PLAYING || state.ActiveHand >= len(p.Hands) {
		return moves
	}
	h := p.Hands[state.ActiveHand]
	if h.done {
		return moves
	}
	twoCards := len(h.Cards) == 2
	moves = append(moves, validMove{MOVE_HIT, "Hit"}, validMove{MOVE_STAND, "Stand"})
	if twoCards && p.Purse >= h.Bet {
		moves = append(moves, validMove{MOVE_DOUBLE, "Double"})
	}
	if twoCards && len(p.Hands) == 1 && p.Purse >= h.Bet && cardValue(h.Cards[0]) == cardValue(h.Cards[1]) {
		moves = append(moves, validMove{MOVE_SPLIT, "Split"})
	}
	if twoCards && len(p.Hands) == 1 {
		moves = append(moves, validMove{MOVE_SURRENDER, "Surrender"})
	}
	return moves
}

func (state *GameState) liveHands() bool {
	for _, p := range state.Players {
		for _, h := range p.Hands {
			total, _ := handValue(h.Cards)
			if total <= 21 && !h.surrendered && !isBlackjack(h.Cards, h.split) {
				return true
			}
		}
	}
	return false
}

func (state *GameState) dealerStep() {
	total, _ := handValue(state.Dealer)
	if !state.holeRevealed {
		state.holeRevealed = true
	} else if total < 17 && state.liveHands() {
		state.Dealer = append(state.Dealer, state.draw())
	} else {
		state.settle()
		return
	}
	total, _ = handValue(state.Dealer)
	state.LastResult = fmt.Sprintf("Dealer has %d", total)
	state.moveExpires = time.Now().Add(DEALER_TIME_LIMIT)
}

func (state *GameState) settle() {
	dealerTotal, _ := handValue(state.Dealer)
	dealerBJ := isBlackjack(state.Dealer, false)

	for i := range state.Players {
		p := &state.Players[i]
		if len(p.Hands) == 0 {
			continue
		}
		net := 0
		for j := range p.Hands {
			h := &p.Hands[j]
			total, _ := handValue(h.Cards)
			bj := isBlackjack(h.Cards, h.split)
			ret := 0
			switch {
			case h.surrendered:
				ret, h.Result = h.Bet/2, RESULT_SURRENDER
			case total > 21:
				h.Result = RESULT_LOSE
			case bj && dealerBJ:
				ret, h.Result = h.Bet, RESULT_PUSH
			case bj:
				ret, h.Result = h.Bet+h.Bet*3/2, RESULT_BLACKJACK
			case dealerBJ:
				h.Result = RESULT_LOSE
			case dealerTotal > 21 || total > dealerTotal:
				ret, h.Result = 2*h.Bet, RESULT_WIN
			case total == dealerTotal:
				ret, h.Result = h.Bet, RESULT_PUSH
			default:
				h.Result = RESULT_LOSE
			}
			h.done = true
			p.Purse += ret
			net += ret - h.Bet
		}
		state.houseNet -= net
		switch {
		case net > 0:
			p.Move = fmt.Sprintf("+%d", net)
		case net < 0:
			p.Move = fmt.Sprintf("%d", net)
		default:
			p.Move = "push"
		}
	}

	switch {
	case dealerBJ:
		state.LastResult = "Dealer has blackjack"
	case dealerTotal > 21:
		state.LastResult = fmt.Sprintf("Dealer busts with %d", dealerTotal)
	default:
		state.LastResult = fmt.Sprintf("Dealer has %d", dealerTotal)
	}
	log.Printf("TABLE %s: %s", state.TableId, state.LastResult)

	state.handsPlayed++
	state.Round = ROUND_OVER
	state.ActivePlayer = -1
	state.ActiveHand = 0
	state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
}

// Returns every outstanding bet; used when the last human leaves mid-hand
func (state *GameState) abortHand() {
	for i := range state.Players {
		p := &state.Players[i]
		if state.Round != ROUND_OVER {
			for _, h := range p.Hands {
				p.Purse += h.Bet
				state.wagered -= h.Bet
			}
		}
		p.Hands = nil
		p.Move = ""
		if p.Status == STATUS_PLAYING {
			p.Status = STATUS_WAITING
		}
	}
	state.Dealer = nil
	state.holeRevealed = false
	state.Round = ROUND_WAITING
	state.ActivePlayer = -1
	state.ActiveHand = 0
	state.LastResult = WAITING_MESSAGE
	log.Printf("TABLE %s: no humans left, hand aborted", state.TableId)
}

func (state *GameState) timerExpired() bool {
	return !time.Now().Before(state.moveExpires)
}

func (state *GameState) resetPlayerTimer() {
	limit := PLAYER_TIME_LIMIT
	if state.Players[state.ActivePlayer].isBot {
		limit = BOT_TIME_LIMIT
	} else if state.countHumans() == 1 {
		limit = PLAYER_TIME_LIMIT_SINGLE_PLAYER
	}
	state.moveExpires = time.Now().Add(limit)
}

func (state *GameState) countHumans() int {
	count := len(state.joinQueue)
	for _, p := range state.Players {
		if !p.isBot && p.Status != STATUS_LEFT {
			count++
		}
	}
	return count
}

// Between hands: drops leavers and timed out players, seats the join queue
// (parking bots to make room) and brings parked bots back into empty seats
func (state *GameState) refreshSeats() {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	stale := func(p Player) bool { return !p.isBot && p.lastPing.Before(cutoff) }

	players := []Player{}
	for _, p := range state.Players {
		if p.Status != STATUS_LEFT && !stale(p) {
			players = append(players, p)
		}
	}
	state.joinQueue = slices.DeleteFunc(state.joinQueue, stale)

	for len(state.joinQueue) > 0 {
		if len(players) >= MAX_PLAYERS {
			bot := slices.IndexFunc(players, func(p Player) bool { return p.isBot })
			if bot < 0 {
				break
			}
			state.botBox = append(state.botBox, players[bot])
			players = slices.Delete(players, bot, bot+1)
		}
		players = append(players, state.joinQueue[0])
		state.joinQueue = state.joinQueue[1:]
	}
	for len(players) < MAX_PLAYERS && len(state.botBox) > 0 && len(state.joinQueue) == 0 {
		players = append(players, state.botBox[len(state.botBox)-1])
		state.botBox = state.botBox[:len(state.botBox)-1]
	}

	// Humans act first so they never wait on the bots for their turn
	slices.SortStableFunc(players, func(a, b Player) int {
		if a.isBot == b.isBot {
			return 0
		}
		if b.isBot {
			return -1
		}
		return 1
	})

	changed := len(players) != len(state.Players)
	state.Players = players
	state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.Name, state.clientName) })
	if changed {
		state.updateLobby()
	}
}

func (state *GameState) setClientPlayerByName(name string) {
	state.clientName = name
	state.clientPlayer = -1
	if name == "" {
		return
	}
	match := func(p Player) bool { return strings.EqualFold(p.Name, name) }

	if i := slices.IndexFunc(state.Players, match); i >= 0 {
		state.clientPlayer = i
		p := &state.Players[i]
		if p.Status == STATUS_LEFT {
			p.Status = STATUS_WAITING
			if len(p.Hands) > 0 && state.Round != ROUND_OVER {
				p.Status = STATUS_PLAYING
			}
		}
		return
	}
	if i := slices.IndexFunc(state.joinQueue, match); i >= 0 {
		state.joinQueue[i].lastPing = time.Now()
		return
	}

	if len(state.Players) < MAX_PLAYERS {
		state.Players = append(state.Players, newPlayer(name, false))
		state.clientPlayer = len(state.Players) - 1
	} else {
		state.joinQueue = append(state.joinQueue, newPlayer(name, false))
	}
	state.updateLobby()
}

func (state *GameState) playerPing() {
	if state.clientPlayer >= 0 {
		state.Players[state.clientPlayer].lastPing = time.Now()
	}
}

func (state *GameState) clientLeave() {
	match := func(p Player) bool { return strings.EqualFold(p.Name, state.clientName) }
	state.joinQueue = slices.DeleteFunc(state.joinQueue, match)
	if state.clientPlayer >= 0 {
		p := &state.Players[state.clientPlayer]
		p.Status = STATUS_LEFT
		p.Move = "left"
	}
	if state.countHumans() == 0 && state.Round != ROUND_WAITING {
		state.abortHand()
		state.refreshSeats()
	}
}

func (state *GameState) createClientState() *clientState {
	cs := &clientState{
		LastResult:   state.LastResult,
		Round:        state.Round,
		ActivePlayer: -1,
		ActiveHand:   state.ActiveHand,
	}

	dealer := state.Dealer
	if !state.holeRevealed && len(dealer) > 1 {
		dealer = dealer[:1]
		cs.Dealer = cardsToString(dealer) + "??"
	} else {
		cs.Dealer = cardsToString(dealer[:min(len(dealer), MAX_DEALER_CARDS)])
	}
	if len(dealer) > 0 {
		dh := hand{Cards: dealer}
		cs.DealerTotal, _ = handValue(dealer)
		cs.DealerFlags = dh.flags() &^ FLAG_DONE
	}

	start := state.clientPlayer
	if start < 0 {
		start = 0
		cs.Viewing = 1
	}

	for i := start; i < start+len(state.Players); i++ {
		idx := i % len(state.Players)
		if idx == state.ActivePlayer {
			cs.ActivePlayer = i - start
		}
		p := state.Players[idx]
		cp := clientPlayer{Name: p.Name, Status: p.Status, Purse: p.Purse, Move: p.Move}
		for _, h := range p.Hands {
			total, _ := handValue(h.Cards)
			cp.Hands = append(cp.Hands, clientHand{
				Bet:    h.Bet,
				Total:  total,
				Flags:  h.flags(),
				Result: h.Result,
				Cards:  cardsToString(h.Cards[:min(len(h.Cards), MAX_HAND_CARDS)]),
			})
		}
		cs.Players = append(cs.Players, cp)
	}

	if cs.ActivePlayer == 0 && cs.Viewing == 0 {
		cs.ValidMoves = state.getValidMoves()
		cs.MoveTime = int(time.Until(state.moveExpires).Seconds()) - MOVE_TIME_GRACE_SECONDS
		cs.MoveTime = max(0, min(cs.MoveTime, 255))
	}

	cs.Hash = "0"
	hash, _ := hashstructure.Hash(cs, hashstructure.FormatV2, nil)
	cs.Hash = fmt.Sprintf("%d", hash)
	return cs
}

func (state *GameState) updateLobby() {
	if !state.registerLobby {
		return
	}
	slots, count := state.getHumanPlayerCountInfo()
	sendStateToLobby(slots, count, true, state.serverName, "?table="+state.TableId)
}

func (state *GameState) getHumanPlayerCountInfo() (int, int) {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	count := 0
	for _, p := range append(slices.Clone(state.Players), state.joinQueue...) {
		if !p.isBot && p.Status != STATUS_LEFT && p.lastPing.After(cutoff) {
			count++
		}
	}
	return MAX_PLAYERS, count
}
