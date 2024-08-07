package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

var isTestMode = false

func setTestMode() {
	isTestMode = true
}

// Serializes the results, either as json (default), or raw (close to FujiNet json parsing result)
// raw=1 -  or as key[char 0]value[char 0] pairs
// - fc=U/L - (may use with raw) force data case all upper or lower

func serializeResults(c *gin.Context, obj any) {
	if isTestMode {
		c.Set("result", obj)
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

	} else {
		c.JSON(http.StatusOK, obj)
	}
}
