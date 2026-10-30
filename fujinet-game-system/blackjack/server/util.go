package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// Serializes as json (default), or with bin=1 as the packed structs the 8 bit
// clients read directly. be=1 makes uint16s big endian (CoCo).
func serializeResults(c *gin.Context, obj any) {
	if debugMode {
		jsonBytes, _ := json.MarshalIndent(obj, "", "  ")
		log.Printf("DEBUG %s %s -> %s", c.Request.URL.Path, c.Request.URL.RawQuery, jsonBytes)
	}

	if c.Query("bin") != "1" {
		c.JSON(http.StatusOK, obj)
		return
	}

	var buf []byte
	bigEndian := c.Query("be") == "1"

	// typedef struct { char table[9]; char name[21]; char players[6]; } Table;
	// typedef struct { uint8_t count; Table tables[]; } Tables;
	if tables, ok := obj.([]GameTable); ok {
		buf = append(buf, byte(len(tables)))
		for _, o := range tables {
			buf = appendFixedLengthString(buf, o.Table, 8)
			buf = appendFixedLengthString(buf, o.Name, 20)
			buf = appendFixedLengthString(buf, fmt.Sprintf("%d / %d", o.CurPlayers, o.MaxPlayers), 5)
		}
	}

	// typedef struct { char move[3]; char name[10]; } ValidMove;
	// typedef struct {
	//   uint16_t bet; uint8_t total; uint8_t flags; uint8_t result; char cards[17];
	// } Hand;
	// typedef struct {
	//   char name[9]; uint8_t status; uint16_t purse; char move[8];
	//   uint8_t handCount; Hand hands[2];
	// } Player;
	// typedef struct {
	//   char lastResult[41];
	//   uint8_t round;
	//   int8_t activePlayer;
	//   uint8_t activeHand;
	//   uint8_t moveTime;
	//   uint8_t viewing;
	//   char dealer[21];
	//   uint8_t dealerTotal;
	//   uint8_t dealerFlags;
	//   uint8_t validMoveCount;
	//   ValidMove validMoves[5];
	//   uint8_t playerCount;
	//   Player players[5];
	// } Game;
	if o, ok := obj.(*clientState); ok {
		buf = appendFixedLengthString(buf, o.LastResult, 40)
		buf = append(buf,
			byte(o.Round),
			byte(o.ActivePlayer),
			byte(o.ActiveHand),
			byte(o.MoveTime),
			byte(o.Viewing))
		buf = appendFixedLengthString(buf, o.Dealer, MAX_DEALER_CARDS*2)
		buf = append(buf, byte(o.DealerTotal), byte(o.DealerFlags))

		buf = append(buf, byte(min(len(o.ValidMoves), 5)))
		for i := 0; i < 5; i++ {
			if i < len(o.ValidMoves) {
				buf = appendFixedLengthString(buf, o.ValidMoves[i].Move, 2)
				buf = appendFixedLengthString(buf, o.ValidMoves[i].Name, 9)
			} else {
				buf = appendFixedLengthString(buf, "", 12)
			}
		}

		buf = append(buf, byte(len(o.Players)))
		for _, p := range o.Players {
			buf = appendFixedLengthString(buf, p.Name, 8)
			buf = append(buf, byte(p.Status))
			buf = appendUint16(buf, min(p.Purse, 0xFFFF), bigEndian)
			buf = appendFixedLengthString(buf, p.Move, 7)
			buf = append(buf, byte(len(p.Hands)))
			for i := 0; i < 2; i++ {
				h := clientHand{}
				if i < len(p.Hands) {
					h = p.Hands[i]
				}
				buf = appendUint16(buf, h.Bet, bigEndian)
				buf = append(buf, byte(h.Total), byte(h.Flags), byte(h.Result))
				buf = appendFixedLengthString(buf, h.Cards, MAX_HAND_CARDS*2)
			}
		}
	}

	c.Data(http.StatusOK, "application/octet-stream", buf)
}

func appendUint16(buf []byte, val int, bigEndian bool) []byte {
	if bigEndian {
		return binary.BigEndian.AppendUint16(buf, uint16(val))
	}
	return binary.LittleEndian.AppendUint16(buf, uint16(val))
}

// Truncates to maxLen, lowercases (the CoCo font has no upper case) and NUL
// pads to maxLen+1
func appendFixedLengthString(buf []byte, s string, maxLen int) []byte {
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	buf = append(buf, strings.ToLower(s)...)
	for i := len(s); i <= maxLen; i++ {
		buf = append(buf, 0)
	}
	return buf
}
