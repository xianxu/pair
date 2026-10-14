package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestNotifyCLIForms(t *testing.T) {
	for _, args := range [][]string{{"--notify", "ready"}, {"--notify", "--layout2"}, {"--notify", "--title", "Review", "ready"}} {
		inv, err := ParseCLI(args, nil)
		if err != nil || inv.kind != cliMessage || inv.messageOp != "notify" || inv.messageBody != args[len(args)-1] {
			t.Errorf("%q: %+v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"--notify"}, {"--notify", "--title", "Review"}, {"--notify", "--title", "", "ready"}, {"--notify", ""}, {"--notify", "\x1b\a"}, {"--notify", "ready", "extra"}, {"--notify", strings.Repeat("x", 4097)}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
}

func TestNotifyCLIExchangeNeverWritesOSCOrRetries(t *testing.T) {
	for _, outcome := range []string{"ok", "refused", "uncertain", "lost-reply"} {
		t.Run(outcome, func(t *testing.T) {
			inv, err := ParseCLI([]string{"--notify", "ready"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			calls := 0
			code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, request, response any) error {
				calls++
				r := request.(couchmessage.Request)
				if r.Op != "notify" || r.Body != "ready" || r.Scope != "scope" || r.Nonce != "nonce" || r.ID != "" {
					t.Fatalf("request %+v", r)
				}
				if outcome == "lost-reply" {
					return errors.New("connection closed")
				}
				*response.(*couchmessage.Response) = couchmessage.Response{Code: outcome, Error: "test outcome"}
				return nil
			})
			if calls != 1 || (code == 0) != (outcome == "ok") {
				t.Fatalf("calls=%d code=%d stderr=%s", calls, code, &stderr)
			}
			if strings.ContainsAny(out.String()+stderr.String(), "\x1b\a") {
				t.Fatal("CLI wrote a terminal effect")
			}
			if outcome == "ok" && !strings.Contains(out.String(), "written") {
				t.Fatalf("success %q", &out)
			}
			if (outcome == "uncertain" || outcome == "lost-reply") && !strings.Contains(stderr.String(), "do not retry automatically") {
				t.Fatalf("uncertainty %q", &stderr)
			}
		})
	}
}
