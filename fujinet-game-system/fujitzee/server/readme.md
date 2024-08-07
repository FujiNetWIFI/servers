# Fujitzee
This is a Yahtzee clone server written in GO. 

It currently provides:
* Multiple concurrent games (tables) via the `?table=[Alphanumeric value]` url parameter
* Bots that simulate players
* Auto moves for players that do not move in time 

## Accessing the Game Server API

1. You may call the public api at https://fujitzee.carr-designs.com/

2. Alternatively, clone and run the server locally:
    ```
    go run .
    ```


## Basic Flow

A game client will perform the following actions:

#### Show tables to join
1. Call `/tables` to present a list of tables to join.

#### Join a table
1. There is no specific call to join a table. Simply retrieving the state for a table will cause the player to join that table.
2. In a loop (waiting for players to ready up):
    1. Call `/state?player=X&table=Y` to retrieve the latest state
    2. Call `/ready?player=X&table=Y` to toggle if that player is ready or not
3. Once all players have readied up, a count down starts and then gameplay begins. Players may unready to abort the countdown.

#### Main gameplay loop
1. In a loop:
    1. Call `/state?player=X&table=Y` to retrieve that latest state
    2. Call `/roll/[KEEP]?player=X&table=Y` to roll dice, keeping the specified index of dice to keep
    3. Call `/score/[INDEX]?player=X&table=Y` with an index from validScores to score for that round
2. If the player wishes to exit the game, the client should call `/leave?player=X&table=Y`


## Retrieving the Table List

Retrieve the list of tables by calling `/tables`.
___
**DEVELOPER TIP** - Call `/tables?dev=1` to retrieve a list of hidden tables for developer usage. You can test your client using these "dev*" tables without impacting live player facing games on the public server.
___

A list of objects with the following properties will be returned:

* `t` - Table id. Pass this as the `table` url parameter to other calls.
* `n` - Friendly name of table to show in a list for the player to choose
* `p` - Number of players currently connected. 0 if none.
* `m` - Number of max available player slots available. Once a game as begun, this will match the current players connected (a player cannot join mid-game to play, but can watch).

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

These tables are psuedo real time. Call `/state` will run any housekeeping tasks (bot or player auto-move). Since a call to `/state` is required to advance the game, a table with bots in it will not actually play until one or more clients are connected and calling `/state`. Each player has a limited amount of time to make a move before the server makes a move on their behalf.

* The game is over when all score locations have been filled in (13 rounds of rolling). The next game will begin automatically after a few seconds.
* The game is waiting on more players when **round 0** is sent.
* Clients should call `/leave` when a player exits the game or table, rather than rely on the server to eventually drop the player due to inactivity.

You can view the state as-is by calling `/view`.

## Api paths

* `/state` - Advance forward (AI/Game Logic) and return updated state
* `/ready` - Toggle if this player is ready. When joining a table that does not have game in progress, all connected players must ready up to start.
* `/roll/[keepRoll]` - Re-roll when it is your player's turn, specifying to either keep or roll the dice at that index.`0` means keep and `1` means re-roll. For example:
    * Given roll `11234`, to keep the ones, call `/roll/00111`. 
    * Given roll `11234`, to keep "1234" and only roll the first die, call `/roll/10000`. 
    * Given roll `31363`, to keep the threes and roll the 1 and 6, call `/roll/01010`. 
* `/score/[index]` - Score the specified index from valid scores array `vs[]` for the current player. The **value** in `vs[]` for that index must be `0` or greater. `-1` indicates an invalid score index (already scored previously), and will not result in a score.
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

A client centric state is returned. This means the player array will always start with your client's player first, though all clients will see all players in the same order.

#### Json Properties

Keys are single character, lower case, to make parsing easier on 8-bit clients. Array keys are 2 character.

(TBD)
    

#### Example state

```json
{
  "p": "ERIC's turn",
  "r": 1,
  "l": 1,
  "a": 0,
  "m": 25,
  "v": 0,
  "d": "15466",
  "pl": [
    {
      "n": "ERIC",
      "a": "E",
      "sc": [-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1]
    },
    {
      "n": "1AI Clyd",
      "a": "1",
      "sc": [-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1]
    },
    {
      "n": "2AI Jim",
      "a": "2",
      "sc": [-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1]
    }
  ],
  "vs": [1,0,0,0,5,18,-1,-1,24,0,0,0,0,24,0]
}
```
