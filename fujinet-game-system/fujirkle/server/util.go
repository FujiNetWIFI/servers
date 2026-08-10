package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

var isTestMode = false

// Serializes the results, either as json (default), or raw (close to FujiNet json parsing result)
// raw=1 - as key[char 0]value[char 0] pairs
// uc/lc=1 - (may use with raw) force data case all upper or lower
// bin=1 - packed binary; the 8 bit clients mirror this layout as a struct
// be=1 - big endian 16 bit values (6809, 6502 clients are little endian)

func serializeResults(c *gin.Context, obj any) {
	if isTestMode {
		c.Set("testResult", obj)
		return
	}
	if c.Query("raw") == "1" {
		lineDelimiter := "\u0000"
		if c.Query("lf") == "1" {
			lineDelimiter = "\n"
		}
		jsonBytes, _ := json.Marshal(obj)
		jsonResult := string(jsonBytes)

		jsonResult = strings.ReplaceAll(jsonResult, "{", "")
		jsonResult = strings.ReplaceAll(jsonResult, "}", "")
		jsonResult = strings.ReplaceAll(jsonResult, "[", "")
		jsonResult = strings.ReplaceAll(jsonResult, "]", "")

		jsonResult = strings.ReplaceAll(jsonResult, ":", lineDelimiter)

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

		appendWord := func(buf []byte, val int) []byte {
			if bigEndian {
				return binary.BigEndian.AppendUint16(buf, uint16(val))
			}
			return binary.LittleEndian.AppendUint16(buf, uint16(val))
		}

		if tables, ok := obj.([]GameTable); ok {
			buf = append(buf, byte(len(tables)))
			for _, o := range tables {
				buf = appendFixedLengthString(buf, o.Table, 8)
				buf = appendFixedLengthString(buf, o.Name, 20)
				buf = appendFixedLengthString(buf, fmt.Sprintf("%d / %d", o.CurPlayers, o.MaxPlayers), 5)
			}
		}

		if o, ok := obj.(*GameState); ok {
			buf = append(buf, byte(len(o.Players)))
			buf = appendFixedLengthString(buf, o.Name, 20)
			buf = appendFixedLengthString(buf, o.Prompt, 40)
			buf = append(buf,
				byte(o.Round),
				byte(o.ActivePlayer),
				byte(o.MoveTime),
				byte(o.Viewing),
				byte(o.ValidMoves))
			buf = appendWord(buf, o.TurnScore)
			buf = appendFixedLengthString(buf, o.Dice, NUM_DICE)
			buf = appendFixedLengthString(buf, o.KeptDice, NUM_DICE)
			buf = appendFixedLengthString(buf, o.Selectable, NUM_DICE)
			buf = appendFixedLengthString(buf, o.KeepRoll, NUM_DICE)

			// 13 bytes per player. The clients hold every field as bytes and
			// recombine the score themselves, so there is nothing here that a
			// compiler could pad differently on one platform.
			for i := 0; i < len(o.Players); i++ {
				buf = appendFixedLengthString(buf, o.Players[i].Name, 8)
				buf = append(buf,
					byte(o.Players[i].Alias),
					byte(o.Players[i].Ready))
				buf = appendWord(buf, o.Players[i].Score)
			}
		}

		c.Data(http.StatusOK, "application/octet-stream", buf)
	} else {
		c.JSON(http.StatusOK, obj)
	}
}

// Returns a byte slice equal to the maxLen+1, padded with zeros
// The extra byte is added to terminate the string
func appendFixedLengthString(buf []byte, s string, maxLen int) []byte {

	if len(s) > maxLen {
		s = s[:maxLen]
	}

	s = strings.ToLower(s)

	buf = append(buf, s...)
	maxLen -= len(s)
	for maxLen >= 0 {
		buf = append(buf, 0)
		maxLen--
	}
	return buf
}

func ifElse[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}
	return no
}
