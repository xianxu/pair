package couchcore

// This composes the stateful continuation executor with a real Zellij pane.
// The deterministic pane checks the exact materialized seed and emits the same
// readiness wire schema as pair-wrap. Source parking and blocked-helper process
// ownership remain fake here; the existing park conformance checks real teardown.
// No paid agent runs. The launch prompt (#522) reaches the stand-in exactly as
// the trusted profile carries it; the wrapper's exec of it is pinned by
// launcher's TestCreateLayoutWrapperPreservesAgentCommand.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestContinuationZellijSeedTransportLive(t *testing.T) {
	liveOnly(t)
	testpair.ApplyEnvironment(t, t.TempDir(), "TMPDIR=/tmp", "SHELL=/bin/sh",
		"XDG_CONFIG_HOME="+t.TempDir(), "XDG_DATA_HOME="+t.TempDir(), "XDG_CACHE_HOME="+t.TempDir())
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	session := fmt.Sprintf("pair-continuation-%d", os.Getpid())
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("controlled zellij cleanup: %v", err)
		}
	})

	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: session, LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	original := filepath.Join(t.TempDir(), "sibling checkpoint.md")
	body := "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nVerify token live-seed-249 with Unicode λ and exact prompt history.\n"
	if err := os.WriteFile(original, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	accepted, err := env.Couch.RequestContinuation(ctx, source.Address, proof, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	indexSessionForConformance(t, dataDir, source.Address, session)
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	reader := OSReadyReader{DataDir: dataDir, Session: checker.PairSession, Proc: OSProcOps{}}
	env.Couch.FreshRegistration = reader.Registered
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.AddOwnerWrapperPane(ctx, paths.ScrollbackRaw("claude")); err != nil {
		t.Fatal(err)
	}
	readyPath, err := paths.AgentReadyChecked("claude")
	if err != nil {
		t.Fatal(err)
	}
	captured := filepath.Join(t.TempDir(), "captured.md")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	scriptDir := t.TempDir()
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		for _, entry := range child.Env {
			if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
				if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
					return err
				}
			}
		}
		if profile.Prompt == "" || profile.FreshNonce == "" {
			return fmt.Errorf("fresh target received no launch prompt: %+v", profile)
		}
		script, err := standInPaneScript(scriptDir, []string{
			"PAIR_CONTINUATION_LIVE_HELPER=1", "PAIR_CONTINUATION_LIVE_PROMPT=" + profile.Prompt,
			"PAIR_CONTINUATION_LIVE_TAG=" + profile.Tag, "PAIR_CONTINUATION_LIVE_NONCE=" + profile.FreshNonce,
			"PAIR_CONTINUATION_LIVE_SEED=" + mustContinuationPath(t, env.Couch, source.Address),
			"PAIR_CONTINUATION_LIVE_READY=" + readyPath, "PAIR_CONTINUATION_LIVE_CAPTURE=" + captured,
		}, executable, "TestContinuationZellijAgentStandIn")
		if err != nil {
			return err
		}
		args := []string{"--session", session, "action", "new-pane", "--", script}
		if out, err := exec.CommandContext(ctx, "zellij", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("start pane: %w: %s", err, out)
		}
		return nil
	}
	result, err := env.Couch.Continue(ctx, source.Address, accepted.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	// Registration completes it (#522): the prompt was part of the launch.
	if result.Status.Phase != checkpoint.Complete || result.Missed != nil {
		t.Fatalf("registration did not complete the request: %+v", result)
	}
	if got, err := os.ReadFile(captured); err != nil || string(got) != body {
		t.Fatalf("pane seed = %q, %v", got, err)
	}
	readyBytes, err := os.ReadFile(readyPath)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := readiness.Decode(string(readyBytes))
	if err != nil {
		t.Fatal(err)
	}
	if ready.Session != session || ready.Nonce != result.Status.Attempt || ready.Tag != string(source.Address.Tag) {
		t.Fatalf("wrong live target: %+v", ready)
	}
	if matched, err := reader.Registered(ctx, source.Address, "claude", "obsolete-attempt"); err != nil || matched {
		t.Fatalf("obsolete target matched: %v, %v", matched, err)
	}
	// The retained receipt is not authority once its exact Zellij session is gone.
	if err := checker.Quiesce(source.Address); err != nil {
		t.Fatal(err)
	}
	if matched, err := reader.Registered(ctx, source.Address, "claude", result.Status.Attempt); err != nil || matched {
		t.Fatalf("dead session retained registration: %v, %v", matched, err)
	}
}

func TestContinuationZellijAgentStandIn(t *testing.T) {
	t.Parallel()

	if os.Getenv("PAIR_CONTINUATION_LIVE_HELPER") != "1" {
		t.Skip("launched only in the controlled Zellij pane")
	}
	prompt := os.Getenv("PAIR_CONTINUATION_LIVE_PROMPT")
	path := os.Getenv("PAIR_CONTINUATION_LIVE_SEED")
	seed, err := checkpoint.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, strconv.Quote(path)) || strings.ContainsAny(prompt, "\r\n") {
		t.Fatalf("the launch prompt does not name the exact continuation in one line: %q", prompt)
	}
	if err := os.WriteFile(os.Getenv("PAIR_CONTINUATION_LIVE_CAPTURE"), []byte(seed.Body), 0600); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: os.Getenv("PAIR_CONTINUATION_LIVE_TAG"), Agent: "claude", Nonce: os.Getenv("PAIR_CONTINUATION_LIVE_NONCE"), Session: os.Getenv("ZELLIJ_SESSION_NAME"), PID: os.Getpid(), AgentIdentity: procutil.Identity(strconv.Itoa(os.Getpid()))}
	raw, err := readiness.Encode(ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicBytes(os.Getenv("PAIR_CONTINUATION_LIVE_READY"), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Second) // keep the exact ready PID live until fixture teardown
}
