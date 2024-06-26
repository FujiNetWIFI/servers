package main

import (
	"strings"
	"testing"
)

func TestGeneratePubKey(t *testing.T) {

	TEST_SERVERKEY := "46;79YV-YmoMwLS·YdWkA!ciIIfBqkq!KK$2EhRX9;:812,,/Fl9GlfmM%R&4YKF"

	tests := []struct {
		privkey    string
		serverkey  string
		wantPubkey string
	}{
		{"rogersm#secreto", TEST_SERVERKEY, "rogersm!orIt#zZqk@)hrDM^H&>)bedq`l6Ck>j2ijBTAZWF"},
		{"rogersm#secreto2", TEST_SERVERKEY, "rogersm!(?MIQmg71fWqV35-dq}e=QzL}h$W-JgB|a0S1idV"},
		{"", TEST_SERVERKEY, ""},
		{"#", TEST_SERVERKEY, ""},
		{"#secreto", TEST_SERVERKEY, ""},
	}
	for _, tt := range tests {
		t.Run(tt.privkey, func(t *testing.T) {
			if gotPubkey := generatePubKey(tt.privkey, tt.serverkey); gotPubkey != tt.wantPubkey {
				t.Errorf("GeneratePubKey() = %v, want %v", gotPubkey, tt.wantPubkey)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {

	var exists struct{}

	set := make(map[string]struct{})

	TIMES := 1000

	t.Run("Check no repeated tokens", func(t *testing.T) {
		for range times(TIMES) {
			gotToken, err := generateToken()

			if err != nil {
				t.Errorf("GenerateToken() error = %v", err)
			}

			if !strings.HasSuffix(gotToken, "99991231235959") {
				t.Errorf("GenerateToken() = %s does not end with 99991231235959", gotToken)
			}

			set[gotToken] = exists

		}
		if len(set) != TIMES {
			t.Errorf("GenerateToken() duplicates were found, expected = %d, found = %d", len(set), TIMES)
		}
	})
}
