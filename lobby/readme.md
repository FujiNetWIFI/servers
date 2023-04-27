# Lobby Spec

##  High Level Flow

1. **Game Type** registeration is done in code, as this seldom changes.
2. Game Server sends **registration json** to Lobby Server on startup for each instance
3. Lobby Server calls **instancePingUrl** to check status of each game instance.
    - Perhaps if game server does not respond to ping in N pings, it gets removed from the list
4. Lobby Client calls Lobby Server, specifying platform (e.g. `?platform=atari` ) to get state json.
5. (Optional) the Lobby could also have a simple Html table view with all details for looking at in browser.

## Reserved App Keys
 App Key 1/1/* is reserved for Lobby Client <-> Game client interaction
 * 1/1/0 - Username (common across Lobby and all game clients)
 * 1/1/N - Instance Url Endpoint for Game Type N*

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

Each game server on startup would send an array of instances (it could be 1, or if the same server supports multiple instances, one object per instance)

* gameType - Registered number for game type
* instance - Friendly name of server instance
* instanceUrl - Lobby Client will save this in appKey for for the client
* instancePingUrl - Lobby Client will ping this url to get instance status
* region - Just a thought - would a region/country code of server location be helpful? US (USA), DE (Germany), etc.

```
[
    {
        "gameType": 1, 
        "instance": "Mock Server - 4 Players",
        "region" : "US",
        "instanceUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
        "instancePingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table1"
    } 
]
```

## Lobby Server Server List

The Lobby Server would keep a list of each game server instance, including all registration
properties plus the following, which would be updated via polling:

 * status - "Online" or "Offline"
 * maxPlayers - Max allowed players on instance
 * curPlayers - Current human players on instance

Example internal Lobby Server state.

```
{
    "gameType": 1, 
    "region": "US",
    "instance": "Mock Server - 4 Players",
    "instanceUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
    "instancePingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table1",
    "status": "Online",
    "maxPlayers": 4,
    "curPlayers": 0
} ,
{
    "gameType": 1, 
    "region": "US",
    "instance": "Mock Server - 8 Players",
    "instanceUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table2&count=8",
    "instancePingUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/ping?table=table2",
    "status": "Online",
    "maxPlayers": 8,
    "curPlayers": 1
} 
```

## Heartbeat Call

The Lobby Server would poll (either on interval or on-demand) each game server, which is expected to return **maxPlayers** and **curPlayers**:

{
    "maxPlayers" : 4,
    "curPlayers": 1
}


## Lobby Client - Flat View

The Json sent to the Lobby client would essentially be the same state, with the following changes:
 - include the **game** name and **gameClientUrl** for the appropriate platform
 - no **instancePingUrl**

There is obvious duplication of data in this flat view, which could be split into different calls if needed for the 8-bit clients as we iterate (e.g. getGameTypes, getServers).

    
```
{
    "gameType": 1,
    "game": "5 Card Stud",
    "region": "US",
    "gameClientUrl" : "TNFS://tnfs.carr-designs.com/5card.xex",
    "instance": "Mock Server - 4 Players",
    "instanceUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table1&count=4",
    "status": "Online",
    "maxPlayers": 4,
    "curPlayers": 0
} ,
{
    "gameType": 1, 
    "game": "5 Card Stud",
    "region": "US",
    "gameClientUrl" : "TNFS://tnfs.carr-designs.com/5card.xex",
    "instance": "Mock Server - 8 Players",
    "instanceUrl" : "https://mock-server-7udvkexssq-uc.a.run.app/state?table=table2&count=8",
    "status": "Offline",
    "maxPlayers": 0,
    "curPlayers": 0
} 
```

The lobby client would then:
1. Set AppKey **1/1/[gameType]** to **instanceUrl**
2. Mount **gameClientUrl** and restart the computer
