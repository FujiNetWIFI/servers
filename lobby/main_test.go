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

func setupRouter() *gin.Engine {

	router := gin.Default()

	router.GET("/viewFull", ShowServers)
	router.GET("/view", ShowServersMinimised)
	router.POST("/server", UpsertServer)

	return router
}

func REMOVEassertHTTPAnswerText(w *httptest.ResponseRecorder, HTTPCode int, HTTPBody string) (err []error) {

	if w.Code != HTTPCode {
		err = append(err, fmt.Errorf("Expecing HTTP %d, received HTTP %d", HTTPCode, w.Code))
	}

	if w.Body.String() != HTTPBody {
		err = append(err, fmt.Errorf("Expecting body: %s, received %s", w.Body.String(), HTTPBody))
	}

	return err

}

func assertHTTPAnswerJSON(w *httptest.ResponseRecorder, HTTPCode int, HTTPBody string) (err []error) {
	if w.Code != HTTPCode {
		err = append(err, fmt.Errorf("Expecing HTTP %d, received HTTP %d", HTTPCode, w.Code))
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
	req, _ := http.NewRequest("POST", "/server", bytes.NewBuffer([]byte(`{
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
    }`)))
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 201, `{"message":"Server correctly updated","success":true}`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}

func TestViewInsertAndRetrieveServer1(t *testing.T) {
	w := httptest.NewRecorder()
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

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/view", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 400, `{"message":"You need to submit a platform","success":false}`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/view?platform=atari", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 200, `[{"g":"Super Chess","t":1,"u":"http://chess.rogersm.net/server","c":"http://chess.rogersm.net/atarichess.xex","s":"chess.rogersm.net","r":"eu","o":1,"m":2,"p":1}]`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/view?platform=spectrum", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 200, `[{"g":"Super Chess","t":1,"u":"http://chess.rogersm.net/server","c":"http://chess.rogersm.net/speccychess.xex","s":"chess.rogersm.net","r":"eu","o":1,"m":2,"p":1}]`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}

func TestViewFullInsertAndRetrieveServer1(t *testing.T) {
	w := httptest.NewRecorder()
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

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/viewFull", nil)
	ROUTER.ServeHTTP(w, req)

	if errors := assertHTTPAnswerJSON(w, 200, `[{
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
    }]`); errors != nil {
		for _, err := range errors {
			t.Errorf("%s %s %s", req.Method, req.URL.Path, err)
		}
	}

}
