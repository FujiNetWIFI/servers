# Blackjack Server

Multiplayer blackjack against the dealer for FujiNet clients. Same API shape as
the other fujinet-game-system servers.

## Rules

- 6 deck shoe, reshuffled once 75% has been dealt
- Dealer stands on soft 17 and peeks for blackjack with an ace or ten up
- Blackjack pays 3:2
- Double on any first two cards, including after a split
- One split per hand; split aces get one card each; 21 after a split is not blackjack
- Late surrender
- No insurance
- A hand that reaches 8 cards stands
- Bets of 10, 20, 50 or 100, starting purse 1000, rebuy to 1000 when under 10

Up to 5 players. Bots play basic strategy and only play while a human is
seated. A player who joins a full table waits as a viewer and takes a bot's
seat at the next hand. A player who does not bet in time sits the hand out; one
who does not move in time stands.

## Running

```
go run .
```

`../play.sh` builds and starts the server and drops you into the terminal
client in `../client`.

## API

- `/tables` - table list (`?dev=1` for the hidden dev1-dev4 tables)
- `/state?table=T&player=P` - joins on first call, then returns the state
- `/move/CODE?table=T&player=P` - moves only for the active player
- `/leave?table=T&player=P`
- `/view?table=T` - read only state for debugging

Add `bin=1` for the packed binary form (layout in `util.go`), plus `be=1` for
big endian uint16s on 6809. In JSON only, add `hash=` with the last state's
`z` to get `1` back when nothing changed.

Move codes: betting `RB` (same as last), `B1`-`B4`; playing `HI` hit, `ST`
stand, `DD` double, `SP` split, `SU` surrender.

Rounds: 0 waiting, 1 betting, 2 playing, 3 dealer, 5 hand over.

Hand flags: 1 soft, 2 blackjack, 4 bust, 8 doubled, 16 surrendered, 32 done.
Hand results: 1 win, 2 lose, 3 push, 4 blackjack, 5 surrender.
