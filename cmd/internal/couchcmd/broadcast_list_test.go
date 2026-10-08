package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestFormatBroadcastStatus(t *testing.T) {
	at := time.Date(2026, 10, 8, 14, 2, 11, 0, time.Local)
	cases := []struct {
		name   string
		status *couchmessage.BroadcastStatus
		want   string
	}{
		{"none", nil, "no broadcast"},
		{"live, two viewers", &couchmessage.BroadcastStatus{State: "live", StartedAt: at, Mode: "tunnel", Viewers: 2}, "live since 14:02:11 (tunnel) — 2 viewers"},
		{"live, one viewer", &couchmessage.BroadcastStatus{State: "live", StartedAt: at, Mode: "local-only", Viewers: 1}, "live since 14:02:11 (local-only) — 1 viewer"},
		{"starting has no session yet", &couchmessage.BroadcastStatus{State: "starting"}, "starting — 0 viewers"},
	}
	for _, tc := range cases {
		if got := formatBroadcastStatus(tc.status); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseBroadcastListForms(t *testing.T) {
	for _, args := range [][]string{{"--broadcast-list"}, {"--broadcast-list", "--json"}} {
		inv, err := ParseCLI(args, nil)
		if err != nil || inv.kind != cliMessage || inv.messageOp != "broadcast-status" || inv.jsonOutput != (len(args) == 2) {
			t.Errorf("%v: %+v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"--broadcast-list", "x"}, {"--broadcast-list", "--json", "--json"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}

// The listing never prints the link or its token (operator decision,
// pair#413), in text or JSON, against a real session's token.
func TestBroadcastListNeverPrintsTheLink(t *testing.T) {
	session, err := broadcast.Start(context.Background(), broadcast.Config{Tunnel: &broadcast.FakeTunnel{}, Ping: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Stop(nil); <-session.Done() })
	link := session.Link()
	token := strings.TrimSuffix(link[strings.LastIndex(strings.TrimSuffix(link, "/"), "/")+1:], "/")
	if len(token) < 20 {
		t.Fatalf("could not find the token in %q", link)
	}
	described := session.Status()
	snapshot := couchmessage.BroadcastStatus{State: "live", StartedAt: described.StartedAt, Mode: described.Mode, Viewers: described.Viewers}
	rt := broadcastListRuntime()
	call := func(_ context.Context, path string, request any, response any) error {
		want, _ := couchmessage.SocketPath(rt.StoreDir(), "broker")
		if path != want {
			t.Fatalf("socket %q, want the store's broker %q", path, want)
		}
		if r := request.(couchmessage.Request); r.Op != "broadcast-status" {
			t.Fatalf("op %q", r.Op)
		}
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Broadcast: &snapshot}
		return nil
	}
	for _, jsonOutput := range []bool{false, true} {
		var out, errout bytes.Buffer
		inv := cliInvocation{kind: cliMessage, messageOp: "broadcast-status", jsonOutput: jsonOutput}
		if code := runMessageCLIWithCall(inv, rt, &out, &errout, call); code != 0 {
			t.Fatalf("json=%v: code %d: %s", jsonOutput, code, errout.String())
		}
		if !strings.Contains(out.String(), "live") {
			t.Errorf("json=%v: output %q lacks the state", jsonOutput, out.String())
		}
		if strings.Contains(out.String(), token) || strings.Contains(out.String(), link) {
			t.Fatalf("json=%v: output leaks the link or token: %q", jsonOutput, out.String())
		}
	}
}

func TestBroadcastListWithNoCouchRunning(t *testing.T) {
	call := func(context.Context, string, any, any) error {
		return errors.New("dial unix: connect: no such file or directory")
	}
	var out, errout bytes.Buffer
	inv := cliInvocation{kind: cliMessage, messageOp: "broadcast-status"}
	if code := runMessageCLIWithCall(inv, broadcastListRuntime(), &out, &errout, call); code == 0 || !strings.Contains(errout.String(), "no running couch") {
		t.Fatalf("code %d, stderr %q; want a refusal naming no running couch", code, errout.String())
	}
}

// broadcastListRuntime is a shell outside any slot: a store, no slot identity.
func broadcastListRuntime() testRT {
	return testRT{dir: "/test/store", env: map[string]string{}}
}

// The service answers broadcast-status for a caller with no slot identity, and
// says unsupported when no console installed a provider (pair#413).
func TestBroadcastStatusNeedsNoSlotIdentity(t *testing.T) {
	s := &messageService{}
	request := couchmessage.Request{Op: "broadcast-status"}
	if got := s.handle(context.Background(), request); got.Code != "unsupported" {
		t.Fatalf("no provider: %+v", got)
	}
	s.SetBroadcastStatus(func() (couchmessage.BroadcastStatus, bool) { return couchmessage.BroadcastStatus{}, false })
	if got := s.handle(context.Background(), request); got.Code != "ok" || got.Broadcast != nil {
		t.Fatalf("no broadcast: %+v", got)
	}
	s.SetBroadcastStatus(func() (couchmessage.BroadcastStatus, bool) {
		return couchmessage.BroadcastStatus{State: "live", Viewers: 3}, true
	})
	if got := s.handle(context.Background(), request); got.Code != "ok" || got.Broadcast == nil || got.Broadcast.Viewers != 3 {
		t.Fatalf("live broadcast: %+v", got)
	}
}

// A couch older than #413 refuses the identity-free request with its generic
// identity check; the CLI says to restart rather than print that raw error.
func TestBroadcastListAgainstAnOlderCouchSaysRestart(t *testing.T) {
	for _, refusal := range []string{"operation requires the calling conversation identity", `unknown message operation "broadcast-status"`} {
		call := func(_ context.Context, _ string, _ any, response any) error {
			*response.(*couchmessage.Response) = couchmessage.Response{Code: "invalid-request", Error: refusal}
			return nil
		}
		var out, errout bytes.Buffer
		inv := cliInvocation{kind: cliMessage, messageOp: "broadcast-status"}
		if code := runMessageCLIWithCall(inv, broadcastListRuntime(), &out, &errout, call); code == 0 || !strings.Contains(errout.String(), "restart Couch") {
			t.Errorf("%q: code %d, stderr %q; want the restart hint", refusal, code, errout.String())
		}
	}
}
