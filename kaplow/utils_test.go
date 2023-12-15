package main

import (
	"testing"
)

func TestCleanServerAddr(t *testing.T) {

	tests := []struct {
		srvaddr string
		want    string
	}{
		{"example.com", "example.com"},
		{"example.com/", "example.com"},
		{"example.com//", "example.com"},
		{"http://example.com", "example.com"},
		{"http://example.com/", "example.com"},
		{"http:/example.com", "example.com"},
		{"http:/example.com/", "example.com"},
		{"http:///example.com//", "example.com"},
		{"///example.com//", "example.com"},
		{"example.com:990", "example.com:990"},
		{"example.com:990/", "example.com:990"},
		{"example.com:990//", "example.com:990"},
		{"http://example.com:990", "example.com:990"},
		{"http://example.com:990/", "example.com:990"},
		{"http:/example.com:990", "example.com:990"},
		{"http:/example.com:990/", "example.com:990"},
		{"http:///example.com:990//", "example.com:990"},
		{"///example.com:990//", "example.com:990"},
	}
	for _, tt := range tests {
		t.Run(tt.srvaddr, func(t *testing.T) {
			if got := CleanServerAddr(tt.srvaddr); got != tt.want {
				t.Errorf("CleanServerAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPrintableAscii(t *testing.T) {

	tests := []struct {
		uri  string
		want bool
	}{
		{"", false},
		{"alfa", true},
		{"alfa beta", false},
		{"alfa,beta", true},
		{"álfa", false},
		{"álfa bëta", false},
		{"😒", false},
		{"_:,", true},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			if got := IsPrintableAscii(tt.uri); got != tt.want {
				t.Errorf("IsPrintableAscii() = %v, want %v", got, tt.want)
			}
		})
	}
}
