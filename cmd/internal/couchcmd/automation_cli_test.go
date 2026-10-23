package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestAutomationCLIContract(t *testing.T) {
	for _, action := range []string{"on", "off", "status"} {
		for _, target := range []string{"", "ops:0"} {
			args := []string{"--automation", action}
			if target != "" {
				args = append(args, target)
			}
			inv, err := ParseCLI(args, nil)
			if err != nil {
				t.Fatalf("%v: %v", args, err)
			}
			if inv.messageOp != "automation" || inv.messageBody != action || inv.ref != target {
				t.Fatalf("%v: unexpected invocation %+v", args, inv)
			}
		}
	}
	for _, args := range [][]string{{"--automation"}, {"--automation", "toggle"}, {"--automation", "open"}, {"--automation", "on", "ops:0", "other"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestAutomationCLIExchangeNeverRetries(t *testing.T) {
	for _, outcome := range []string{"ok", "refused", "lost-reply", "missing-status"} {
		t.Run(outcome, func(t *testing.T) {
			inv, err := ParseCLI([]string{"--automation", "status", "ops:0"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			calls := 0
			code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, request, response any) error {
				calls++
				r := request.(couchmessage.Request)
				if r.Op != "automation" || r.Body != "status" || r.Target != "ops:0" || r.Nonce != "nonce" || r.ID != "" {
					t.Fatalf("request %+v", r)
				}
				if outcome == "lost-reply" {
					return errors.New("connection closed")
				}
				got := couchmessage.Response{Code: "refused", Error: "not the hub"}
				if outcome == "ok" {
					got = couchmessage.Response{Code: "ok", Automation: &couchcore.HubAutomationStatus{Hub: "/ops", Startup: "unknown", MissingHubs: []string{"/other"}}}
				}
				if outcome == "missing-status" {
					got.Code = "ok"
				}
				*response.(*couchmessage.Response) = got
				return nil
			})
			if calls != 1 || (code == 0) != (outcome == "ok") {
				t.Fatalf("calls=%d code=%d stderr=%s", calls, code, &stderr)
			}
			if outcome == "ok" && (!strings.Contains(out.String(), "hub automation off") || !strings.Contains(out.String(), "missing at startup: /other")) {
				t.Fatalf("status %q", out.String())
			}
			if outcome == "lost-reply" && !strings.Contains(stderr.String(), "query status before retrying") {
				t.Fatalf("uncertainty %q", stderr.String())
			}
		})
	}
}

// Captured verbatim from the live staged build after both hubs were manually
// attached; authenticated TL receipt fe9ff602-b0c0-4b87-a0b8-a9d9f28aaeb4.
const recoveredAutomationStatusBeforeFix = `/Users/xianxu/workspace/ops: hub automation on (startup: full-shutdown)
missing hub: /Users/xianxu/workspace/ops
missing hub: /Users/xianxu/workspace/ops-echo
`

func TestAutomationCLIRecoveredHubLabelsStartupHistory(t *testing.T) {
	const hub = "/Users/xianxu/workspace/ops"
	s := &messageService{automation: newHubAutomationOwner(map[string]bool{hub: false}, couchcore.StartupFullShutdown, []string{hub, "/Users/xianxu/workspace/ops-echo"})}
	status, err := s.controlAutomation(hub, "on")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := ParseCLI([]string{"--automation", "status"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, _, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Automation: &status}
		return nil
	})
	if code != 0 {
		t.Fatalf("status failed: %s", &stderr)
	}
	want := strings.ReplaceAll(recoveredAutomationStatusBeforeFix, "\nmissing hub:", "\nmissing at startup:")
	if out.String() != want {
		t.Fatalf("recovered status obscures startup history:\ngot %q\nwant %q", out.String(), want)
	}
}
