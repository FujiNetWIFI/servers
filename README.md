# servers
Source to servers for games and apps that work with FujiNet

- "Lobby" - for FujiNet Game system - moved to new repo: https://github.com/FujiNetWIFI/fujinet-lobby

- "apod" - Astronomy Picture of the Day fetcher.
  - Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
    convert it to a format suitable for quickly loading on an Atari (e.g.,
    80x192 16 grey shade `GRAPHICS 9`), and make it available via HTTP for
    an Atari with a #FujiNet and its `N:` device.

- "cherrysrv" - A simple chat multi-channel server that works over TCP.

- "fujinet-game-server" - The beginnings of a multi-player game server that works over UDP.

- "kaplow" - A server to play a scorched earth like game.

- "networds" - A server for a two-player word game played via mostly-RESTful HTTP requests.

- "5cardstud" - A Multi-player/Multi-Platform implementation of 5 Card Stud Poker
  - Server
    - [fujinet-game-system/5cardstud/server](fujinet-game-system/5cardstud/server) - Game server written in Go
  - Clients
    - [fujinet-apps/5cardstud](https://github.com/FujiNetWIFI/fujinet-apps/tree/master/5cardstud) - 8 bit clients (C, FastBasic)
  

- "fujitzee" - A Multi-player/Multi-Platform implementation of Yahtzee
  - Server
    - [fujinet-game-system/fujzee/server](fujinet-game-system/fujitzee/server) - Game server written in Go
  - Clients
    - [github.com/FujiNetWIFI/fujinet-fujitzee](https://github.com/FujiNetWIFI/fujinet-fujitzee) -  8 bit clients


- "battleship" - A multiplayer pimplementation of the classic sea battle game
  - Server
    - [fujinet-game-system/battleship](fujinet-game-system/battleship) - Game server written in Go
  - Clients
    - TBD