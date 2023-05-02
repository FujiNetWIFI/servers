Some testing scenarios:

GET localhost:8080/viewFull --> HTTP/1.1 404 Not Found --> OK
GET localhost:8080/view --> HTTP/1.1 404 Not Found --> OK

curl -X POST http://localhost:8080/server  -H 'Content-Type: application/json' -d '    {
        "game": "Super Chess",
        "gametype": 1,
        "server": "chess.rogersm.net",
        "serverURL": "http://chess.rogersm.net/server",
        "region": "eu",
        "instance": "Table A",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {"platform":"atari", "url":"http://chess.rogersm.net/atarichess.xex" },
            {"platform": "spectrum", "url":"http://chess.rogersm.net/speccychess.xex"}
        ]
    }'

--> {"message":"Server correctly updated","success":true} --> OK

GET http://localhost:8080/view --> HTTP/1.1 400 Bad Request (you need to submit a platform) --> OK
GET localhost:8080/viewFull HTTP/1.1 200 OK (shows Super Chess) --> OK
GET localhost:8080/view?platform=atari --> HTTP/1.1 200 OK (shows Super Chess minimised) --> OK
GET localhost:8080/view?platform=spectru --> HTTP/1.1 404 NOT FOUND --> {"message":"No servers available for spectru","success":false}