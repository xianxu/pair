package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type fakeProbeGit struct {
	head   string
	status string // porcelain v2; "" makes status fail
}

func (g fakeProbeGit) Run(dir string, args ...string) (string, error) {
	return g.RunContext(context.Background(), dir, args...)
}
func (g fakeProbeGit) RunContext(_ context.Context, dir string, args ...string) (string, error) {
	for _, a := range args {
		if a == "status" {
			if g.status == "" {
				return "", errors.New("not a work tree")
			}
			return g.status, nil
		}
	}
	if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD" {
		if g.head == "" {
			return "", errors.New("not a repository")
		}
		return g.head + "\n", nil
	}
	return "", errors.New("unexpected git " + strings.Join(args, " "))
}

// The probe's binary half: `pair` as Couch resolves it, its checkout (bin/'s
// parent), PAIR_DEV, and the running wrapper's hash from its session.
func TestLiveRestartProbeBinaryFacts(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "pair", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	pair := filepath.Join(bin, "pair")
	if err := os.WriteFile(pair, []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	p := &liveRestartProbe{
		git:      fakeProbeGit{head: "headrev"},
		lookPath: func(string) (string, error) { return pair, nil },
		getenv:   func(k string) string { return env[k] },
		build: func(string) (couchmessage.BuildIdentity, error) {
			return couchmessage.BuildIdentity{SHA256: "disk", Revision: "builtrev", Modified: true}, nil
		},
	}
	running := couchmessage.SlotLiveness{Build: &couchmessage.BuildIdentity{SHA256: "running"}}
	resolved, _ := filepath.EvalSymlinks(pair)
	got := p.binaryFacts(context.Background(), running)
	want := couchcore.BinaryFacts{RunningSHA: "running", OnDiskPath: resolved, OnDiskSHA: "disk", OnDiskRevision: "builtrev", OnDiskModified: true,
		Checkout: filepath.Dir(filepath.Dir(resolved)), CheckoutHEAD: "headrev"}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	env["PAIR_DEV"] = "1"
	if !p.binaryFacts(context.Background(), running).DevRebuild {
		t.Fatal("PAIR_DEV not seen")
	}
	// A legacy wrapper (no build) and a binary outside a git checkout.
	p.git = fakeProbeGit{}
	got = p.binaryFacts(context.Background(), couchmessage.SlotLiveness{})
	if got.RunningSHA != "" || got.Checkout != "" || got.CheckoutHEAD != "" {
		t.Fatalf("unknowns invented: %+v", got)
	}
}

func TestNotedResultForwardsAndJoins(t *testing.T) {
	n := notedResult{value: couchcore.RelaunchResult{Outcome: couchcore.Relaunched}, note: "built from x"}
	if _, ok := n.Started(); !ok {
		t.Fatal("Started not forwarded")
	}
	out := slotOperationOutcome(n, nil)
	if out.Status != couchmessage.ReceiptSucceeded || out.Warning != "built from x" {
		t.Fatalf("outcome %+v", out)
	}
}

// BR-8 (M2 review): the safety guard's wiring. The probe must find the slot's
// wrapper by THREAD (scope + tag), carry its Settled claim only when the
// wrapper made one, and read dirtiness from the slot's checkout.
func TestLiveRestartProbeFactsMapping(t *testing.T) {
	yes, no := true, false
	svc := &messageService{}
	live := map[string]couchmessage.SlotLiveness{
		"pair:1": {Binding: couchmessage.Binding{Slot: "pair:1", Scope: "s1", Tag: "t1"}, Settled: &yes},
		"pair:2": {Binding: couchmessage.Binding{Slot: "pair:2", Scope: "s1", Tag: "t2"}, Settled: &no},
		"pair:3": {Binding: couchmessage.Binding{Slot: "pair:3", Scope: "s1", Tag: "t3"}}, // legacy: no claim
	}
	svc.liveness.Store(&live)
	clean := "# branch.head main\n"
	dirty := clean + "1 .M N... 100644 100644 100644 a a file\n"
	row := func(tag string, liveRow bool) couchcore.ActionableThreadSummary {
		r := couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: "s1", Tag: couchcore.ThreadTag(tag)}}
		if liveRow {
			r.State = couchcore.ThreadLive
		}
		return r
	}
	for _, tc := range []struct {
		name   string
		row    couchcore.ActionableThreadSummary
		status string
		want   couchcore.LiveRestartFacts
	}{
		{"settled clean", row("t1", true), clean, couchcore.LiveRestartFacts{Live: true, Session: true, SettledKnown: true, Settled: true, GitKnown: true}},
		{"busy dirty", row("t2", true), dirty, couchcore.LiveRestartFacts{Live: true, Session: true, SettledKnown: true, GitKnown: true, Dirty: true}},
		{"legacy wrapper", row("t3", true), clean, couchcore.LiveRestartFacts{Live: true, Session: true, GitKnown: true}},
		{"no session, git unreadable", row("t9", true), "", couchcore.LiveRestartFacts{Live: true}},
		{"not live", row("t1", false), clean, couchcore.LiveRestartFacts{Session: true, SettledKnown: true, Settled: true, GitKnown: true}},
	} {
		// reload-context: the probe gathers no binary facts, so no lookPath.
		p := &liveRestartProbe{git: fakeProbeGit{status: tc.status}}
		p.service.Store(svc)
		got, err := p.LiveRestartFacts(context.Background(), "reload-context", tc.row, "/slot")
		if err != nil || got != tc.want {
			t.Errorf("%s: got %+v %v\nwant %+v", tc.name, got, err, tc.want)
		}
	}
	// No service attached yet (requests racing startup): no session, never idle.
	p := &liveRestartProbe{git: fakeProbeGit{status: clean}}
	if got, _ := p.LiveRestartFacts(context.Background(), "reload-context", row("t1", true), "/slot"); got.Session || got.SettledKnown {
		t.Fatalf("detached probe invented a session: %+v", got)
	}
	// A different scope with the same tag is a different thread.
	if _, ok := svc.LivenessForThread("s2", "t1"); ok {
		t.Fatal("matched across scopes")
	}
}

// BR-7 (M2 review): a relaunch that parked but did not resume needs a
// different recovery than one that never parked; the receipt carries which,
// and the admission note survives the failure.
func TestRelaunchFailureOutcomeIsTyped(t *testing.T) {
	for _, outcome := range []couchcore.RelaunchOutcome{couchcore.ParkIncomplete, couchcore.ParkedNotResumed} {
		value := notedResult{value: couchcore.RelaunchResult{Outcome: outcome}, note: "built from x"}
		out := slotOperationOutcome(value, errors.New("relaunch failed"))
		if out.Status != couchmessage.ReceiptFailed || out.Code != string(outcome) || out.Warning != "built from x" {
			t.Errorf("%s: %+v", outcome, out)
		}
	}
	if out := slotOperationOutcome(couchcore.RelaunchResult{Outcome: couchcore.Relaunched}, nil); out.Code != "" {
		t.Fatalf("success carried a failure code: %+v", out)
	}
}

// pair#421 M3: reload-context signals the broker-held wrapper only after its
// identity re-checks, and succeeds only on a NEW session (a re-exec keeps the
// binding byte-identical).
func TestRestartConversationVerifiedAndConfirmed(t *testing.T) {
	addr := couchcore.ThreadAddress{RepoScope: "s1", Tag: "t1"}
	binding := couchmessage.Binding{Slot: "pair:1", Scope: "s1", Tag: "t1", PID: 4242, Start: "start-4242"}
	setup := func(identity string) (*liveRestartProbe, *messageService, *couchcore.FakeProcOps) {
		svc := &messageService{}
		live := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: binding, Session: 7}}
		svc.liveness.Store(&live)
		proc := couchcore.NewFakeProcOps()
		proc.Set(4242, identity)
		p := &liveRestartProbe{proc: proc, confirmWithin: 200 * time.Millisecond, pollEvery: 5 * time.Millisecond}
		p.service.Store(svc)
		return p, svc, proc
	}
	// The PID now names another process: nothing is signalled.
	p, _, proc := setup("someone-else")
	if err := p.RestartConversation(context.Background(), addr); err == nil || len(proc.Signals[4242]) != 0 {
		t.Fatalf("mismatched identity: %v, signals %v", err, proc.Signals[4242])
	}
	// Verified, signalled, and a new session appears: confirmed.
	p, svc, proc := setup("start-4242")
	go func() {
		time.Sleep(20 * time.Millisecond)
		next := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: binding, Session: 8}}
		svc.liveness.Store(&next)
	}()
	if err := p.RestartConversation(context.Background(), addr); err != nil {
		t.Fatalf("confirmed restart: %v", err)
	}
	if got := proc.Signals[4242]; len(got) != 1 || got[0] != syscall.SIGUSR2 {
		t.Fatalf("signals %v", got)
	}
	// Signalled but the same session stays: unconfirmed, never success.
	p, _, _ = setup("start-4242")
	var unconfirmed *couchcore.ReloadUnconfirmed
	if err := p.RestartConversation(context.Background(), addr); !errors.As(err, &unconfirmed) {
		t.Fatalf("no new session: %v", err)
	}
	if out := slotOperationOutcome(nil, &couchcore.ReloadUnconfirmed{Detail: "x"}); out.Status != couchmessage.ReceiptFailed || out.Code != "unconfirmed" {
		t.Fatalf("receipt %+v", out)
	}
	// No session at all: refused, nothing signalled.
	empty := &liveRestartProbe{proc: couchcore.NewFakeProcOps()}
	empty.service.Store(&messageService{})
	if err := empty.RestartConversation(context.Background(), addr); err == nil {
		t.Fatal("restart without a session")
	}
}

// M2 review advisory: the adapter that attaches the admission note must keep
// it on a failed job too.
func TestWithAdmissionNoteOnSuccessAndFailure(t *testing.T) {
	for _, failed := range []error{nil, errors.New("park did not complete")} {
		var got couchmessage.ReceiptOutcome
		note := "built from x"
		withAdmissionNote(&note, func(v any, err error) { got = slotOperationOutcome(v, err) })(couchcore.RelaunchResult{Outcome: couchcore.ParkIncomplete}, failed)
		if got.Warning != "built from x" {
			t.Errorf("err=%v: note lost: %+v", failed, got)
		}
	}
	none := ""
	var raw any
	withAdmissionNote(&none, func(v any, _ error) { raw = v })("plain", nil)
	if raw != "plain" {
		t.Fatalf("empty note wrapped the value: %#v", raw)
	}
}

func TestParseReloadContext(t *testing.T) {
	got, err := ParseCLI([]string{"--reload-context", "pair:2", "--confirm", "--force-unknown"}, couchcore.Operations())
	if err != nil || got.messageOp != "reload-context" || !got.confirmed || !got.forceUnknown {
		t.Fatalf("%#v %v", got, err)
	}
	if _, err := ParseCLI([]string{"--reload-context", "pair:2"}, couchcore.Operations()); err == nil {
		t.Fatal("reload-context without --confirm")
	}
}
