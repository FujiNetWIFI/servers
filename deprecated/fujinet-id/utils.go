package main

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// Encode takes a byte array and returns a string of encoded Ascii85 data variant implemented in RFC 1924
// https://github.com/darkwyrm/b85 MIT BSD
func EncodeAscii85(inData []byte) string {

	const b85chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz!#$%&()*+-;<=>?@^_`{|}~"

	var outData strings.Builder

	length := len(inData)
	chunkCount := uint32(length / 4)
	var dataIndex uint32

	for i := uint32(0); i < chunkCount; i++ {
		var decnum, remainder uint32
		decnum = uint32(inData[dataIndex])<<24 | uint32(inData[dataIndex+1])<<16 |
			uint32(inData[dataIndex+2])<<8 | uint32(inData[dataIndex+3])
		outData.WriteByte(b85chars[decnum/52200625])
		remainder = decnum % 52200625
		outData.WriteByte(b85chars[remainder/614125])
		remainder %= 614125
		outData.WriteByte(b85chars[remainder/7225])
		remainder %= 7225
		outData.WriteByte(b85chars[remainder/85])
		outData.WriteByte(b85chars[remainder%85])
		dataIndex += 4
	}

	extraBytes := length % 4
	if extraBytes != 0 {
		lastChunk := uint32(0)
		for i := length - extraBytes; i < length; i++ {
			lastChunk <<= 8
			lastChunk |= uint32(inData[i])
		}

		// Pad extra bytes with zeroes
		for i := (4 - extraBytes); i > 0; i-- {
			lastChunk <<= 8
		}
		outData.WriteByte(b85chars[lastChunk/52200625])
		remainder := lastChunk % 52200625
		outData.WriteByte(b85chars[remainder/614125])
		if extraBytes > 1 {
			remainder %= 614125
			outData.WriteByte(b85chars[remainder/7225])
			if extraBytes > 2 {
				remainder %= 7225
				outData.WriteByte(b85chars[remainder/85])
			}
		}
	}
	return outData.String()
}

// atoi but return ResultIfFail if not possible to do atoi
func Atoi(StrNum string, ResultIfFail int) int {
	num, err := strconv.Atoi(StrNum)

	if err != nil {
		return ResultIfFail
	}

	return num
}

// https://stackoverflow.com/questions/7052693/how-to-get-the-name-of-a-function-in-go
// https://gist.github.com/HouLinwei/16df41bee7d799f0928e717b23d97a9b
//
//lint:ignore U1000 To be used in the future
func currentFnName() string {

	current, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}

	return strings.Split(runtime.FuncForPC(current).Name(), ".")[1]
}

func extendedFnName() string {

	current_name := "unknown"
	parent_name := "unknown"

	current, _, _, ok := runtime.Caller(1)
	if ok {
		current_name = strings.Split(runtime.FuncForPC(current).Name(), ".")[1]
	}
	parent, _, _, ok := runtime.Caller(2)
	if ok {
		parent_name = strings.Split(runtime.FuncForPC(parent).Name(), ".")[1]
	}

	return parent_name + "/" + current_name
}

// copy a file from src to dest with permission 0644
func CopyFile(src string, dest string) error {

	bytesRead, err := os.ReadFile(src)

	if err != nil {
		return err
	}

	return os.WriteFile(dest, bytesRead, 0644)
}

// safely move a file from src to dest with permission 0644
func MoveFile(src string, dest string) error {

	bytesRead, err := os.ReadFile(src)

	if err != nil {
		return err
	}

	if err = os.WriteFile(dest, bytesRead, 0644); err != nil {
		return err
	}

	return os.Remove(src)
}

// ternary if operator
func IfElse[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}

	return no
}

// return value if doesn't meet condition, otherwise nil of T
func IfNot[T any](condition bool, value T) T {

	if !condition {
		return value
	}

	var zero T

	return zero
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

	return err == nil
}

func split2(s string, sep string) (first string, second string) {

	split := strings.SplitN(s, sep, 2)

	switch len(split) {
	case 0:
		return "", ""
	case 1:
		return split[0], ""
	}

	return split[0], split[1]
}

func trim(input string) string {
	// TODO: should we use TrimSpace?
	return strings.Trim(input, " \t\n\r")
}

func no(x any) bool {

	v := reflect.ValueOf(x)
	k := v.Kind()

	if k == reflect.Pointer {
		return v.IsNil()
	}

	if k == reflect.Map || k == reflect.Slice || k == reflect.Array || k == reflect.String {
		return v.Len() == 0
	}

	if k == reflect.Struct {
		return v.IsZero()
	}

	return false
}

/*
  ____            _                  _
 / ___|___  _ __ | |_ _ __ __ _  ___| |_
| |   / _ \| '_ \| __| '__/ _` |/ __| __|
| |__| (_) | | | | |_| | | (_| | (__| |_
 \____\___/|_| |_|\__|_|  \__,_|\___|\__|
*/

//lint:ignore U1000 To be used as needed
func requires(condition bool, description string) {
	if RELEASE_MODE { // we don't apply contracts to production code
		return
	}

	if !condition {
		panic(fmt.Sprintf("condition required: '%s' not meet at %s", description, extendedFnName()))
	}
}

//lint:ignore U1000 To be used as needed
func expects(condition bool, description string) {
	if RELEASE_MODE { // we don't apply contracts to production code
		return
	}

	if !condition {
		panic(fmt.Sprintf("condition expected: '%s' not meet at %s()", description, extendedFnName()))
	}
}

//lint:ignore U1000 To be used as needed
func invariant(condition bool, description string) {

	if RELEASE_MODE { // we don't apply contracts to production code
		return
	}

	if !condition {
		panic(fmt.Sprintf("invariant '%s' not meet at %s", description,
			extendedFnName()))
	}
}

// for range times(4) {
// do_something()
// }
//
//lint:ignore U1000 Ignore unused function. Will be used in the future
func times(n int) []struct{} {
	if n < 0 {
		n = 0
	}

	return make([]struct{}, n)
}
