package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// Serializes the results, either as json (default), or raw (close to FujiNet json parsing result)
// raw=1 -  or as key[char 0]value[char 0] pairs
// - fc=U/L - (may use with raw) force data case all upper or lower

func serializeResults(c *gin.Context, obj any) {

	if debugMode {
		// Log the request details
		log.Printf("DEBUG Request - Method: %s, Path: %s, Query: %s", c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery)

		// Log the request body if present
		// Read the body into a byte slice
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("DEBUG Error reading request body: %v", err)
		} else if len(bodyBytes) > 0 {
			log.Printf("DEBUG Request Body: %s", string(bodyBytes))
		}
		// Restore the body for subsequent handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log the response
		jsonBytes, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			log.Printf("DEBUG Error marshaling response for logging: %v", err)
		} else {
			log.Printf("DEBUG Response: %s", string(jsonBytes))
		}
	}

	if c.Query("raw") == "1" {
		lineDelimiter := "\u0000"
		if c.Query("lf") == "1" {
			lineDelimiter = "\n"
		}
		jsonBytes, _ := json.Marshal(obj)
		jsonResult := string(jsonBytes)

		// Strip out [,],{,}
		jsonResult = strings.ReplaceAll(jsonResult, "{", "")
		jsonResult = strings.ReplaceAll(jsonResult, "}", "")
		jsonResult = strings.ReplaceAll(jsonResult, "[", "")
		jsonResult = strings.ReplaceAll(jsonResult, "]", "")

		// Convert : to new line
		jsonResult = strings.ReplaceAll(jsonResult, ":", lineDelimiter)

		// Convert commas to new line
		jsonResult = strings.ReplaceAll(jsonResult, "\",", lineDelimiter)
		jsonResult = strings.ReplaceAll(jsonResult, ",\"", lineDelimiter)
		jsonResult = strings.ReplaceAll(jsonResult, "\"", "")

		if c.Query("uc") == "1" {
			jsonResult = strings.ToUpper(jsonResult)
		}

		if c.Query("lc") == "1" {
			jsonResult = strings.ToLower(jsonResult)
		}

		c.String(http.StatusOK, jsonResult)

	} else if c.Query("bin") == "1" {
		// Packed binary serialization for 8-bit clients (cc65/cmoc). The layout
		// must byte-for-byte match the packed Game/Tables structs in the client's
		// src/misc.h. Strings are fixed-length, NUL-terminated, and lowercased
		// (the client's card glyphs expect lowercase, e.g. "ks" "ah" "??").
		// Pass be=1 for big-endian uint16s (CoCo); default is little-endian (6502).
		var buf []byte

		bigEndian := c.Query("be") == "1"

		// Binary version of Table list
		if tables, ok := obj.([]GameTable); ok {
			buf = append(buf, byte(len(tables)))
			for _, o := range tables {
				buf = appendFixedLengthString(buf, o.Table, 8)
				buf = appendFixedLengthString(buf, o.Name, 20)
				buf = appendFixedLengthString(buf, fmt.Sprintf("%d / %d", o.CurPlayers, o.MaxPlayers), 5)
			}
		}

		// Binary version of the client state. Client-side struct (src/misc.h):
		//
		//	typedef struct {
		//	  char lastResult[81];
		//	  uint8_t round;
		//	  uint16_t pot;
		//	  int8_t activePlayer;
		//	  uint8_t moveTime;
		//	  uint8_t viewing;
		//	  char community[11];       // Texas Hold'em addition
		//	  uint8_t validMoveCount;
		//	  ValidMove validMoves[5];  // { char move[3]; char name[10]; }
		//	  uint8_t playerCount;
		//	  Player players[8];        // { char name[9]; uint8_t status; uint16_t bet;
		//	                            //   char move[8]; uint16_t purse; char hand[11]; }
		//	} Game;

		if o, ok := obj.(*clientState); ok {
			buf = appendFixedLengthString(buf, o.LastResult, 80)
			buf = append(buf, byte(o.Round))
			buf = appendUint16(buf, o.Pot, bigEndian)
			buf = append(buf,
				byte(o.ActivePlayer),
				byte(o.MoveTime),
				byte(o.Viewing))
			buf = appendFixedLengthString(buf, o.Community, 10)

			// Valid moves array (fixed 5 slots)
			moves := len(o.ValidMoves)
			if moves > 5 {
				moves = 5
			}
			buf = append(buf, byte(moves))
			for i := 0; i < 5; i++ {
				if i < moves {
					buf = appendFixedLengthString(buf, o.ValidMoves[i].Move, 2)
					buf = appendFixedLengthString(buf, trimToWord(o.ValidMoves[i].Name, 9), 9)
				} else {
					// Append empty values
					buf = appendFixedLengthString(buf, "", 12)
				}
			}

			// Players array
			buf = append(buf, byte(len(o.Players)))
			for i := 0; i < len(o.Players); i++ {
				p := o.Players[i]
				status := p.Status
				if status == STATUS_ALL_IN {
					// 8-bit clients treat status 1 as "in the hand"; all-in players
					// are still in the hand (their move text shows ALLIN)
					status = STATUS_PLAYING
				}
				buf = appendFixedLengthString(buf, p.Name, 8)
				buf = append(buf, byte(status))
				buf = appendUint16(buf, p.Bet, bigEndian)
				buf = appendFixedLengthString(buf, p.Move, 7)
				buf = appendUint16(buf, p.Purse, bigEndian)
				buf = appendFixedLengthString(buf, p.Hand, 10)
			}
		}

		c.Data(http.StatusOK, "application/octet-stream", buf)

	} else {
		c.JSON(http.StatusOK, obj)
	}
}

// Appends a uint16 value to the byte slice in either big-endian or little-endian format
func appendUint16(buf []byte, val int, bigEndian bool) []byte {
	if bigEndian {
		buf = binary.BigEndian.AppendUint16(buf, uint16(val))
	} else {
		buf = binary.LittleEndian.AppendUint16(buf, uint16(val))
	}
	return buf
}

// Returns a byte slice equal to the maxLen+1, padded with zeros
// The extra byte is added to terminate the string
func appendFixedLengthString(buf []byte, s string, maxLen int) []byte {

	// Truncate string to honor contract
	if len(s) > maxLen {
		s = s[:maxLen]
	}

	// Convert to lowercase
	s = strings.ToLower(s)

	buf = append(buf, s...)
	maxLen -= len(s)
	for maxLen >= 0 {
		buf = append(buf, 0)
		maxLen--
	}
	return buf
}

// trimToWord shortens a string to fit maxLen by dropping trailing
// space-delimited words rather than cutting mid-word, so "All-in 1990"
// becomes "All-in" instead of the misleading "all-in 19"
func trimToWord(s string, maxLen int) string {
	for len(s) > maxLen {
		idx := strings.LastIndex(s, " ")
		if idx < 0 {
			return s[:maxLen]
		}
		s = s[:idx]
	}
	return s
}
