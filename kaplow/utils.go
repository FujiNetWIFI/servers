package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
)

/*
 __  __    _    ____
|  \/  |  / \  |  _ \
| |\/| | / _ \ | |_) |
| |  | |/ ___ \|  __/
|_|  |_/_/   \_\_|
*/

type Map map[string]interface{}

func (m Map) M() Map {
	return m
}

func (m Map) NamedM(name string) Map {
	return Map{"Name": name, "Value": m}
}

type MapSlice []Map

func (m MapSlice) M() Map {
	return Map{"Name": "list-of-maps", "Value": m}
}

func (m MapSlice) NamedM(name string) Map {
	return Map{"Name": name, "Value": m}
}

type Mapeable interface {
	M() Map
}

type SliceMapeable interface {
	NamedM(name string) Map
}

/*
 _   _ _   _ __  __  ____ _____ _   _
| \ | | | | |  \/  |/ ___| ____| \ | |
|  \| | | | | |\/| | |  _|  _| |  \| |
| |\  | |_| | |  | | |_| | |___| |\  |
|_| \_|\___/|_|  |_|\____|_____|_| \_|
*/

type NumGen struct {
	rand         *rand.Rand
	initial_seed int64
}

func makeNumGenInt64(seed int64) (n NumGen) {

	return NumGen{
		rand:         rand.New(rand.NewSource(seed)),
		initial_seed: seed,
	}
}

func makeNumGen() (n NumGen) {

	seed := time.Now().UnixNano()

	return makeNumGenInt64(seed)
}

/*
 _   _ _____ _     ____  _____ ____  ____
| | | | ____| |   |  _ \| ____|  _ \/ ___|
| |_| |  _| | |   | |_) |  _| | |_) \___ \
|  _  | |___| |___|  __/| |___|  _ < ___) |
|_| |_|_____|_____|_|   |_____|_| \_\____/
*/

func Atoi(StrNum string, ResultIfFail int) int {
	num, err := strconv.Atoi(StrNum)

	if err != nil {
		return ResultIfFail
	}

	return num
}

// return err if meets condition
func ErrorIf(condition bool, err error) error {
	if condition {
		return err
	}

	return nil
}

func IsValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)

	return err != nil
}

// uri should be ascii-8 string
// true if between 33 and 126 inclusive. Does not include space (is 32)
func IsPrintableAscii(uri string) bool {

	if uri == "" {
		return false
	}

	for i := 0; i < len(uri); i++ {
		if !(uri[i] >= 33 && uri[i] <= 126) {
			return false
		}
	}

	return true
}

// If exist, remove leading http:// and / at the tail
func CleanServerAddr(srvaddr string) string {
	srvaddr = strings.TrimLeft(srvaddr, "http://")
	srvaddr = strings.TrimRight(srvaddr, "/")
	return srvaddr
}

// return value if doesn't meet condition, otherwise nil of T
func IfNot[T any](condition bool, value T) T {

	if !condition {
		return value
	}

	var zero T

	return zero
}

// ternary if operator
func IfElse[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}

	return no
}

/*
 _   _ _____ _____ ____
| | | |_   _|_   _|  _ \
| |_| | | |   | | | |_) |
|  _  | | |   | | |  __/
|_| |_| |_|   |_| |_|

*/

// write data []byte and write it to w http.ResponseWriter

func HTTPByteResponse(w http.ResponseWriter, httpStatus int, data []byte) bool {
	w.WriteHeader(httpStatus)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	w.Write(data)

	return true
}

// from httpStatus code -> "httpStatus code", "<description>"
func httpStatusToText(httpStatus int) (number string, description string) {
	/*
		https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
		1xx: Informational - Request received, continuing process
		2xx: Success - The action was successfully received, understood, and accepted
		3xx: Redirection - Further action must be taken in order to complete the request
		4xx: Client Error - The request contains bad syntax or cannot be fulfilled
		5xx: Server Error - The server failed to fulfill an apparently valid request
	*/

	Mapper := []string{
		"unknown",
		"informational",
		"success",
		"redirection",
		"clienterror",
		"servererror",
	}

	if httpStatus/100 >= 6 {
		return strconv.Itoa(httpStatus), "unknown"
	}

	return strconv.Itoa(httpStatus), Mapper[httpStatus/100]
}

// transform data to a JSON and write it to w http.ResponseWriter
func HTTPJsonResponse(w http.ResponseWriter, httpStatus int, data Mapeable) bool {

	dataMap := data.M()

	if len(dataMap) == 0 {
		return false
	}

	dataMap["code"], dataMap["codetext"] = httpStatusToText(httpStatus)

	byteData, err := json.MarshalIndent(dataMap, "", "\t")

	if err != nil {
		slog.Error("HTTPJsonResponse", "unable to Marshal json", dataMap)
	}

	w.WriteHeader(httpStatus)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(byteData)))

	w.Write(byteData)

	return true
}

// transform data using tpl template and write it to w http.ResponseWriter
func HTTPTemplateResponse(w http.ResponseWriter, tpl string, data Mapeable) bool {

	dataMap := data.M()

	t, err := template.ParseGlob("./templates/*.gtpl")
	if err != nil {
		slog.Error("HTTPTemplateResponse", "template is incorrect: ", tpl, "err: ", err)
		http.Error(w, "500 - Something bad happened parsing the template!", http.StatusInternalServerError)

		return false
	}

	err = t.ExecuteTemplate(w, tpl, dataMap)
	if err != nil {
		slog.Error("HTTPTemplateResponse", "error processing template: ", tpl, "err: ", err)
		http.Error(w, "500 - Something bad happened parsing the template!", http.StatusInternalServerError)

		return false
	}

	return true
}

// decode structure from req.Body (using io.Reader interface) and store it in v Checkeable
func HTTPDecodeStructFromPost(r io.Reader, v Checkeable) error {
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return err
	}

	return v.CheckInput()
}
