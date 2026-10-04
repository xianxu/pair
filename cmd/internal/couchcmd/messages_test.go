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
	"time"

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

// slotScript answers the n-th call of one slot-operation invocation: call 0
// is the admission, every later call a status poll.
type slotScript func(n int, r couchmessage.Request) (couchmessage.Response, error)

func receiptResponse(code string, status couchmessage.ReceiptStatus, mutate ...func(*couchmessage.OperationReceipt)) couchmessage.Response {
	receipt := couchmessage.OperationReceipt{ID: "id", Op: "resume", Target: "pair:1", Status: status}
	for _, m := range mutate {
		m(&receipt)
	}
	return couchmessage.Response{Code: code, Operation: &receipt}
}

// admitThen admits, then answers polls from statuses in order (the last one
// repeats).
func admitThen(statuses ...couchmessage.Response) slotScript {
	return func(n int, r couchmessage.Request) (couchmessage.Response, error) {
		if n == 0 {
			return receiptResponse("accepted", couchmessage.ReceiptQueued), nil
		}
		return statuses[min(n-1, len(statuses)-1)], nil
	}
}

type slotRun struct {
	code          int
	stdout        string
	stderr        string
	calls         []couchmessage.Request
	deadlines     []time.Time
	contextFaults int
	elapsed       time.Duration
}

func runSlotScript(t *testing.T, inv cliInvocation, rt testRT, script slotScript) slotRun {
	t.Helper()
	var run slotRun
	var out, errout bytes.Buffer
	now := time.Unix(5000, 0)
	clock := slotPollClock{now: func() time.Time { return now }, sleep: func(d time.Duration) { now = now.Add(d) }}
	call := func(ctx context.Context, _ string, request any, response any) error {
		r := request.(couchmessage.Request)
		deadline, ok := ctx.Deadline()
		if !ok || ctx.Err() != nil || time.Until(deadline) > couchmessage.AdmissionTimeout {
			run.contextFaults++
		}
		run.deadlines = append(run.deadlines, deadline)
		n := len(run.calls)
		run.calls = append(run.calls, r)
		resp, err := script(n, r)
		if err != nil {
			return err
		}
		*(response.(*couchmessage.Response)) = resp
		return nil
	}
	start := now
	run.code = runSlotOperationCLI(inv, rt, &out, &errout, call, clock)
	run.stdout, run.stderr, run.elapsed = out.String(), errout.String(), now.Sub(start)
	return run
}

const uncertainLine = "outcome uncertain; verify with couch --recover-plan-from-sdlc"

// TestSlotOperationCLIPollLoop: one strategy per outcome of the admit-then-poll
// loop. Every lost or unreadable outcome after admission is uncertain, never a
// raw transport error alone.
func TestSlotOperationCLIPollLoop(t *testing.T) {
	resume := cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1"}
	succeeded := receiptResponse("ok", couchmessage.ReceiptSucceeded, func(r *couchmessage.OperationReceipt) { r.Tag = "t9" })
	running := receiptResponse("ok", couchmessage.ReceiptRunning)
	for _, tc := range []struct {
		why    string
		script slotScript
		code   int
		stdout string // substring
		stderr string // substring
		calls  int    // 0 = any
	}{
		{"queued → running → succeeded reports the tag", admitThen(running, succeeded), 0, "pair:1 resume succeeded (tag t9)", "", 3},
		{"a refusal at completion names its code and fact", admitThen(receiptResponse("ok", couchmessage.ReceiptRefused, func(r *couchmessage.OperationReceipt) { r.Code, r.Detail = "not-offered", "pair:1 is live" })), 1, "", "refused (not-offered): pair:1 is live", 2},
		{"a failure names its diagnostic", admitThen(receiptResponse("ok", couchmessage.ReceiptFailed, func(r *couchmessage.OperationReceipt) { r.Diagnostic, r.Detail = "attach-failed", "pane gone" })), 1, "", "failed (attach-failed): pane gone", 2},
		{"an admission refusal is a refusal, not uncertain", func(int, couchmessage.Request) (couchmessage.Response, error) {
			return couchmessage.Response{Code: "unavailable", Error: "caller not live"}, nil
		}, 1, "", "couch: unavailable: caller not live", 1},
		{"polling past the budget is uncertain", admitThen(running), 1, "", uncertainLine, 0},
		{"a receipt Couch no longer holds is uncertain", admitThen(receiptResponse("ok", couchmessage.ReceiptUnknown)), 1, "", uncertainLine, 2},
		{"a dial error mid-poll is uncertain", func(n int, r couchmessage.Request) (couchmessage.Response, error) {
			if n == 0 {
				return receiptResponse("accepted", couchmessage.ReceiptQueued), nil
			}
			return couchmessage.Response{}, errors.New("dial unix broker.sock: connect: no such file or directory")
		}, 1, "", uncertainLine, 2},
		{"an unavailable caller mid-poll (Couch restarted) is uncertain", admitThen(couchmessage.Response{Code: "unavailable", Error: "no binding"}), 1, "", uncertainLine, 2},
		{"a per-call timeout mid-poll is uncertain", func(n int, r couchmessage.Request) (couchmessage.Response, error) {
			if n == 0 {
				return receiptResponse("accepted", couchmessage.ReceiptQueued), nil
			}
			return couchmessage.Response{}, context.DeadlineExceeded
		}, 1, "", uncertainLine, 2},
		{"an admission transport error is uncertain and names the request", func(int, couchmessage.Request) (couchmessage.Response, error) {
			return couchmessage.Response{}, context.DeadlineExceeded
		}, 1, "", uncertainLine, 1},
		{"an already-terminal duplicate admission needs no poll", func(int, couchmessage.Request) (couchmessage.Response, error) {
			return receiptResponse("accepted", couchmessage.ReceiptSucceeded, func(r *couchmessage.OperationReceipt) { r.Tag = "t9" }), nil
		}, 0, "pair:1 resume succeeded (tag t9)", "", 1},
	} {
		t.Run(tc.why, func(t *testing.T) {
			run := runSlotScript(t, resume, messageRuntime(), tc.script)
			if run.code != tc.code || !strings.Contains(run.stdout, tc.stdout) || !strings.Contains(run.stderr, tc.stderr) || tc.calls != 0 && len(run.calls) != tc.calls {
				t.Fatalf("exit %d, %d calls, stdout %q, stderr %q", run.code, len(run.calls), run.stdout, run.stderr)
			}
			if tc.stderr == uncertainLine && !strings.Contains(run.stderr, run.calls[0].ID) {
				t.Fatalf("uncertain outcome omits the request ID: %q", run.stderr)
			}
			if run.contextFaults != 0 {
				t.Fatalf("%d calls ran on an expired or overlong context", run.contextFaults)
			}
		})
	}
}

// The admission and every poll carry the caller's identity, one request ID,
// and a fresh per-call context: the 6th poll, past the 2 s a single
// invocation-wide context allowed, still succeeds.
func TestSlotOperationCLIPollsWithFreshContexts(t *testing.T) {
	statuses := []couchmessage.Response{}
	for range 5 {
		statuses = append(statuses, receiptResponse("ok", couchmessage.ReceiptRunning))
	}
	statuses = append(statuses, receiptResponse("ok", couchmessage.ReceiptSucceeded))
	run := runSlotScript(t, cliInvocation{kind: cliMessage, messageOp: "reboot", ref: "pair:1", confirmed: true}, messageRuntime(), admitThen(statuses...))
	if run.code != 0 || len(run.calls) != 7 || run.elapsed <= couchmessage.AdmissionTimeout {
		t.Fatalf("exit %d after %d calls over %s: %q %q", run.code, len(run.calls), run.elapsed, run.stdout, run.stderr)
	}
	admit := run.calls[0]
	if admit.Op != "reboot" || admit.Target != "pair:1" || !admit.Confirmed || admit.Scope != "scope" || admit.Nonce != "nonce" || len(admit.ID) != 36 {
		t.Fatalf("admission %+v", admit)
	}
	for i, poll := range run.calls[1:] {
		if poll.Op != "operation-status" || poll.ID != admit.ID || poll.Target != "" || poll.Tag != "tag" {
			t.Fatalf("poll %d = %+v", i, poll)
		}
	}
	for i := 1; i < len(run.deadlines); i++ {
		if !run.deadlines[i].After(run.deadlines[i-1]) {
			t.Fatalf("call %d reused an earlier context deadline", i)
		}
	}
	if run.contextFaults != 0 {
		t.Fatalf("%d calls ran on an expired or overlong context", run.contextFaults)
	}
}

func TestSlotOperationCLIJSONPrintsTheFinalReceipt(t *testing.T) {
	run := runSlotScript(t, cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1", jsonOutput: true}, messageRuntime(),
		admitThen(receiptResponse("ok", couchmessage.ReceiptSucceeded, func(r *couchmessage.OperationReceipt) { r.Tag = "t9" })))
	var receipt couchmessage.OperationReceipt
	if err := strictjson.Decode([]byte(run.stdout), &receipt); err != nil || run.code != 0 || receipt.Status != couchmessage.ReceiptSucceeded || receipt.Tag != "t9" {
		t.Fatalf("exit %d stdout %q: %v", run.code, run.stdout, err)
	}
}

func TestSlotOperationCLIRequiresALiveSlot(t *testing.T) {
	run := runSlotScript(t, cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1"}, testRT{}, admitThen())
	if run.code != 1 || len(run.calls) != 0 || !strings.Contains(run.stderr, "requires a live Couch slot") {
		t.Fatalf("exit %d calls %d stderr %q", run.code, len(run.calls), run.stderr)
	}
}
