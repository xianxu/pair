package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func messageRuntime() testRT {
	return testRT{env: map[string]string{"COUCH_STORE_DIR": "/test/namespace", "COUCH_THREAD_SCOPE": "scope", "COUCH_THREAD_TAG": "tag", "PAIR_SESSION_NAME": "session", "PAIR_LAUNCH_NONCE": "nonce"}}
}

func TestMessageCLIUsesCallerContextAndRetainedReceipt(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "couch-cli-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	socket := filepath.Join(root, "broker.sock")
	var mu sync.Mutex
	var saved couchmessage.Request
	s, err := couchmessage.StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		if err := couchmessage.ValidateRequest(request); err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		response := couchmessage.Response{Code: "ok"}
		switch request.Op {
		case "send":
			saved = request
			response.Code = "accepted"
			response.Receipt = &couchmessage.Receipt{Message: couchmessage.Message{ID: request.ID, From: couchmessage.Binding{Slot: "brain:0"}, To: couchmessage.Binding{Slot: "pair:1"}, Body: request.Body}, Status: couchmessage.Queued}
		case "status":
			if request.ID != saved.ID {
				return nil, errors.New("unknown ID")
			}
			response.Receipt = &couchmessage.Receipt{Message: couchmessage.Message{ID: saved.ID, Body: saved.Body}, Status: couchmessage.Submitted}
		case "actors":
			response.Actors = []couchmessage.Candidate{{Binding: couchmessage.Binding{Slot: "pair:1"}, Known: true, Resting: true, Remaining: 8, Supported: true}}
		}
		return json.Marshal(response)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rt := messageRuntime()
	call := func(ctx context.Context, path string, request any, response any) error {
		want, _ := couchmessage.SocketPath(rt.Getenv("COUCH_STORE_DIR"), "broker")
		if path != want {
			t.Fatalf("wrong namespace path %q", path)
		}
		return couchmessage.Call(ctx, socket, request, response)
	}
	var out, errout bytes.Buffer
	inv := cliInvocation{kind: cliMessage, messageOp: "send", ref: "pair:1", messageBody: "literal --layout2"}
	if code := runMessageCLIWithCall(inv, rt, &out, &errout, call); code != 0 {
		t.Fatalf("code %d: %s", code, errout.String())
	}
	mu.Lock()
	sent := saved
	mu.Unlock()
	if sent.Scope != "scope" || sent.Tag != "tag" || sent.Session != "session" || sent.Nonce != "nonce" || sent.Body != inv.messageBody || len(sent.ID) != 36 {
		t.Fatalf("request %#v", sent)
	}
	if !strings.Contains(out.String(), sent.ID) || !strings.Contains(out.String(), "pair:1") || !strings.Contains(out.String(), "queued") {
		t.Fatalf("send output %q", out.String())
	}
	out.Reset()
	errout.Reset()
	inv = cliInvocation{kind: cliMessage, messageOp: "status", ref: sent.ID, jsonOutput: true}
	if code := runMessageCLIWithCall(inv, rt, &out, &errout, call); code != 0 {
		t.Fatalf("status %d %s", code, errout.String())
	}
	var response couchmessage.Response
	if err := strictjson.Decode(out.Bytes(), &response); err != nil || response.Receipt == nil || response.Receipt.Message.Body != sent.Body {
		t.Fatalf("JSON %q %v", out.String(), err)
	}
	out.Reset()
	inv = cliInvocation{kind: cliMessage, messageOp: "actors", jsonOutput: true}
	if code := runMessageCLIWithCall(inv, rt, &out, &errout, call); code != 0 {
		t.Fatalf("actors %d %s", code, errout.String())
	}
	if err := strictjson.Decode(out.Bytes(), &response); err != nil || len(response.Actors) != 1 {
		t.Fatalf("actors JSON %q %v", out.String(), err)
	}
}

func TestMessageCLINoFreeRefusalAndUncertainSend(t *testing.T) {
	for _, tc := range []struct {
		code string
		err  error
		exit int
	}{{code: "not-dispatched"}, {code: "busy", exit: 1}, {err: context.DeadlineExceeded, exit: 1}} {
		var out, errout bytes.Buffer
		calls := 0
		id := ""
		call := func(_ context.Context, _ string, request any, response any) error {
			calls++
			id = request.(couchmessage.Request).ID
			if tc.err != nil {
				return tc.err
			}
			*(response.(*couchmessage.Response)) = couchmessage.Response{Code: tc.code, Error: "no recipient"}
			return nil
		}
		got := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "send", ref: "pair", messageBody: "work"}, messageRuntime(), &out, &errout, call)
		if got != tc.exit || calls != 1 {
			t.Fatalf("%s: exit %d calls %d", tc.code, got, calls)
		}
		if tc.err != nil && (!strings.Contains(errout.String(), id) || !strings.Contains(errout.String(), "--message-status")) {
			t.Fatalf("uncertain send omitted recovery ID: %q", errout.String())
		}
	}
}

func TestMessageCLIRejectsMissingContextBeforeRPC(t *testing.T) {
	var out, errout bytes.Buffer
	called := false
	call := func(context.Context, string, any, any) error { called = true; return nil }
	if code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "actors"}, testRT{}, &out, &errout, call); code == 0 || called {
		t.Fatalf("code %d called %v", code, called)
	}
}

func TestMessageSkillRunsOutsideCouch(t *testing.T) {
	var out, errout bytes.Buffer
	// A nil runtime proves this informational command needs neither env nor a
	// namespace lease, and emits the embedded canonical source verbatim.
	if code := RunWithRuntime([]string{"--skill"}, nil, &out, &errout, nil); code != 0 || out.String() != couchSkill || errout.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errout.String())
	}
}

func TestMessageCLIForwardsAgentAndPrintsAlias(t *testing.T) {
	var out, errout bytes.Buffer
	var sent couchmessage.Request
	call := func(_ context.Context, _ string, request, response any) error {
		r := request.(couchmessage.Request)
		res := response.(*couchmessage.Response)
		switch r.Op {
		case "send":
			sent = r
			res.Code = "accepted"
			res.Receipt = &couchmessage.Receipt{Message: couchmessage.Message{ID: r.ID}, Status: couchmessage.Queued}
		case "actors":
			res.Code = "ok"
			b := couchmessage.Binding{Slot: "xianxu.dev:1", Agent: "claude"}
			res.Actors = []couchmessage.Candidate{{Binding: b, Alias: "blog", Supported: true, Remaining: 8}}
		}
		return nil
	}
	if code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "send", ref: "blog", messageAgent: "codex", messageBody: "work"}, messageRuntime(), &out, &errout, call); code != 0 || sent.Agent != "codex" || sent.Target != "blog" {
		t.Fatalf("send %d %+v %q", code, sent, errout.String())
	}
	out.Reset()
	if code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "actors"}, messageRuntime(), &out, &errout, call); code != 0 || !strings.HasPrefix(out.String(), "xianxu.dev:1 (blog)\tclaude\t") {
		t.Fatalf("actors %d %q", code, out.String())
	}
}
