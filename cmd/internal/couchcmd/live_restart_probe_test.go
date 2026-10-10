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
		settled := true // admission's real case: the wrapper said settled
		live := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: binding, Session: 7, Settled: &settled}}
		svc.liveness.Store(&live)
		proc := couchcore.NewFakeProcOps()
		proc.Set(4242, identity)
		p := &liveRestartProbe{proc: proc, confirmWithin: 200 * time.Millisecond, pollEvery: 5 * time.Millisecond}
		p.service.Store(svc)
		return p, svc, proc
	}
	// The PID now names another process: nothing is signalled.
	p, _, proc := setup("someone-else")
	if err := p.RestartConversation(context.Background(), addr, false); err == nil || len(proc.Signals[4242]) != 0 {
		t.Fatalf("mismatched identity: %v, signals %v", err, proc.Signals[4242])
	}
	// Verified, signalled, and a new session appears: confirmed.
	p, svc, proc := setup("start-4242")
	go func() {
		time.Sleep(20 * time.Millisecond)
		next := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: binding, Session: 8}} // a fresh session has not judged yet
		svc.liveness.Store(&next)
	}()
	if err := p.RestartConversation(context.Background(), addr, false); err != nil {
		t.Fatalf("confirmed restart: %v", err)
	}
	if got := proc.Signals[4242]; len(got) != 1 || got[0] != syscall.SIGUSR2 {
		t.Fatalf("signals %v", got)
	}
	// Signalled but the same session stays: unconfirmed, never success.
	p, _, _ = setup("start-4242")
	var unconfirmed *couchcore.ReloadUnconfirmed
	if err := p.RestartConversation(context.Background(), addr, false); !errors.As(err, &unconfirmed) {
		t.Fatalf("no new session: %v", err)
	}
	if out := slotOperationOutcome(nil, &couchcore.ReloadUnconfirmed{Detail: "x"}); out.Status != couchmessage.ReceiptFailed || out.Code != "unconfirmed" {
		t.Fatalf("receipt %+v", out)
	}
	// No session at all: refused, nothing signalled.
	empty := &liveRestartProbe{proc: couchcore.NewFakeProcOps()}
	empty.service.Store(&messageService{})
	if err := empty.RestartConversation(context.Background(), addr, false); err == nil {
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

// BR-15: a wrapper that is busy NOW is refused at the effect, before any
// signal; unknown passes (admission gated it); a cancelled wait is unconfirmed.
func TestRestartEffectRechecksBusyAndCancellation(t *testing.T) {
	addr := couchcore.ThreadAddress{RepoScope: "s1", Tag: "t1"}
	binding := couchmessage.Binding{Slot: "pair:1", Scope: "s1", Tag: "t1", PID: 4242, Start: "start-4242"}
	no := false
	probe := func(settled *bool) (*liveRestartProbe, *couchcore.FakeProcOps) {
		svc := &messageService{}
		live := map[string]couchmessage.SlotLiveness{"pair:1": {Binding: binding, Session: 7, Settled: settled}}
		svc.liveness.Store(&live)
		proc := couchcore.NewFakeProcOps()
		proc.Set(4242, "start-4242")
		p := &liveRestartProbe{proc: proc, confirmWithin: time.Second, pollEvery: 5 * time.Millisecond}
		p.service.Store(svc)
		return p, proc
	}
	p, proc := probe(&no)
	if err := p.RestartConversation(context.Background(), addr, false); err == nil || len(proc.Signals[4242]) != 0 {
		t.Fatalf("busy at effect: %v signals %v", err, proc.Signals[4242])
	}
	if err := p.ConfirmNotBusy(context.Background(), addr, true); err == nil {
		t.Fatal("relaunch re-check passed a busy slot, even forced")
	}
	p, _ = probe(nil)
	if err := p.ConfirmNotBusy(context.Background(), addr, true); err != nil {
		t.Fatalf("unknown at effect after a forced admission must pass: %v", err)
	}
	// Admission saw it known-settled; now it is unknown: never assumed idle.
	if err := p.ConfirmNotBusy(context.Background(), addr, false); err == nil {
		t.Fatal("unknown at effect passed without a forced admission")
	}
	if err := p.RestartConversation(context.Background(), addr, false); err == nil {
		t.Fatal("reload signalled an unknown wrapper without a forced admission")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var unconfirmed *couchcore.ReloadUnconfirmed
	if err := p.RestartConversation(ctx, addr, true); !errors.As(err, &unconfirmed) || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("cancelled wait: %v", err)
	}
	if out := slotOperationOutcome(couchcore.ReloadContextResult{Address: addr}, nil); out.Tag != "t1" {
		t.Fatalf("reload success names no thread: %+v", out)
	}
}

func TestReloadContextRefusesSameBinary(t *testing.T) {
	if _, err := ParseCLI([]string{"--reload-context", "pair:2", "--confirm", "--same-binary"}, couchcore.Operations()); err == nil {
		t.Fatal("--same-binary accepted on reload-context")
	}
}

// pair#427 Spec 3: a relaunch or reload-context receipt reports success only
// once a NEW session says it is settled. Driven from admission (prepare's
// readinessAfter) through awaitReadiness to the receipt the CLI polls.
func TestRestartReceiptWaitsForReadiness(t *testing.T) {
	binding := couchmessage.Binding{Slot: "pair:1", Scope: "s1", Tag: "t1", PID: 4242, Start: "start-4242"}
	args := map[string]string{"repo-scope": "s1", "tag": "t1"}
	yes, no := true, false
	store := func(svc *messageService, l couchmessage.SlotLiveness) {
		live := map[string]couchmessage.SlotLiveness{"pair:1": l}
		svc.liveness.Store(&live)
	}
	setup := func(t *testing.T, op string) (*liveRestartProbe, *messageService, func() couchmessage.OperationReceipt, func(any, error)) {
		svc := &messageService{}
		store(svc, couchmessage.SlotLiveness{Binding: binding, Session: 7, Build: &couchmessage.BuildIdentity{}, Settled: &yes})
		p := &liveRestartProbe{readyWithin: time.Second, pollEvery: 2 * time.Millisecond}
		p.service.Store(svc)
		r, runner, _ := slotRig(t)
		b := r.world.add(0)
		req := slotRequest(b, op, "id", "pair:1")
		req.Confirmed = op != "resume"
		if resp := r.s.handle(context.Background(), req); resp.Code != "accepted" {
			t.Fatalf("admit %+v", resp)
		}
		job := runner.job(0)
		job.started()
		await := p.readinessAfter(op, args) // prepare, before the effect
		status := func() couchmessage.OperationReceipt {
			return *r.s.handle(context.Background(), statusRequest(b, "id")).Operation
		}
		return p, svc, status, awaitReadiness(&await, job.finished)
	}
	relaunched := couchcore.RelaunchResult{Outcome: couchcore.Relaunched}

	for _, op := range []string{couchcore.OpRelaunch, couchcore.OpReloadContext} {
		t.Run(op+" ready", func(t *testing.T) {
			_, svc, status, finish := setup(t, op)
			finish(relaunched, nil)
			time.Sleep(20 * time.Millisecond)
			if got := status(); got.Status != couchmessage.ReceiptRunning {
				t.Fatalf("the old session's settled counted: %+v", got)
			}
			store(svc, couchmessage.SlotLiveness{Binding: binding, Session: 8, Build: &couchmessage.BuildIdentity{}, Settled: &no})
			time.Sleep(20 * time.Millisecond)
			if got := status(); got.Status != couchmessage.ReceiptRunning {
				t.Fatalf("a busy new session counted: %+v", got)
			}
			store(svc, couchmessage.SlotLiveness{Binding: binding, Session: 8, Build: &couchmessage.BuildIdentity{}, Settled: &yes})
			if !restartWaitFor(func() bool { return status().Status == couchmessage.ReceiptSucceeded }) {
				t.Fatalf("settled new session: %+v", status())
			}
		})
	}
	t.Run("unready", func(t *testing.T) {
		p, svc, status, finish := setup(t, couchcore.OpRelaunch)
		p.readyWithin = 30 * time.Millisecond
		store(svc, couchmessage.SlotLiveness{Binding: binding, Session: 8, Build: &couchmessage.BuildIdentity{}, Settled: &no})
		finish(relaunched, nil)
		if !restartWaitFor(func() bool { return status().Status.Terminal() }) {
			t.Fatal("the readiness wait is unbounded")
		}
		if got := status(); got.Status != couchmessage.ReceiptFailed || got.Code != "unready" || !strings.Contains(got.Detail, "do not restart it again") {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("wrapper cannot report", func(t *testing.T) {
		_, svc, status, finish := setup(t, couchcore.OpReloadContext)
		store(svc, couchmessage.SlotLiveness{Binding: binding, Session: 8})
		finish(couchcore.ReloadContextResult{}, nil)
		if !restartWaitFor(func() bool { return status().Status.Terminal() }) || status().Code != "unready" {
			t.Fatalf("%+v", status())
		}
	})
	t.Run("other verbs and failures do not wait", func(t *testing.T) {
		p := &liveRestartProbe{}
		p.service.Store(&messageService{})
		if p.readinessAfter("resume", args) != nil || p.readinessAfter(couchcore.OpRelaunch, map[string]string{}) != nil || (*liveRestartProbe)(nil).readinessAfter(couchcore.OpRelaunch, args) != nil {
			t.Fatal("a wait for a verb that is not a restart")
		}
		_, _, status, finish := setup(t, couchcore.OpRelaunch)
		finish(nil, errors.New("park did not complete"))
		if got := status(); got.Status != couchmessage.ReceiptFailed || got.Code == "unready" {
			t.Fatalf("a failed restart waited: %+v", got)
		}
	})
}

func restartWaitFor(ok func() bool) bool {
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		if ok() {
			return true
		}
	}
	return ok()
}
