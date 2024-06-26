# two-players

An example TCP server explicitly for two player games. It performs these functions:

1. Open a listening socket on address 0.0.0.0; with port specified by the first argument.
2. Wait for two distinct player connections.
3. Accept each one.
4. Once accepted, check for disconnections, if still connected, send data from socket 1 to socket 2, and vice versa.
5. If any disconnections happen, close both sockets, close the listening socket, and end the server program.
6. Otherwise, Loop back to step 4.

The program is implemented in C, in such a way that it can be read and understood from top to bottom. 

## Building

The program can be simply built by running its Makefile with make:

```sh
make
```

## Using

The server expects a single argument, the port on which to listen. If one is not provided, the program will let you know that you need to provide one.

### Example: Listening for connections on port 6502

To listen for two connections on port 6502:

```sh
./two-players 6502
```

The server will respond with:

```sh
Socket successfully created..
Socket successfully bound.
```

You can test this, with a program such as netcat(1):

Open two terminals, from each one do:

```sh
nc localhost 6502
```

**NOTE:** Replace localhost with the correct host name, if you're not running netcat from the same machine.

When each of the two connections are made, you will see the following from the terminal running the two-players server:

```sh
Socket successfully created..
Socket successfully bound.
reflect()
```

This means that both connections  have been made, and you can now type in the two netcat windows. Pressing RETURN will send the line you just typed, to the other connection, and vice-versa.

If one or both of the connections disconnect, the server will stop. You can re-run it to accept more connections.

## Installing

This is intended to be installed as part of a system service, such as with a systemd unit. 

### Example systemd unit

```service
[Unit]
Description=Example 2players service

[Service]
Type=simple
ExecStart=/usr/local/sbin/two-players 6502
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
```

This can subsequently be installed into your systemd services area:

```sh
sudo cp example.service /etc/systemd/system
```

and started with the following commands:

```sh
sudo systemctl daemon-reload
sudo systemctl start example
```

and it can be enabled to load at start-up with:

```sh
sudo systemctl enable example
```

## Author

This was written by Thomas Cherryhomes <thom dot cherryhomes at gmail dot com>
