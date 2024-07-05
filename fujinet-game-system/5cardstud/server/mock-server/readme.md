# 5 Card Stud Server
This is a 5 Card Stud server written in GO. This is my first project in GO, so do not expect expert use of the language.  All game logic is written by scratch, with the exception of using github.com/gin-gonic/gin to rank the final 5 card hands.

It currently provides:
* Multiple concurrent games (tables) via the `?table=[Alphanumeric value]` url parameter
* Bots that simulate players (They will check, bet, raise, and fold based on combination of random/simple decision logic)
* Auto moves for players that do not move in time (fold, check, or forced post)
* Auto drops players that have not interacted with the server after some time (timed out)

## Accessing the Game Server API

1. You may call the public api at https://5card.carr-designs.com/

2. Alternatively, clone and run the server locally:
    ```
    go run .
    ```


## Basic Flow

A game client is expected to:

1. Call `/tables` to present a list of tables to join.
2. There is no specific call to join a table. Simply retrieving the state will cause the player to join that table.
2. In a loop:
    A. Call `/state?player=X&table=Y` to retrieve that latest state
    B. Call `/move/[CODE]?player=X&table=Y` to place a move if it is the current player's turn
3. If the player wishes to exit the game, the client should call `/leave?player=X&table=Y`


## Retrieving the Table List

Retrieve the list of tables by calling `/tables`. 
___
**DEVELOPER TIP** - Call `/tables?dev=1` to retrieve a list of hidden tables for developer usage. You can test your client using these "dev*" tables without impacting live player facing games on the public server.
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
}, ...]
```

These tables are psuedo real time. Call `/state` will run any housekeeping tasks (bot or player auto-move, deal card, proceed with dealing). Since a call to `/state` is required to advance the game, a table with bots in it will not actually play until one or more clients are connected and calling `/state`. Each player has a limited amount of time to make a move before the server makes a move on their behalf. BOTs take a second to move.

* The game is over when **round 5** is sent. The next game will begin automatically after a few seconds.
* The game is waiting on more players when **round 0** is sent.
* Clients should call `/leave` when a player exits the game or table, rather than rely on the server to eventually drop the player due to inactivity.

You can view the state as-is by calling `/view`.

## Api paths

* `/state` - Advance forward (AI/Game Logic) and return updated state as compact json
* `/move/[code]` - Apply your player's move and return updated state as compact json. e.g. ``/move/CH`` to "Check", ``/move/BL`` to "Bet 5 (low)".
* `/leave` - Leave the table. Each client should call this when a player exits the game
* `/view?table=N` - View the current state as-is without advancing, as formatted json. Useful for debugging in a browser alongside the client. **NOTE:** If you call this for an uninitated game, a different randomly initiated game will be returned every time. Only `table` query parameter is required.
* `/tables` - Returns a list of available REAL tables along with player information. No query parameters are required
* `/updateLobby` - Use to manually force a refresh of state to the Lobby. No query parameters are required.

All paths accept GET or POST for ease of use.

## Query parameters

### Required
All paths require the query parameters below, unless otherwise specified.
* `TABLE=[Alphanumeric]` - **Required** - Use to play in an isolated game. Case insensitive.
* `PLAYER=[Alphanumeric]` - **Required for Real** - Player's name. Treated as case insensitive unique ID.

### Optional
* `RAW=1` - **Optional** - Use to return key[byte 0]value[byte 0] pairs instead of json output - similar to FujiNet json parsing, with 0x00 used as delimiter instead of line end
* `UC=1` - **Optional** - Use with raw, to make the result data upper case
* `LC=1` - **Optional** - Use with raw, to make the result data lower case

## State structure
This is focused on a low nested structure and speed of parsing for 8-bit clients.

A client centric state is returned. This means that your client will only see the values of cards it is meant to see, and the player array will always start with your client's player first, though all clients will see all players in the same order.

#### Json Properties
Keys are single character, lower case, to make parsing easier on 8-bit clients. Array keys are 2 character.

* `l` - Will be filled with text when round=`5` to signal the current game is over. e.g. "So and so won with 2 pairs", or when round=`0` to indicate waiting for more players to join.
* `r` - The current round (1-5). Round 5 means the game has ended and pot awarded to winning player(s).
* `p` - The current value of the pot for the current game
* `a` - The currently active player. Your client is always player 0. This will be `-1` at the end of a round (or end of game) to allow the client to show the last move before starting the next round.
* `m` - Move time - Number of seconds remaining for current player to make their move, or until the next game will start. If a player does not send a move within this time, the server will auto-move for them (post/check if possible, otherwise a fold)
* `v` - Viewing - If all player spots are full, your client's player will not join the game, but instead view the game as a spectator.  In this case, this will be `1` to indicate that you are only viewing. Otherwise, this will be `0` during normal play. 
* `vm` - An array of Valid Moves
    * `m` - The move code to send to `/move`
    * `n` - The friendly name of the move to show onscreen in the client
* `pl` - An array of player objects
    * `n` - Name - The name of the player, or `You` for the client
    * `s` - Status - The player's current in-game status
        * 0 - Just joined, waiting to play the next game
        * 1 - In Game, playing
        * 2 - In Game, Folded
        * 3 - Left the table (will be gone next game)
    * `b` - Bet - The total of the player's bet for the current round
    * `m` - Move - Friendly text of the player's most recent move this round
    * `p` - Purse - The player's remaining amount available to bet
    * `h` - Hand - A string of multiple 2 character representation of cards in the player's hand:
        * First char - Value : 2 to 9, T=10, J=Jack, Q=Queen, K=King, A=Ace
        * Second char - Suit : C,S,D,H stand for Clubs, Spades, Diamonds, and Hearts
        * `??` - A hidden card. Also represents a folded hand when `hand` is just `??` and followed by no other cards
    
    

#### Example state

```json
{
    "l": "Thom won with Full House, Eights full of Sixes",
    "r": 1,
    "p": 0,
    "a": 0,
    "m": 25,
    "v": 0,
    "vm": [
        {
            "m": "FO",
            "n": "Fold"
        },
        {
            "m": "CA",
            "n": "Call"
        },
        {
            "m": "RL",
            "n": "Raise 5"
        }
    ],
    "pl": [
        {
            "n": "You",
            "s": 1,
            "b": 0,
            "m": "",
            "p": 199,
            "h": "KSKH"
        },
        {
            "n": "Thom",
            "s": 1,
            "b": 5,
            "m": "BET",
            "p": 194,
            "h": "??6H"
        },
        {
            "n": "Mozzwald",
            "s": 0,
            "b": 0,
            "m": "",
            "p": 200,
            "h": ""
        },
    ]
}
```
