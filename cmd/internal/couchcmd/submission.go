package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	core "github.com/xianxu/pair/cmd/internal/submission"
)

const submissionFlag = "--broadcast-web-submit"

func parseSubmissionCLI(args []string) (cliInvocation, error) {
	if len(args) != 3 || !couchmessage.IsSubmissionOp("submission-"+args[1]) || !core.ValidID(args[2]) {
		return cliInvocation{}, fmt.Errorf("usage: couch %s prepare|status|begin|ack ID", submissionFlag)
	}
	return cliInvocation{kind: cliSubmission, messageOp: "submission-" + args[1], ref: args[2]}, nil
}

// runSubmissionCLI talks only to the live draft's Couch. It does not resolve
// another store or launch a supervisor; a missing conversation fails closed.
func runSubmissionCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	r := callerIdentity(rt, inv.messageOp)
	r.ID = inv.ref
	if err := couchmessage.ValidateRequest(r); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: submission requires a live Couch draft")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, r, &result); err != nil {
		fmt.Fprintf(stderr, "couch: submission %s outcome uncertain: %v; query status with the same ID\n", r.ID, err)
		return 1
	}
	// Keep the structured outcome even on a refusal: the client can retain the
	// exact ID/target while displaying the failure without repeating a send.
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	target := core.Target{Scope: r.Scope, Tag: r.Tag, Session: r.Session, Nonce: r.Nonce}
	if result.Code != "ok" || result.Submission == nil || result.Submission.ID != r.ID || result.Submission.Target != target {
		detail := result.Error
		if detail == "" {
			detail = "submission response missing or mismatched"
		}
		fmt.Fprintln(stderr, "couch:", detail)
		return 1
	}
	if result.Submission.Error != "" || result.Submission.State == "failed" {
		detail := result.Submission.Error
		if detail == "" {
			detail = "submission failed"
		}
		fmt.Fprintln(stderr, "couch:", detail)
		return 1
	}
	return 0
}

func (s *messageService) SetSubmission(run func(context.Context, string, string, core.Target) (couchmessage.SubmissionStatus, error)) {
	s.submission.Store(&run)
}

func (s *messageService) handleSubmission(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	caller, err := s.broker.Caller(r.Scope, r.Tag, r.Session, r.Nonce)
	if err == nil {
		_, err = s.authority.current(ctx, caller)
	}
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: err.Error()}
	}
	run := s.submission.Load()
	if run == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch does not submit cast notes"}
	}
	target := core.Target{Scope: caller.Scope, Tag: caller.Tag, Session: caller.Session, Nonce: caller.Nonce}
	status, err := (*run)(ctx, strings.TrimPrefix(r.Op, "submission-"), r.ID, target)
	response := couchmessage.Response{Code: "ok", Submission: &status}
	if err != nil {
		response.Code, response.Error = "refused", err.Error()
	}
	return response
}
