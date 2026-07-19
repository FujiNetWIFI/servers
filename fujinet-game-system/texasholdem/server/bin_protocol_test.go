package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Byte layout of the packed structs in the 8-bit client's src/misc.h.
// These constants ARE the wire contract - a change here must be mirrored in the
// client repo (fujinet-texasHoldEm/src/misc.h) and vice versa.
const (
	binOffRound      = 81  // after lastResult[81]
	binOffPot        = 82  // uint16
	binOffActive     = 84  // int8
	binOffMoveTime   = 85  // uint8
	binOffViewing    = 86  // uint8
	binOffCommunity  = 87  // char[11]
	binOffMoveCount  = 98  // uint8
	binOffMoves      = 99  // 5 x 13 bytes { move[3]; name[10] }
	binOffPlayerCnt  = 164 // uint8
	binOffPlayers    = 165 // N x 33 bytes { name[9]; status; bet u16; move[8]; purse u16; hand[11] }
	binSizeValidMove = 13
	binSizePlayer    = 33
)

func cstr(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

type binPlayer struct {
	Name   string
	Status int
	Bet    int
	Move   string
	Purse  int
	Hand   string
}

type binGame struct {
	LastResult   string
	Round        int
	Pot          int
	ActivePlayer int
	MoveTime     int
	Viewing      int
	Community    string
	Moves        []validMove
	Players      []binPlayer
}

// decodeBinGame parses the packed blob exactly as the 6502 client's struct
// overlay would (little-endian)
func decodeBinGame(t *testing.T, b []byte) binGame {
	t.Helper()
	require.GreaterOrEqual(t, len(b), binOffPlayers, "blob must include the full fixed header")
	g := binGame{
		LastResult:   cstr(b[0:81]),
		Round:        int(b[binOffRound]),
		Pot:          int(binary.LittleEndian.Uint16(b[binOffPot:])),
		ActivePlayer: int(int8(b[binOffActive])),
		MoveTime:     int(b[binOffMoveTime]),
		Viewing:      int(b[binOffViewing]),
		Community:    cstr(b[binOffCommunity : binOffCommunity+11]),
	}
	moveCount := int(b[binOffMoveCount])
	for i := 0; i < moveCount; i++ {
		off := binOffMoves + i*binSizeValidMove
		g.Moves = append(g.Moves, validMove{Move: cstr(b[off : off+3]), Name: cstr(b[off+3 : off+13])})
	}
	playerCount := int(b[binOffPlayerCnt])
	require.Equal(t, binOffPlayers+playerCount*binSizePlayer, len(b), "blob length must match player count")
	for i := 0; i < playerCount; i++ {
		off := binOffPlayers + i*binSizePlayer
		g.Players = append(g.Players, binPlayer{
			Name:   cstr(b[off : off+9]),
			Status: int(b[off+9]),
			Bet:    int(binary.LittleEndian.Uint16(b[off+10:])),
			Move:   cstr(b[off+12 : off+20]),
			Purse:  int(binary.LittleEndian.Uint16(b[off+20:])),
			Hand:   cstr(b[off+22 : off+33]),
		})
	}
	return g
}

// TestBinaryStateLayout verifies the bin=1 packed serialization matches the
// 8-bit client's struct layout byte-for-byte
func TestBinaryStateLayout(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 3, 77)

	client := newSimClient(server.URL, tableId, "BinTest", nil)

	// First state call joins and starts the hand
	_, err := client.get(fmt.Sprintf("/state?table=%s&player=BinTest", tableId))
	require.NoError(t, err)

	blob, err := client.get(fmt.Sprintf("/state?table=%s&player=BinTest&bin=1", tableId))
	require.NoError(t, err)

	g := decodeBinGame(t, blob)

	assert.GreaterOrEqual(t, g.Round, 1, "hand should have started")
	assert.LessOrEqual(t, g.Round, 5)
	assert.Equal(t, SB+BB, g.Pot, "pot holds the blinds pre-flop")
	assert.Equal(t, 4, len(g.Players), "3 bots + our client")
	assert.Equal(t, "bintest", g.Players[0].Name, "client is players[0], lowercased")
	assert.Empty(t, g.Community, "no community cards pre-flop")

	// Our hole cards: 4 lowercase chars, valid ranks/suits; opponents masked
	own := g.Players[0].Hand
	require.Len(t, own, 4, "two hole cards as 2-char codes")
	for i := 0; i+1 < len(own); i += 2 {
		assert.Contains(t, "23456789tjqka", string(own[i]), "rank char")
		assert.Contains(t, "cdhs", string(own[i+1]), "suit char")
	}
	for _, p := range g.Players[1:] {
		if p.Status == 1 {
			assert.Equal(t, "????", p.Hand, "opponent hole cards masked")
		}
	}

	// Field caps honored
	for _, m := range g.Moves {
		assert.LessOrEqual(t, len(m.Move), 2, "move codes are 2 chars")
		assert.LessOrEqual(t, len(m.Name), 9, "move names fit the client buffer")
	}
	for _, p := range g.Players {
		assert.LessOrEqual(t, len(p.Name), 8)
		assert.LessOrEqual(t, len(p.Move), 7)
		assert.LessOrEqual(t, len(p.Hand), 10)
		assert.Contains(t, []int{0, 1, 2, 3}, p.Status, "all-in is mapped to playing for 8-bit clients")
	}
}

// TestBinaryFullHand drives a complete hand through the binary protocol only,
// exactly as an 8-bit client would (poll bin state, post 2-char move codes)
func TestBinaryFullHand(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 2, 78)
	client := newSimClient(server.URL, tableId, "A2Client", nil)

	sawCommunity := false
	var final binGame
	for i := 0; i < 4000; i++ {
		blob, err := client.get(fmt.Sprintf("/state?table=%s&player=A2Client&bin=1", tableId))
		require.NoError(t, err)
		g := decodeBinGame(t, blob)

		if len(g.Community) >= 6 {
			sawCommunity = true // flop or later visible as card string
		}
		if g.Round == 5 {
			final = g
			break
		}
		if g.ActivePlayer == 0 && len(g.Moves) > 0 {
			// Pick check/call like a simple client; codes arrive lowercased on the
			// wire and the server accepts them case-insensitively
			code := "fo"
		pick:
			for _, want := range []string{"ch", "ca"} {
				for _, m := range g.Moves {
					if m.Move == want {
						code = want
						break pick
					}
				}
			}
			_, err := client.get(fmt.Sprintf("/move/%s?table=%s&player=A2Client&bin=1", code, tableId))
			require.NoError(t, err)
		}
	}

	require.Equal(t, 5, final.Round, "hand must complete via binary protocol")
	assert.NotEmpty(t, final.LastResult, "result banner present")
	if !strings.Contains(final.LastResult, "default") {
		assert.True(t, sawCommunity, "community cards were dealt and serialized")
		// At showdown, surviving opponents' hands are revealed (no ? chars)
		for _, p := range final.Players {
			if p.Status == 1 {
				assert.NotContains(t, p.Hand, "?", "hands revealed at showdown")
			}
		}
	}
}

// TestBinaryBigEndian verifies be=1 flips uint16 byte order (CoCo clients)
func TestBinaryBigEndian(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 3, 81)
	client := newSimClient(server.URL, tableId, "CocoTest", nil)

	// Join and start the hand, then fetch both endiannesses
	_, err := client.get(fmt.Sprintf("/state?table=%s&player=CocoTest", tableId))
	require.NoError(t, err)
	le, err := client.get(fmt.Sprintf("/state?table=%s&player=CocoTest&bin=1", tableId))
	require.NoError(t, err)
	be, err := client.get(fmt.Sprintf("/state?table=%s&player=CocoTest&bin=1&be=1", tableId))
	require.NoError(t, err)

	require.Equal(t, len(le), len(be), "blob length must not depend on endianness")
	lePot := binary.LittleEndian.Uint16(le[binOffPot:])
	bePot := binary.BigEndian.Uint16(be[binOffPot:])
	assert.Equal(t, lePot, bePot, "pot value identical across endianness")
	assert.Equal(t, uint16(SB+BB), bePot, "pre-flop pot is the blinds")

	// Player purse field too (offset 20 within the player record)
	lePurse := binary.LittleEndian.Uint16(le[binOffPlayers+20:])
	bePurse := binary.BigEndian.Uint16(be[binOffPlayers+20:])
	assert.Equal(t, lePurse, bePurse, "purse value identical across endianness")
}

// TestBinaryTablesLayout verifies /tables?bin=1 matches the client's Tables struct
func TestBinaryTablesLayout(t *testing.T) {
	server, _ := newHTTPTable(t, 2, 79)
	client := newSimClient(server.URL, "", "X", nil)

	blob, err := client.get("/tables?bin=1&dev=1")
	require.NoError(t, err)
	require.NotEmpty(t, blob)

	count := int(blob[0])
	// Table entry: table[9] + name[21] + players[6] = 36 bytes
	require.Equal(t, 1+count*36, len(blob), "blob length matches Tables struct")
	if count > 0 {
		entry := blob[1:37]
		table := cstr(entry[0:9])
		name := cstr(entry[9:30])
		players := cstr(entry[30:36])
		assert.NotEmpty(t, table)
		assert.NotEmpty(t, name)
		assert.Contains(t, players, "/", "players rendered as cur / max")
	}
}
