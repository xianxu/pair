package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"reflect"
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
		if extra == "--json" || extra == "--interrupt-first" {
			return
		}
		args := []string{"--switch-agent", "pair:2", "--agent", "codex", "--confirm", extra}
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted extra %q", extra)
		}
	})
}

func TestSwitchAgentCLIExactOptionalArguments(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, tc := range []struct {
			name      string
			extra     []string
			want      json.RawMessage
			interrupt bool
		}{
			{name: "absent"},
			{name: "empty", extra: []string{"--agent-args", ""}, want: json.RawMessage(`[]`)},
			{name: "literal", extra: []string{"--agent-args", `--model 'a b' '' '$HOME' '$(nope)'`}, want: json.RawMessage(`["--model","a b","","$HOME","$(nope)"]`)},
			{name: "interrupt", extra: []string{"--agent-args", "", "--interrupt-first"}, want: json.RawMessage(`[]`), interrupt: true},
		} {
			t.Run(agent+"/"+tc.name, func(t *testing.T) {
				args := append([]string{"--switch-agent", "pair:2", "--agent", agent, "--confirm"}, tc.extra...)
				inv, err := ParseCLI(args, nil)
				if err != nil {
					t.Fatal(err)
				}
				var out, errout bytes.Buffer
				calls := 0
				code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(_ context.Context, _ string, raw, reply any) error {
					calls++
					request := raw.(couchmessage.Request)
					encoded, err := json.Marshal(request)
					if err != nil {
						t.Fatal(err)
					}
					var shape struct {
						AgentArgs      json.RawMessage
						InterruptFirst bool
					}
					if err := json.Unmarshal(encoded, &shape); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(shape.AgentArgs, tc.want) || shape.InterruptFirst != tc.interrupt {
						t.Fatalf("changed intent: %s", encoded)
					}
					*reply.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: request.ID, Op: request.Op, Status: couchmessage.ReceiptQueued}}
					return nil
				})
				if code != 0 || calls != 1 {
					t.Fatalf("code=%d calls=%d stderr=%s", code, calls, &errout)
				}
			})
		}
	}
}

func TestSwitchAgentCLIRejectsInvalidArgumentIntent(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, extra := range [][]string{
			{"--agent-args"}, {"--agent-args", "'unterminated"}, {"--agent-args", "bad\x00arg"},
			{"--agent-args", strings.Repeat("x", 4097)}, {"--agent-args", "", "--agent-args", ""},
			{"--interrupt-first", "--interrupt-first"}, {"--force"},
		} {
			if _, err := ParseCLI(append([]string{"--switch-agent", "pair:2", "--agent", agent, "--confirm"}, extra...), nil); err == nil {
				t.Fatalf("accepted invalid %s intent %q", agent, extra)
			}
		}
	}
}
