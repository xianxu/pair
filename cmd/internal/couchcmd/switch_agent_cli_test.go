package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchAgentCLIForms(t *testing.T) {
	for _, args := range [][]string{
		{"--switch-agent", "self", "--agent", "codex", "--confirm"},
		{"--switch-agent", "pair:2", "--handoff-file", "brief", "--agent", "claude", "--json", "--confirm"},
		{"--switch-agent", "self", "--agent", "codex", "--handoff-file", "brief", "--request", "parent", "--confirm"},
		{"--operation-status", "parent", "--json"},
	} {
		if _, err := ParseCLI(args, nil); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, tail := range []string{
		"self --agent codex", "pair --agent codex --confirm", "self --agent bogus --confirm",
		"self --agent codex --confirm --confirm", "self --agent codex --confirm --layout2",
		"self --agent codex --confirm --request parent", "pair:2 --agent codex --confirm --handoff-file brief --request parent",
		"self --agent codex --confirm --handoff-file", "self --agent codex --agent claude --confirm",
	} {
		if _, err := ParseCLI(append([]string{"--switch-agent"}, strings.Fields(tail)...), nil); err == nil {
			t.Errorf("accepted %s", tail)
		}
	}
}

func TestSwitchAgentCLIAdmissionOneCallAndExactFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brief")
	body := "  context\n\tNext step\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	inv, err := ParseCLI([]string{"--switch-agent", "self", "--agent", "codex", "--handoff-file", path, "--request", "parent", "--confirm", "--json"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	calls := 0
	call := func(_ context.Context, _ string, raw, out any) error {
		calls++
		if calls > 1 {
			t.Fatal("polled admission")
		}
		r := raw.(couchmessage.Request)
		encoded, _ := json.Marshal(r)
		var fields map[string]any
		json.Unmarshal(encoded, &fields)
		if r.Body != body || r.Agent != "codex" || r.Target != "self" || fields["SwitchRequest"] != "parent" || r.Scope != "scope" || strings.Contains(string(encoded), path) {
			t.Fatalf("request %s", encoded)
		}
		*out.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: r.ID, Op: r.Op, Status: couchmessage.ReceiptQueued}}
		return nil
	}
	if code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, call); code != 0 {
		t.Fatalf("%d: %s", code, &errout)
	}
	var response couchmessage.Response
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.Operation == nil || calls != 1 {
		t.Fatalf("response %s calls %d: %v", &out, calls, err)
	}
}

func TestSwitchAgentHandoffFileRefusesBeforeTransport(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	os.WriteFile(regular, []byte("ok"), 0600)
	paths := []string{filepath.Join(dir, "symlink"), filepath.Join(dir, "fifo"), dir}
	os.Symlink(regular, paths[0])
	if err := unix.Mkfifo(paths[1], 0600); err != nil {
		t.Fatal(err)
	}
	for i, body := range []string{"", "\r\n", "bad\x00", "\xff", strings.Repeat("x", 16385)} {
		path := filepath.Join(dir, string(rune('a'+i)))
		os.WriteFile(path, []byte(body), 0600)
		paths = append(paths, path)
	}
	for _, path := range paths {
		inv, err := ParseCLI([]string{"--switch-agent", "self", "--agent", "codex", "--confirm", "--handoff-file", path}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var out, errout bytes.Buffer
		code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(context.Context, string, any, any) error { t.Fatal("invalid file sent"); return nil })
		if code == 0 || !strings.Contains(errout.String(), path) {
			t.Errorf("%s: %d %s", path, code, &errout)
		}
	}
}

func TestSwitchAgentCLIUncertainJSONRetainsID(t *testing.T) {
	inv, err := ParseCLI([]string{"--switch-agent", "self", "--agent", "codex", "--confirm", "--json"}, nil)
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
		args := []string{"--switch-agent", "self", "--agent", "codex", "--confirm", extra}
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted extra %q", extra)
		}
	})
}
