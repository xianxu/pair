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
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// Real source process/session observations, real exact ledger/readiness files,
// and a deterministic Zellij pane; target Couch helpers use the existing fake.
// Command-package acceptance separately proves Console adoption and keyboard IO.
func TestRecoveryRealHelperAndSessionConformanceLive(t *testing.T) {
	liveOnly(t)
	// Darwin's default TMPDIR can exhaust Unix socket path capacity before
	// Zellij appends the session name. Keep its socket root short and isolate
	// configuration/data so the operator's shell and plugins do not run here.
	setupRecoveryLiveEnvironment(t)
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
	capture := newRecoveryQueryCapture(t)
	ctx = capture.Context(ctx)
	capture.Mark(recoveryLiveEnvironmentMark())
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	session := "pair-recovery-" + hex.EncodeToString(entropy[:])
	capture.Mark("source-session-start-enter")
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	capture.Mark(fmt.Sprintf("source-session-start-return err=%v", err))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		capture.Mark("controlled-session-cleanup-enter")
		err := fixture.Close()
		capture.Mark(fmt.Sprintf("controlled-session-cleanup-return err=%v", err))
		if err != nil {
			t.Errorf("controlled zellij cleanup: %v", err)
		}
	})
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	sibling := filepath.Join(base, "other-worktree")
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Env = testpair.Environment(t, base, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
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
	if err := fixture.AddOwnerWrapperPane(ctx, paths.ScrollbackRaw("claude")); err != nil {
		t.Fatal(err)
	}
	checker := NewScopedThreadArtifactCollisionChecker(data)
	c.Artifacts = checker
	ledger := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	if _, err := ledger.Append(paths.Ledger(), sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: scope.Key, Tag: string(source.Address.Tag), Agent: "claude"}); err != nil {
		t.Fatal(err)
	}
	c.PairLifecycle = &PairLifecycleController{Threads: c.Threads, DataDir: data, Lifecycle: PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}, Sessions: checker, Proc: c.Proc, Clock: c.Clock}
	c.ContinuationSource = (OSContinuationSourceReader{DataDir: data}).Read
	reader := OSReadyReader{DataDir: data, Session: checker.PairSession, Proc: OSProcOps{}}
	c.FreshRegistration = reader.Registered
	c.ContinuationGeneration = reader.Generation
	agentPIDFile := filepath.Join(base, "source-agent.pid")
	inputFile := filepath.Join(base, "source-input")
	paneIDFile := filepath.Join(base, "source-pane")
	sourceScript := filepath.Join(base, "source-agent.sh")
	script := "#!/bin/sh\n" + recoveryFixturePublisher("") + `
publish_fixture_value "$3" "$ZELLIJ_PANE_ID" || exit 1
publish_fixture_value "$1" "$$" || exit 1
while IFS= read -r line; do printf '%s\n' "$line" >> "$2"; printf 'AGENT:%s\n' "$line"; done
`
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
	capture.Mark(fmt.Sprintf("source-agent pid=%d identity=%q", agentPID, agentIdentity))
	capture.Mark("local-client-stop-enter")
	stopErr := fixture.KillClient()
	capture.Mark(fmt.Sprintf("local-client-stop-return err=%v; server detach unconfirmed", stopErr))
	if stopErr != nil {
		t.Fatal(stopErr)
	}
	capture.Mark("source-helper-kill-enter")
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = helper.Wait()
	capture.Mark("source-helper-wait-return")
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
		// A warm client is acknowledged before any guarded durable repair;
		// recovery may already have retired its dead predecessor. The local
		// handle, not a not-yet-published incarnation, owns this new process.
		handle := &fakeHandle{id: id, runner: env.Runner}
		env.Proc.Set(handle.PID(), handle.Identity())
		if warm {
			return nil
		}
		capture.Mark("checkpoint-session-start-enter")
		fixture, err = pairlifecycletest.StartControlledZellij(ctx, session)
		capture.Mark(fmt.Sprintf("checkpoint-session-start-return err=%v", err))
		if err != nil {
			return err
		}
		if err := fixture.AddOwnerWrapperPane(ctx, paths.ScrollbackRaw("claude")); err != nil {
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
		if profile.Prompt == "" || !profile.FreshRequired || profile.FreshNonce == "" {
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
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		script, err := standInPaneScript(base, []string{"PAIR_RECOVERY_LIVE_AGENT=1", "PAIR_RECOVERY_LIVE_PROMPT=" + profile.Prompt, "PAIR_RECOVERY_LIVE_TAG=" + profile.Tag, "PAIR_RECOVERY_LIVE_NONCE=" + profile.FreshNonce, "PAIR_RECOVERY_LIVE_READY=" + readyPath, "PAIR_RECOVERY_LIVE_ORDINAL=" + strconv.FormatUint(target.Ordinal, 10), "PAIR_RECOVERY_LIVE_CAPTURE=" + filepath.Join(base, "target-seed")}, executable, "TestRecoveryLiveAgentStandIn")
		if err != nil {
			return err
		}
		args := []string{"--session", session, "action", "new-pane", "--", script}
		if out, err := exec.CommandContext(ctx, "zellij", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("target pane: %w %s", err, out)
		}
		return nil
	}
	capture.Mark("recover-thread-enter")
	result, err := c.RecoverThread(ctx, source.Address, checkpointPath)
	capture.Mark(fmt.Sprintf("recover-thread-return err=%v", err))
	if err != nil {
		// The deferred recorder retains the failed request itself; another native
		// query would observe a later state and cannot explain its missing reply.
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
		waitUntil(t, "same agent input receipt", func() bool {
			input, err := os.ReadFile(inputFile)
			return err == nil && strings.Contains(string(input), "WARM-INPUT-250")
		})
		input, err := os.ReadFile(inputFile)
		if err != nil || !strings.Contains(string(input), "WARM-INPUT-250") {
			t.Fatalf("same agent input: %q %v", input, err)
		}
	} else {
		captured, err := os.ReadFile(filepath.Join(base, "target-seed"))
		if err != nil || string(captured) != body {
			t.Fatalf("exact target seed: %q %v", captured, err)
		}
		// Registration completed it (#522); continuing again starts nothing.
		if result.Status.Phase != checkpoint.Complete {
			t.Fatalf("target receipt: %+v", result.Status)
		}
		if _, err := c.Continue(ctx, source.Address, result.Status.RequestID); err != nil {
			t.Fatal(err)
		}
		if starts != 1 {
			t.Fatalf("continuing duplicated a surviving checkpoint target: %d starts", starts)
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
	t.Parallel()

	if os.Getenv("PAIR_RECOVERY_LIVE_AGENT") != "1" {
		t.Skip("controlled Zellij only")
	}
	prompt := os.Getenv("PAIR_RECOVERY_LIVE_PROMPT")
	path, err := continuationPromptPath(prompt)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := checkpoint.ReadFile(path)
	if err != nil {
		t.Fatalf("exact snapshot %v", err)
	}
	if err := os.WriteFile(os.Getenv("PAIR_RECOVERY_LIVE_CAPTURE"), []byte(cp.Body), 0600); err != nil {
		t.Fatal(err)
	}
	ordinal, err := strconv.ParseUint(os.Getenv("PAIR_RECOVERY_LIVE_ORDINAL"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := readiness.Encode(readiness.ReadyRecord{Tag: os.Getenv("PAIR_RECOVERY_LIVE_TAG"), Agent: "claude", Session: os.Getenv("ZELLIJ_SESSION_NAME"), Nonce: os.Getenv("PAIR_RECOVERY_LIVE_NONCE"), PID: os.Getpid(), AgentIdentity: procutil.Identity(strconv.Itoa(os.Getpid())), LaunchOrdinal: ordinal})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicBytes(os.Getenv("PAIR_RECOVERY_LIVE_READY"), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Second)
}
