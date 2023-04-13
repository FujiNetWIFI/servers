# MIDIMaze Server

This is a work in progress server to enable easier MIDIMaze connections. Currently (2023-04-12), this requires custom built firmware from the `midimaze` branch of `fujinet-platformio`.

## Objective

The intent of the server is to allow UDP connections from clients without the need for the client to open a port on their firewall. Clients send packets to the server and the server routes them in a ring (A sends to B, B sends to C, C sends to A). The server will wait until all clients are connected before routing any packets.

## Building

There is no Makefile. You can build it on a unixy system to create a binary called `midimaze-serv`:

```
gcc -o midimaze-serv main.c
```

## Usage

```
Usage: midimaze-serv <port> <number of clients>
```

The server takes 2 arguments on the command line:
 * <port>: UDP port (this must be open on the server firewall)
 * <number of clients>: the number of clients to wait for

FujiNet clients should follow this process for connecting to a running server:
 * If using MIDIMaze on Cartridge
 1. Insert cart and power on the Atari
 2. On a PC/Phone/Tablet browser, connect to the FujiNet web interface
 3. Eject any disk in D1 and select `MOUNT ALL SLOTS` under MOUNT LIST
 4. Press `RESET` on the Atari and MIDIMaze will load
 5. On the FujiNet web interface, enter the address of the server into the UDP STREAM box and click `Save`
 6. In MIDIMaze select the `MIDIMATE` option
 7. Follow this process (or the process for ATR image) on each client machine

 * If using MIDIMaze ATR Disk Image
  1. Boot to FujiNet CONFIG
  2. Mount MIDIMaze ATR to D1 and boot the disk
  3. On the FujiNet web interface, enter the address of the server into the UDP STREAM box and click `Save`
  4. In MIDIMaze select the `MIDIMATE` option
  5. Follow this process (or the process for Cartridge) on each client machine

As each client connects to the server it will display a message and when all are connected it will indicate this as well:

```
Client 1 registered.
Client 2 registered.
Client 3 registered.
All clients registered.
```

The last client to connect becomes the Master and has control of the game settings. When all clients have connected, packets will begin to appear on the server and be routed in the ring:

```
Packet #1 from address 192.168.1.70:5004 to 192.168.1.70:5004
PKT: 00
Packet #1 from address 192.168.1.33:5004 to 192.168.1.33:5004
PKT: 00
```

## Notes

During testing some timeouts did occur during game play and the game will exit to the main menu. You can start a new game by selecting `MIDIMATE` again from the menu without the need to reconnect each FujiNet to the server as outlined above. The connection remains active. Some games would last for 15 minutes, and some would only last for a few minutes.