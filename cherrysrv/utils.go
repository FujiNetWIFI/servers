package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"golang.org/x/crypto/bcrypt"
)

// Gensym creates a random sting pre-fixing the parameter provided.
func gensym(prefix string) string {
	// yes, I and O are missing not to confuse them with 1 and 0

	return prefix + "-" + uniuri.NewLenChars(8, []byte("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789"))
}

// encrypt password with bcrypt
//
//lint:ignore U1000 we will use it in the future
func encrypt(password string) string {

	// htpasswd -bnBC 10 "" <passwd> | tr -d ':\n' | sed 's/$2y/$2a/'
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword)
}

// check if passwd submitted for player is correct
func check_passwd(bcrypt_hash string, passwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(bcrypt_hash), []byte(passwd))
}

// difference returns the elements in `a` that aren't in `b`. Tables need to be sorted.
func difference(a, b []string) []string {

	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// x>y --> x
// else --> y
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// x --> x
// -x --> x
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func ValidUsername(username string) (validusername string, err error) {

	var notvalid string

	if username[0] == '@' {
		username = username[1:]
	}

	if len(username) > 16 {
		return notvalid, fmt.Errorf("username cannot be longer than 16 chars")
	}

	if username[0] >= '0' && username[0] <= '9' {
		return notvalid, fmt.Errorf("username cannot start with a number")
	}

	if username == "srv" {
		return notvalid, fmt.Errorf("this is a reserved name that cannot be used")
	}

	for _, r := range username {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return notvalid, fmt.Errorf("username can only contain ASCII chars and numbers")
		}
	}

	return username, nil
}

/*
This code can be placed near the top of your code, and then used to time any function like this:

	func factorial(n *big.Int) (result *big.Int) {
	    defer timeTrack(time.Now())
	    // ... do some things, maybe even return under some condition
	    return n
	}
*/
func timeTrack(start time.Time) time.Duration {
	return time.Since(start)
}

// https://stackoverflow.com/questions/7052693/how-to-get-the-name-of-a-function-in-go
// https://gist.github.com/HouLinwei/16df41bee7d799f0928e717b23d97a9b
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

// return goroutine id
// https://blog.sgmansfield.com/2015/12/goroutine-ids/

func goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

// if needle is in haystack --> true
func contains[T comparable](haystack []T, needle T) bool {
	for _, a := range haystack {
		if a == needle {
			return true
		}
	}
	return false
}

// no(x) -> bool
// len(x is Map, Slice, Array or String) == 0 --> true
// (x is Struct) == empty interface --> true
// otherwise --> false

func no(x interface{}) bool {

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

// confirm v is a slice
func IsSlice[T any](v T) bool {
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

type Indexable[T int | string] interface {
	Key() (value T, name string)
}

func split2(s string, sep string) (first string, second string) {

	split := strings.SplitN(s, sep, 2)

	return split[0], split[1]
}
