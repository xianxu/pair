package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	core "github.com/xianxu/pair/cmd/internal/submission"
)

func TestSubmissionCLIClosedShape(t *testing.T) {
	for _, op := range []string{"prepare", "status", "begin", "ack"} {
		inv, err := ParseCLI([]string{"--broadcast-web-submit", op, "submission-1"}, couchcore.Operations())
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if inv.messageOp != "submission-"+op || inv.ref != "submission-1" {
			t.Fatalf("%s: %#v", op, inv)
		}
	}
	for _, args := range [][]string{
		{"--broadcast-web-submit"}, {"--broadcast-web-submit", "prepare"},
		{"--broadcast-web-submit", "unknown", "id"}, {"--broadcast-web-submit", "prepare", "../id"},
		{"--broadcast-web-submit", "prepare", "id", "extra"}, {"--broadcast-web-submit", "prepare", "id", "--layout2"},
	} {
		if inv, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %q: %#v", args, inv)
		}
	}
}

func TestSubmissionCLIJSONAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, state, code, detail string
		transport                 bool
		want                      int
	}{
		{name: "preparing", state: "preparing", code: "ok"},
		{name: "resolved", state: "resolved", code: "ok"},
		{name: "empty", state: "empty", code: "ok"},
		{name: "failed", state: "failed", code: "ok", detail: "capture failed", want: 1},
		{name: "refused", state: "prepared", code: "refused", detail: "target changed", want: 1},
		{name: "transport", transport: true, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := messageRuntime()
			inv, err := ParseCLI([]string{submissionFlag, "prepare", "submission-1"}, couchcore.Operations())
			if err != nil {
				t.Fatal(err)
			}
			var out, errout bytes.Buffer
			call := func(ctx context.Context, socket string, request, response any) error {
				r := request.(couchmessage.Request)
				wantSocket, _ := couchmessage.SocketPath(rt.Getenv("COUCH_STORE_DIR"), "broker")
				if socket != wantSocket || r.Scope != "scope" || r.Tag != "tag" || r.Session != "session" || r.Nonce != "nonce" || r.ID != "submission-1" || r.Op != "submission-prepare" {
					t.Fatalf("request: %s %#v", socket, r)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("missing RPC deadline")
				}
				if tc.transport {
					return errors.New("socket gone")
				}
				v := response.(*couchmessage.Response)
				*v = couchmessage.Response{Code: tc.code, Submission: &couchmessage.SubmissionStatus{ID: r.ID, State: tc.state, Path: "/store/submission-1/submission.json", Target: core.Target{Scope: r.Scope, Tag: r.Tag, Session: r.Session, Nonce: r.Nonce}}}
				if tc.code != "ok" {
					v.Error = tc.detail
				} else {
					v.Submission.Error = tc.detail
				}
				return nil
			}
			if got := runSubmissionCLI(inv, rt, &out, &errout, call); got != tc.want {
				t.Fatalf("exit=%d stdout=%s stderr=%s", got, &out, &errout)
			}
			if tc.transport {
				if !strings.Contains(errout.String(), "same ID") {
					t.Fatalf("lost retry advice: %s", &errout)
				}
				return
			}
			var result couchmessage.Response
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Submission == nil || result.Submission.State != tc.state {
				t.Fatalf("lost structured status: %s", &out)
			}
			if tc.want != 0 && !strings.Contains(errout.String(), tc.detail) {
				t.Fatalf("missing error: %s", &errout)
			}
		})
	}
}

func TestSubmissionCLIRequiresLiveDraftContext(t *testing.T) {
	for _, key := range []string{"COUCH_STORE_DIR", "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE"} {
		t.Run(key, func(t *testing.T) {
			rt := messageRuntime()
			delete(rt.env, key)
			inv := cliInvocation{kind: cliSubmission, messageOp: "submission-status", ref: "id"}
			call := func(context.Context, string, any, any) error {
				t.Fatal("called RPC without complete live draft context")
				return nil
			}
			var out, errout bytes.Buffer
			if got := runSubmissionCLI(inv, rt, &out, &errout, call); got == 0 || errout.Len() == 0 {
				t.Fatalf("code=%d stderr=%s", got, &errout)
			}
		})
	}
}

func TestSubmissionCLIRefusesMismatchedTarget(t *testing.T) {
	rt := messageRuntime()
	inv := cliInvocation{kind: cliSubmission, messageOp: "submission-begin", ref: "id"}
	var out, errout bytes.Buffer
	call := func(_ context.Context, _ string, _, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Submission: &couchmessage.SubmissionStatus{ID: "id", State: "delivering", Target: core.Target{Scope: "scope", Tag: "other", Session: "session", Nonce: "nonce"}}}
		return nil
	}
	if got := runSubmissionCLI(inv, rt, &out, &errout, call); got == 0 {
		t.Fatalf("accepted another draft's submission: %s", &out)
	}
}

func TestSubmissionServiceRejectsInvalidAndUnregisteredCaller(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	r.s.SetSubmission(func(context.Context, string, string, core.Target) (couchmessage.SubmissionStatus, error) {
		t.Error("invalid caller reached submission")
		return couchmessage.SubmissionStatus{}, nil
	})
	req := callerRequest(b, "submission-prepare")
	req.ID = "id"
	req.Body = "forbidden"
	if got := r.s.handle(context.Background(), req); got.Code != "invalid-request" {
		t.Fatalf("invalid request: %#v", got)
	}
	req.Body = ""
	req.Nonce = "obsolete"
	if got := r.s.handle(context.Background(), req); got.Code != "unavailable" {
		t.Fatalf("unregistered caller: %#v", got)
	}
}

// Drive the service through its real socket and registered wrapper identity.
// Submission requests must not spend peer budget or admit a stale incarnation.
func TestSubmissionServiceIdentityAndNoPeerQuota(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	target := core.Target{Scope: b.Scope, Tag: b.Tag, Session: b.Session, Nonce: b.Nonce}
	states := map[string]string{"prepare": "prepared", "status": "prepared", "begin": "delivering", "ack": "resolved"}
	calls := 0
	r.s.SetSubmission(func(_ context.Context, op, id string, got core.Target) (couchmessage.SubmissionStatus, error) {
		calls++
		if got != target || id != "id" || states[op] == "" {
			t.Fatalf("callback: %s %s %#v", op, id, got)
		}
		return couchmessage.SubmissionStatus{ID: id, State: states[op], Target: got}, nil
	})
	for i := 0; i < couchmessage.InboundAllowance+2; i++ {
		for _, op := range []string{"prepare", "status", "begin", "ack"} {
			req := callerRequest(b, "submission-"+op)
			req.ID = "id"
			var result couchmessage.Response
			if err := couchmessage.Call(context.Background(), r.brokerSock, req, &result); err != nil {
				t.Fatal(err)
			}
			if result.Code != "ok" || result.Submission == nil || result.Submission.State != states[op] {
				t.Fatalf("response: %#v", result)
			}
		}
	}
	before := calls
	r.world.set(b, func(s *worldSlot) { s.recorded = false })
	for _, op := range []string{"prepare", "status", "begin", "ack"} {
		req := callerRequest(b, "submission-"+op)
		req.ID = "id"
		result := r.s.handle(context.Background(), req)
		if result.Code != "unavailable" || calls != before {
			t.Fatalf("stale identity reached callback: %#v calls=%d", result, calls)
		}
	}
}

func TestSubmissionServicePreservesFailureStatus(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	r.s.SetSubmission(func(_ context.Context, op, id string, target core.Target) (couchmessage.SubmissionStatus, error) {
		return couchmessage.SubmissionStatus{ID: id, State: "delivering", Target: target}, errors.New("ack write failed")
	})
	req := callerRequest(b, "submission-ack")
	req.ID = "id"
	got := r.s.handle(context.Background(), req)
	if got.Code != "refused" || got.Error != "ack write failed" || got.Submission == nil || got.Submission.State != "delivering" {
		t.Fatalf("failure lost outcome: %#v", got)
	}
}
