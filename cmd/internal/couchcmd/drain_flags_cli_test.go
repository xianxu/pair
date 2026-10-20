package couchcmd

import "testing"

func TestDrainTransientPublicGrammar(t *testing.T) {
	for _, args := range [][]string{{"--drain"}, {"--drain", "--fast", "--json"}, {"--drain", "--status"}, {"--undrain"}, {"--undrain", "--message", "continue after review", "--json"}} {
		if _, err := ParseCLI(args, nil); err != nil {
			t.Fatalf("public transient command %q refused: %v", args, err)
		}
	}
	for _, args := range [][]string{{"--drain", "--ack", "00000000-0000-4000-8000-000000000001"}, {"--drain", "--cancel"}, {"--drain", "--timeout", "1m"}, {"--drain", "--reason", "restart"}, {"--undrain", "--fast"}, {"--drain", "--status", "--fast"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("obsolete or mixed command accepted: %q", args)
		}
	}
}
