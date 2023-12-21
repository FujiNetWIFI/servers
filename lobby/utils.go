package main

import (
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
)

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

// TODO: Change err != nil to err == nil and review full codebase
func IsValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)

	return err != nil
}
