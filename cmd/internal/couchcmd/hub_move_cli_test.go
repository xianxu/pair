package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
)

func TestHubMoveCLITransmitsIntentAndReportsEvidence(t *testing.T) {
	inv := cliInvocation{kind: cliMessage, messageOp: "move-slot", ref: "pair:7,pair:8", toHub: "bench:0", persistHub: true}
	var out, errout bytes.Buffer
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(_ context.Context, _ string, req any, resp any) error {
		r := req.(couchmessage.Request)
		if r.Target != inv.ref || r.ToHub != inv.toHub || !r.Persist || r.ID == "" {
			t.Fatalf("request %+v", r)
		}
		*resp.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: r.ID, Op: r.Op, Target: r.Target, ToHub: r.ToHub, Persist: r.Persist, Status: couchmessage.ReceiptSucceeded}}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "bench:0") || !strings.Contains(out.String(), "persisted") {
		t.Fatalf("code %d out %q err %q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	code = runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, func(context.Context, string, any, any) error { return errors.New("lost reply") })
	if code == 0 || !strings.Contains(errout.String(), "--operation-status") || strings.Contains(errout.String(), "once the slot is live") {
		t.Fatalf("uncertain %q", errout.String())
	}
}
