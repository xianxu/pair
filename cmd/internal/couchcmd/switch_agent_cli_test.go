package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
)

func TestSwitchAgentCLIForms(t *testing.T) {
	for _, args := range [][]string{
		{"--switch-agent", "pair:2", "--agent", "codex", "--confirm"},
		{"--switch-agent", "pair:0", "--agent", "claude", "--json", "--confirm"},
		{"--operation-status", "parent", "--json"},
	} {
		if _, err := ParseCLI(args, nil); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, tail := range []string{
		"self --agent codex --confirm", "pair --agent codex --confirm", "pair:2 --agent bogus --confirm",
		"pair:2 --agent codex", "pair:2 --agent codex --confirm --confirm", "pair:2 --agent codex --confirm --layout2",
		"pair:2 --agent codex --confirm --request parent", "pair:2 --agent codex --confirm --handoff-file brief",
		"pair:2 --agent codex --confirm --argv []", "pair:2 --agent codex --agent claude --confirm",
	} {
		if _, err := ParseCLI(append([]string{"--switch-agent"}, strings.Fields(tail)...), nil); err == nil {
			t.Errorf("accepted %s", tail)
		}
	}
}

func TestSwitchAgentCLIAdmissionCarriesOnlySelection(t *testing.T) {
	inv, err := ParseCLI([]string{"--switch-agent", "pair:2", "--agent", "codex", "--confirm", "--json"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(_ context.Context, _ string, raw, reply any) error {
		calls++
		r := raw.(couchmessage.Request)
		if r.Body != "" || r.Target != "pair:2" || r.Agent != "codex" || !r.Confirmed || r.Scope != "scope" {
			t.Fatalf("request %#v", r)
		}
		*reply.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: r.ID, Op: r.Op, Status: couchmessage.ReceiptQueued}}
		return nil
	})
	if code != 0 || calls != 1 {
		t.Fatalf("code %d calls %d stderr %s", code, calls, &errout)
	}
}

func TestSwitchAgentCLIUncertainJSONRetainsID(t *testing.T) {
	inv, err := ParseCLI([]string{"--switch-agent", "pair:2", "--agent", "codex", "--confirm", "--json"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	calls := 0
	id := ""
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(_ context.Context, _ string, r, reply any) error {
		calls++
		id = r.(couchmessage.Request).ID
		return context.DeadlineExceeded
	})
	var response couchmessage.Response
	json.Unmarshal(out.Bytes(), &response)
	if code == 0 || calls != 1 || response.Operation == nil || response.Operation.ID != id || !strings.Contains(errout.String(), "--operation-status "+id) {
		t.Fatalf("code %d calls %d output %s stderr %s", code, calls, &out, &errout)
	}
}

func TestSwitchAgentOperationStatusOneRead(t *testing.T) {
	inv, err := ParseCLI([]string{"--operation-status", "parent", "--json"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(_ context.Context, _ string, r, reply any) error {
		calls++
		req := r.(couchmessage.Request)
		if req.Op != "operation-status" || req.ID != "parent" || req.Scope != "scope" {
			t.Fatalf("%#v", req)
		}
		*reply.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Operation: &couchmessage.OperationReceipt{ID: "parent", Status: couchmessage.ReceiptRunning}}
		return nil
	})
	if code != 0 || calls != 1 {
		t.Fatalf("code %d calls %d stderr %s", code, calls, &errout)
	}
}

func FuzzSwitchCLIClosedShape(f *testing.F) {
	f.Add("--layout2")
	f.Add("--same-binary")
	f.Add("extra")
	f.Fuzz(func(t *testing.T, extra string) {
		if extra == "--json" {
			return
		}
		args := []string{"--switch-agent", "pair:2", "--agent", "codex", "--confirm", extra}
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted extra %q", extra)
		}
	})
}
