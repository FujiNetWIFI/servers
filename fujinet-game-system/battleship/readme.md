# FUJI BATTLE SHIPS
This is a Battleship style game server written in GO. 

********
## NOTE - THIS PROJECT IS IN PROGRESS
##### INFORMATION BELOW MAY BE INCOMPLETE AND IS SUBJECT TO CHANGE
********

## Game Server Features
* Multiple concurrent games
* Support for up to 4 players per game
* Bots that simulate players

## Accessing the Game Server API

1. You may call the public api at https://battleship.carr-designs.com/

2. Alternatively, clone and run the server locally:
    ```
    go run .
    ```


## Basic Flow

A game client should perform the following actions:

#### Show tables to join
1. Call `/tables` to present a list of tables to join.

#### Join a table
1. There is no specific call to join a table. Simply retrieving the state for a table will cause the player to join that table.
2. In a loop (waiting for players to ready up):
    1. Call `/state?player=X&table=Y` to retrieve the latest state
    2. Call `/ready?player=X&table=Y` to toggle if that player is ready or not
3. Once all players have readied up, a count down starts and then gameplay begins. Players may unready to abort the countdown.

#### Place ships
1. Call `/place/N,N,N,N,N?player=X&table=Y` to place your ships

#### Main gameplay loop
1. In a loop:
    1. Call `/state?player=X&table=Y` to retrieve that latest state
    2. Call `/attack/[POSITION]?player=X&table=Y` to attack when it is the player's turn
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

* The game is over when there is only one player left with ships intact. The next game will begin automatically after a few seconds.
* The game is waiting on more players when **status 0** is sent.
* Clients should call `/leave` when a player exits the game or table, rather than rely on the server to eventually drop the player due to inactivity.

You can view the state as-is by calling `/view`.

## Api paths

* `/state` - Advance forward (AI/Game Logic) and return updated state
* `/ready` - Toggle if this player is ready. When joining a table that does not have game in progress, all connected players must ready up to start.
* `/place/[N,N,N,N,N]` - 5 comma separated positions. Place 5 ships on the grid. Position is top left. 0-99 - horizontal. 100-199 - vertical
* `/attack/[POSITION]` - Attack a position on the grid (0-99)
* `/leave` - Leave the game. Each client should call this when a player exits
* `/view?table=N` - View the current state as-is without advancing, as formatted json. Useful for debugging in a browser alongside the client. **NOTE:** If you call this for an uninitated game, a different randomly initiated game will be returned every time. Only `table` query parameter is required.
* `/tables` - Returns a list of available REAL tables along with player information. No query parameters are required
* `/updateLobby` - Use to manually force a refresh of state to the Lobby. No query parameters are required.

All paths accept GET or POST for ease of use.

## Query parameters

### Required
All paths require the query parameters below, unless otherwise specified.
* `TABLE=[Alphanumeric]` - Use to play in an isolated game. Case insensitive.
* `PLAYER=[Alphanumeric]` - Player's name. Treated as case insensitive unique ID.

### Optional
* `BIN=1` - **Optional** - Use to return in binary format instead of user friendly json. Designed for 8 bit clients to easily dump directly to memory (e.g. C structs).


## State structure
A client centric state is returned. This means the player array will always start with your client's player first, though all clients will see all players in the same order.

#### Json Properties

```json
// TODO
```
    

#### Example state

```json
// TODO
```
