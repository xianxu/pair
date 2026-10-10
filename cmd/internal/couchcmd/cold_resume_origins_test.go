package couchcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// coldResumeRig is a running Console over a test Couch whose only thread is a
// parked :0 (verified park, conversation bound), so resume is a COLD launch:
// a fresh Pair session that must register before the thread is live. The
// runner's pane comes up after the helper is released, as Pair's does.
type coldResumeRig struct {
	rt      testRT
	c       *couchcore.Couch
	console *couchtty.Console
	input   io.Writer
	host    *hostty.FakeHost
	parked  couchcore.ThreadRecord
	calls   chan dispatchedCall
	address string
}

type dispatchedCall struct {
	call couchcore.OperationCall
	err  error
}

func newColdResumeRig(t *testing.T) *coldResumeRig {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(root, "repo")
	if err := os.MkdirAll(primary, 0o700); err != nil {
		t.Fatal(err)
	}
	rt := newRT(t, primary)
	rt.runner = couchcore.NewFakeRunner()
	parked := seedVerifiedPark(t, rt, primary)
	rt.artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	rt.runner.AfterAcknowledge = func(id string) error {
		// Pair's create path: the agent pane is born, then its session is
		// up and owned.
		rt.artifacts.SetPaneSidecar(parked.Address, "claude")
		rt.artifacts.SetPairSession(parked.Address, managedChildSession(t, rt.runner, id), true)
		return nil
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	address, slot, rest := "repo:0", 0, "main"
	identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: root,
		EnvironmentRoot: root, WorktreeRoot: primary, Kind: "primary", Address: &address, Slot: &slot, RestingBranch: &rest}
	c.Slots = &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{primary: identity}, Repositories: map[string]couchcore.SlotRepository{primary: {Identity: identity}}}
	if err := c.Threads.EnrollSlotRepository(context.Background(), couchcore.SlotRepository{Identity: identity}); err != nil {
		t.Fatal(err)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
	reader, input := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = input.Close() })
	console := couchtty.New(host, reader)
	t.Cleanup(console.Stop)
	initial := ptychild.NewFakeChild(nil)
	console.Attach("initial-console-pane", "initial", initial)
	t.Cleanup(func() { initial.Exit(0) })
	rt.runner.AfterBlockedStart = func(id string) {
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	wireResolver(console, c)
	r := &coldResumeRig{rt: rt, c: c, console: console, input: input, host: host, parked: parked, calls: make(chan dispatchedCall, 16), address: address}
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		value, err := actual(call)
		r.calls <- dispatchedCall{call, err}
		return value, err
	})
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Console did not stop")
		}
	})
	return r
}

// awaitResumeAndAttach waits for the resume and its adoption and returns the
// resume's error.
func (r *coldResumeRig) awaitResumeAndAttach(t *testing.T) couchcore.OperationCall {
	t.Helper()
	var resume couchcore.OperationCall
	for {
		select {
		case d := <-r.calls:
			if d.err != nil {
				t.Fatalf("%s failed: %v", d.call.Name, d.err)
			}
			if d.call.Name == "resume" {
				resume = d.call
			}
			if d.call.Name == "attach" {
				return resume
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("no resume and attach: %s", r.host.Written())
		}
	}
}

// assertRegisteredLive: the cold launch registered (the record's incarnation
// is live) and the thread reads live.
func (r *coldResumeRig) assertRegisteredLive(t *testing.T) {
	t.Helper()
	record, err := r.c.Threads.GetThread(r.parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	live := false
	for _, inc := range record.Incarnations {
		if inc.State == couchcore.IncarnationLive {
			live = true
			r.rt.proc.Set(inc.PID, inc.Identity)
		}
	}
	if !live {
		t.Fatalf("cold resume left no live incarnation: %+v", record.Incarnations)
	}
	rows, err := r.c.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Address == r.parked.Address && row.State != couchcore.ThreadLive {
			t.Fatalf("thread reads %s (%s) after the cold resume", row.State, row.Reason)
		}
	}
}

// A cold resume of a parked :0 registers and ends live from either origin:
// the operator's switcher Enter, and a remote `couch --resume repo:0` from a
// live slot (pair#367 smoke test, tools:0).
func TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins(t *testing.T) {
	t.Run("switcher", func(t *testing.T) {
		r := newColdResumeRig(t)
		if _, err := r.input.Write([]byte{0}); err != nil {
			t.Fatal(err)
		}
		waitWarmAcceptance(t, "switcher row", func() bool { return strings.Contains(r.host.Written(), "threads") })
		if _, err := r.input.Write([]byte{'\r'}); err != nil {
			t.Fatal(err)
		}
		if resume := r.awaitResumeAndAttach(t); resume.Args["warm-only"] != "" {
			t.Fatalf("a parked row resumed warm-only: %+v", resume)
		}
		r.assertRegisteredLive(t)
	})
	t.Run("remote", func(t *testing.T) {
		r := newColdResumeRig(t)
		service := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}, slotOps: consoleSlotOperations(r.console, r.c, nil)}
		service.init()
		caller := service.connect(1)
		env := testRT{env: map[string]string{"COUCH_STORE_DIR": "/couch/namespace", "COUCH_THREAD_SCOPE": caller.Scope, "COUCH_THREAD_TAG": caller.Tag,
			"PAIR_SESSION_NAME": caller.Session, "PAIR_LAUNCH_NONCE": caller.Nonce}}
		call := func(ctx context.Context, _ string, request any, response any) error {
			return couchmessage.Call(ctx, service.brokerSock, request, response)
		}
		var out, errout bytes.Buffer
		if code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume", ref: r.address}, env, &out, &errout, call); code != 0 {
			t.Fatalf("remote resume: exit %d, stdout %q, stderr %q", code, out.String(), errout.String())
		}
		if resume := r.awaitResumeAndAttach(t); resume.Args["warm-only"] != "" {
			t.Fatalf("a parked row resumed warm-only: %+v", resume)
		}
		r.assertRegisteredLive(t)
	})
}
