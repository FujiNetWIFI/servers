package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const (
	OFS_ROUND        = 41
	OFS_ACTIVE       = 42
	OFS_ACTIVE_HAND  = 43
	OFS_MOVE_TIME    = 44
	OFS_VIEWING      = 45
	OFS_DEALER       = 46
	OFS_DEALER_TOTAL = 67
	OFS_DEALER_FLAGS = 68
	OFS_MOVE_COUNT   = 69
	OFS_MOVES        = 70
	OFS_PLAYER_COUNT = 135
	OFS_PLAYERS      = 136
	PLAYER_SIZE      = 65
	HAND_SIZE        = 22
	// Within a player
	OFS_P_STATUS     = 9
	OFS_P_PURSE      = 10
	OFS_P_MOVE       = 12
	OFS_P_HAND_COUNT = 20
	OFS_P_HANDS      = 21
)

var routerOnce sync.Once
var router http.Handler

func get(t *testing.T, path string) []byte {
	t.Helper()
	routerOnce.Do(func() {
		setupServer()
		router = setupRouter()
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d", path, rec.Code)
	}
	return body
}

func cstr(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

func TestBinaryLayout(t *testing.T) {
	useFastTimers(t)
	q := "?table=dev1&player=Wire&bin=1"

	b := get(t, "/state"+q+"&be=1")
	expectEqual(t, "size", len(b), OFS_PLAYERS+2*PLAYER_SIZE)
	expectEqual(t, "round", b[OFS_ROUND], byte(ROUND_BETTING))
	expectEqual(t, "my turn", int8(b[OFS_ACTIVE]), 0)
	expectEqual(t, "move count", b[OFS_MOVE_COUNT], 4)
	expectEqual(t, "move 0 code", cstr(b[OFS_MOVES:]), "b1")
	expectEqual(t, "move 0 name", cstr(b[OFS_MOVES+3:]), "bet 10")
	expectEqual(t, "move 3 name", cstr(b[OFS_MOVES+3*13+3:]), "bet 100")
	expectEqual(t, "player count", b[OFS_PLAYER_COUNT], 2)
	p := b[OFS_PLAYERS:]
	expectEqual(t, "name lowercased", cstr(p), "wire")
	expectEqual(t, "purse big endian", binary.BigEndian.Uint16(p[OFS_P_PURSE:]), 1000)
	bot := b[OFS_PLAYERS+PLAYER_SIZE:]
	expectEqual(t, "bot name", cstr(bot), "clyd bot")

	le := get(t, "/state"+q)
	expectEqual(t, "purse little endian", binary.LittleEndian.Uint16(le[OFS_PLAYERS+OFS_P_PURSE:]), 1000)

	b = get(t, "/move/b2"+q+"&be=1")
	if b[OFS_ROUND] == ROUND_PLAYING {
		expectEqual(t, "hole card masked", strings.HasSuffix(cstr(b[OFS_DEALER:]), "??"), true)
		expectEqual(t, "dealer shows one card", len(cstr(b[OFS_DEALER:])), 4)
	}
	if b[OFS_DEALER_TOTAL] == 0 {
		t.Errorf("no dealer total")
	}
	p = b[OFS_PLAYERS:]
	expectEqual(t, "status", p[OFS_P_STATUS], STATUS_PLAYING)
	expectEqual(t, "purse after bet", binary.BigEndian.Uint16(p[OFS_P_PURSE:]), 980)
	expectEqual(t, "hand count", p[OFS_P_HAND_COUNT], 1)
	h := p[OFS_P_HANDS:]
	expectEqual(t, "hand bet", binary.BigEndian.Uint16(h), 20)
	expectEqual(t, "two cards", len(cstr(h[5:])), 4)
	if h[2] < 2 || h[2] > 21 {
		t.Errorf("hand total %d", h[2])
	}
	empty := p[OFS_P_HANDS+HAND_SIZE:]
	expectEqual(t, "second hand empty", binary.BigEndian.Uint16(empty)+uint16(empty[2]), 0)

	get(t, "/leave?table=dev1&player=Wire")
}

func TestHashShortcut(t *testing.T) {
	useFastTimers(t)
	var cs clientState
	if err := json.Unmarshal(get(t, "/state?table=dev2&player=Hash"), &cs); err != nil {
		t.Fatal(err)
	}
	// moveTime ticks every second, so ask from a viewer that has no timer
	var view clientState
	json.Unmarshal(get(t, "/state?table=dev2"), &view)
	expectEqual(t, "unchanged", string(get(t, "/state?table=dev2&hash="+view.Hash)), `"1"`)
	get(t, "/leave?table=dev2&player=Hash")
}

func TestTablesList(t *testing.T) {
	var live, dev []GameTable
	json.Unmarshal(get(t, "/tables"), &live)
	json.Unmarshal(get(t, "/tables?dev=1"), &dev)
	expectEqual(t, "live tables", len(live), 4)
	expectEqual(t, "dev tables", len(dev), MAX_PLAYERS-1)
	expectEqual(t, "max players", live[0].MaxPlayers, MAX_PLAYERS)

	b := get(t, "/tables?bin=1")
	expectEqual(t, "binary count", b[0], 4)
	expectEqual(t, "binary size", len(b), 1+4*(9+21+6))
	expectEqual(t, "first table", cstr(b[1:]), "basement")
}

func TestMoveOutOfTurnIgnored(t *testing.T) {
	useFastTimers(t)
	get(t, "/state?table=dev3&player=A")
	get(t, "/state?table=dev3&player=B")
	var cs clientState
	json.Unmarshal(get(t, "/move/B1?table=dev3&player=B"), &cs)
	if len(cs.Players[0].Hands) != 0 {
		t.Errorf("B bet out of turn")
	}
	get(t, "/leave?table=dev3&player=A")
	get(t, "/leave?table=dev3&player=B")
}
