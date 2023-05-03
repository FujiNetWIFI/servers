package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nsf/jsondiff" // TODO: can we use some core golang functionality?
)

var ROUTER = setupRouter()

var GameServersIn = []string{
	`{
        "game": "Super Chess",
        "gametype": 1,
        "server": "chess.rogersm.net",
        "serverurl": "http://chess.rogersm.net/server",
        "region": "eu",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {"platform":"atari", "url":"http://chess.rogersm.net/atarichess.xex" },
            {"platform": "spectrum", "url":"http://chess.rogersm.net/speccychess.xex"},
            {"platform": "c64", "url":"http://chess.rogersm.net/c64chess.xex"}

        ]
    }`,
	`{
        "game": "Battleship",
        "gametype": 1,
        "region": "au",
        "server": "8bitBattleship.com",
        "serverurl": "https://8bitBattleship.com/battlebots",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {"platform":"atari", "url":"https://8bitBattleship.com/atariship.xex" },
            {"platform": "spectrum", "url":"https://8bitBattleship.com/specship.xex"},
            {"platform": "c64", "url":"https://8bitBattleship.com/c64ship.xex"},
            {"platform": "amiga", "url":"https://8bitBattleship.com/amigaship.xex"}
        ]
	}`,
	`{
        "game": "5 CARD STUD",
        "gametype": 1,
        "region": "us",
        "server": "erichomeserver.com",
        "serverurl": "tcp://thomcorner.com/pokerbots",
        "status": "online",
        "maxplayers": 8,
        "curplayers": 1,
        "clients": [
            {"platform":"atari", "url":"tcp://thomcorner.com/clientus/ataripoker.xex" },
            {"platform": "spectrum", "url":"tcp://thomcorner.com/clientus/specpoker.xex"},
            {"platform": "c64", "url":"tcp://thomcorner.com/clientus/c64poker.xex"},
            {"platform": "lynx", "url":"tcp://thomcorner.com/clientus/lynxpoker.xex"}
        ]
	}`,
	`{
        "game": "Battleship",
        "gametype": 1,
        "region": "apac",
        "server": "8bitBattleship.com",
        "serverurl": "https://8bitBattleship.com/battlehuman",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 0,
        "clients": [
            {"platform":"atari", "url":"https://8bitBattleship.com/atariship.xex" },
            {"platform": "spectrum", "url":"https://8bitBattleship.com/specship.xex"},
            {"platform": "c64", "url":"https://8bitBattleship.com/c64ship.xex"},
            {"platform": "amiga", "url":"https://8bitBattleship.com/amigaship.xex"},
            {"platform": "vic20", "url":"https://8bitBattleship.com/vic20ship.xex"}

        ]
    }`,
	`{
        "game": "5 CARD STUD",
        "gametype": 1,
        "region": "all",
        "server": "erichomeserver.com",
        "serverurl": "tcp://thomcorner.com/server5",
        "status": "offline",
        "maxplayers": 3,
        "curplayers": 0,
        "clients": [
            {"platform":"atari", "url":"tcp://thomcorner.com/ataripoker.xex" },
            {"platform": "spectrum", "url":"tcp://thomcorner.com/specpoker.xex"},
            {"platform": "c64", "url":"tcp://thomcorner.com/c64poker.xex"},
            {"platform": "amiga", "url":"tcp://thomcorner.com/amigapoker.xex"}
        ]
    }`,
	`{
        "game": "5 CARD STUD",
        "gametype": 1,
        "region": "vatican",
        "server": "thomcorner.com",
        "serverurl": "tcp://thomcorner.com/pokerhuman",
        "status": "online",
        "maxplayers": 8,
        "curplayers": 4,
        "clients": [
            {"platform":"atari", "url":"tcp://thomcorner.com/clt/ataripoker.xex" }  
        ]
    }`}

var GameServersOut = `"[
    {
        "game": "5 CARD STUD",
        "gametype": 1,
        "server": "thomcorner.com",
        "region": "vatican",
        "serverurl": "tcp://thomcorner.com/pokerhuman",
        "status": "online",
        "maxplayers": 8,
        "curplayers": 4,
        "clients": [
            {
                "platform": "atari",
                "url": "tcp://thomcorner.com/clt/ataripoker.xex"
            }
        ]
    },
    {
        "game": "Battleship",
        "gametype": 1,
        "server": "8bitBattleship.com",
        "region": "apac",
        "serverurl": "https://8bitBattleship.com/battlehuman",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 0,
        "clients": [
            {
                "platform": "atari",
                "url": "https://8bitBattleship.com/atariship.xex"
            },
            {
                "platform": "spectrum",
                "url": "https://8bitBattleship.com/specship.xex"
            },
            {
                "platform": "c64",
                "url": "https://8bitBattleship.com/c64ship.xex"
            },
            {
                "platform": "amiga",
                "url": "https://8bitBattleship.com/amigaship.xex"
            },
            {
                "platform": "vic20",
                "url": "https://8bitBattleship.com/vic20ship.xex"
            }
        ]
    },
    {
        "game": "5 CARD STUD",
        "gametype": 1,
        "server": "erichomeserver.com",
        "region": "us",
        "serverurl": "tcp://thomcorner.com/pokerbots",
        "status": "online",
        "maxplayers": 8,
        "curplayers": 1,
        "clients": [
            {
                "platform": "atari",
                "url": "tcp://thomcorner.com/clientus/ataripoker.xex"
            },
            {
                "platform": "spectrum",
                "url": "tcp://thomcorner.com/clientus/specpoker.xex"
            },
            {
                "platform": "c64",
                "url": "tcp://thomcorner.com/clientus/c64poker.xex"
            },
            {
                "platform": "lynx",
                "url": "tcp://thomcorner.com/clientus/lynxpoker.xex"
            }
        ]
    },
    {
        "game": "Battleship",
        "gametype": 1,
        "server": "8bitBattleship.com",
        "region": "au",
        "serverurl": "https://8bitBattleship.com/battlebots",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {
                "platform": "atari",
                "url": "https://8bitBattleship.com/atariship.xex"
            },
            {
                "platform": "spectrum",
                "url": "https://8bitBattleship.com/specship.xex"
            },
            {
                "platform": "c64",
                "url": "https://8bitBattleship.com/c64ship.xex"
            },
            {
                "platform": "amiga",
                "url": "https://8bitBattleship.com/amigaship.xex"
            }
        ]
    },
    {
        "game": "Super Chess",
        "gametype": 1,
        "server": "chess.rogersm.net",
        "region": "eu",
        "serverurl": "http://chess.rogersm.net/server",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {
                "platform": "atari",
                "url": "http://chess.rogersm.net/atarichess.xex"
            },
            {
                "platform": "spectrum",
                "url": "http://chess.rogersm.net/speccychess.xex"
            },
            {
                "platform": "c64",
                "url": "http://chess.rogersm.net/c64chess.xex"
            }
        ]
    },
    {
        "game": "5 CARD STUD",
        "gametype": 1,
        "server": "erichomeserver.com",
        "region": "all",
        "serverurl": "tcp://thomcorner.com/server5",
        "status": "offline",
        "maxplayers": 3,
        "curplayers": 0,
        "clients": [
            {
                "platform": "atari",
                "url": "tcp://thomcorner.com/ataripoker.xex"
            },
            {
                "platform": "spectrum",
                "url": "tcp://thomcorner.com/specpoker.xex"
            },
            {
                "platform": "c64",
                "url": "tcp://thomcorner.com/c64poker.xex"
            },
            {
                "platform": "amiga",
                "url": "tcp://thomcorner.com/amigapoker.xex"
            }
        ]
    }
]`

func setupRouter() *gin.Engine {

	router := gin.Default()

	router.GET("/viewFull", ShowServers)
	router.GET("/view", ShowServersMinimised)
	router.POST("/server", UpsertServer)

	return router
}

func assertHTTPAnswerJSON(w *httptest.ResponseRecorder, HTTPCode int, HTTPBody string) (err []error) {
	if w.Code != HTTPCode {
		err = append(err, fmt.Errorf("Expecing HTTP %d, received HTTP %d", HTTPCode, w.Code))
	}

	if w.Body.String() == HTTPBody {
		return err
	}

	opts := jsondiff.DefaultJSONOptions()
	ret, diff := jsondiff.Compare(w.Body.Bytes(), []byte(HTTPBody), &opts)

	// This includes ExactMatch and SupersededMatch (useful for the ping value)
	// but SupersededMatch may accept some errors.
	// TODO: use ExactMatch but doing a manual check for the ping time that will be different.
	if ret == jsondiff.NoMatch {
		err = append(err, fmt.Errorf(diff))
	}

	return err
}

func TestEmptyViewFull(t *testing.T) {

	w := httptest.NewRecorder()
	w.Header().Add("Content-Type", "application/json")

	req, _ := http.NewRequest("GET", "/viewFull", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 404, `{"message":"No servers available","success":false}`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}
}

func TestEmptyView(t *testing.T) {

	w := httptest.NewRecorder()
	w.Header().Add("Content-Type", "application/json")

	req, _ := http.NewRequest("GET", "/view", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 404, `{"message":"No servers available","success":false}`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}
}

func TestInsertServer1(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Add("Content-Type", "application/json")

	req, _ := http.NewRequest("POST", "/server", bytes.NewBuffer([]byte(`{
        "game": "Super Chess",
        "gametype": 1,
        "server": "chess.rogersm.net",
        "serverurl": "http://chess.rogersm.net/server",
        "region": "eu",
        "status": "online",
        "maxplayers": 2,
        "curplayers": 1,
        "clients": [
            {"platform":"atari", "url":"http://chess.rogersm.net/atarichess.xex" },
            {"platform": "spectrum", "url":"http://chess.rogersm.net/speccychess.xex"}
        ]
    }`)))
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 201, `{"message":"Server correctly updated","success":true}`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}

func TestViewFullInsertAndRetrieveServerN(t *testing.T) {

	for _, ServerJson := range GameServersIn {

		w := httptest.NewRecorder()
		w.Header().Add("Content-Type", "application/json")
		req, _ := http.NewRequest("POST", "/server", bytes.NewBuffer([]byte(ServerJson)))
		ROUTER.ServeHTTP(w, req)

		if errors := assertHTTPAnswerJSON(w, 201, `{"message":"Server correctly updated","success":true}`); errors != nil {
			for _, err := range errors {
				t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
			}
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/viewFull", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 200, GameServersOut); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}

func TODOTestViewInsertAndRetrieveServerN(t *testing.T) {

	for _, ServerJson := range GameServersIn {

		w := httptest.NewRecorder()
		w.Header().Add("Content-Type", "application/json")
		req, _ := http.NewRequest("POST", "/server", bytes.NewBuffer([]byte(ServerJson)))
		ROUTER.ServeHTTP(w, req)

		if errors := assertHTTPAnswerJSON(w, 201, `{"message":"Server correctly updated","success":true}`); errors != nil {
			for _, err := range errors {
				t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
			}
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/view", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 200, GameServersOut); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}
