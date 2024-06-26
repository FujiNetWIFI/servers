package main

import (
	"testing"
)

func Test_currentFnName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"basecase", "Test_currentFnName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentFnName(); got != tt.want {
				t.Errorf("currentFnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extendedFnName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"basecase", "tRunner/Test_extendedFnName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extendedFnName(); got != tt.want {
				t.Errorf("extendedFnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeAscii85(t *testing.T) {

	tests := []struct {
		inData string
		want   string
	}{
		{"a", "VE"},
		{"aa", "VPO"},
		{"aaa", "VPRn"},
		{"aaaa", "VPRom"},
		{"aaaaa", "VPRomVE"},
		{"aaaaaa", "VPRomVPO"},
		{"aaaaaaa", "VPRomVPRn"},
		{"aaaaaaaa", "VPRomVPRom"},
	}
	for _, tt := range tests {
		t.Run(tt.inData, func(t *testing.T) {
			if got := EncodeAscii85([]byte(tt.inData)); got != tt.want {
				t.Errorf("EncodeAscii85() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidURI(t *testing.T) {

	tests := []struct {
		uri  string
		want bool
	}{
		{"http://example.com/", true},
		{"nothing", false},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			if got := IsValidURI(tt.uri); got != tt.want {
				t.Errorf("IsValidURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
