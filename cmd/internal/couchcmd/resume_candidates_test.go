package couchcmd

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
	"time"
)

func TestResumeCandidatesCLI(t *testing.T) {
	for _, args := range [][]string{{"--resume-candidates", "pair:1", "--json"}, {"--resume", "pair:1", "--candidate", "v1.key"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"--resume-candidates", "pair"}, {"--resume", "pair:1", "--candidate"}, {"--reboot", "pair:1", "--confirm", "--candidate", "key"}, {"--resume", "pair:1", "--candidate", "one", "--candidate", "two"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestResumeCandidatesCLIUsesReadOnlyBoundedRequest(t *testing.T) {
	var out, diagnostic bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1", jsonOutput: true}, messageRuntime(), &out, &diagnostic, func(ctx context.Context, _ string, request any, response any) error {
		calls++
		r := request.(couchmessage.Request)
		if r.Op != "resume-candidates" || r.Target != "pair:1" || r.ID != "" || r.CandidateKey != "" {
			t.Fatalf("unexpected preview request %+v", r)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 35*time.Second || time.Until(deadline) <= 2*time.Second {
			t.Fatalf("bad preview deadline %v", deadline)
		}
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{Key: "opaque", Label: "best"}}}}
		return nil
	})
	if code != 0 || calls != 1 || !strings.Contains(out.String(), "opaque") {
		t.Fatalf("preview code%d calls%d out%s diagnostic%s", code, calls, &out, &diagnostic)
	}
}
