package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dchest/uniuri"
)

// Gensym creates a random sting pre-fixing the parameter provided.
func gensym(prefix string) string {
	// yes, I and O are missing not to confuse them with 1 and 0

	return prefix + "-" + uniuri.NewLenChars(8, []byte("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789"))
}

// check if char is a digit (0..9)
func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

// check if str only contains ASCII letters & numbers.
func isASCIIPrintable(str string) bool {

	for _, r := range str {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}

	return true
}

func ValidUsername(username string) (validusername string, err error) {

	var notvalid string

	if username == "srv" {
		return notvalid, fmt.Errorf("this is a reserved name that cannot be used")
	}

	if username[0] != '@' {
		return notvalid, fmt.Errorf("username must start with '@'")
	}

	if len(username) > 16 {
		return notvalid, fmt.Errorf("username cannot be longer than 16 chars")
	}

	if isDigit(username[1]) {
		return notvalid, fmt.Errorf("username cannot start with a number")
	}

	if !isASCIIPrintable(username[1:]) {
		return notvalid, fmt.Errorf("username can only contain ASCII chars and numbers")
	}

	return username, nil
}

func ValidChannelname(channelname string) (vaalidchannelname string, err error) {

	var notvalid string

	if channelname == "#main" {
		return notvalid, fmt.Errorf("this is a reserved name that cannot be used")
	}

	if channelname[0] != '#' {
		return notvalid, fmt.Errorf("channelname must start with '#'")
	}

	if len(channelname) > 16 {
		return notvalid, fmt.Errorf("channelname cannot be longer than 16 chars")
	}

	if isDigit(channelname[1]) {
		return notvalid, fmt.Errorf("channelname cannot start with a number")
	}

	if !isASCIIPrintable(channelname[1:]) {
		return notvalid, fmt.Errorf("channelname can only contain ASCII chars and numbers")
	}

	return channelname, nil
}

// no(x) -> bool
// len(x is Map, Slice, Array or String) == 0 --> true
// (x is Struct) == empty interface --> true
// otherwise --> false

func no(x interface{}) bool {

	if x == nil { // to check for nil interface
		return true
	}

	v := reflect.ValueOf(x)
	k := v.Kind()

	if k == reflect.Map || k == reflect.Slice || k == reflect.Array || k == reflect.String {
		return v.Len() == 0
	}

	if k == reflect.Struct {
		return v.IsZero()
	}

	return false
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

func trim(s string) string {
	return strings.Trim(s, " \t\n\r")
}
