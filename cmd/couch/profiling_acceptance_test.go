package main

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// This is a collection control, not a performance acceptance budget. Run with
// a go-test watchdog of at least 15m: four 90s windows also need setup/cleanup.
func TestDisposableCouchProfile(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_PROFILE") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_PROFILE=1 for disposable native profile collection")
	}
	evidence := os.Getenv("PAIR_COUCH_PROFILE_EVIDENCE")
	if evidence == "" {
		evidence = filepath.Join(t.TempDir(), "profiles")
	}
	if !filepath.IsAbs(evidence) || filepath.Clean(evidence) != evidence {
		t.Fatal("PAIR_COUCH_PROFILE_EVIDENCE must name a new clean absolute directory")
	}
	if err := os.Mkdir(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	t.Logf("profile evidence: %s (default temporary evidence expires with the test; set PAIR_COUCH_PROFILE_EVIDENCE to retain)", evidence)
	binaries := testcouch.Build(t)
	retainedBinary := filepath.Join(evidence, "couch")
	hash, err := profileCopyFile(binaries.Couch, retainedBinary, 128<<20, 0700)
	if err != nil {
		t.Fatal(err)
	}
	build, err := buildinfo.ReadFile(retainedBinary)
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			for _, workload := range []string{"quiet", "busy"} {
				if !t.Run(workload, func(t *testing.T) {
					profileFleetCase(t, binaries, evidence, retainedBinary, hash, build, agent, workload)
				}) {
					break
				}
			}
		}) {
			break
		}
	}
}

type profileStateChange struct {
	Actor    int                `json:"actor"`
	Phase    agentfixture.Phase `json:"phase"`
	Started  time.Time          `json:"started"`
	Finished time.Time          `json:"finished"`
}

type profileCollectionManifest struct {
	CandidateSHA256       string               `json:"candidate_sha256"`
	CandidateBuild        *debug.BuildInfo     `json:"candidate_build"`
	BuildEvidence         map[string]string    `json:"build_evidence"`
	Agent                 string               `json:"fixture_agent"`
	RunID                 string               `json:"run_id"`
	Cohort                []agentfixture.Ready `json:"cohort"`
	Workload              string               `json:"workload"`
	WorkloadDescription   string               `json:"workload_description"`
	Started               time.Time            `json:"started"`
	ActorsReady           time.Time            `json:"actors_ready"`
	Finished              time.Time            `json:"finished"`
	Rows                  int                  `json:"rows"`
	Columns               int                  `json:"columns"`
	Cast                  bool                 `json:"cast"`
	Viewers               int                  `json:"viewers"`
	CompletedChanges      []profileStateChange `json:"completed_changes"`
	CompletedCommandCount int                  `json:"completed_command_count"`
	Limitations           string               `json:"limitations"`
}

type profileCompletion struct {
	Status         string           `json:"status"`
	Failure        string           `json:"failure"`
	Started        time.Time        `json:"started"`
	CaptureStarted time.Time        `json:"capture_started"`
	Finished       time.Time        `json:"finished"`
	WarmupSeconds  int              `json:"warmup_seconds"`
	CPUSeconds     int              `json:"cpu_seconds"`
	Build          *debug.BuildInfo `json:"build"`
}

func profileFleetCase(t *testing.T, binaries testcouch.Binaries, evidence, retainedBinary, hash string, build *debug.BuildInfo, agent, workload string) {
	t.Helper()
	destination := filepath.Join(evidence, agent+"-"+workload)
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 3, ExplicitHub: true, AgentKind: agent, RuntimeBudget: 6 * time.Minute})
	leaf := filepath.Join(fleet.Roots.Mutable["artifacts"], "profile")
	fleet.Environment = append(fleet.Environment, "COUCH_PROFILE_DIR="+leaf)
	manifest := profileCollectionManifest{
		CandidateSHA256: hash, CandidateBuild: build, BuildEvidence: binaries.Evidence,
		Agent: agent, RunID: fleet.RunID, Workload: workload, Started: time.Now(), Rows: 32, Columns: 120,
		WorkloadDescription: "quiet: no drive after startup; busy: at most one round across four actors per 500ms, alternating Working/Idle; missed ticks do not catch up",
		Limitations:         "Native fixture agents are not live models. No Chrome coverage. Cast off, zero viewers. No throughput, CPU or allocation acceptance budget; this test does not select an optimization.",
	}
	// Copy fixed bounded outputs before the native owner's cleanup removes them.
	// Failures still go through Fleet.Close's existing evidence owner, without retry.
	defer func() {
		manifest.Finished = time.Now()
		manifest.CompletedCommandCount = len(manifest.CompletedChanges)
		for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof", "metadata.json"} {
			limit := int64(32 << 20)
			if name == "metadata.json" {
				limit = 64 << 10
			}
			if _, err := profileCopyFile(filepath.Join(leaf, name), filepath.Join(destination, name), limit, 0600); err != nil {
				t.Errorf("retain %s before fleet cleanup: %v", name, err)
			}
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(destination, "manifest.json"), data, 0600)
		}
		if err != nil {
			t.Errorf("retain manifest: %v", err)
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := fleet.Close(cleanup); err != nil {
			t.Errorf("profile fleet cleanup: %v", err)
		}
		if err := fleet.AssertCleaned(); err != nil {
			t.Errorf("profile fleet cleanup proof: %v", err)
		}
	}()
	setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
	err := fleet.Prepare(setup)
	cancelSetup()
	if err != nil {
		t.Fatal(err)
	}
	startup, cancelStartup := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancelStartup()
	if err := fleet.Start(startup); err != nil {
		t.Fatal(err)
	}
	if err := fleet.Resize(32, 120); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		if err := fleet.StartWorker(startup, n); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n <= 3; n++ {
		ready, err := fleet.Agent(n)
		if err != nil || ready.Agent != agent {
			t.Fatalf("actor %d personality: %+v: %v", n, ready, err)
		}
		manifest.Cohort = append(manifest.Cohort, ready)
	}
	manifest.ActorsReady = time.Now()
	cancelStartup()
	scenario, cancelScenario := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancelScenario()
	completion, err := profileCollect(scenario, fleet, leaf, workload, &manifest)
	if err != nil {
		t.Fatal(err)
	}
	if completion.Status != "complete" || completion.Build == nil || completion.WarmupSeconds != 60 || completion.CPUSeconds != 30 || completion.CaptureStarted.IsZero() || !completion.Finished.After(completion.CaptureStarted) {
		t.Fatalf("incomplete profile window: %+v", completion)
	}
	if !completion.CaptureStarted.After(manifest.ActorsReady) {
		t.Fatal("actor startup overlapped capture; this sample does not establish the declared steady workload")
	}
	// Parsing occurs before the deferred owner cleanup, from independent copies.
	for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof"} {
		retained := filepath.Join(destination, name)
		if _, err := profileCopyFile(filepath.Join(leaf, name), retained, 32<<20, 0600); err != nil {
			t.Fatal(err)
		}
		parse, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		cmd := exec.CommandContext(parse, "go", "tool", "pprof", "-top", retainedBinary, retained)
		output := &profileReport{remaining: 1 << 20}
		cmd.Stdout, cmd.Stderr = output, output
		err := cmd.Run()
		cancel()
		if writeErr := os.WriteFile(filepath.Join(destination, name+".top.txt"), output.data, 0600); writeErr != nil {
			t.Fatal(writeErr)
		}
		if err != nil {
			t.Fatalf("parse %s with retained candidate: %v\n%s", name, err, output.data)
		}
	}
}

func profileCollect(ctx context.Context, fleet *testcouch.Fleet, leaf, workload string, manifest *profileCollectionManifest) (profileCompletion, error) {
	poll := time.NewTicker(250 * time.Millisecond)
	defer poll.Stop()
	var drive <-chan time.Time
	var timer *time.Timer
	if workload == "busy" {
		timer = time.NewTimer(500 * time.Millisecond)
		drive = timer.C
		defer timer.Stop()
	}
	phase := agentfixture.Working
	for {
		select {
		case <-ctx.Done():
			return profileCompletion{}, ctx.Err()
		case <-poll.C:
			data, err := profileReadBounded(filepath.Join(leaf, "metadata.json"), 64<<10)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return profileCompletion{}, err
			}
			if len(data) == 0 {
				continue
			}
			var metadata profileCompletion
			if err := json.Unmarshal(data, &metadata); err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) {
					continue
				}
				var syntax *json.SyntaxError
				if errors.As(err, &syntax) && syntax.Offset == int64(len(data)) {
					continue
				}
				return profileCompletion{}, fmt.Errorf("decode profile metadata: %w", err)
			}
			if metadata.Status != "" {
				return metadata, nil
			}
		case <-drive:
			for actor := 0; actor <= 3; actor++ {
				change := profileStateChange{Actor: actor, Phase: phase, Started: time.Now()}
				if err := fleet.SetState(ctx, actor, phase); err != nil {
					return profileCompletion{}, err
				}
				change.Finished = time.Now()
				manifest.CompletedChanges = append(manifest.CompletedChanges, change)
			}
			if phase == agentfixture.Working {
				phase = agentfixture.Idle
			} else {
				phase = agentfixture.Working
			}
			// Reset after the completed round: slow receipts never create catch-up bursts.
			timer.Reset(500 * time.Millisecond)
		}
	}
}

func profileReadBounded(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("artifact exceeds %d bytes: %s", limit, path)
	}
	return data, err
}

func profileCopyFile(source, destination string, limit int64, mode os.FileMode) (string, error) {
	data, err := profileReadBounded(source, limit)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(destination, data, mode); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type profileReport struct {
	data      []byte
	remaining int
}

func (r *profileReport) Write(p []byte) (int, error) {
	n := min(len(p), r.remaining)
	r.data = append(r.data, p[:n]...)
	r.remaining -= n
	if n != len(p) {
		return n, fmt.Errorf("pprof report exceeds 1MiB")
	}
	return n, nil
}
