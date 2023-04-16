# servers
Source to servers for games and apps that work with FujiNet

- "apod" - Astronomy Picture of the Day fetcher.
  - Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
    convert it to a format suitable for quickly loading on an Atari (e.g.,
    80x192 16 grey shade `GRAPHICS 9`), and make it available via HTTP for
    an Atari with a #FujiNet and its `N:` device.

- "fujinet-game-server" - The beginnings of a multi-player game server that works over UDP.

- "networds" - A server for a two-player word game played via mostly-RESTful HTTP requests.

- "5cardstud" - A collection of client and servers that impliment 5 Card Stud poker game. This is very much a work in progress.
  - Clients
    - "client/pc/python" - PC client, written in Python.
  - Servers
    - "dummy-server/pc/Python" - Json server written in Python, serves random hands for client testing.
    - "server/mock-server" - Json Api server written in Go, emulates much of the game server logic, including player bots to assist in writing/testing persisted play over multiple games on a single client.
