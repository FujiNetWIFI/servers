package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var isTestMode = false

// Serializes the results, either as json (default), or bin (for 8 bit clients)
func serializeResults(c *gin.Context, obj any) {
	if isTestMode {
		c.Set("testResult", obj)
		return
	}
	if c.Query("bin") == "1" {
		var buf []byte

		// Binary version of Table list
		if tables, ok := obj.([]GameTable); ok {
			buf = append(buf, byte(len(tables)))
			for _, o := range tables {
				buf = appendFixedLengthString(buf, o.Table, 8)
				buf = appendFixedLengthString(buf, o.Name, 20)
				buf = appendFixedLengthString(buf, fmt.Sprintf("%d / %d", o.CurPlayers, o.MaxPlayers), 5)
			}
		}
		
		// // Binary version of GameState
		if o, ok := obj.(*GameState); ok {
			buf = append(buf, byte(len(o.Players)))
			buf = appendFixedLengthString(buf, o.Prompt, 32)
			buf = append(buf,
				byte(o.Status),
				byte(o.PlayerStatus),
				byte(o.ActivePlayer),
				byte(o.MoveTime))

			if o.Status == STATUS_LOBBY {
				// include server name
				buf = appendFixedLengthString(buf, o.serverName, 20)
			} else {
				buf = append(buf,byte(o.LastAttackPos))
				
				if o.clientPlayer == 0 && o.PlayerStatus != PLAYER_STATUS_VIEWING && o.Players[0].ships != nil {
					for j := 0; j < 5; j++ {
						buf = append(buf, byte(o.Players[0].ships[j].Pos + (100*o.Players[0].ships[j].Dir)))
					}
				} else {
					for j := 0; j < 5; j++ {
						buf = append(buf, byte(0))
					}
				}	
			}

		
			for i := 0; i < len(o.Players); i++ {
				buf = appendFixedLengthString(buf, o.Players[i].Name, 8)
				buf = append(buf, byte(o.Players[i].status))
				
				if o.Status != STATUS_LOBBY {
					// Include gamefield and ships only if avilable (the game has started)
					if len(o.Players[i].Gamefield) >0 && len(o.Players[i].ShipsLeft) >0 {
						for j := 0; j < FIELD_SIZE; j++ {
							buf = append(buf, byte(o.Players[i].Gamefield[j]))
						}
						for j := 0; j < len(o.Players[i].ShipsLeft); j++ {
							buf = append(buf, byte(o.Players[i].ShipsLeft[j]))
						}
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
