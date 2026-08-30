package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type conformanceFaultRuntime struct {
	pairlifecycle.OSRuntime
	mu       sync.Mutex
	failSync bool
}

func (r *conformanceFaultRuntime) FailNextSync() {
	r.mu.Lock()
	r.failSync = true
	r.mu.Unlock()
}

func (r *conformanceFaultRuntime) SyncDirectory(path string) error {
	r.mu.Lock()
	fail := r.failSync
	r.failSync = false
	r.mu.Unlock()
	if fail {
		return errors.New("conformance crash after rename")
	}
	return r.OSRuntime.SyncDirectory(path)
}

type realLifecycleOps struct {
	session   string
	runtime   launcher.OSRuntime
	fault     *conformanceFaultRuntime
	stages    []pairlifecycle.CleanupStage
	completed time.Time
}

func (o *realLifecycleOps) step(stage pairlifecycle.CleanupStage, effect func() error) error {
	if effect != nil {
		if err := effect(); err != nil {
			return err
		}
	}
	o.stages = append(o.stages, stage)
	if stage == pairlifecycle.StageCmuxCleanup {
		o.fault.FailNextSync()
	}
	return nil
}
func (o *realLifecycleOps) QuiesceSession(ctx context.Context) error {
	return o.step(pairlifecycle.StageSessionQuiescence, func() error { return o.runtime.DeleteSessionContext(ctx, o.session) })
}
func (o *realLifecycleOps) ReapEditors(context.Context) error {
	return o.step(pairlifecycle.StageEditorReap, nil)
}
func (o *realLifecycleOps) PreserveScrollback(context.Context, pairlifecycle.CleanupIntent) error {
	return o.step(pairlifecycle.StageScrollbackPreserve, nil)
}
func (o *realLifecycleOps) CleanupSidecars(context.Context) error {
	return o.step(pairlifecycle.StageSidecarCleanup, nil)
}
func (o *realLifecycleOps) CleanupPoller(context.Context) error {
	return o.step(pairlifecycle.StagePollerCleanup, nil)
}
func (o *realLifecycleOps) CleanupCmux(context.Context) error {
	return o.step(pairlifecycle.StageCmuxCleanup, nil)
}
func (o *realLifecycleOps) Now() time.Time { return o.completed }

type realParkConformanceDriver struct {
	paths   artifactpath.LifecyclePaths
	store   pairlifecycle.Store
	fault   *conformanceFaultRuntime
	checker ScopedThreadArtifactCollisionChecker
	threads *ThreadStore
	current ThreadRecord
	child   *exec.Cmd
	childID ProcessIdentity
	ops     *realLifecycleOps
}

func (d *realParkConformanceDriver) PrepareRequest(request pairlifecycle.QuitRequest) error {
	d.fault.FailNextSync()
	err := d.store.PublishRequest(d.paths, request)
	if pairlifecycle.PublicationOutcomeOf(err) != pairlifecycle.Indeterminate {
		return fmt.Errorf("prepare request outcome = %s: %w", pairlifecycle.PublicationOutcomeOf(err), err)
	}
	return nil
}
func (d *realParkConformanceDriver) Restart() error {
	d.store = pairlifecycle.Store{Runtime: d.fault}
	d.checker = NewScopedThreadArtifactCollisionChecker(d.checker.GlobalDataDir)
	return nil
}
func (d *realParkConformanceDriver) CommitRequest(request pairlifecycle.QuitRequest) error {
	if err := d.store.Reconcile(d.paths, pairlifecycle.RecordRequest, request.Attempt); err != nil {
		return err
	}
	current, err := d.threads.GetThread(d.current.Address)
	if err != nil {
		return err
	}
	current, err = d.threads.AdvancePark(current.Address, current.Revision, ParkEvent{
		Kind: ParkRequestCommitted, Identity: current.Park.Identity, Attempt: request.Attempt,
	})
	d.current = current
	return err
}
func (d *realParkConformanceDriver) DeliverTrigger(request pairlifecycle.QuitRequest) error {
	intent := launcher.QuitIntent{Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch,
		Request: &launcher.QuitRequestReference{
			DataDir: d.checker.GlobalDataDir, RepoScope: request.Identity.RepoScope,
			Tag: request.Identity.Tag, Nonce: request.Identity.Nonce, Attempt: request.Attempt,
		}}
	return d.checker.TriggerQuit(request.Session, intent)
}
func (d *realParkConformanceDriver) CleanupAndPrepareCompletion(ctx context.Context, request pairlifecycle.QuitRequest) (pairlifecycle.CleanupResult, []pairlifecycle.CleanupStage, error) {
	runtime := launcher.NewScopedOSRuntime(d.checker.GlobalDataDir, d.checker.GlobalDataDir, "")
	intent, present, err := runtime.TakeQuitIntent(request.Session)
	if err != nil || !present || intent.Kind != launcher.QuitIntentCouch {
		return pairlifecycle.CleanupResult{}, nil, fmt.Errorf("production trigger intent = %+v present=%v: %w", intent, present, err)
	}
	ref := *intent.Request
	result, consumeErr := launcher.ConsumeCouchAttempt(ctx, d.store, d.paths, ref, request.Session, d.ops)
	if pairlifecycle.PublicationOutcomeOf(consumeErr) != pairlifecycle.Indeterminate {
		return result, nil, fmt.Errorf("prepare completion outcome = %s: %w", pairlifecycle.PublicationOutcomeOf(consumeErr), consumeErr)
	}
	return result, append([]pairlifecycle.CleanupStage(nil), d.ops.stages...), nil
}
func (d *realParkConformanceDriver) CommitCompletion(request pairlifecycle.QuitRequest, _ pairlifecycle.CleanupResult) error {
	return d.store.Reconcile(d.paths, pairlifecycle.RecordCompletion, request.Attempt)
}
func (d *realParkConformanceDriver) ObserveChildDeath(pairlifecycle.QuitRequest) error {
	if d.child != nil && d.child.Process != nil {
		_ = d.child.Process.Kill()
		_ = d.child.Wait()
	}
	if state := observeExactProcess(OSProcOps{}, d.childID); state != Dead {
		return fmt.Errorf("exact child state = %s", state)
	}
	return nil
}
func (d *realParkConformanceDriver) Finalize(request pairlifecycle.QuitRequest) error {
	controller := PairLifecycleController{
		Threads: d.threads, DataDir: d.checker.GlobalDataDir,
		Lifecycle: PairLifecycleStoreIO{Store: d.store}, Sessions: d.checker,
		Proc: OSProcOps{}, Clock: FixedClock{T: d.ops.completed},
		Nonce: func() (string, error) { return "unused", nil },
	}
	if err := controller.ReconcileActive(context.Background()); err != nil {
		return err
	}
	final, err := d.threads.GetThread(d.current.Address)
	if err != nil {
		return err
	}
	if final.Park != nil || final.VerifiedPark == nil || final.VerifiedPark.Attempt != request.Attempt {
		return fmt.Errorf("thread did not finalize verified park: %+v", final)
	}
	return nil
}

func TestParkLifecycleLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	session := fmt.Sprintf("pair-park-live-%d", os.Getpid())
	zellij, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zellij.Close() })

	child := exec.Command("sh", "-c", "exec sleep 60")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if child.Process != nil {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	})
	identity := procutil.Identity(strconv.Itoa(child.Process.Pid))
	if identity == "" {
		t.Fatal("controlled child identity is unavailable")
	}

	namespace := testCouchNamespace(t)
	threads := NewThreadStore(namespace)
	workdir := t.TempDir()
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = workdir, workdir
	record.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: child.Process.Pid, Identity: identity, State: IncarnationLive, LaunchProfile: &profile,
	}}
	record.LatestLaunchProfile = &profile
	created, err := threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	parkIdentity := ParkIdentity{
		Nonce: "park-live-conformance", Address: created.Address,
		PID: child.Process.Pid, ProcessIdentity: identity,
	}
	current, err := threads.BeginPark(created.Address, created.Revision, parkIdentity)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	address, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: dataDir, RepoScope: created.Address.RepoScope, Tag: string(created.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	lifecyclePaths, err := address.Lifecycle(parkIdentity.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	scoped := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: created.Address.RepoScope}, string(created.Address.Tag))
	if err := os.MkdirAll(scoped.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: session, ScopeKey: created.Address.RepoScope, RepoRoot: workdir,
		RepoName: filepath.Base(workdir), Tag: string(created.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scoped.SessionBindings(), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	request := pairlifecycle.QuitRequest{
		SchemaVersion: pairlifecycle.SchemaVersion,
		Identity: pairlifecycle.Identity{
			Nonce: parkIdentity.Nonce, RepoScope: created.Address.RepoScope, Tag: string(created.Address.Tag),
			PID: child.Process.Pid, ProcessIdentity: identity,
		},
		Attempt: 1, Session: session, Mode: pairlifecycle.CleanupPreserveScrollback,
		CompletionKey: "quit-completion-1",
	}
	fault := &conformanceFaultRuntime{}
	driver := &realParkConformanceDriver{
		paths: lifecyclePaths, store: pairlifecycle.Store{Runtime: fault}, fault: fault,
		checker: NewScopedThreadArtifactCollisionChecker(dataDir), threads: threads, current: current,
		child: child, childID: ProcessIdentity{PID: child.Process.Pid, Identity: identity},
		ops: &realLifecycleOps{session: session, fault: fault, completed: time.Unix(100, 0).UTC()},
	}

	liveTrace, err := pairlifecycletest.RunConformanceScenario(ctx, driver, request)
	if err != nil {
		t.Fatal(err)
	}
	fake := pairlifecycletest.New(time.Unix(100, 0).UTC())
	fake.SetSession(request.Session, true)
	fakeTrace, err := pairlifecycletest.RunConformanceScenario(ctx, pairlifecycletest.NewFakeConformanceDriver(fake), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(liveTrace, fakeTrace) {
		t.Fatalf("fake/live lifecycle traces differ:\nfake %v\nlive %v", fakeTrace, liveTrace)
	}
}

var _ pairlifecycletest.ConformanceDriver = (*realParkConformanceDriver)(nil)
var _ pairlifecycle.QuitLifecycleOps = (*realLifecycleOps)(nil)
