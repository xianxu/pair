package couchcore

// This composes the stateful continuation executor with a real Zellij pane.
// The deterministic pane checks the exact materialized seed and emits the same
// readiness wire schema as pair-wrap. Source parking and blocked-helper process
// ownership remain fake here; the existing park conformance checks real teardown.
// No paid agent runs, and this does not claim real composer recognition/submission.

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
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestContinuationZellijSeedTransportLive(t *testing.T) {
	liveOnly(t)
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	session := fmt.Sprintf("pair-continuation-%d", os.Getpid())
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })

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
	reader := OSOrientationStatusReader{DataDir: dataDir, Session: checker.PairSession, Proc: OSProcOps{}}
	env.Couch.FreshRegistration = reader.Registered
	env.Couch.OrientationStatus = reader.Read
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	readyPath, err := paths.AgentReadyChecked("claude")
	if err != nil {
		t.Fatal(err)
	}
	captured := filepath.Join(t.TempDir(), "captured.md")
	release := filepath.Join(t.TempDir(), "submit")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		raw := ""
		for _, entry := range child.Env {
			if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
				var profile launcher.TrustedLaunchProfile
				if err := json.Unmarshal([]byte(strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")), &profile); err != nil {
					return err
				}
				if profile.Orientation != nil {
					encoded, err := json.Marshal(profile.Orientation)
					if err != nil {
						return err
					}
					raw = string(encoded)
				}
			}
		}
		if raw == "" {
			return fmt.Errorf("fresh target received no orientation request")
		}
		args := []string{"--session", session, "action", "new-pane", "--", "env",
			"PAIR_CONTINUATION_LIVE_HELPER=1", orientation.Env + "=" + raw,
			"PAIR_CONTINUATION_LIVE_SEED=" + env.Couch.continuationPath(source.Address),
			"PAIR_CONTINUATION_LIVE_READY=" + readyPath, "PAIR_CONTINUATION_LIVE_CAPTURE=" + captured,
			"PAIR_CONTINUATION_LIVE_RELEASE=" + release,
			executable, "-test.run=^TestContinuationZellijAgentStandIn$"}
		if out, err := exec.CommandContext(ctx, "zellij", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("start pane: %w: %s", err, out)
		}
		return nil
	}
	result, err := env.Couch.Continue(ctx, source.Address, accepted.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status.Phase != checkpoint.Running || result.Orientation == nil {
		t.Fatalf("registration completed too early: %+v", result)
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
	env.Proc.Set(result.Record.PID, result.Record.Identity)
	pending, err := env.Couch.ReconcileContinuation(ctx, source.Address, accepted.RequestID, result.Status.Attempt)
	if err != nil || pending.Phase != checkpoint.Running {
		t.Fatalf("waiting receipt = %+v, %v", pending, err)
	}
	if err := os.WriteFile(release, []byte("submit"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		completed, err := env.Couch.ReconcileContinuation(ctx, source.Address, accepted.RequestID, result.Status.Attempt)
		if err != nil {
			t.Fatal(err)
		}
		if completed.Phase == checkpoint.Complete {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
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
	if os.Getenv("PAIR_CONTINUATION_LIVE_HELPER") != "1" {
		t.Skip("launched only in the controlled Zellij pane")
	}
	var request orientation.Request
	if err := json.Unmarshal([]byte(os.Getenv(orientation.Env)), &request); err != nil {
		t.Fatal(err)
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	path := os.Getenv("PAIR_CONTINUATION_LIVE_SEED")
	seed, err := checkpoint.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request.Body, strconv.Quote(path)) || !strings.Contains(request.Body, seed.Digest) {
		t.Fatal("orientation does not bind the exact checkpoint path and digest")
	}
	if err := os.WriteFile(os.Getenv("PAIR_CONTINUATION_LIVE_CAPTURE"), []byte(seed.Body), 0600); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: request.Tag, Agent: request.Agent, Nonce: request.Attempt, Session: os.Getenv("ZELLIJ_SESSION_NAME"), PID: os.Getpid(), Orientation: &orientation.DeliveryState{Phase: orientation.DeliveryWaiting}}
	publish := func() {
		raw, err := readiness.Encode(ready)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeAtomicBytes(os.Getenv("PAIR_CONTINUATION_LIVE_READY"), []byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	publish()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(os.Getenv("PAIR_CONTINUATION_LIVE_RELEASE")); err == nil {
			ready.Orientation = &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}
			publish()
			time.Sleep(20 * time.Second) // keep the exact ready PID live until fixture teardown
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("stand-in never received submission release")
}
