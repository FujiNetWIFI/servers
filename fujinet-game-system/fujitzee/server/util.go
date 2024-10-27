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
// raw=1 -  or as key[char 0]value[char 0] pairs
// - fc=U/L - (may use with raw) force data case all upper or lower

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
		var val int
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
		if o, ok := obj.(*GameState); ok {
			buf = append(buf, byte(len(o.Players)))
			buf = appendFixedLengthString(buf, o.Name, 20)
			buf = appendFixedLengthString(buf, o.Prompt, 40)
			buf = append(buf,
				byte(o.Round),
				byte(o.RollsLeft),
				byte(o.ActivePlayer),
				byte(o.MoveTime),
				byte(o.Viewing))
			buf = appendFixedLengthString(buf, o.Dice, 5)
			buf = appendFixedLengthString(buf, o.KeepRoll, 5)
			for i := 0; i < 15; i++ {
				if i < len(o.ValidScores) {
					val = o.ValidScores[i]
				} else {
					val = 0
				}
				buf = append(buf, byte(val))
			}
			for i := 0; i < len(o.Players); i++ {
				buf = appendFixedLengthString(buf, o.Players[i].Name, 8)
				buf = append(buf, byte(o.Players[i].Alias))

				for j := 0; j < 16; j++ {
					if j < len(o.Players[i].Scores) {
						val = o.Players[i].Scores[j]
					} else {
						val = 0
					}
					if bigEndian {
						buf = binary.BigEndian.AppendUint16(buf, uint16(val))
					} else {
						buf = binary.LittleEndian.AppendUint16(buf, uint16(val))
					}
				}
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

// ternary if operator
func ifFuncElseFunc[T any](condition bool, yes func() T, no func() T) T {
	if condition {
		return yes()
	}

	return no()
}

// ternary if operator
func ifFuncElse[T any](condition bool, yes func() T, no T) T {
	if condition {
		return yes()
	}

	return no
}

// ternary if operator
func ifElse[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}

	return no
}
