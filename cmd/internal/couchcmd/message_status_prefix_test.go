package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Status prefixes cross the real CLI/Unix socket path, after a message has
// actually completed delivery; a miss must not falsely imply lost mail.
func TestMessageCLIStatusPrefixFindsSubmittedMessage(t *testing.T) {
	r := newServiceRig(t)
	from, to := r.connect(1), r.connect(2)
	const id = "97fcc5ef-0000-4000-8000-000000000001"
	if _, err := r.s.broker.Send(context.Background(), from, id, couchmessage.Route{Target: to.Slot}, "prefix status test"); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "submitted receipt", func() bool {
		receipt, err := r.s.broker.Status(from, id)
		return err == nil && receipt.Status == couchmessage.Submitted
	})
	rt := messageRuntime()
	rt.env["COUCH_THREAD_SCOPE"], rt.env["COUCH_THREAD_TAG"] = from.Scope, from.Tag
	rt.env["PAIR_SESSION_NAME"], rt.env["PAIR_LAUNCH_NONCE"] = from.Session, from.Nonce
	call := func(ctx context.Context, _ string, request any, response any) error {
		return couchmessage.Call(ctx, r.brokerSock, request, response)
	}
	for _, query := range []string{id, id[:8]} {
		var out, errout bytes.Buffer
		started := time.Now()
		code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "status", ref: query, jsonOutput: true}, rt, &out, &errout, call)
		t.Logf("query=%s elapsed=%s exit=%d output=%s stderr=%s", query, time.Since(started), code, strings.TrimSpace(out.String()), errout.String())
		var response couchmessage.Response
		if err := json.Unmarshal(out.Bytes(), &response); err != nil || code != 0 || response.Receipt == nil || response.Receipt.Status != couchmessage.Submitted || response.Receipt.Message.ID != id {
			t.Errorf("status %q: exit=%d response=%s error=%v stderr=%s", query, code, out.String(), err, errout.String())
		}
	}
}
