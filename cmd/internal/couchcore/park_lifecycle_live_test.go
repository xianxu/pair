package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
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
	paths     artifactpath.LifecyclePaths
	store     pairlifecycle.Store
	fault     *conformanceFaultRuntime
	checker   ScopedThreadArtifactCollisionChecker
	threads   *ThreadStore
	current   ThreadRecord
	child     *exec.Cmd
	childWait <-chan error
	childID   ProcessIdentity
	stagePath string
	trigger   func(string, launcher.QuitIntent) error
	intent    *launcher.OSRuntime
	completed time.Time
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
	if d.child == nil {
		return errors.New("production handoff helper is not running")
	}
	intent := launcher.QuitIntent{Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch,
		Request: &launcher.QuitRequestReference{
			DataDir: d.checker.GlobalDataDir, RepoScope: request.Identity.RepoScope,
			Tag: request.Identity.Tag, Nonce: request.Identity.Nonce, Attempt: request.Attempt,
		}}
	if d.trigger != nil {
		return d.trigger(request.Session, intent)
	}
	return d.checker.TriggerQuit(request.Session, intent)
}
func (d *realParkConformanceDriver) CleanupAndPrepareCompletion(ctx context.Context, request pairlifecycle.QuitRequest) (pairlifecycle.CleanupResult, []pairlifecycle.CleanupStage, error) {
	observer := PairLifecycleStoreIO{Store: d.store}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		completion, found, err := observer.ObserveCompletion(d.paths, request)
		if err != nil {
			return pairlifecycle.CleanupResult{}, nil, err
		}
		if found {
			raw, err := os.ReadFile(d.stagePath)
			if errors.Is(err, os.ErrNotExist) {
				select {
				case <-ctx.Done():
					return pairlifecycle.CleanupResult{}, nil, fmt.Errorf("observe helper stages: %w", ctx.Err())
				case <-ticker.C:
					continue
				}
			}
			if err != nil {
				return pairlifecycle.CleanupResult{}, nil, fmt.Errorf("read helper cleanup stages: %w", err)
			}
			var stages []pairlifecycle.CleanupStage
			for _, value := range strings.Fields(string(raw)) {
				stages = append(stages, pairlifecycle.CleanupStage(value))
			}
			return pairlifecycle.CleanupResult{Outcome: completion.Outcome, CompletedAt: completion.CompletedAt}, stages, nil
		}
		select {
		case <-ctx.Done():
			return pairlifecycle.CleanupResult{}, nil, fmt.Errorf("observe helper completion: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
func (d *realParkConformanceDriver) CommitCompletion(request pairlifecycle.QuitRequest, _ pairlifecycle.CleanupResult) error {
	return d.store.Reconcile(d.paths, pairlifecycle.RecordCompletion, request.Attempt)
}
func (d *realParkConformanceDriver) ObserveChildDeath(pairlifecycle.QuitRequest) error {
	if d.childWait == nil {
		return errors.New("production handoff helper was not started")
	}
	select {
	case err := <-d.childWait:
		if err != nil {
			return fmt.Errorf("production handoff helper: %w", err)
		}
	case <-time.After(5 * time.Second):
		return errors.New("production handoff helper remained live")
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
		Proc: OSProcOps{}, Clock: FixedClock{T: d.completed},
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
	ctx, cancel := liveParkContext(t, 20*time.Second)
	defer cancel()
	driver, request := newRealParkConformance(t, ctx, false)

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

func TestParkLifecycleLiveIntentOnlyMutation(t *testing.T) {
	ctx, cancel := liveParkContext(t, 750*time.Millisecond)
	defer cancel()
	driver, request := newRealParkConformance(t, ctx, true)

	trace, err := pairlifecycletest.RunConformanceScenario(ctx, driver, request)
	wantTrace := pairlifecycletest.EffectTrace{
		"request:prepared", "restart:prepared-request", "request:committed", "restart:committed-request",
		"trigger:delivered", "trigger:delivered",
	}
	if !reflect.DeepEqual(trace, wantTrace) {
		t.Fatalf("intent-only mutation failed before its required precondition:\ngot  %v\nwant %v\nerr  %v", trace, wantTrace, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "observe helper completion") {
		t.Fatalf("intent-only mutation failed outside completion observation: %v", err)
	}
	intent, present, intentErr := driver.intent.TakeQuitIntent(request.Session)
	if intentErr != nil || !present || intent.Kind != launcher.QuitIntentCouch || intent.Request == nil {
		t.Fatalf("intent-only mutation did not durably publish a Couch intent: %+v, present=%t, err=%v", intent, present, intentErr)
	}
	ref := intent.Request
	if ref.DataDir != driver.checker.GlobalDataDir || ref.RepoScope != request.Identity.RepoScope ||
		ref.Tag != request.Identity.Tag || ref.Nonce != request.Identity.Nonce || ref.Attempt != request.Attempt {
		t.Fatalf("durable intent reference = %+v, request=%+v", ref, request)
	}
	if state := observeExactProcess(OSProcOps{}, driver.childID); state != Live {
		t.Fatalf("intent-only trigger released the production handoff: %s", state)
	}
}

func liveParkContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1")
	}
	return context.WithTimeout(t.Context(), timeout)
}

func newRealParkConformance(t *testing.T, ctx context.Context, intentOnly bool) (*realParkConformanceDriver, pairlifecycle.QuitRequest) {
	t.Helper()
	variant := "ok"
	if intentOnly {
		variant = "mut"
	}
	session := fmt.Sprintf("pair-park-%d-%s", os.Getpid(), variant)
	zellij, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zellij.Close() })

	namespace := testCouchNamespace(t)
	threads := NewThreadStore(namespace)
	workdir := t.TempDir()
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = workdir, workdir
	record.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.LatestLaunchProfile = &profile
	parkNonce := "park-live-conformance"

	dataDir := t.TempDir()
	address, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: dataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	lifecyclePaths, err := address.Lifecycle(parkNonce)
	if err != nil {
		t.Fatal(err)
	}
	scoped := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: record.Address.RepoScope}, string(record.Address.Tag))
	if err := os.MkdirAll(scoped.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: session, ScopeKey: record.Address.RepoScope, RepoRoot: workdir,
		RepoName: filepath.Base(workdir), Tag: string(record.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scoped.SessionBindings(), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stagePath := filepath.Join(t.TempDir(), "cleanup-stages")
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pairHome := filepath.Clean(filepath.Join(packageDir, "..", "..", ".."))
	child, childPTY, childWait := startParkLifecycleHelper(t, ctx, parkHelperConfig{
		GlobalDataDir: dataDir, Scope: record.Address.RepoScope, Tag: string(record.Address.Tag),
		Nonce: parkNonce, Attempt: 1, Session: session, PairHome: pairHome,
		StagePath: stagePath, CompletedAt: time.Unix(100, 0).UTC(),
	})
	identity := procutil.Identity(strconv.Itoa(child.Process.Pid))
	if identity == "" {
		t.Fatal("production handoff helper identity is unavailable")
	}
	record.Incarnations = []ThreadIncarnation{{
		PID: child.Process.Pid, Identity: identity, State: IncarnationLive, LaunchProfile: &profile,
	}}
	created, err := threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	parkIdentity := ParkIdentity{Nonce: parkNonce, Address: created.Address, PID: child.Process.Pid, ProcessIdentity: identity}
	current, err := threads.BeginPark(created.Address, created.Revision, parkIdentity)
	if err != nil {
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
	intentRuntime := launcher.NewScopedOSRuntime(dataDir, scoped.ScopeDir(), pairHome)
	driver := &realParkConformanceDriver{
		paths: lifecyclePaths, store: pairlifecycle.Store{Runtime: fault}, fault: fault,
		checker: NewScopedThreadArtifactCollisionChecker(dataDir), threads: threads, current: current,
		child: child, childWait: childWait,
		childID:   ProcessIdentity{PID: child.Process.Pid, Identity: identity},
		stagePath: stagePath, intent: intentRuntime, completed: time.Unix(100, 0).UTC(),
	}
	if intentOnly {
		driver.trigger = intentRuntime.WriteQuitIntent
	}
	t.Cleanup(func() {
		runtime := launcher.NewScopedOSRuntime(dataDir, scoped.ScopeDir(), pairHome)
		_, _, _ = runtime.TakeQuitIntent(session)
		if child.Process != nil {
			_ = child.Process.Kill()
		}
		if childPTY != nil {
			_ = childPTY.Close()
		}
	})
	return driver, request
}

type parkHelperConfig struct {
	GlobalDataDir, Scope, Tag, Nonce, Session, PairHome, StagePath string
	Attempt                                                        uint64
	CompletedAt                                                    time.Time
}

func startParkLifecycleHelper(t *testing.T, ctx context.Context, cfg parkHelperConfig) (*exec.Cmd, *os.File, <-chan error) {
	t.Helper()
	baseline, err := zellijClientCount(ctx, cfg.Session)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		"PAIR_TEST_COUCH_PARK_HELPER=1",
		"PAIR_TEST_COUCH_LAUNCH_NATIVE_SIDECAR=1",
		"PAIR_TEST_PARK_GLOBAL_DATA_DIR="+cfg.GlobalDataDir,
		"PAIR_TEST_PARK_SCOPE="+cfg.Scope,
		"PAIR_TEST_PARK_TAG="+cfg.Tag,
		"PAIR_TEST_PARK_NONCE="+cfg.Nonce,
		"PAIR_TEST_PARK_ATTEMPT="+strconv.FormatUint(cfg.Attempt, 10),
		"PAIR_TEST_PARK_SESSION="+cfg.Session,
		"PAIR_TEST_PARK_HOME="+cfg.PairHome,
		"PAIR_TEST_PARK_STAGE_PATH="+cfg.StagePath,
		"PAIR_TEST_PARK_COMPLETED_AT="+strconv.FormatInt(cfg.CompletedAt.Unix(), 10),
	)
	terminal, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = io.Copy(io.Discard, terminal) }()
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		count, countErr := zellijClientCount(ctx, cfg.Session)
		if countErr == nil && count > baseline {
			return cmd, terminal, wait
		}
		select {
		case err := <-wait:
			t.Fatalf("production handoff helper exited before attaching: %v", err)
		case <-ctx.Done():
			t.Fatalf("production handoff helper did not attach: %v (last query: %v)", ctx.Err(), countErr)
		case <-ticker.C:
		}
	}
}

func zellijClientCount(ctx context.Context, session string) (int, error) {
	out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "list-clients").Output()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func runParkLifecycleHelper() int {
	attempt, err := strconv.ParseUint(os.Getenv("PAIR_TEST_PARK_ATTEMPT"), 10, 64)
	if err != nil {
		return 2
	}
	completedUnix, err := strconv.ParseInt(os.Getenv("PAIR_TEST_PARK_COMPLETED_AT"), 10, 64)
	if err != nil {
		return 2
	}
	globalDataDir := os.Getenv("PAIR_TEST_PARK_GLOBAL_DATA_DIR")
	scope := os.Getenv("PAIR_TEST_PARK_SCOPE")
	tag := os.Getenv("PAIR_TEST_PARK_TAG")
	nonce := os.Getenv("PAIR_TEST_PARK_NONCE")
	session := os.Getenv("PAIR_TEST_PARK_SESSION")
	pairHome := os.Getenv("PAIR_TEST_PARK_HOME")
	stagePath := os.Getenv("PAIR_TEST_PARK_STAGE_PATH")
	scoped := launcher.NewScopedPaths(globalDataDir, launcher.RepoScope{Key: scope}, tag)
	runtime := launcher.NewScopedOSRuntime(globalDataDir, scoped.ScopeDir(), pairHome)
	env := launcher.Env{DataDir: scoped.ScopeDir()}
	if _, err := launcher.AttachExistingSession(launcher.LaunchOptions{Env: env, PairHome: pairHome}, env, runtime, tag, session, "claude"); err != nil {
		return 3
	}
	intent, present, err := runtime.TakeQuitIntent(session)
	if err != nil || !present || intent.Kind != launcher.QuitIntentCouch || intent.Request == nil {
		return 4
	}
	ref := *intent.Request
	if ref.RepoScope != scope || ref.Tag != tag || ref.Nonce != nonce || ref.Attempt != attempt {
		return 5
	}
	address, err := artifactpath.Resolve(artifactpath.Address{DataDir: globalDataDir, RepoScope: scope, Tag: tag})
	if err != nil {
		return 6
	}
	paths, err := address.Lifecycle(nonce)
	if err != nil {
		return 6
	}
	fault := &conformanceFaultRuntime{}
	ops := &realLifecycleOps{
		session: session, runtime: *runtime, fault: fault, completed: time.Unix(completedUnix, 0).UTC(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, consumeErr := launcher.ConsumeCouchAttempt(ctx, pairlifecycle.Store{Runtime: fault}, paths, ref, session, ops)
	stageNames := make([]string, 0, len(ops.stages))
	for _, stage := range ops.stages {
		stageNames = append(stageNames, string(stage))
	}
	if err := os.WriteFile(stagePath, []byte(strings.Join(stageNames, "\n")+"\n"), 0o600); err != nil {
		return 7
	}
	if pairlifecycle.PublicationOutcomeOf(consumeErr) != pairlifecycle.Indeterminate {
		return 8
	}
	return 0
}

var _ pairlifecycletest.ConformanceDriver = (*realParkConformanceDriver)(nil)
var _ pairlifecycle.QuitLifecycleOps = (*realLifecycleOps)(nil)
