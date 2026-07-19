# Texas Hold'em Server

This is a Texas Hold'em server written in GO, derived from the FujiNet 5 Card Stud
server and using the same compact wire protocol designed for 8-bit clients
(Atari, Apple II, etc.). Hand ranking uses github.com/cardrank/cardrank.

It currently provides:
* Multiple concurrent games (tables) via the `?table=[Alphanumeric value]` url parameter
* Bots with distinct personalities (VPIP/PFR/bluff profiles) that play pre-flop and post-flop
* Bots only play while at least one human is seated - if the last human leaves, all bots fold and the table parks until a human returns
* Blinds, button rotation, heads-up rules, all-ins with main/side pots
* Auto moves for players that do not move in time (check if free, otherwise fold)
* Auto drops players that have not interacted with the server after some time (timed out)

## Accessing the Game Server API

Clone and run the server locally:
```
go run .
```

A reference client (Bubbletea TUI + scriptable headless mode) lives in
`test-go-client`:
```
cd test-go-client
go run . -server http://localhost:8080            # interactive, with table picker
go run . -server http://localhost:8080 -headless -table dev3 -hands 2   # scripted
```

## Basic Flow

A game client is expected to:

1. Call `/tables` to present a list of tables to join.
2. There is no specific call to join a table. Simply retrieving the state will cause the player to join that table.
3. In a loop:
    * Call `/state?player=X&table=Y` to retrieve the latest state
    * Call `/move/[CODE]?player=X&table=Y` to place a move if it is the current player's turn
4. If the player wishes to exit the game, the client should call `/leave?player=X&table=Y`

## Retrieving the Table List

Retrieve the list of tables by calling `/tables`.
___
**DEVELOPER TIP** - Call `/tables?dev=1` to retrieve a list of hidden tables for developer usage. You can test your client using these "dev*" tables (dev1..dev7, preloaded with 1..7 bots) without impacting live player facing games on the public server.
___

A list of objects with the following properties will be returned:

* `t` - Table id. Pass this as the `table` url parameter to other calls.
* `n` - Friendly name of table to show in a list for the player to choose
* `p` - Number of players currently connected. 0 if none.
* `m` - Number of max available player slots available.

Example response of `/tables` call
```json
[{
    "t":"basement",
    "n":"The Basement",
    "p":3,
    "m":8
},{
    "t":"ai2",
    "n":"AI Room - 2 bots",
    "p":0,
    "m":6
}]
```

These tables are pseudo real time. Calling `/state` will run any housekeeping tasks (bot or player auto-move, deal cards, advance streets). Since a call to `/state` is required to advance the game, a table with bots in it will not actually play until one or more clients are connected and calling `/state`. Each player has a limited amount of time to make a move before the server makes a move on their behalf. BOTs take a few seconds to move.

* The game is over when **round 5** is sent. The next game will begin automatically after a few seconds.
* The game is waiting on more players when **round 0** is sent.
* Clients should call `/leave` when a player exits the game or table, rather than rely on the server to eventually drop the player due to inactivity.

You can view the state as-is by calling `/view`.

## Api paths

* `/state` - Advance forward (AI/Game Logic) and return updated state as compact json. Pass `hash=[z value from previous state]` to receive `"1"` instead of the full body when nothing changed.
* `/move/[code]` - Apply your player's move and return updated state. e.g. `/move/CH` to Check, `/move/CA` to Call. Codes are always 2 characters (see Move codes below); amounts are computed server-side.
* `/leave` - Leave the table. Each client should call this when a player exits the game
* `/view?table=N` - View the current state as-is without advancing, as formatted json. Useful for debugging in a browser alongside the client. Only `table` query parameter is required.
* `/tables` - Returns a list of available REAL tables along with player information. No query parameters are required. Pass `dev=1` for the hidden developer tables.
* `/version` - Returns the server version string (also logged at startup), e.g. "texasholdem-server v1.1.0 (commit abc12345, ...)". No query parameters are required.
* `/updateLobby` - Use to manually force a refresh of state to the Lobby. No query parameters are required.

All paths accept GET or POST for ease of use.

## Query parameters

### Required
All paths require the query parameters below, unless otherwise specified.
* `TABLE=[Alphanumeric]` - **Required** - Use to play in an isolated game. Case insensitive.
* `PLAYER=[Alphanumeric]` - **Required for Real** - Player's name. Treated as case insensitive unique ID.

### Optional
* `HASH=[z value]` - **Optional, /state only** - Pass the `z` value from the previously received state. If the state has not changed, the server returns just `"1"`, saving bandwidth and parse time.
* `BIN=1` - **Optional** - Return a packed binary struct instead of json (see "Binary protocol" below). This is what the cc65/cmoc 8-bit clients use.
* `BE=1` - **Optional** - Use with bin, emit uint16 values big-endian (CoCo). Default is little-endian (6502).
* `RAW=1` - **Optional** - Use to return key[byte 0]value[byte 0] pairs instead of json output - similar to FujiNet json parsing, with 0x00 used as delimiter instead of line end
* `UC=1` - **Optional** - Use with raw, to make the result data upper case
* `LC=1` - **Optional** - Use with raw, to make the result data lower case

## Binary protocol (bin=1)

8-bit clients read `/state?bin=1` directly into a packed C struct - no parsing.
The layout must byte-for-byte match `src/misc.h` in the client repo
(fujinet-texasHoldEm). All strings are fixed-length, NUL-terminated, and
**lowercase** (card codes like `ks` `ah`, `??` for hidden). uint16 values are
little-endian unless `be=1`.

```c
typedef struct {              // offset
  char lastResult[81];        //   0
  uint8_t round;              //  81
  uint16_t pot;               //  82
  int8_t activePlayer;        //  84
  uint8_t moveTime;           //  85
  uint8_t viewing;            //  86
  char community[11];         //  87  Texas Hold'em addition (up to 5 cards)
  uint8_t validMoveCount;     //  98
  ValidMove validMoves[5];    //  99  { char move[3]; char name[10]; } x5
  uint8_t playerCount;        // 164
  Player players[8];          // 165  { char name[9]; uint8_t status; uint16_t bet;
                              //        char move[8]; uint16_t purse; char hand[11]; }
} Game;                       // total 429 bytes max (165 + playerCount*33)
```

Notes:
* Only `playerCount` player records are sent (the blob is `165 + 33*N` bytes).
* Status 4 (all-in) is mapped to 1 (playing) in binary mode - 8-bit clients
  treat status 1 as "in the hand".
* Valid move display names are word-trimmed to 9 characters (e.g. "All-in").
* `/tables?bin=1` returns `{ uint8_t count; { char table[9]; char name[21];
  char players[6]; } x count }`.
* The layout is locked by tests on both sides: `bin_protocol_test.go` here and
  `support/host-test/holdem_host_test.c` in the client repo.

## Move codes

Clients only ever send 2-character move codes; the server computes all chip
amounts. The `vm` array in the state lists which codes are currently legal along
with a friendly name that includes the amount (e.g. `"Call 10"`).

| Code | Meaning |
|------|---------------------------------------------|
| `FO` | Fold |
| `CH` | Check (includes the big-blind option) |
| `CA` | Call the current bet |
| `BL` | Bet low (1x big blind) |
| `BH` | Bet high (2x big blind) |
| `RL` | Minimum raise |
| `RH` | Bigger raise (2x the minimum increment) |
| `AI` | All-in (also serves as a call-for-less) |

## State structure

This is focused on a low nested structure and speed of parsing for 8-bit clients.

A client centric state is returned. This means that your client will only see the values of cards it is meant to see, and the player array will always start with your client's player first, though all clients will see all players in the same order.

#### Json Properties
Keys are single character, lower case, to make parsing easier on 8-bit clients. Array keys are 2 character.

* `l` - Will be filled with text when round=`5` to signal the current game is over. e.g. "Thom won with Two Pair, Kings and Eights", or when round=`0` to indicate waiting for more players to join.
* `r` - The current round. `0`=waiting for players, `1`=Pre-flop, `2`=Flop, `3`=Turn, `4`=River, `5`=hand complete (pot has been awarded).
* `p` - The current value of the pot for the current hand
* `a` - The currently active player. Your client is always player 0. This will be `-1` at the end of a betting round (or end of game) to allow the client to show the last move before the next street begins.
* `m` - Move time - Number of seconds remaining for current player to make their move. If a player does not send a move within this time, the server will auto-move for them (check if possible, otherwise fold)
* `v` - Viewing - If all player spots are full, your client's player will not join the game, but instead view the game as a spectator. In this case, this will be `1` to indicate that you are only viewing. Otherwise, this will be `0` during normal play.
* `c` - Community cards as a card string (see hand format below), e.g. `"AS5H2D"` on the flop, growing to 10 characters by the river. Empty pre-flop. *(Texas Hold'em addition)*
* `z` - State hash. Pass back as the `hash` query parameter on `/state` to skip unchanged states. *(Addition to the original spec)*
* `vm` - An array of Valid Moves, present only when it is your turn
    * `m` - The 2-character move code to send to `/move`
    * `n` - The friendly name of the move to show onscreen in the client, including the amount (e.g. "Call 10", "Raise 20", "All-in 990")
* `pl` - An array of player objects
    * `n` - Name - The name of the player
    * `s` - Status - The player's current in-game status
        * 0 - Just joined, waiting to play the next game
        * 1 - In Game, playing
        * 2 - In Game, Folded
        * 3 - Left the table (will be gone next game)
        * 4 - In Game, All-in *(Texas Hold'em addition - still in the hand but cannot act further)*
    * `b` - Bet - The total of the player's bet for the current betting round
    * `m` - Move - Friendly text of the player's most recent move this round (e.g. "CALL", "RAISE", "POST 10")
    * `p` - Purse - The player's remaining amount available to bet
    * `h` - Hand - A string of 2-character card representations of the player's hole cards:
        * First char - Value : 2 to 9, T=10, J=Jack, Q=Queen, K=King, A=Ace
        * Second char - Suit : C,S,D,H stand for Clubs, Spades, Diamonds, and Hearts
        * `??` - A hidden card. Other players' hole cards appear as `????` during the hand and are revealed at showdown (round 5, unless the hand ended by everyone folding). A folded player's hand is just `??`.

#### Example state

```json
{
    "l": "",
    "r": 2,
    "p": 25,
    "a": 0,
    "m": 25,
    "v": 0,
    "c": "2HKC9C",
    "vm": [
        { "m": "FO", "n": "Fold" },
        { "m": "CH", "n": "Check" },
        { "m": "BL", "n": "Bet 10" },
        { "m": "BH", "n": "Bet 20" },
        { "m": "AI", "n": "All-in 990" }
    ],
    "pl": [
        { "n": "You",      "s": 1, "b": 0, "m": "",      "p": 990, "h": "8D7H" },
        { "n": "Thom",     "s": 1, "b": 0, "m": "CHECK", "p": 985, "h": "????" },
        { "n": "Mozzwald", "s": 2, "b": 0, "m": "FOLD",  "p": 1000, "h": "??" }
    ],
    "z": "9040156043085247566"
}
```
