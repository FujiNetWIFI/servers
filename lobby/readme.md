# Lobby Spec

This is intended to find agreement on general behavior of the Lobby and Game Servers. More digestable documentation would follow.

##  High Level Flow

1. **Game Type** registeration is done in code, as this seldom changes.
2. Game Server sends **registration json** to Lobby Server on startup for each instance. A game server may support multiple instances, in which case it will send an array, since from the Lobby's perspective, each server is a stand alone instance.
3. The game server state is maintained by the Lobby Server, by combination of:
    - Lobby calls **serverPingUrl** to check status of each game server
    - Game server updates Lobby with status as state changes
    - If game server does not respond to ping in N pings, it gets eventually removed from the list

4. Lobby Client calls Lobby Server, specifying platform (e.g. `?platform=atari` ) to get state json.
5. (Optional) the Lobby could have a simple Html table view with all details for looking at in browser.

## Reserved App Keys
 App Key 1/1/* is reserved for Lobby Client <-> Game client interaction
 * 1/1/0 - Username (common across Lobby and all game clients)
 * 1/1/N - server Url Endpoint for Game Type N*

\*Supports up to 255 unique game types, which should be more than sufficient. Over 255 game types would be a great problem to have.


## Game Type Registration
* Each game type (e.g. "5 Card Stud", "Battleship") would be be assigned a unique number that corresponds with the App Key number that is hard coded in each client.
* The Lobby Client will read the proper gameType from the Lobby Server and set AppKey **1/1/[gameType]** before mounting and loading the client application. The client application will then be hard coded to read **1/1/[gameType]** for the endpoint url. This way, the endpoint url will survive crashes/restarts, and if you just start up your client after playing other games, each game client will remember the last server it was connected to.
* To start out, given low frequency of new game types, we may store this as a json file BitBucket, and added with PR. The Lobby Server could just read it on start-up (HTTP GET a well known Bit Bucket location?).

Example gameType representation:

```
[
    { 
        "gameType": 1, 
        "name": "5 Card Stud",
        "clients": [
            "adam" :  "TNFS://tnfs.carr-designs.com/5card.ddp",
            "atari" : "TNFS://tnfs.carr-designs.com/5card.xex", ..
        ]
    }, ..
]
```





### Game Server Registration Json

Each game server on startup would send an array of server isntances (it could be 1, or if the same server supports multiple instances, one object per instance). The Lobby views each instance as just another server. Each **serverUrl** is used as the unique ID for the server and is used as the ID for this server for any updates.

* serverUrl - **(Unique Key)** The Lobby will save this in appKey for for the client. e.g. `HTTPS://x.com/x` or `TCP://x.x.x.x:6502/`
* gameType - Registered number for game type
* server - Friendly name of server instance
* serverPingUrl - Lobby Client will ping this url to get server status
* TTL - (optional) If specified, the Lobby will not ping the game server if it receives an update within the specified number of seconds. Used if the game server sends regular updates to the Lobby as opposed to relying solely on polling.
* region - Just a thought - would a region/country code of server location be helpful? US (USA), DE (Germany), etc.
```
[
    {
        "gameType": 1, 
        "server": "Mock Server - 4 Players",
        "region" : "US",
        "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
        "serverPingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table1"
    } 
]
```

## Lobby Server List

The Lobby Server would keep a list of each game server instance, including all registration
properties plus the following, which would be updated via polling:

 * status - "Online" or "Offline"
 * maxPlayers - Max allowed players on server
 * curPlayers - Current human players on server

Example internal Lobby Server state.

```
{
    "gameType": 1, 
    "region": "US",
    "server": "Mock Server - 4 Players",
    "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
    "serverPingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table1",
    "status": "Online",
    "maxPlayers": 4,
    "curPlayers": 0
} ,
{
    "gameType": 1, 
    "region": "US",
    "server": "Mock Server - 8 Players",
    "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table2&count=8",
    "serverPingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table2",
    "status": "Online",
    "maxPlayers": 8,
    "curPlayers": 1,
    "TTL" : 3600
} 
```

## Game Server Status Updates

1. The Lobby Server may poll (either on interval or on-demand) each game server, which is expected to return **maxPlayers** and **curPlayers**.
2. The game server can specify a TTL during registration, and then POST the json object to the Lobby whenever player information changes. It must including the **serverUrl** so the Lobby knows which server to update. As long as a game server updates the Lobby within the TTL time, the Lobby will have no need to poll the game server.

```
{
    "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table2&count=8",
    "maxPlayers" : 4,
    "curPlayers": 1
}
```

## Lobby Client - Flat View

The Json sent to the Lobby client would essentially be the Lobby server state, with the following changes:
 - include the **game** name and **gameClientUrl** for the appropriate platform
 - exclude **serverPingUrl** and **TTL**

There is obvious duplication of data in this flat view, which could be split into different calls if needed for the 8-bit clients as we iterate (e.g. getGameTypes, getServers).

    
```
{
    "gameType": 1,
    "game": "5 Card Stud",
    "region": "US",
    "gameClientUrl" : "TNFS://tnfs.carr-designs.com/5card.xex",
    "server": "Mock Server - 4 Players",
    "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
    "status": "Online",
    "maxPlayers": 4,
    "curPlayers": 0
} ,
{
    "gameType": 1, 
    "game": "5 Card Stud",
    "region": "US",
    "gameClientUrl" : "TNFS://tnfs.carr-designs.com/5card.xex",
    "server": "Mock Server - 8 Players",
    "serverUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table2&count=8",
    "status": "Offline",
    "maxPlayers": 0,
    "curPlayers": 0
} 
```

The lobby client would then:
1. Set AppKey **1/1/[gameType]** to **serverUrl**
2. Mount **gameClientUrl** and restart the computer
