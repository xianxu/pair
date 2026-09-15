package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// Real source process/session observations, real exact ledger/readiness files,
// and a deterministic Zellij pane; target Couch helpers use the existing fake.
// Command-package acceptance separately proves Console adoption and keyboard IO.
func TestRecoveryRealHelperAndSessionConformanceLive(t *testing.T) {
	liveOnly(t)
	// Darwin's default TMPDIR can exhaust Unix socket path capacity before
	// Zellij appends the session name. Keep its socket root short and isolate
	// configuration/data so the operator's shell and plugins do not run here.
	t.Setenv("TMPDIR", "/tmp")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")
	// This is a separate fixture client, not an attach from the operator's pane.
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	for _, warm := range []bool{true, false} {
		name := "checkpoint"
		if warm {
			name = "warm"
		}
		t.Run(name, func(t *testing.T) { runRecoveryLiveFixture(t, warm) })
	}
}

type recoveryLiveProc struct {
	*FakeProcOps
	source int
}

func (p recoveryLiveProc) Exists(pid int) Liveness {
	if pid == p.source {
		return (OSProcOps{}).Exists(pid)
	}
	return p.FakeProcOps.Exists(pid)
}
func (p recoveryLiveProc) Identity(pid int) (string, error) {
	if pid == p.source {
		return (OSProcOps{}).Identity(pid)
	}
	return p.FakeProcOps.Identity(pid)
}

func runRecoveryLiveFixture(t *testing.T, warm bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
	defer cancel()
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	session := "pair-recovery-" + hex.EncodeToString(entropy[:])
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	sibling := filepath.Join(base, "other-worktree")
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	git("init", "-q", "-b", "main", repo)
	git("-C", repo, "-c", "user.name=Recovery Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "source")
	git("-C", repo, "worktree", "add", "-q", "-b", "checkpoint", sibling)
	env := newTestEnv(t, repo)
	c := env.Couch
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	source := warmDetachedRecord(t)
	source.Address.RepoScope = scope.Key
	source.StartingPath = repo
	source.WorkingPath = repo
	source.Reservation = false
	helper := exec.Command("sleep", "60")
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill(); _, _ = helper.Process.Wait() })
	identity, err := (OSProcOps{}).Identity(helper.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	source.Incarnations = []ThreadIncarnation{{PID: helper.Process.Pid, Identity: identity, State: IncarnationLive, LaunchProfile: source.LatestLaunchProfile}}
	source, err = c.Threads.CreateThread(source)
	if err != nil {
		t.Fatal(err)
	}
	c.Proc = recoveryLiveProc{FakeProcOps: env.Proc, source: helper.Process.Pid}
	sentinel := exec.Command("sleep", "60")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sentinel.Process.Kill(); _, _ = sentinel.Process.Wait() })
	sentinelIdentity, err := (OSProcOps{}).Identity(sentinel.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	untouched := warmDetachedRecord(t)
	untouched.Address.RepoScope = scope.Key
	untouched.Address.Tag = "couch-ffffffffffffffff"
	untouched.StartingPath, untouched.WorkingPath = repo, repo
	untouched.Reservation = false
	untouched.Incarnations = []ThreadIncarnation{{PID: sentinel.Process.Pid, Identity: sentinelIdentity, State: IncarnationLive, LaunchProfile: untouched.LatestLaunchProfile}}
	untouched, err = c.Threads.CreateThread(untouched)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(base, "data")
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := launcher.EnsureThreadAddressForPair(data, scope, string(source.Address.Tag), false); err != nil {
		t.Fatal(err)
	}
	indexSessionForConformance(t, data, source.Address, session)
	checker := NewScopedThreadArtifactCollisionChecker(data)
	c.Artifacts = checker
	ledger := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	if _, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: scope.Key, Tag: string(source.Address.Tag), Agent: "claude"}); err != nil {
		t.Fatal(err)
	}
	c.PairLifecycle = &PairLifecycleController{Threads: c.Threads, DataDir: data, Lifecycle: PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}, Sessions: checker, Proc: c.Proc, Clock: c.Clock}
	c.ContinuationSource = (OSContinuationSourceReader{DataDir: data}).Read
	reader := OSOrientationStatusReader{DataDir: data, Session: checker.PairSession, Proc: OSProcOps{}}
	c.FreshRegistration, c.OrientationStatus = reader.Registered, reader.Read
	c.ContinuationGeneration = reader.Generation
	agentPIDFile := filepath.Join(base, "source-agent.pid")
	inputFile := filepath.Join(base, "source-input")
	paneIDFile := filepath.Join(base, "source-pane")
	sourceScript := filepath.Join(base, "source-agent.sh")
	script := "#!/bin/sh\nprintf '%s' \"$ZELLIJ_PANE_ID\" > \"$3\"\nprintf '%s' \"$$\" > \"$1\"\nwhile IFS= read -r line; do printf '%s\\n' \"$line\" >> \"$2\"; printf 'AGENT:%s\\n' \"$line\"; done\n"
	if err := os.WriteFile(sourceScript, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "new-pane", "--", sourceScript, agentPIDFile, inputFile, paneIDFile).CombinedOutput(); err != nil {
		t.Fatalf("source pane: %v %s", err, out)
	}
	waitFile(t, agentPIDFile)
	waitFile(t, paneIDFile)
	paneRaw, err := os.ReadFile(paneIDFile)
	if err != nil {
		t.Fatal(err)
	}
	paneID := strings.TrimSpace(string(paneRaw))
	raw, err := os.ReadFile(agentPIDFile)
	if err != nil {
		t.Fatal(err)
	}
	agentPID, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	agentIdentity, err := (OSProcOps{}).Identity(agentPID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.KillClient(); err != nil {
		t.Fatal(err)
	}
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = helper.Wait()
	if c.Proc.Exists(helper.Process.Pid) != Dead {
		t.Fatal("actual source helper not dead")
	}
	t.Logf("owned session=%s helper=%d/%s dead; agent=%d/%s", session, helper.Process.Pid, identity, agentPID, agentIdentity)
	checkpointPath := ""
	body := "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nLIVE-RECOVERY-250 exact sibling seed.\n"
	if !warm {
		if err := checker.Quiesce(source.Address); err != nil {
			t.Fatal(err)
		}
		checkpointPath = filepath.Join(sibling, "checkpoint.md")
		if err := os.WriteFile(checkpointPath, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	starts := 0
	env.Runner.AfterAcknowledge = func(id string) error {
		starts++
		record, err := c.Threads.GetThread(source.Address)
		if err != nil {
			return err
		}
		inc := record.Incarnations[0]
		env.Proc.Set(inc.PID, inc.Identity)
		if warm {
			return nil
		}
		fixture, err = pairlifecycletest.StartControlledZellij(ctx, session)
		if err != nil {
			return err
		}
		var profile launcher.TrustedLaunchProfile
		for _, entry := range env.Runner.Child(id).Env {
			if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
				if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
					return err
				}
			}
		}
		if profile.Orientation == nil || !profile.FreshRequired {
			return fmt.Errorf("recovery lost fresh seed")
		}
		target, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: scope.Key, Tag: string(source.Address.Tag), Agent: "claude"})
		if err != nil {
			return err
		}
		readyPath, err := paths.AgentReadyChecked("claude")
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(profile.Orientation)
		if err != nil {
			return err
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		args := []string{"--session", session, "action", "new-pane", "--", "env", "PAIR_RECOVERY_LIVE_AGENT=1", orientation.Env + "=" + string(encoded), "PAIR_RECOVERY_LIVE_READY=" + readyPath, "PAIR_RECOVERY_LIVE_ORDINAL=" + strconv.FormatUint(target.Ordinal, 10), "PAIR_RECOVERY_LIVE_CAPTURE=" + filepath.Join(base, "target-seed"), executable, "-test.run=^TestRecoveryLiveAgentStandIn$"}
		if out, err := exec.CommandContext(ctx, "zellij", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("target pane: %w %s", err, out)
		}
		return nil
	}
	result, err := c.RecoverThread(ctx, source.Address, checkpointPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Handle == nil || result.Record.Thread != source.Address || starts != 1 {
		t.Fatalf("recovery result=%+v starts=%d", result, starts)
	}
	t.Cleanup(func() { env.Runner.SetExited(result.Handle.ID(), 0) })
	if warm {
		if got, err := (OSProcOps{}).Identity(agentPID); err != nil || got != agentIdentity {
			t.Fatalf("surviving agent changed %q %v", got, err)
		}
		if out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "write-chars", "--pane-id", paneID, "WARM-INPUT-250").CombinedOutput(); err != nil {
			t.Fatalf("write agent: %v %s", err, out)
		}
		if out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "write", "--pane-id", paneID, "13").CombinedOutput(); err != nil {
			t.Fatalf("submit agent: %v %s", err, out)
		}
		waitFile(t, inputFile)
		input, err := os.ReadFile(inputFile)
		if err != nil || !strings.Contains(string(input), "WARM-INPUT-250") {
			t.Fatalf("same agent input: %q %v", input, err)
		}
	} else {
		captured, err := os.ReadFile(filepath.Join(base, "target-seed"))
		if err != nil || string(captured) != body {
			t.Fatalf("exact target seed: %q %v", captured, err)
		}
		if _, err := c.RetryContinuation(ctx, source.Address, result.Status.RequestID); err != nil {
			t.Fatal(err)
		}
		if starts != 1 {
			t.Fatalf("retry duplicated a surviving checkpoint target: %d starts", starts)
		}
		state, err := c.ReconcileContinuation(ctx, source.Address, result.Status.RequestID, result.Status.Attempt)
		if err != nil || state.Phase != checkpoint.Complete {
			t.Fatalf("target receipt: %+v %v", state, err)
		}
	}
	after, err := c.Threads.GetThread(untouched.Address)
	if err != nil || !reflect.DeepEqual(after, untouched) {
		t.Fatalf("recovery changed another record: %+v %v", after, err)
	}
	if got, err := (OSProcOps{}).Identity(sentinel.Process.Pid); err != nil || got != sentinelIdentity {
		t.Fatalf("recovery disturbed another process: %q %v", got, err)
	}
	if err := checker.Quiesce(source.Address); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryLiveAgentStandIn(t *testing.T) {
	if os.Getenv("PAIR_RECOVERY_LIVE_AGENT") != "1" {
		t.Skip("controlled Zellij only")
	}
	var request orientation.Request
	if err := json.Unmarshal([]byte(os.Getenv(orientation.Env)), &request); err != nil {
		t.Fatal(err)
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	quoted, err := strconv.QuotedPrefix(strings.TrimPrefix(request.Body, "Read the saved continuation at "))
	if err != nil {
		t.Fatal(err)
	}
	path, err := strconv.Unquote(quoted)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := checkpoint.ReadFile(path)
	if err != nil || !strings.Contains(request.Body, cp.Digest) {
		t.Fatalf("exact snapshot %v", err)
	}
	if err := os.WriteFile(os.Getenv("PAIR_RECOVERY_LIVE_CAPTURE"), []byte(cp.Body), 0600); err != nil {
		t.Fatal(err)
	}
	ordinal, err := strconv.ParseUint(os.Getenv("PAIR_RECOVERY_LIVE_ORDINAL"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := readiness.Encode(readiness.ReadyRecord{Tag: request.Tag, Agent: request.Agent, Session: os.Getenv("ZELLIJ_SESSION_NAME"), Nonce: request.Attempt, PID: os.Getpid(), LaunchOrdinal: ordinal, Orientation: &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicBytes(os.Getenv("PAIR_RECOVERY_LIVE_READY"), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Second)
}
