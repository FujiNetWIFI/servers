## 5 Card Stud Server
This is an incomplete 5 Card Stud server written in GO for the purposes of writing/testing 5 Card Stud clients. This is my first project in GO, so do not expect expert use of the language.  

As this is focused on assisting in designing a client, it currently:
* Provides bots that simulate a game (They will CHECK whenever they can, and occasionally bet/raise or fold)
* Provides end of game detection and starting a new game
* Provides only legal moves to each player
* **Does not** support multiple clients
* **Does not** support all betting/raising
* **Does not** determine the winner of each game

## How to use

The server is **not** real time. Each time you call ``/state`` it will step forward in time, either playing playing a BOT's move, or giving end of round details (with activePlayer set to -1 indicating no play is left). Calling ``/state`` will then begin the next round.

The game is over when **round 5** is sent. The next call to ``/state`` will begin a new game.

The game currently does not determine the winner, so while each player's purse will shrink, nobody will win the pot.

## Endpoints
The following endpoints are available

* GET `/state` - Advance forward (AI/Game Logic) and return updated state
* GET ``/move/[code]`` - Apply your player's move and return updated state. e.g. ``/move/CH`` to "Check", ``/move/BL`` to "Bet 5 (low)".

These are GETs to easily test in the browser in addition to the client. `/move/` also supports POST.

## State Structure
This is highly subject to change, but focused on a low nested structyre and speed of parsing for 8-bit clients.

* `lastResult` - Will be filled in when round=5 to signal the current game is over
* `validMoves` - An array of strings, each string is a 2 character code (to send to `/move`), followed by a space, followed by friendly text to show onscreen in the client.
* `activePlayer` - The currently active player. Your client is always player 0. This will be `-1` at the end of a round (or end of game) to allow the client to show the last move before clearing and starting on the net round.
* `players/status` - The player's current in-game status
    * 0 - Waiting to play the next game (joined the table late - future use)
    * 1 - In Game, playing
    * 2 - In Game, Folded
* `players/hand` - List of 2 digit representation of cards in the player's hand:
    * `??` - A hidden card. Also represents a folded hand when `hand` is just `??` and followed by no other cards
    * First char - Value : 2 to 9, 0=10, J=Jack, Q=Queen, K=King, A=Ace
    * Second char - Suit : C,S,D,H stand for Clubs, Spades, Diamonds, and Hearts (pretty cryptic, I know)
* `players/bet` - The total of the player's bet for the current round
* `players/move` - Friendly text of the player's most recent move this round

```json
{
    "lastResult": "",
    "round": 1,
    "pot": 0,
    "activePlayer": 0,
    "startingPlayer": 0,
    "validMoves": [
        "FO Fold",
        "CH Check",
        "BL Bet 5"
    ],
    "players": [
        {
            "name": "Player",
            "status": 1,
            "bet": 0,
            "move": "",
            "purse": 150,
            "hand": "JDQH"
        },
        {
            "name": "Mozz Bot",
            "status": 1,
            "bet": 0,
            "move": "",
            "purse": 150,
            "hand": "??QS"
        },
        {
            "name": "Thom Bot",
            "status": 1,
            "bet": 0,
            "move": "CHECK",
            "purse": 150,
            "hand": "??JH"
        },
        {
            "name": "Chat GPT",
            "status": 1,
            "bet": 0,
            "move": "CHECK",
            "purse": 150,
            "hand": "??2H"
        }
    ]
}
```
