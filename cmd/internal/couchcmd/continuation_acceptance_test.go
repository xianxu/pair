package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const continuationPublicationFixtureEnv = "PAIR249_PUBLICATION_FIXTURE"

type continuationPublicationFixture struct {
	Namespace string
	DataDir   string
	Repo      string
	Address   couchcore.ThreadAddress
	Session   string
	PID       int
	Identity  string
	Mode      string
}

// This runtime keeps actual CLI dispatch, persisted thread ownership and the
// source ledger/index reader. Only external helper liveness is simulated.
type continuationPublicationRuntime struct {
	testRT
	dataDir string
	owner   *couchcore.Couch
}

func (r continuationPublicationRuntime) NewCouchWith(runner couchcore.Runner, namespace couchcore.CouchNamespace) (*couchcore.Couch, error) {
	if r.owner != nil {
		return r.owner, nil
	}
	c, err := r.testRT.NewCouchWith(runner, namespace)
	if err == nil {
		c.ContinuationSource = (couchcore.OSContinuationSourceReader{DataDir: r.dataDir}).Read
	}
	return c, err
}

// The real writer reinvokes itself as `pair continue`; that production launcher
// invokes this helper as its external Couch process. There is no hand-written
// path/digest reconstruction between those two production boundaries.
func TestContinuationPublicationProcessHelper(t *testing.T) {
	path := os.Getenv(continuationPublicationFixtureEnv)
	if path == "" {
		t.Skip("subprocess-only portable Couch publication helper")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture continuationPublicationFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	args := []string(nil)
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if len(args) != 3 || args[0] != "--internal" || args[1] != "request-continuation" {
		t.Fatalf("unexpected production Couch argv: %q", args)
	}
	namespace, err := couchcore.ResolveCouchNamespace(fixture.Namespace, "/unused")
	if err != nil {
		t.Fatal(err)
	}
	rt := newRT(t, fixture.Repo)
	rt.namespace, rt.dir = namespace, namespace.Dir()
	rt.proc.Set(fixture.PID, fixture.Identity)
	rt.artifacts.SetPairSession(fixture.Address, fixture.Session, true)
	for _, field := range []string{"COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PAIR_SCOPE_KEY", "PAIR_TAG", "PAIR_AGENT", "PAIR_SESSION_NAME", "ZELLIJ_SESSION_NAME", "PAIR_LAUNCH_ORDINAL", checkpoint.DigestEnv} {
		rt.env[field] = os.Getenv(field)
	}
	switch fixture.Mode {
	case "missing-digest":
		rt.env[checkpoint.DigestEnv] = ""
	case "tampered":
		doc, err := os.ReadFile(args[2])
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(args[2], bytes.ReplaceAll(doc, []byte("PUBLICATION-EXACT-TOKEN"), []byte("ALTERED-AFTER-LAUNCHER-READ")), 0600); err != nil {
			t.Fatal(err)
		}
	case "obsolete-source":
		rt.env["PAIR_LAUNCH_ORDINAL"] = "1"
	}
	code := RunWithRuntime(args, strings.NewReader(""), os.Stdout, os.Stderr, continuationPublicationRuntime{testRT: rt, dataDir: fixture.DataDir})
	os.Exit(code)
}

func TestContinuationWriterPublishesExactCheckpointAcrossWorktrees(t *testing.T) {
	assetRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	pair := filepath.Join(bin, "pair")
	build := exec.Command("go", "build", "-o", pair, "../../pair-go")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real writer/launcher: %v\n%s", err, out)
	}
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	shim := "#!/bin/sh\nexec \"$PAIR249_TEST_BINARY\" -test.run '^TestContinuationPublicationProcessHelper$' -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "couch"), []byte(shim), 0700); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"initial", "warm-reattached", "missing-digest", "tampered", "obsolete-source"} {
		t.Run(mode, func(t *testing.T) {
			runContinuationPublicationAcceptance(t, pair, helper, assetRoot, mode)
		})
	}
}

func runContinuationPublicationAcceptance(t *testing.T, pair, helper, assetRoot, mode string) {
	t.Helper()
	base := t.TempDir()
	home, repo, worktree := filepath.Join(base, "home"), filepath.Join(base, "outer-checkout"), filepath.Join(base, "writer-worktree")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Env = append(os.Environ(), "HOME="+home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main", repo)
	git("-C", repo, "config", "user.name", "Continuation Test")
	git("-C", repo, "config", "user.email", "continuation@example.invalid")
	git("-C", repo, "commit", "--allow-empty", "-m", "source")
	git("-C", repo, "worktree", "add", "-b", "writer-checkpoint", worktree)

	rt := newRT(t, repo)
	rt.runner = couchcore.NewFakeRunner()
	rt.env["PAIR_AGENT"] = "claude"
	var source couchcore.ThreadRecord
	var actor couchcore.ActorRecord
	var handle couchcore.Handle
	var c *couchcore.Couch
	var err error
	session := "pair-publication-source"
	if mode == "warm-reattached" {
		source = seedDetachedThread(t, rt, repo)
		rt.artifacts.SetDetachedSession(source.Address, session)
		rt.runner.AfterAcknowledge = func(string) error {
			rt.artifacts.SetDetachedSession(source.Address, "")
			rt.artifacts.SetPairSession(source.Address, session, true)
			return nil
		}
		c, err = rt.NewCouch()
		if err == nil {
			actor, handle, err = c.ResumeContextWith(context.Background(), source.Address, couchcore.ResumeOptions{WarmOnly: true})
		}
	} else {
		c, err = rt.NewCouch()
		if err == nil {
			actor, handle, err = c.Spawn(couchcore.StartArgs{Worktree: couchcore.Worktree(repo)})
		}
	}
	if err != nil {
		t.Fatalf("source setup: %v", err)
	}
	rt.proc.Set(handle.PID(), handle.Identity())
	rt.artifacts.SetPairSession(actor.Thread, session, true)
	t.Cleanup(func() { rt.runner.SetExited(handle.ID(), 0) })
	source, err = c.Threads.GetThread(actor.Thread)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(base, "pair-data")
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := launcher.ClaimNewThreadAddress(data, scope, string(source.Address.Tag))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })
	if err := launcher.EnsureThreadAddressForPair(data, scope, string(source.Address.Tag), true); err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Log(), []byte("PROMPT-HISTORY-EXACT-249\n"), 0600); err != nil {
		t.Fatal(err)
	}
	beforeClaim, err := os.ReadFile(paths.ThreadClaim())
	if err != nil {
		t.Fatal(err)
	}
	ledger := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	for _, agent := range []string{"codex", "claude"} {
		if _, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: scope.Key, Tag: string(source.Address.Tag), Agent: agent}); err != nil {
			t.Fatal(err)
		}
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: session, ScopeKey: scope.Key, Tag: string(source.Address.Tag), RepoRoot: scope.Root, RepoName: scope.DisplayName})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fixture := continuationPublicationFixture{Namespace: rt.dir, DataDir: data, Repo: repo, Address: source.Address, Session: session, PID: handle.PID(), Identity: handle.Identity(), Mode: mode}
	raw, _ := json.Marshal(fixture)
	fixturePath := filepath.Join(base, "fixture.json")
	if err := os.WriteFile(fixturePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, pair, "continuation", "--repo-root", worktree, "--slug", "publication", "--agent", "claude", "--issues", "249", "--body-file", "-")
	command.Dir = repo // deliberately differs from --repo-root and the checkpoint path
	command.Stdin = strings.NewReader("## NEXT ACTION\nVerify PUBLICATION-EXACT-TOKEN and preserve the same Pair thread.\n")
	command.Env = append(os.Environ(), "HOME="+home, "XDG_DATA_HOME="+filepath.Join(home, "data"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "PAIR_HOME="+assetRoot, "PAIR_DATA_DIR="+paths.ScopeDir(), "PAIR_SCOPE_KEY="+scope.Key, "PAIR_TAG="+string(source.Address.Tag), "PAIR_AGENT=claude", "PAIR_SESSION_NAME="+session, "ZELLIJ_SESSION_NAME="+session, "PAIR_LAUNCH_ORDINAL=2", "COUCH_THREAD_SCOPE="+scope.Key, "COUCH_THREAD_TAG="+string(source.Address.Tag), "COUCH_STORE_DIR="+rt.dir, launcher.CouchLaunchProfileEnv+"=", checkpoint.DigestEnv+"=", continuationPublicationFixtureEnv+"="+fixturePath, "PAIR249_TEST_BINARY="+helper, "PAIR_KILL_CMD=__continuation_test_must_never_kill__")
	output, runErr := command.CombinedOutput()
	shouldPublish := mode == "initial" || mode == "warm-reattached"
	if (runErr == nil) != shouldPublish {
		t.Fatalf("writer publication result: %v\n%s", runErr, output)
	}
	if !shouldPublish {
		want := map[string]string{"missing-digest": "requires the committed checkpoint digest", "tampered": "checkpoint changed after writer validation", "obsolete-source": "continuation source generation is obsolete"}[mode]
		if !strings.Contains(string(output), want) {
			t.Fatalf("wrong rejection: want %q, got %s", want, output)
		}
	}
	docs, err := filepath.Glob(filepath.Join(worktree, "workshop", "continuation", "*.md"))
	if err != nil || len(docs) != 1 {
		t.Fatalf("writer did not retain one exact document: %v %v", docs, err)
	}
	rel, _ := filepath.Rel(worktree, docs[0])
	committed := git("-C", worktree, "show", "HEAD:"+filepath.ToSlash(rel)) + "\n"
	if !strings.Contains(committed, "PUBLICATION-EXACT-TOKEN") {
		t.Fatal("writer commit lost the original checkpoint")
	}
	after, err := c.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.Incarnations, source.Incarnations) || !handle.Alive() || rt.proc.Exists(handle.PID()) != couchcore.Live {
		t.Fatalf("publication changed source ownership: before %+v after %+v", source.Incarnations, after.Incarnations)
	}
	afterClaim, err := os.ReadFile(paths.ThreadClaim())
	if err != nil || !bytes.Equal(beforeClaim, afterClaim) || launcher.RegisterExistingCouchThread(data, scope, string(source.Address.Tag)) != nil {
		t.Fatalf("publication weakened established claim: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "pair", "restart-"+session)); !os.IsNotExist(err) {
		t.Fatalf("hosted writer touched standalone marker: %v", err)
	}
	if !shouldPublish {
		if after.Continuation != nil {
			t.Fatalf("failed publication retained authority: %+v", after.Continuation)
		}
		return
	}
	if after.Continuation == nil || after.Continuation.Phase != checkpoint.Pending || after.Continuation.Source.LaunchOrdinal != 2 || after.Continuation.Source.Helper.PID != handle.PID() || after.Continuation.Checkpoint.Body != committed || after.Continuation.Checkpoint.SourcePath != docs[0] {
		t.Fatalf("wrong durable request: %+v", after.Continuation)
	}
	if err := after.Continuation.Checkpoint.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, rel)); !os.IsNotExist(err) {
		t.Fatalf("fixture did not separate worktrees: %v", err)
	}
	if err := os.Remove(docs[0]); err != nil {
		t.Fatal(err)
	}
	reopened, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	retained, err := reopened.Threads.GetThread(source.Address)
	if err != nil || retained.Continuation == nil || retained.Continuation.Checkpoint.Body != committed {
		t.Fatalf("request depends on removed worktree file: %v", err)
	}
	completeContinuationAcceptance(t, rt, c, retained, handle, data, scope, committed, mode)
	if got, err := os.ReadFile(paths.Log()); err != nil || string(got) != "PROMPT-HISTORY-EXACT-249\n" {
		t.Fatalf("handoff replaced prompt history: %q %v", got, err)
	}
	if got, err := os.ReadFile(paths.ThreadClaim()); err != nil || !bytes.Equal(got, beforeClaim) {
		t.Fatalf("replacement changed established address claim: %v", err)
	}
}

// Publication and completion use actual durable protocol files; the trigger
// substitutes only for the external Pair process executing its cleanup.
type continuationAcceptanceLifecycle struct {
	couchcore.PairLifecycleStoreIO
	paths   artifactpath.LifecyclePaths
	request pairlifecycle.QuitRequest
}

func (l *continuationAcceptanceLifecycle) PublishRequest(paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	l.paths, l.request = paths, request
	return l.PairLifecycleStoreIO.PublishRequest(paths, request)
}

func completeContinuationAcceptance(t *testing.T, rt testRT, c *couchcore.Couch, source couchcore.ThreadRecord, old couchcore.Handle, data string, scope launcher.RepoScope, committed string, mode string) {
	t.Helper()
	lifecycle := &continuationAcceptanceLifecycle{PairLifecycleStoreIO: couchcore.PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}}
	c.PairLifecycle = &couchcore.PairLifecycleController{Threads: c.Threads, DataDir: data, Lifecycle: lifecycle, Sessions: rt.artifacts, Proc: rt.proc, Clock: c.Clock, Nonce: func() (string, error) { return "acceptance-park-249", nil }, CompletionTimeout: time.Second, PollInterval: time.Millisecond}
	c.ContinuationSource = (couchcore.OSContinuationSourceReader{DataDir: data}).Read
	rt.artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
		if intent.Kind != launcher.QuitIntentCouch || session != source.Continuation.Source.Session {
			return fmt.Errorf("wrong typed source quit: %+v %s", intent, session)
		}
		_, err := lifecycle.Store.ConsumeAttempt(context.Background(), lifecycle.paths, lifecycle.request.Attempt, func(context.Context, *pairlifecycle.LockedAttempt, pairlifecycle.QuitRequest) pairlifecycle.CleanupResult {
			rt.proc.Kill(old.PID())
			rt.runner.SetExited(old.ID(), 0)
			rt.artifacts.SetPairSession(source.Address, session, false)
			return pairlifecycle.CleanupResult{Outcome: pairlifecycle.CompletionSuccess, CompletedAt: c.Clock.Now()}
		})
		return err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	readyPath, err := paths.AgentReadyChecked("claude")
	if err != nil {
		t.Fatal(err)
	}
	reader := couchcore.OSOrientationStatusReader{DataDir: data, Session: rt.artifacts.PairSession, Proc: rt.proc}
	c.FreshRegistration, c.OrientationStatus = reader.Registered, reader.Read
	var ready readiness.ReadyRecord
	var targetSeed string
	rt.runner.AfterAcknowledge = func(id string) error {
		record, err := c.Threads.GetThread(source.Address)
		if err != nil {
			return err
		}
		if record.VerifiedPark == nil || record.Continuation.SourcePark == "" || old.Alive() || rt.proc.Exists(old.PID()) == couchcore.Live {
			return fmt.Errorf("target started before verified source teardown")
		}
		if err := launcher.RegisterExistingCouchThread(data, scope, string(source.Address.Tag)); err != nil {
			return err
		}
		child := rt.runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		for _, entry := range child.Env {
			if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
				if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
					return err
				}
			}
		}
		if !profile.FreshRequired || profile.ResumeRequired || profile.RequiredSessionID != "" || profile.Tag != string(source.Address.Tag) || profile.Agent != "claude" {
			return fmt.Errorf("replacement lost fresh launch identity: %+v", profile)
		}
		if profile.Orientation == nil || profile.Orientation.Attempt != record.Continuation.Attempt {
			return fmt.Errorf("target lost orientation attempt")
		}
		quoted, err := strconv.QuotedPrefix(strings.TrimPrefix(profile.Orientation.Body, "Read the saved continuation at "))
		if err != nil {
			return err
		}
		targetSeed, err = strconv.Unquote(quoted)
		if err != nil {
			return err
		}
		seed, err := checkpoint.ReadFile(targetSeed)
		if err != nil {
			return err
		}
		if seed.Body != committed || seed.Digest != source.Continuation.Checkpoint.Digest || !strings.Contains(profile.Orientation.Body, seed.Digest) {
			return fmt.Errorf("fresh target received different checkpoint")
		}
		inc := record.Incarnations[0]
		rt.proc.Set(inc.PID, inc.Identity)
		rt.artifacts.SetPairSession(source.Address, source.Continuation.Source.Session, true)
		ready = readiness.ReadyRecord{Tag: string(source.Address.Tag), Agent: "claude", Nonce: record.Continuation.Attempt, Session: source.Continuation.Source.Session, PID: inc.PID, Orientation: &orientation.DeliveryState{Phase: orientation.DeliveryWaiting}}
		raw, err := readiness.Encode(ready)
		if err != nil {
			return err
		}
		return os.WriteFile(readyPath, []byte(raw), 0600)
	}
	var start couchcore.StartResult
	if mode == "warm-reattached" {
		// Explicit retry bootstraps a real owner lease and reaches the same attachment
		// finisher used by initial CLI launch, with the accepted on-disk request.
		master, slave, err := pty.Open()
		if err != nil {
			t.Fatal(err)
		}
		if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
			t.Fatal(err)
		}
		defer master.Close()
		defer slave.Close()
		op, _ := Resolve("retry-continuation")
		var stdout, stderr bytes.Buffer
		rt.currentRepoScope = scope.Key
		runtime := continuationPublicationRuntime{testRT: rt, dataDir: data, owner: c}
		finish := func(console *couchtty.Console, owner *couchcore.Couch, got couchcore.StartResult, _ io.Writer) int {
			defer console.Stop()
			if rt.supervisor.acquired != 1 || rt.supervisor.released != 0 || owner != c {
				t.Error("retry finisher lacks live owner lease")
			}
			wireResolver(console, owner)
			if err := dispatchInitialAttach(console, got); err != nil {
				t.Fatalf("retry attachment while lease held: %v", err)
			}
			start = got
			return 0
		}
		code := runTypedOperationWithConsole(op, map[string]string{"ref": string(source.Address.Tag)}, nil, true, "", slave, slave, slave, &stdout, &stderr, runtime, finish)
		if code != 0 || rt.supervisor.acquired != 1 || rt.supervisor.released != 1 {
			t.Fatalf("retry owner bootstrap: code=%d lease=%+v stderr=%s", code, rt.supervisor, stderr.String())
		}
	} else {
		result, err := c.Continue(context.Background(), source.Address, source.Continuation.ID)
		if err != nil {
			t.Fatal(err)
		}
		var ok bool
		start, ok = result.Started()
		if !ok {
			t.Fatalf("no replacement: %+v", result)
		}
	}
	if start.Handle == nil || start.Record.Thread != source.Address || start.Handle.ID() == old.ID() {
		t.Fatalf("wrong replacement: %+v", start)
	}
	t.Cleanup(func() { rt.runner.SetExited(start.Handle.ID(), 0) })
	state, err := c.ReconcileContinuation(context.Background(), source.Address, source.Continuation.ID, ready.Nonce)
	if err != nil || state.Phase != checkpoint.Running {
		t.Fatalf("registration must wait for submission: %+v %v", state, err)
	}
	ready.Orientation = &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}
	raw, err := readiness.Encode(ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	state, err = c.ReconcileContinuation(context.Background(), source.Address, source.Continuation.ID, ready.Nonce)
	if err != nil || state.Phase != checkpoint.Complete {
		t.Fatalf("target receipt incomplete: %+v %v", state, err)
	}
	if got, err := os.ReadFile(targetSeed); err != nil || string(got) != committed {
		t.Fatalf("materialized exact snapshot: %v", err)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
	inputReader, input := io.Pipe()
	console := couchtty.New(host, inputReader)
	terminal := start.Handle.(couchcore.TerminalHandle).Terminal()
	terminal.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error {
		return console.Deliver(ctx, start.Handle.ID(), batch)
	})
	wireResolver(console, c)
	if err := dispatchInitialAttach(console, start); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	defer func() {
		console.Stop()
		_ = input.Close()
		_ = inputReader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Console did not stop")
		}
	}()
	terminal.Feed([]byte("CONTINUATION-EXACT-OUTPUT"))
	waitWarmAcceptance(t, "replacement output", func() bool { return strings.Contains(host.Written(), "CONTINUATION-EXACT-OUTPUT") })
	if _, err := input.Write([]byte("continuation-input")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "replacement input", func() bool { return bytes.Contains(bytes.Join(terminal.Writes(), nil), []byte("continuation-input")) })
	final, err := c.Threads.GetThread(source.Address)
	if err != nil || final.Continuation.Phase != checkpoint.Complete || len(final.ParkHistory) != 1 {
		t.Fatalf("handoff receipt/history: %+v %v", final, err)
	}
}
