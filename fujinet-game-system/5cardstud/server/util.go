package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// Serializes the results, either as json (default), or raw (close to FujiNet json parsing result)
// raw=1 -  or as key[char 0]value[char 0] pairs
// - fc=U/L - (may use with raw) force data case all upper or lower

func serializeResults(c *gin.Context, obj any) {

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

		// Binary version of GameState

		/*
			typedef struct {
			  char lastResult[80];
			  uint8_t round;
			  uint16_t pot;
			  int8_t activePlayer;
			  uint8_t moveTime;
			  uint8_t viewing;
			  uint8_t validMoveCount;
			  ValidMove validMoves[5];
			  uint8_t playerCount;
			  Player players[8];
			} Game;
		*/

		if o, ok := obj.(*GameState); ok {
			buf = appendFixedLengthString(buf, o.LastResult, 80)
			buf = append(buf, byte(o.Round))
			buf = appendUint16(buf, o.Pot, bigEndian)
			buf = append(buf,
				byte(o.ActivePlayer),
				byte(o.MoveTime),
				byte(o.Viewing))

			// Valid moves array
			moves := len(o.ValidMoves)
			if moves > 5 {
				moves = 5 // Limit to 5 valid moves
			}

			buf = append(buf, byte(len(o.ValidMoves)))
			for i := 0; i < 5; i++ {
				if i < moves {
					buf = appendFixedLengthString(buf, o.ValidMoves[i].Move, 2)
					buf = appendFixedLengthString(buf, o.ValidMoves[i].Name, 9)
				} else {
					// Append empty values
					buf = appendFixedLengthString(buf, "", 12)
				}
			}

			// Players array
			buf = append(buf, byte(len(o.Players)))
			for i := 0; i < len(o.Players); i++ {
				buf = appendFixedLengthString(buf, o.Players[i].Name, 8)
				buf = append(buf, byte(o.Players[i].Status))
				buf = appendUint16(buf, o.Players[i].Bet, bigEndian)
				buf = appendFixedLengthString(buf, o.Players[i].Move, 7)
				buf = appendUint16(buf, o.Players[i].Purse, bigEndian)
				buf = appendFixedLengthString(buf, o.Players[i].Hand, 10)
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
