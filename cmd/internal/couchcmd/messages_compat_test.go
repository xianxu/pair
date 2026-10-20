package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// legacy540Request freezes Request from ccc0ee88ef133fdcf1d5902e9d505e014d046e48,
// cmd/internal/couchmessage/protocol.go (the supervisor running before #540).
// In particular it has no Title. Do not replace this with the current Request:
// the old supervisor strictly rejects fields it does not recognize.
type legacy540Request struct {
	Op, Scope, Tag, Session, Nonce, ID, Target, Body string
	Agent                                            string `json:",omitempty"`
	Confirmed                                        bool   `json:",omitempty"`
	SameBinary                                       bool   `json:",omitempty"`
	ForceUnknown                                     bool   `json:",omitempty"`
	TailScope                                        string `json:",omitempty"`
	TailTag                                          string `json:",omitempty"`
	Lines                                            int    `json:",omitempty"`
	Binding                                          *couchmessage.Binding
}

func TestMessageCLIPre540SupervisorCompatibility(t *testing.T) {
	root, err := os.MkdirTemp("", "pcc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	broker := couchmessage.NewBroker(context.Background(), time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
	t.Cleanup(func() { _ = broker.Close() })
	caller := couchmessage.Binding{Slot: "pair:7", Repository: "/repo/.git", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "codex", Version: "legacy", PID: 1, Start: "start"}
	target := caller
	target.Slot, target.Tag, target.Session = "pair:2", "target", "target-session"
	for _, binding := range []couchmessage.Binding{caller, target} {
		if err := broker.Register(binding, serviceEndpointFake{}); err != nil {
			t.Fatal(err)
		}
	}
	var requests atomic.Int32
	socket := filepath.Join(root, "broker.sock")
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		requests.Add(1)
		var legacy legacy540Request
		// Exactly the old message_service.go strict-decode error response.
		if err := strictjson.Decode(raw, &legacy); err != nil {
			return json.Marshal(couchmessage.Response{Code: "invalid-request", Error: err.Error()})
		}
		// Existing operations use the real stateful broker: receipt ownership and
		// caller/recipient liveness are not synthesized success responses.
		var request couchmessage.Request
		encoded, err := json.Marshal(legacy)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(encoded, &request); err != nil {
			return nil, err
		}
		switch legacy.Op {
		case "actors", "send", "status":
			return json.Marshal(couchmessage.Handle(ctx, broker, request))
		default:
			return json.Marshal(couchmessage.Response{Code: "invalid-request", Error: "unknown message operation"})
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	call := func(ctx context.Context, _ string, request, response any) error {
		return couchmessage.Call(ctx, socket, request, response)
	}
	run := func(inv cliInvocation) couchmessage.Response {
		t.Helper()
		inv.jsonOutput = true
		var out, stderr bytes.Buffer
		if code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, call); code != 0 {
			t.Fatalf("%s: exit %d, stdout=%s stderr=%s", inv.messageOp, code, &out, &stderr)
		}
		var response couchmessage.Response
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	actors := run(cliInvocation{kind: cliMessage, messageOp: "actors"})
	found := false
	for _, actor := range actors.Actors {
		if actor.Binding == target {
			found = true
		}
	}
	if !found {
		t.Fatalf("registered target missing: %+v", actors)
	}
	sent := run(cliInvocation{kind: cliMessage, messageOp: "send", ref: target.Slot, messageBody: "legacy delivery"})
	if sent.Code != "accepted" || sent.Receipt == nil || sent.Receipt.Message.To != target || sent.Receipt.Message.From != caller {
		t.Fatalf("send: %+v", sent)
	}
	status := run(cliInvocation{kind: cliMessage, messageOp: "status", ref: sent.Receipt.Message.ID})
	if status.Receipt == nil || status.Receipt.Message.ID != sent.Receipt.Message.ID || status.Receipt.Message.Body != "legacy delivery" {
		t.Fatalf("status: %+v", status)
	}
	if requests.Load() != 3 {
		t.Fatalf("existing operations made %d requests", requests.Load())
	}

	// A restart may leave the legacy broker without this running wrapper's
	// registration. Exercise that real broker state, not a fabricated response.
	broker.Disconnect(caller)
	for _, op := range []string{"actors", "send"} {
		t.Run("missing-caller-"+op, func(t *testing.T) {
			var out, stderr bytes.Buffer
			before := requests.Load()
			inv := cliInvocation{kind: cliMessage, messageOp: op, ref: target.Slot, messageBody: "must not dispatch"}
			if code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, call); code != 1 {
				t.Fatalf("missing caller: exit=%d stdout=%s stderr=%s", code, &out, &stderr)
			}
			if !strings.Contains(stderr.String(), "caller lookup") || strings.Contains(stderr.String(), "restart Couch") {
				t.Errorf("missing registration lost caller evidence or gained restart advice: %s", &stderr)
			}
			if strings.Contains(stderr.String(), "older than") {
				t.Errorf("registration absence does not prove version skew: %s", &stderr)
			}
			if requests.Load() != before+1 {
				t.Fatalf("retried request: %d -> %d", before, requests.Load())
			}
		})
	}

	// The newly added verb cannot be interpreted by the frozen legacy decoder.
	// It must refuse without retrying or claiming a desktop notification happened.
	inv, err := ParseCLI([]string{"--notify", "ready"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, call); code != 1 || out.Len() != 0 {
		t.Fatalf("notify: exit=%d stdout=%s stderr=%s", code, &out, &stderr)
	}
	if !strings.Contains(stderr.String(), "running Couch does not support --notify") || !strings.Contains(stderr.String(), "restart Couch") {
		t.Fatalf("notify lacks unsupported-command recovery: %s", &stderr)
	}
	if requests.Load() != 6 {
		t.Fatalf("notify repeated a write: %d requests", requests.Load())
	}
}
