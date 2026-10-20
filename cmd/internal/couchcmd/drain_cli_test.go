package couchcmd

import (
	"testing"
)

func TestDrainCLIClosedShapes(t *testing.T) {
	inv, err := ParseCLI([]string{"--drain", "--fast", "--reason", "--layout3", "--timeout", "2m", "--json"}, nil)
	if err != nil || inv.drain == nil || inv.drain.Action != "start" || !inv.drain.Fast || inv.drain.Reason != "--layout3" || inv.drain.TimeoutMillis != 120000 {
		t.Fatalf("%+v %v", inv, err)
	}
	for _, args := range [][]string{{"--drain", "--status", "--fast"}, {"--drain", "--cancel", "--timeout", "1m"}, {"--drain", "--fast", "--fast"}, {"--drain", "--timeout", "0s"}, {"--drain", "--reason"}, {"--drain", "--ack", "bad-id"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
	for _, args := range [][]string{{"--drain"}, {"--drain", "--status"}, {"--drain", "--cancel"}, {"--drain", "--ack", "00000000-0000-4000-8000-000000000001"}} {
		if _, err := ParseCLI(args, nil); err != nil {
			t.Fatalf("%q: %v", args, err)
		}
	}
}
