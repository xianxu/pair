package couchcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func FuzzParseDrainCLI(f *testing.F) {
	for _, seed := range []string{"--drain", "--drain\x00--status", "--drain\x00--fast\x00--json", "--drain\x00--ack\x0000000000-0000-4000-8000-000000000001"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		args := strings.Split(input, "\x00")
		if len(args) > 64 || len(input) > 8192 {
			return
		}
		args[0] = "--drain"
		inv, err := ParseCLI(args, nil)
		if err != nil {
			return
		}
		if inv.kind != cliMessage || inv.messageOp != "drain" || inv.drain == nil {
			t.Fatalf("accepted unrelated operation: %+v", inv)
		}
		r := couchmessage.Request{Op: "drain", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Drain: inv.drain}
		if inv.drain.Action == "start" {
			r.ID = "00000000-0000-4000-8000-000000000001"
		}
		if err := couchmessage.ValidateRequest(r); err != nil {
			t.Fatalf("CLI accepted invalid wire request: %q: %v", args, err)
		}
	})
}
