package couchcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

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

func TestDrainTransientLostReplyDoesNotRepeat(t *testing.T) {
	for _, args := range [][]string{{"--drain"}, {"--undrain", "--message", "continue"}} {
		inv, err := ParseCLI(args, nil)
		if err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		calls := 0
		code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, req, _ any) error {
			calls++
			if req.(couchmessage.Request).ID == "" {
				t.Fatal("mutation lacks request identity")
			}
			return context.DeadlineExceeded
		})
		if code == 0 || calls != 1 || !strings.Contains(stderr.String(), "outcome uncertain") || !strings.Contains(stderr.String(), "--drain --status") {
			t.Fatalf("uncertain mutation repeated or hidden: code=%d calls=%d stderr=%q", code, calls, stderr.String())
		}
	}
}

func TestDrainStatusNamesActiveAndDisconnected(t *testing.T) {
	inv, err := ParseCLI([]string{"--drain", "--status"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	active := false
	var out, stderr bytes.Buffer
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Drain: &couchcore.DrainBoard{Rows: []couchcore.DrainRow{{Workspace: "pair:1", Connected: true, Settled: &active}, {Workspace: "pair:2"}}}}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "pair:1: active") || !strings.Contains(out.String(), "pair:2: disconnected") {
		t.Fatalf("activity collapsed into unknown: code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}
