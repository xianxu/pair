package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestDriveCLIShapes(t *testing.T) {
	for _, args := range [][]string{{"--keys", "pair:2", "text:/model", "enter", "wait:200", "esc"}, {"--set-model", "pair:2", "--model", "gpt-6.1-sol", "--confirm"}, {"--set-model", "pair:2", "--model", "opus", "--scope", "default", "--effort", "high", "--confirm", "--json"}} {
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil || inv.kind != cliMessage {
			t.Fatalf("%v: %#v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"--keys", "pair", "enter"}, {"--keys", "pair:2", "text:x\n"}, {"--keys", "pair:2", "wait:1001"}, {"--set-model", "pair:2", "--model", "opus"}, {"--set-model", "pair:2", "--model", "opus", "--scope", "global", "--confirm"}, {"--keys", "pair:2", "enter", "--confirm"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestDriveCLIPollsWithoutReplayingAdmission(t *testing.T) {
	inv, err := ParseCLI([]string{"--keys", "pair:2", "enter", "--json"}, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	rt := newRT(t)
	rt.env["COUCH_STORE_DIR"] = "/tmp/couch-drive-test"
	rt.env["COUCH_THREAD_SCOPE"] = "scope"
	rt.env["COUCH_THREAD_TAG"] = "tag"
	rt.env["PAIR_SESSION_NAME"] = "session"
	rt.env["PAIR_LAUNCH_NONCE"] = "nonce"
	var out, diagnostic bytes.Buffer
	admissions, statuses := 0, 0
	call := func(_ context.Context, _ string, raw any, result any) error {
		r := raw.(couchmessage.Request)
		response := result.(*couchmessage.Response)
		response.Code = "ok"
		if r.Op == "keys" {
			admissions++
			response.Operation = &couchmessage.OperationReceipt{ID: r.ID, Status: couchmessage.ReceiptQueued}
		} else if r.Op == "operation-status" {
			statuses++
			response.Operation = &couchmessage.OperationReceipt{ID: r.ID, Status: couchmessage.ReceiptSucceeded}
		} else {
			t.Fatalf("unexpected %s", r.Op)
		}
		return nil
	}
	if code := runAgentDriveCLI(inv, rt, &out, &diagnostic, call); code != 0 {
		t.Fatalf("code %d: %s", code, diagnostic.String())
	}
	if admissions != 1 || statuses != 1 {
		t.Fatalf("admissions %d statuses %d", admissions, statuses)
	}
}
func TestDriveCLILostReplyGivesStatusAndPeek(t *testing.T) {
	inv, _ := ParseCLI([]string{"--keys", "pair:2", "enter"}, couchcore.Operations())
	rt := newRT(t)
	rt.env["COUCH_STORE_DIR"] = "/tmp/couch-drive-test"
	rt.env["COUCH_THREAD_SCOPE"] = "scope"
	rt.env["COUCH_THREAD_TAG"] = "tag"
	rt.env["PAIR_SESSION_NAME"] = "session"
	rt.env["PAIR_LAUNCH_NONCE"] = "nonce"
	var out, diagnostic bytes.Buffer
	calls := 0
	code := runAgentDriveCLI(inv, rt, &out, &diagnostic, func(context.Context, string, any, any) error { calls++; return errors.New("lost") })
	if code != 1 || calls != 1 || !strings.Contains(diagnostic.String(), "--operation-status") || !strings.Contains(diagnostic.String(), "--peek pair:2") || strings.Contains(diagnostic.String(), "recover-plan") {
		t.Fatalf("%d %d %s", code, calls, diagnostic.String())
	}
}

func TestDriveCLIRetainsRecoveredChoicesThroughReceipt(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		args := []string{"--set-model", "pair:2", "--model", "missing", "--confirm"}
		if jsonOutput {
			args = append(args, "--json")
		}
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil {
			t.Fatal(err)
		}
		rt := newRT(t)
		for key, value := range map[string]string{"COUCH_STORE_DIR": "/tmp/couch-drive-test", "COUCH_THREAD_SCOPE": "scope", "COUCH_THREAD_TAG": "tag", "PAIR_SESSION_NAME": "session", "PAIR_LAUNCH_NONCE": "nonce"} {
			rt.env[key] = value
		}
		var out, diagnostic bytes.Buffer
		detail := "requested model not among observed choices: opus 5.5, sonnet 4.6; back at empty prompt"
		code := runAgentDriveCLI(inv, rt, &out, &diagnostic, func(_ context.Context, _ string, raw any, result any) error {
			req := raw.(couchmessage.Request)
			receipt, err := couchmessage.ApplyReceiptEvent(couchmessage.OperationReceipt{}, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: req.ID, Op: "set-model"})
			if err != nil {
				return err
			}
			receipt, err = couchmessage.ApplyReceiptEvent(receipt, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptFinish, ID: req.ID, Outcome: couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptFailed, Drive: &agentdrive.Result{Outcome: agentdrive.OutcomeFailed, Detail: detail, Tail: []string{"empty prompt"}}}})
			if err != nil {
				return err
			}
			response := result.(*couchmessage.Response)
			response.Code = "ok"
			response.Operation = &receipt
			return nil
		})
		if code != 1 || !strings.Contains(out.String(), detail) || !strings.Contains(out.String(), "empty prompt") {
			t.Fatalf("json=%v code=%d output=%s diagnostic=%s", jsonOutput, code, out.String(), diagnostic.String())
		}
	}
}
