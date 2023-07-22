package main

import (
	"fmt"
	"testing"
)

func Test_no(t *testing.T) {

	tests := []struct {
		name string
		x    interface{}
		want bool
	}{
		{"Empty slice of strings", []string{}, true},
		{"Empty slice of ints", []int{}, true},
		{"Empty map int->int", map[int]int{}, true},
		{"Empty map string->int", map[string]int{}, true},
		{"Empty map int->string", map[int]string{}, true},
		{"Slice of ints", []int{1, 2, 33}, false},
		{"Map int->int", map[int]int{2: 3, 4: 5}, false},
		{"Nil", nil, true},
		{"Empty string", "", true},
		{"String", "casa", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := no(tt.x); got != tt.want {
				t.Errorf("no() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidUsername(t *testing.T) {

	var NOSTRING string

	tests := []struct {
		name              string
		username          string
		wantValidusername string
		wantErr           bool
	}{
		//		{"empty string", "", NOSTRING, true},
		{"valid name", "@JohnnyCash", "@JohnnyCash", false},
		{"valid name w/numbers", "@JohnnyCash12", "@JohnnyCash12", false},
		{"name with space", "@Johnny Cash", NOSTRING, true},
		{"srv", "srv", NOSTRING, true},
		{"name too long", "@a1234567890123456", NOSTRING, true},
		{"name at limit", "@a12345678901234", "@a12345678901234", false},
		{"name starts w/number", "1John", NOSTRING, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValidusername, err := ValidUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotValidusername != tt.wantValidusername {
				t.Errorf("ValidUsername() = %v, want %v", gotValidusername, tt.wantValidusername)
			}
		})
	}
}

func Test_split2(t *testing.T) {

	tests := []struct {
		s          string
		sep        string
		wantFirst  string
		wantSecond string
	}{
		{"", " ", "", ""},
		{"AA", " ", "AA", ""},
		{"AA BB", " ", "AA", "BB"},
		{"AA BB CC", " ", "AA", "BB CC"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf(`'%s' '%s'`, tt.s, tt.sep), func(t *testing.T) {
			gotFirst, gotSecond := split2(tt.s, tt.sep)
			if gotFirst != tt.wantFirst {
				t.Errorf("split2() gotFirst = %v, want %v", gotFirst, tt.wantFirst)
			}
			if gotSecond != tt.wantSecond {
				t.Errorf("split2() gotSecond = %v, want %v", gotSecond, tt.wantSecond)
			}
		})
	}
}
