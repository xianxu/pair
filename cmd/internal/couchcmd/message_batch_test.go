package couchcmd

import (
	"strings"
	"testing"
)

func TestPeerBatchConfig(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int
	}{{"", 16384}, {"1", 1}, {"16384", 16384}, {"32768", 32768}} {
		got, err := peerBatchMaxChars(func(key string) string {
			if key != "COUCH_PEER_BATCH_MAX_CHARS" {
				t.Fatalf("wrong variable %q", key)
			}
			return tc.value
		})
		if err != nil || got != tc.want {
			t.Errorf("%q => %d,%v; want %d", tc.value, got, err, tc.want)
		}
	}
	for _, value := range []string{"0", "-1", "32769", "1.5", "bad", " 5", "5 ", "999999999999999999999999999999"} {
		if _, err := peerBatchMaxChars(func(string) string { return value }); err == nil || !strings.Contains(err.Error(), "COUCH_PEER_BATCH_MAX_CHARS") {
			t.Errorf("%q: expected named configuration error, got %v", value, err)
		}
	}
}
