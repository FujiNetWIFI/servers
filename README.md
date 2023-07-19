# servers
Source to servers for games and apps that work with FujiNet

- "apod" - Astronomy Picture of the Day fetcher.
  - Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
    convert it to a format suitable for quickly loading on an Atari (e.g.,
    80x192 16 grey shade `GRAPHICS 9`), and make it available via HTTP for
    an Atari with a #FujiNet and its `N:` device.

- "cherrysrv" - A simple chat multu-channel server that works over TCP.

- "fujinet-game-server" - The beginnings of a multi-player game server that works over UDP.

- "networds" - A server for a two-player word game played via mostly-RESTful HTTP requests.

- "5cardstud" - A Multi-player/Multi-Platform Poker Server and Clients that impliment 5 Card Stud poker game. This is very much a work in progress.
  - Clients
    - "client/pc/python" - PC client, written in Python.
  - Servers
    - "dummy-server/pc/Python" - Json server written in Python, serves random hands for client testing.
    - "[server/mock-server](5cardstud/server/mock-server)" - Json Api server written in Go. It started as a mock server for the purpose of writing 5 Card Stud clients and migrated into a full server supporting multiple clients, with bots. It still supports mock tables to assist in writing/testing new clients.
