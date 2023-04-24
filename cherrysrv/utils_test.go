package main

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestDifference(t *testing.T) {

	result := difference([]string{}, []string{})

	if nil != result {
		t.Error("difference([]string{}, []string{}) must be []", result)
	}

}

func TestDifferenceEmptyString(t *testing.T) {

	result := difference([]string{""}, []string{""})

	if nil != result {
		t.Error(`difference([]string{""}, []string{""}) must be []/nil`, result)
	}

}

func TestDifferenceEmptyStrings_1(t *testing.T) {

	result := difference([]string{"", ""}, []string{""})

	if nil != result {
		t.Error(`difference([]string{"", ""}, []string{""}) must be []/nil`, result)
	}

}

func TestDifferenceEmptyStrings_2(t *testing.T) {

	result := difference([]string{"", ""}, []string{"", ""})

	if nil != result {
		t.Error(`difference([]string{"", ""}, []string{"", ""}) must be []/nil`, result)
	}

}

func TestDifference_1_0(t *testing.T) {

	result := difference([]string{"1"}, []string{""})

	if !slices.Equal([]string{"1"}, result) {
		t.Error(`difference([]string{"1"}, []string{""}) must be []string{"1"}`, result)
	}

}

func TestDifference_0_1(t *testing.T) {

	result := difference([]string{""}, []string{"1"})

	if !slices.Equal([]string{""}, result) {
		t.Error(`difference([]string{""}, []string{"1"}) must be []string{""} and is `, result)
	}

}

func TestDifference_1_1(t *testing.T) {

	result := difference([]string{"1"}, []string{"1"})

	if !slices.Equal([]string{}, result) {
		t.Error(`difference([]string{"1"}, []string{"1"}) must be []/nil`, result)
	}

}

func TestDifference_2_1(t *testing.T) {

	result := difference([]string{"1", "2"}, []string{"1"})

	if !slices.Equal([]string{"2"}, result) {
		t.Error(`difference([]string{"1", "2"}, []string{"1"}) must be []string{"2"}`, result)
	}

}

func TestDifference_1_2(t *testing.T) {

	result := difference([]string{"1"}, []string{"1", "2"})

	if !slices.Equal([]string{}, result) {
		t.Error(`difference([]string{"1"}, []string{"1", "2"}) must be []/nil `, result)
	}

}

func TestPasswd_1(t *testing.T) {

	plain_passwd := "sesame"
	bcrypt_hash := "$2a$10$teH6sUdr48HflVGtMLj/LeiT2r7ENebIiJlxfmoEHUpL.hx5gVRli"

	if check_passwd(bcrypt_hash, plain_passwd) != nil {
		t.Error(`checkpassword is unable to check password "sesame" correctly`)
	}
}

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
		{"valid name", "JohnnyCash", "JohnnyCash", false},
		{"valid name w/numbers", "JohnnyCash12", "JohnnyCash12", false},
		{"name with space", "Johnny Cash", NOSTRING, true},
		{"srv", "srv", NOSTRING, true},
		{"name too long", "a1234567890123456", NOSTRING, true},
		{"name at limit", "a123456789012345", "a123456789012345", false},
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
