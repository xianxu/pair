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
)

// TestSlotOperationSocketAcceptance is report step → socket → Couch, end to
// end (pair#367 Task 2.7): `couch --resume repo:0` from a connected caller's
// environment crosses the real message service, the production slot-operation
// wiring, the console's queue and dispatcher, the resume transaction and the
// console's background adoption, then polls its receipt to success. Only the
// terminal and the external session processes are fake.
func TestSlotOperationSocketAcceptance(t *testing.T) {
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
	thread := seedDetachedThread(t, rt, primary)
	session := "pair-" + string(thread.Address.Tag)
	rt.artifacts.SetDetachedSession(thread.Address, session)
	rt.runner.AfterAcknowledge = func(string) error {
		rt.artifacts.SetDetachedSession(thread.Address, "")
		rt.artifacts.SetPairSession(thread.Address, session, true)
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
	type dispatched struct {
		call couchcore.OperationCall
		err  error
	}
	calls := make(chan dispatched, 16)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		value, err := actual(call)
		calls <- dispatched{call, err}
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

	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}, slotOps: consoleSlotOperations(console, c, nil)}
	r.init()
	caller := r.connect(0)
	callerEnv := testRT{env: map[string]string{"COUCH_STORE_DIR": "/couch/namespace", "COUCH_THREAD_SCOPE": caller.Scope, "COUCH_THREAD_TAG": caller.Tag,
		"PAIR_SESSION_NAME": caller.Session, "PAIR_LAUNCH_NONCE": caller.Nonce}}
	call := func(ctx context.Context, _ string, request any, response any) error {
		return couchmessage.Call(ctx, r.brokerSock, request, response)
	}
	resume := cliInvocation{kind: cliMessage, messageOp: "resume", ref: address}

	// Refused before anything is enqueued: outside a slot, and a caller
	// whose identity is no live binding.
	var out, errout bytes.Buffer
	if code := runMessageCLIWithCall(resume, testRT{}, &out, &errout, call); code != 1 || !strings.Contains(errout.String(), "requires a live Couch slot") {
		t.Fatalf("outside a slot: exit %d, %q", code, errout.String())
	}
	forged := testRT{env: map[string]string{}}
	for k, v := range callerEnv.env {
		forged.env[k] = v
	}
	forged.env["PAIR_LAUNCH_NONCE"] = "forged"
	out.Reset()
	errout.Reset()
	if code := runMessageCLIWithCall(resume, forged, &out, &errout, call); code != 1 || !strings.Contains(errout.String(), "unavailable") {
		t.Fatalf("forged caller: exit %d, %q", code, errout.String())
	}
	select {
	case d := <-calls:
		t.Fatalf("a refused caller dispatched %s", d.call.Name)
	default:
	}

	out.Reset()
	errout.Reset()
	if code := runMessageCLIWithCall(resume, callerEnv, &out, &errout, call); code != 0 {
		t.Fatalf("resume: exit %d, stdout %q, stderr %q", code, out.String(), errout.String())
	}
	if !strings.HasPrefix(out.String(), address+" resume succeeded (tag "+string(thread.Address.Tag)+")") {
		t.Fatalf("stdout %q", out.String())
	}
	var seen []string
	for len(seen) < 2 {
		select {
		case d := <-calls:
			seen = append(seen, d.call.Name)
			switch d.call.Name {
			case "resume":
				if d.err != nil || d.call.Args["warm-only"] != "true" || d.call.Args["tag"] != string(thread.Address.Tag) {
					t.Fatalf("resume call %+v: %v", d.call, d.err)
				}
			case "attach":
				if d.err != nil || d.call.Args["background"] != "true" {
					t.Fatalf("attach call %+v: %v", d.call, d.err)
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("dispatched only %v", seen)
		}
	}
	if seen[0] != "resume" || seen[1] != "attach" {
		t.Fatalf("dispatch order %v", seen)
	}
}
