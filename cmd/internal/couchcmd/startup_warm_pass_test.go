package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// startupWarmScreen projects terminal writes into visible cells. A delta repaint
// may reuse an existing prefix, so concatenated write bytes are not screen text.
func startupWarmScreen(t *testing.T, cols, rows int, written func() string) func() string {
	t.Helper()
	em := vt.NewEmulator(cols, rows)
	em.SetReplyWriter(io.Discard)
	t.Cleanup(func() { _ = em.Close() })
	consumed := 0
	return func() string {
		stream := written()
		if _, err := em.WriteString(stream[consumed:]); err != nil {
			t.Fatal(err)
		}
		consumed = len(stream)
		return em.String()
	}
}

func TestStartupWarmScreenReadsFailureAcrossDeltaRepaint(t *testing.T) {
	written := "worker attachment"
	screen := startupWarmScreen(t, 80, 2, func() string { return written })
	if got := screen(); !strings.Contains(got, "worker attachment") {
		t.Fatal(got)
	}
	// Reuse the already painted prefix, just as the production renderer does.
	written += "\x1b[1;18H unavailable"
	if strings.Contains(written, "worker attachment unavailable") {
		t.Fatal("fixture did not split the write-stream phrase")
	}
	if got := screen(); !strings.Contains(got, "worker attachment unavailable") {
		t.Fatalf("visible failure reason missing: %q", got)
	}
}

type startupWarmRunner struct {
	couchcore.Runner
	mu                sync.Mutex
	entered           chan couchcore.ThreadAddress
	release           <-chan struct{}
	attempts          map[couchcore.ThreadAddress]int
	failed            couchcore.ThreadAddress
	allowFailure      bool
	active, maxActive int
}

func (r *startupWarmRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (couchcore.BlockedHandle, error) {
	address := couchcore.ThreadAddress{RepoScope: envValue(env, "COUCH_THREAD_SCOPE"), Tag: couchcore.ThreadTag(envValue(env, "COUCH_THREAD_TAG"))}
	r.mu.Lock()
	r.attempts[address]++
	r.active++
	r.maxActive = max(r.maxActive, r.active)
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.active--; r.mu.Unlock() }()
	if len(argv) != 4 || argv[0] != "pair" || argv[1] != launcher.CouchSessionFlag || argv[2] != "resume" || argv[3] != string(address.Tag) {
		return nil, errors.New("automatic attachment attempted a native/fresh launch")
	}
	select {
	case r.entered <- address:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	r.mu.Lock()
	fail := address == r.failed && !r.allowFailure
	r.mu.Unlock()
	if fail {
		return nil, errors.New("worker attachment unavailable")
	}
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

type startupWarmArtifacts struct {
	*couchcore.FakeThreadArtifactCollisionChecker
	mu          sync.Mutex
	nativeReads map[couchcore.ThreadAddress]int
}

func (a *startupWarmArtifacts) ResolveEstablished(ctx context.Context, scope, tag, agent string) (couchcore.NativeBindingResolution, error) {
	address := couchcore.ThreadAddress{RepoScope: scope, Tag: couchcore.ThreadTag(tag)}
	a.mu.Lock()
	a.nativeReads[address]++
	a.mu.Unlock()
	return a.FakeThreadArtifactCollisionChecker.ResolveEstablished(ctx, scope, tag, agent)
}

func seedWarmPassRow(t *testing.T, r *hubStartupRig, label, kind string, pid int) couchcore.ThreadRecord {
	t.Helper()
	path := filepath.Join(filepath.Dir(r.paths[0]), label)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	number := pid - 5150 + 1
	address, branch := "ops:"+fmt.Sprint(number), "main-slot"+fmt.Sprint(number)
	catalog := r.c.Slots.(*couchcore.SlotCatalogFake)
	catalog.Workspaces[path] = couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "ops", RepoIdentity: filepath.Join(r.paths[0], ".git"), PrimaryRoot: r.paths[0], FleetRoot: filepath.Dir(r.paths[0]), EnvironmentRoot: filepath.Dir(r.paths[0]), WorktreeRoot: path, Kind: "worktree", Slot: &number, Address: &address, RestingBranch: &branch}
	profile := &couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	record := couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0203040506070809"}, StartingPath: path, WorkingPath: path, CreatedAt: time.Unix(1, 0), Revision: 1, LatestLaunchProfile: profile, LastActiveAt: time.Unix(3, 0), SessionBinding: &couchidentity.SessionBinding{Legacy: true, Name: "pair-" + label, ScopeKey: scope.Key, Tag: "couch-0203040506070809", StartNonce: "start-0123456789abcdef"}}
	if kind != "exited" {
		record.Incarnations = []couchcore.ThreadIncarnation{{PID: pid, Identity: "old-" + label, State: couchcore.IncarnationLive, StartedAt: time.Unix(2, 0), LaunchProfile: profile}}
	}
	if kind == "claim" {
		record.Incarnations[0].State = couchcore.IncarnationCreating
		record.Incarnations[0].LaunchProfile = nil
		record.Incarnations[0].Start = &couchcore.ThreadStartClaim{Nonce: "start-2222222222222222", OwnerPID: pid + 100, OwnerIdentity: "independent-owner", LaunchProfile: profile}
		r.rt.proc.Set(pid+100, "independent-owner")
	}
	if kind == "live" {
		r.rt.proc.Set(pid, "old-"+label)
	}
	created, err := r.c.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	r.rt.artifacts.SetPairSession(created.Address, record.SessionBinding.Name, kind != "exited")
	presence := couchcore.SessionPresent
	if kind == "exited" {
		presence = couchcore.SessionAbsent
	}
	r.rt.artifacts.SetSessionPresence(created.Address, couchcore.SessionObservation{State: presence})
	if kind == "exited" {
		r.rt.artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-exited")
	}
	return created
}

func TestStartupWarmPassPaintsAndAcceptsInputWhileMixedAttachmentsBlock(t *testing.T) {
	previous := couchcore.LifecycleParallelism
	couchcore.LifecycleParallelism = 2
	t.Cleanup(func() { couchcore.LifecycleParallelism = previous })
	var dead, live, failed, claim, exited couchcore.ThreadRecord
	release := make(chan struct{})
	var once sync.Once
	var runner *startupWarmRunner
	var artifacts *startupWarmArtifacts
	r := newHubStartupRig(t, false, func(r *hubStartupRig) {
		dead = seedWarmPassRow(t, r, "worker-dead", "dead", 5151)
		live = seedWarmPassRow(t, r, "worker-live", "live", 5152)
		failed = seedWarmPassRow(t, r, "worker-failure", "dead", 5153)
		claim = seedWarmPassRow(t, r, "worker-claim", "claim", 5154)
		exited = seedWarmPassRow(t, r, "worker-exited", "exited", 5155)
		runner = &startupWarmRunner{Runner: r.rt.runner, entered: make(chan couchcore.ThreadAddress, 32), release: release, attempts: map[couchcore.ThreadAddress]int{}, failed: failed.Address}
		r.c.Runner = runner
		artifacts = &startupWarmArtifacts{FakeThreadArtifactCollisionChecker: r.rt.artifacts, nativeReads: map[couchcore.ThreadAddress]int{}}
		r.c.Artifacts = artifacts
	})
	r.host.SetSize(ptychild.Size{Rows: 30, Cols: 240})
	screen := startupWarmScreen(t, 240, 30, r.host.Written)
	if !r.console.ReattachPassArmed() {
		t.Fatal("startup did not arm the automatic warm pass")
	}
	// Preserve actual typed dispatch and queue ownership; only gate the external
	// client start, so the UI must paint and accept input before it resolves.
	actual := r.console.Ops()
	r.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		runner.mu.Lock()
		allowRetry := runner.allowFailure
		runner.mu.Unlock()
		if call.Name == "resume" && call.Args["warm-only"] != "true" && !allowRetry {
			return nil, errors.New("startup attempted a cold resume")
		}
		return actual(call)
	})
	r.run(t)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	select {
	case <-runner.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("automatic attachment never entered existing worker pool")
	}
	runner.mu.Lock()
	blocked := runner.active
	peak := runner.maxActive
	runner.mu.Unlock()
	if blocked < 1 || peak > 2 {
		t.Fatalf("worker parallelism active=%d peak=%d", blocked, peak)
	}
	r.assertFirstUsable(t)
	if _, err := r.input.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "mixed switcher paints before attach completion", func() bool {
		paint := screen()
		return strings.Contains(paint, "Ctrl+Y copy last notice") && strings.Contains(paint, "worker-exited") && strings.Contains(paint, "worker-failure")
	})
	once.Do(func() { close(release) })
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		runner.mu.Lock()
		t.Logf("helper attempts=%v active=%d peak=%d", runner.attempts, runner.active, runner.maxActive)
		runner.mu.Unlock()
		written := r.host.Written()
		if len(written) > 6000 {
			written = written[len(written)-6000:]
		}
		t.Logf("last host writes=%q; visible screen=%q", written, screen())
	})
	waitWarmAcceptance(t, "two failed automatic helper attempts", func() bool {
		runner.mu.Lock()
		n := runner.attempts[failed.Address]
		runner.mu.Unlock()
		return n == 2
	})
	waitWarmAcceptance(t, "rendered Continue failure reason", func() bool {
		paint := screen()
		return strings.Contains(paint, "Continue") && strings.Contains(paint, "worker attachment unavailable")
	})
	waitWarmAcceptance(t, "dead client repaired by automatic Continue", func() bool {
		current, err := r.c.Threads.GetThread(dead.Address)
		return err == nil && current.Revision == dead.Revision+1
	})
	for _, before := range []couchcore.ThreadRecord{live, claim, failed, exited} {
		after, err := r.c.Threads.GetThread(before.Address)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("automatic warm pass changed retained claim/binding/history: %s before=%+v after=%+v err=%v", before.WorkingPath, before, after, err)
		}
	}
	repaired, err := r.c.Threads.GetThread(dead.Address)
	if err != nil {
		t.Fatal(err)
	}
	expected := dead
	expected.Incarnations = append([]couchcore.ThreadIncarnation(nil), dead.Incarnations...)
	expected.Incarnations[0].PID = repaired.Incarnations[0].PID
	expected.Incarnations[0].Identity = repaired.Incarnations[0].Identity
	expected.Revision++
	if !reflect.DeepEqual(expected, repaired) || repaired.Incarnations[0].PID == dead.Incarnations[0].PID {
		t.Fatal("automatic Continue changed more than the dead client identity")
	}
	runner.mu.Lock()
	coldStarts := runner.attempts[exited.Address]
	peak = runner.maxActive
	runner.mu.Unlock()
	if coldStarts != 0 || peak > 2 || len(r.rt.artifacts.Quiesces()) != 0 {
		t.Fatalf("cold/teardown/parallel effects: exited=%d peak=%d quiesces=%v", coldStarts, peak, r.rt.artifacts.Quiesces())
	}
	artifacts.mu.Lock()
	for address, reads := range artifacts.nativeReads {
		if address != exited.Address && reads > 0 {
			artifacts.mu.Unlock()
			t.Fatalf("automatic live attach resolved native session for %+v", address)
		}
	}
	artifacts.mu.Unlock()
	// Failed live rows keep Continue, and the existing Enter action retries via
	// the same public attach path after its bounded automatic attempt exhausted.
	runner.mu.Lock()
	runner.allowFailure = true
	runner.mu.Unlock()
	if _, err := r.input.Write([]byte("worker-failure")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "failed Continue selected for explicit retry", func() bool { return strings.Contains(screen(), "filter: worker-failure") })
	if _, err := r.input.Write([]byte{'\r'}); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "failed Continue user retry repairs its dead client", func() bool {
		current, err := r.c.Threads.GetThread(failed.Address)
		return err == nil && current.Revision == failed.Revision+1
	})
	runner.mu.Lock()
	attempts := runner.attempts[failed.Address]
	runner.mu.Unlock()
	if attempts != 3 {
		t.Fatalf("failed Continue retry attempts=%d, want two automatic and one explicit", attempts)
	}
}
