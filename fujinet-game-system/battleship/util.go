package main

import (
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
		
		// TODO: Implement binary serialization
		
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
