# Fujirkle
This is a Farkle clone server written in GO.

It currently provides:
* Multiple concurrent games (tables) via the `?table=[Alphanumeric value]` url parameter
* Bots that simulate players
* Auto moves for players that do not move in time

## Accessing the Game Server API

1. You may call the public api at https://fujirkle.carr-designs.com/

2. Alternatively, clone and run the server locally:
    ```
    go run .
    ```

A terminal client is included for play testing the server on Linux. From the project root:
```
./play.sh
```
This builds the server and client into a temp directory, starts the server, and joins a dev table. Run `./play.sh -auto -games 1` to watch it play itself.


## Rules

Six dice are rolled. Each turn the player sets aside at least one scoring die and then chooses to roll the remaining dice or bank the turn score.

| Combination | Score |
| --- | --- |
| Each 1 | 100 |
| Each 5 | 50 |
| Three of a kind | face value x 100 (three 1s = 1000) |
| Four of a kind | three of a kind x 2 |
| Five of a kind | three of a kind x 4 |
| Six of a kind | three of a kind x 8 |
| Straight 1-6 | 1500 |
| Three pairs | 1500 |

* Every die set aside must take part in a scoring combination. An illegal set-aside is rejected and the state is left untouched.
* **Hot dice** - setting aside all six dice earns a fresh set of six, and the turn continues.
* **Fujirkle** - if a roll contains no scoring dice at all, the turn ends and the entire turn score is lost. The roll stays on screen briefly so all players can see it.
* First player to reach **10000** triggers a final round, giving everyone else one last turn. Highest score then wins.


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
    1. Call `/state?player=X&table=Y` to retrieve the latest state
    2. When `c` indicates a move is valid, call either:
        * `/roll/[KEEP]?player=X&table=Y` to set aside the masked dice and roll the rest
        * `/bank/[KEEP]?player=X&table=Y` to set aside the masked dice and bank the turn score
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
* `m` - Number of max available player slots available. Once a game has begun, this will match the current players connected (a player cannot join mid-game to play, but can watch).

Example response of `/tables` call
```json
[{
    "t":"bar",
    "n":"The Bar",
    "p":3,
    "m":6
},{
    "t":"ai2",
    "n":"AI Room - 2 bots",
    "p":0,
    "m":6
}, ...]
```

These tables are psuedo real time. Calling `/state` will run any housekeeping tasks (bot or player auto-move). Since a call to `/state` is required to advance the game, a table with bots in it will not actually play until one or more clients are connected and calling `/state`. Each player has a limited amount of time to make a move before the server makes a move on their behalf.

* The game is over when a player reaches 10000 and the final round completes. The next game will begin automatically after a few seconds.
* The game is waiting on more players when **round 0** is sent.
* Clients should call `/leave` when a player exits the game or table, rather than rely on the server to eventually drop the player due to inactivity.

You can view the state as-is by calling `/view`.

## Api paths

* `/state` - Advance forward (AI/Game Logic) and return updated state
* `/ready` - Toggle if this player is ready. When joining a table that does not have a game in progress, all connected players must ready up to start.
* `/roll/[keep]` - Set aside the masked dice and roll whatever is left.
* `/bank/[keep]` - Set aside the masked dice and add the turn score to the player's total, ending the turn.
* `/leave` - Leave the table. Each client should call this when a player exits the game
* `/view?table=N` - View the current state as-is without advancing, as formatted json. Useful for debugging in a browser alongside the client. Only `table` query parameter is required.
* `/tables` - Returns a list of available REAL tables along with player information. No query parameters are required
* `/updateLobby` - Use to manually force a refresh of state to the Lobby. No query parameters are required.

All paths accept GET or POST for ease of use.

### The keep mask

`/roll` and `/bank` both take a mask the same length as the current dice `d`, where **`1` means keep/score that die** and `0` leaves it in the pool.
___
**NOTE** - This is the opposite of Fujitzee, where `1` means re-roll. A Fujirkle set-aside is permanent, so the mask marks what you are scoring.
___

* Given roll `152346`, to keep the 1 and the 5, call `/roll/110000`.
* Given roll `111345`, to keep the three 1s (1000) and bank, call `/bank/111000`.
* Given roll `152346`, `/roll/100100` is rejected - the 2 does not score, so the state is unchanged.

Use `x` from the state to find which dice are worth keeping. Every die marked `1` in `x` can take part in some scoring combination, and the marked set as a whole is always a legal selection, so a client can offer a single "keep everything that scores" action by passing `x` straight back as the mask.

## Query parameters

### Required
All paths require the query parameters below, unless otherwise specified.
* `TABLE=[Alphanumeric]` - Use to play in an isolated game. Case insensitive.
* `PLAYER=[Alphanumeric]` - Player's name. Treated as case insensitive unique ID.

### Optional
* `hash=[value]` - Pass the `h` value from the previous state. If nothing has changed, the server returns just `1` instead of the full state, which saves a parse on 8-bit clients.
* `pov=[player]` - Return the state as that player sees it. Useful for debugging alongside a client.

### Response format
The default response is json. Include one of the below parameters to change the format.
* `bin=1` - Return a binary representation suitable for copying directly into a struct. Little-endian is the default. See below for big-endian clients.
* `raw=1` - Use to return key[byte 0]value[byte 0] pairs instead of json output - similar to FujiNet json parsing, with 0x00 used as delimiter instead of line end

#### Bin Options

When `bin=1` is sent, the following **optional** parameter may be sent:
* `be=1` - Use to request big-endian values from the server. If this parameter is not included, the server defaults to little-endian.

#### Raw Options
When `raw=1` is sent, the following **optional** parameters may be sent:
* `uc=1` - Use to make the result data upper case
* `lc=1` - Use to make the result data lower case
* `lf=1` - Use a line feed as the delimiter instead of 0x00


## State structure
This is focused on a low nested structure and speed of parsing for 8-bit clients.

A client centric state is returned. This means the player array will always start with your client's player first, though all clients will see all players in the same order.

#### Json Properties

Keys are single character, lower case, to make parsing easier on 8-bit clients. Array keys are 2 character.

* `n` - Name of the table
* `p` - Prompt to show the player
* `r` - Round. `0` is the lobby, `99` is game over.
* `a` - Index of the active player in `pl`
* `m` - Seconds left for the active player to move
* `v` - `1` when this client is only watching, `0` when playing
* `d` - Dice currently in the pool, as digits
* `k` - The keep mask last applied
* `t` - Turn score accumulated so far this turn
* `e` - Dice already set aside this turn
* `x` - Mask of which dice in `d` can take part in a scoring combination
* `c` - Valid moves, as a bit field. `1` roll, `2` bank. `0` means it is not this client's move.
* `h` - Hash of this state. Pass back as the `hash` parameter to skip an unchanged state.
* `pl` - Players

Each player in `pl` has:

* `n` - Player name
* `a` - Index into the name of the character that makes this player unique in the list
* `s` - Banked score
* `y` - Ready flag. `1` ready, `0` not ready, `-2` watching.

#### Binary format

When `bin=1` is sent, `/state` returns the following packed layout. Strings are fixed length and zero terminated, so a field of "n bytes" occupies n+1 in the stream.

| Field | Size |
| --- | --- |
| Player count | 1 |
| Table name | 20 |
| Prompt | 40 |
| Round | 1 |
| Active player | 1 |
| Move time | 1 |
| Viewing | 1 |
| Valid moves | 1 |
| Turn score | 2 |
| Dice | 6 |
| Kept dice | 6 |
| Selectable | 6 |
| Keep mask | 6 |

Followed by one 13 byte record per player:

| Field | Size |
| --- | --- |
| Name | 8 |
| Alias | 1 |
| Ready | 1 |
| Score | 2 |

`/tables` returns a count byte followed by one record per table: table id (8), name (20), and a "cur / max" string (5).

#### Example state

```json
{
  "n": "The Bar",
  "p": "ERIC's turn",
  "r": 3,
  "a": 0,
  "m": 45,
  "v": 0,
  "d": "152346",
  "k": "",
  "t": 0,
  "e": "",
  "x": "110000",
  "c": 3,
  "pl": [
    {
      "n": "ERIC",
      "a": 0,
      "s": 2350,
      "y": 1
    },
    {
      "n": "1AI Clyd",
      "a": 0,
      "s": 3100,
      "y": 1
    },
    {
      "n": "2AI Meg",
      "a": 0,
      "s": 1800,
      "y": 1
    }
  ]
}
```
