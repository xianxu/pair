package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// This collection control has no CPU budget. Select one personality initially;
// the full matrix needs a watchdog of at least45m, including owned cleanup.
func TestDisposableCouchProfileScale(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_PROFILE_SCALE") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_PROFILE_SCALE=1 for disposable scaled profile collection")
	}
	evidence := os.Getenv("PAIR_COUCH_PROFILE_EVIDENCE")
	if evidence == "" {
		evidence = filepath.Join(t.TempDir(), "scale-profiles")
	}
	if !filepath.IsAbs(evidence) || filepath.Clean(evidence) != evidence {
		t.Fatal("PAIR_COUCH_PROFILE_EVIDENCE must name a new clean absolute directory")
	}
	if err := os.Mkdir(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	t.Logf("scaled profile evidence: %s (temporary evidence expires with the test unless PAIR_COUCH_PROFILE_EVIDENCE is set)", evidence)
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
	cases := []profileScaleCase{
		{Name: "N1", Actors: 1}, {Name: "N5", Actors: 5}, {Name: "N10", Actors: 10}, {Name: "N20", Actors: 20},
		{Name: "N20_history2000", Actors: 20, History: 2000},
		{Name: "N20_age", Actors: 20, History: 2000, Age: true},
	}
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			for _, scenario := range cases {
				if !t.Run(scenario.Name, func(t *testing.T) {
					profileScaleRun(t, binaries, evidence, retainedBinary, hash, build, agent, scenario)
				}) {
					break
				}
			}
		}) {
			break
		}
	}
}

type profileScaleCase struct {
	Name            string
	Actors, History int
	Age             bool
}
type profileScaleIdentity struct {
	Slot              string             `json:"slot"`
	WrapperPID        int                `json:"wrapper_pid"`
	WrapperGeneration string             `json:"wrapper_generation"`
	Ready             agentfixture.Ready `json:"fixture_ready"`
}
type profileScaleCohort struct {
	At     time.Time              `json:"at"`
	Actors []profileScaleIdentity `json:"actors"`
}
type profileScaleWorkload struct {
	Actor             int                               `json:"actor"`
	Request           agentfixture.WorkloadRequest      `json:"request"`
	Start             *agentfixture.WorkloadObservation `json:"start,omitempty"`
	StartAcknowledged time.Time                         `json:"start_acknowledged"`
	StartError        string                            `json:"start_error,omitempty"`
	StopAttempted     bool                              `json:"stop_attempted"`
	Stop              *agentfixture.WorkloadObservation `json:"stop,omitempty"`
	StopAcknowledged  time.Time                         `json:"stop_acknowledged"`
	StopError         string                            `json:"stop_error,omitempty"`
}
type profileScaleResourceSample struct {
	ObservedAt         time.Time              `json:"observed_at"`
	ObservationStarted time.Time              `json:"observation_started"`
	Owner              testcouch.Resource     `json:"owned_supervisor"`
	Process            procutil.Process       `json:"process"`
	Usage              procutil.ResourceUsage `json:"usage"`
	TableCalls         int                    `json:"table_calls"`
	UsageCalls         int                    `json:"usage_calls"`
}
type profileScaleResourceWindow struct {
	Name           string `json:"name"`
	Before, After  profileScaleResourceSample
	ElapsedNS      int64   `json:"elapsed_ns"`
	CPUTimeNS      uint64  `json:"cpu_delta_ns"`
	DiskWriteBytes uint64  `json:"physical_disk_write_delta_bytes"`
	CPUPercent     float64 `json:"cpu_percent_one_core"`
	Error          string  `json:"error,omitempty"`
}
type profileScaleManifest struct {
	CandidateSHA256                                    string            `json:"candidate_sha256"`
	CandidateBuild                                     *debug.BuildInfo  `json:"candidate_build"`
	BuildEvidence                                      map[string]string `json:"build_evidence"`
	Agent                                              string            `json:"fixture_agent"`
	RunID                                              string            `json:"run_id"`
	Case                                               profileScaleCase  `json:"case"`
	Rows, Columns                                      int
	Cast                                               bool `json:"cast"`
	Viewers                                            int  `json:"viewers"`
	Started, Finished                                  time.Time
	InitialCohort, ReattachedCohort, FinalCohort       profileScaleCohort
	InitialSupervisor, MeasuredSupervisor              testcouch.Resource
	DetachedAt, ReattachedAt, AllWorkloadsAcknowledged time.Time
	Workloads                                          []profileScaleWorkload       `json:"workloads"`
	Profile                                            profileCompletion            `json:"profile"`
	ResourceWindows                                    []profileScaleResourceWindow `json:"resource_windows"`
	AgeWaitStarted, AgeWaitFinished                    time.Time
	CleanupStarted, CleanupFinished                    time.Time
	CleanupError                                       string `json:"cleanup_error,omitempty"`
	CleanupProved                                      bool   `json:"cleanup_proved"`
	Limitations                                        string `json:"limitations"`
}

func profileScaleRun(t *testing.T, binaries testcouch.Binaries, evidence, retainedBinary, hash string, build *debug.BuildInfo, agent string, scenario profileScaleCase) {
	t.Helper()
	destination := filepath.Join(evidence, agent+"-"+scenario.Name)
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	fleet := testcouch.New(t, binaries, testcouch.Options{Workers: scenario.Actors - 1, ExplicitHub: true, AgentKind: agent, RuntimeBudget: 20 * time.Minute})
	leaf := filepath.Join(fleet.Roots.Mutable["artifacts"], "scale-profile")
	manifest := profileScaleManifest{
		CandidateSHA256: hash, CandidateBuild: build, BuildEvidence: binaries.Evidence, Agent: agent, RunID: fleet.RunID,
		Case: scenario, Rows: 32, Columns: 120, Started: time.Now(),
		Limitations: "Native fixture agents are not live models. Cast off; zero viewers; no Chrome coverage. Fixed100ms bounded spinner workload; acknowledgments and stop observations report actual completed ticks/bytes, not promised throughput. Runtime profile captures the fresh reattached Couch at60–90s. External30s CPU/RSS/physical-write counter windows run AFTER profiling and refer to different times. The aged contrast uses the same Couch generation after five EXTRA minutes; its CPU is measured only by external counters, not by the earlier profile. No CPU, allocation or throughput acceptance budgets; no optimization selected. Owned profiles are bounded32MiB each, metadata64KiB, top reports1MiB each, manifest1MiB; retained executable128MiB. Requested retained evidence remains operator-owned for explicit removal.",
	}
	// Record partial outcomes before removing native roots, then add cleanup proof
	// to that same bounded manifest. Fleet.Close retains its existing failure evidence.
	defer func() {
		stop, cancelStop := context.WithTimeout(context.Background(), 30*time.Second)
		profileScaleStop(t, stop, fleet, &manifest)
		cancelStop()
		manifest.Finished = time.Now()
		for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof", "metadata.json"} {
			limit := int64(32 << 20)
			if name == "metadata.json" {
				limit = 64 << 10
			}
			if _, err := profileCopyFile(filepath.Join(leaf, name), filepath.Join(destination, name), limit, 0600); err != nil {
				t.Errorf("retain scale %s before fleet cleanup: %v", name, err)
			}
		}
		profileScaleWriteManifest(t, destination, manifest)
		manifest.CleanupStarted = time.Now()
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := fleet.Close(cleanup)
		cancel()
		manifest.CleanupFinished = time.Now()
		if err != nil {
			manifest.CleanupError = err.Error()
			t.Errorf("scaled fleet cleanup: %v", err)
		}
		if err := fleet.AssertCleaned(); err != nil {
			manifest.CleanupError += "; " + err.Error()
			t.Errorf("scaled fleet cleanup proof: %v", err)
		} else {
			manifest.CleanupProved = true
		}
		profileScaleWriteManifest(t, destination, manifest)
	}()
	setup, cancelSetup := context.WithTimeout(t.Context(), 3*time.Minute)
	err := fleet.Prepare(setup)
	cancelSetup()
	if err != nil {
		t.Fatal(err)
	}
	startup, cancelStartup := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancelStartup()
	if err := fleet.Start(startup); err != nil {
		t.Fatal(err)
	}
	if err := fleet.Resize(32, 120); err != nil {
		t.Fatal(err)
	}
	for n := 1; n < scenario.Actors; n++ {
		if err := fleet.StartWorker(startup, n); err != nil {
			t.Fatal(err)
		}
	}
	manifest.InitialCohort, err = profileScaleObserveCohort(startup, fleet, scenario.Actors, agent, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest.InitialSupervisor, err = fleet.SupervisorIdentity(startup)
	if err != nil {
		t.Fatal(err)
	}
	if err := fleet.Detach(startup); err != nil {
		t.Fatal(err)
	}
	manifest.DetachedAt = time.Now()
	cancelStartup()
	scenarioCtx, cancelScenario := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancelScenario()
	fleet.Environment = append(fleet.Environment, "COUCH_PROFILE_DIR="+leaf)
	if err := fleet.RestartSupervisor(scenarioCtx); err != nil {
		t.Fatal(err)
	}
	manifest.ReattachedAt = time.Now()
	if err := fleet.WaitActors(scenarioCtx, scenario.Actors); err != nil {
		t.Fatal(err)
	}
	manifest.ReattachedCohort, err = profileScaleObserveCohort(scenarioCtx, fleet, scenario.Actors, agent, &manifest.InitialCohort)
	if err != nil {
		t.Fatal(err)
	}
	manifest.MeasuredSupervisor, err = fleet.SupervisorIdentity(scenarioCtx)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.InitialSupervisor == manifest.MeasuredSupervisor {
		t.Fatal("reattach did not select a new owned Couch generation")
	}
	for n := 0; n < scenario.Actors; n++ {
		request := agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart, ScrollbackLines: scenario.History}
		manifest.Workloads = append(manifest.Workloads, profileScaleWorkload{Actor: n, Request: request})
		workload := &manifest.Workloads[len(manifest.Workloads)-1]
		observation, err := fleet.ConfigureWorkload(scenarioCtx, n, request)
		if err != nil {
			workload.StartError = err.Error()
			t.Fatalf("start actor%d workload: %v", n, err)
		}
		workload.Start, workload.StartAcknowledged = &observation, time.Now()
		if observation.Status != agentfixture.WorkloadRunning || observation.Started.IsZero() || observation.IntervalMS != 100 || observation.ScrollbackLines != scenario.History {
			t.Fatalf("actor%d workload not acknowledged as configured: %+v", n, observation)
		}
	}
	manifest.AllWorkloadsAcknowledged = time.Now()
	working, cancelWorking := context.WithTimeout(scenarioCtx, 15*time.Second)
	err = waitFleet(working, func() (bool, error) { return strings.Contains(fleet.Screen(), "Working"), nil })
	cancelWorking()
	if err != nil {
		t.Fatalf("steward Working projection: %v", err)
	}
	// The original collector's quiet mode only polls final profile metadata; the
	// native fixture owns every redraw, with no per-tick mailbox traffic here.
	collector := profileCollectionManifest{}
	manifest.Profile, err = profileCollect(scenarioCtx, fleet, leaf, "quiet", &collector)
	if err != nil {
		t.Fatal(err)
	}
	p := manifest.Profile
	if p.Status != "complete" || p.Build == nil || p.WarmupSeconds != 60 || p.CPUSeconds != 30 || p.Started.IsZero() || p.CaptureStarted.IsZero() || !p.Finished.After(p.CaptureStarted) || !p.CaptureStarted.After(manifest.AllWorkloadsAcknowledged) {
		t.Fatalf("scaled profile does not establish complete configured capture: %+v; all_acks=%s", p, manifest.AllWorkloadsAcknowledged)
	}
	// Real cumulative counters quantify later activity; sampled pprof CPU totals
	// are never substituted for these external counter deltas.
	first, err := profileScaleMeasure(scenarioCtx, fleet, manifest.MeasuredSupervisor, "after-profile")
	if err != nil {
		first.Error = err.Error()
	}
	manifest.ResourceWindows = append(manifest.ResourceWindows, first)
	if err != nil {
		t.Fatal(err)
	}
	if scenario.Age {
		manifest.AgeWaitStarted = time.Now()
		if err := profileScaleWait(scenarioCtx, fleet, 5*time.Minute); err != nil {
			t.Fatal(err)
		}
		manifest.AgeWaitFinished = time.Now()
		late, err := profileScaleMeasure(scenarioCtx, fleet, manifest.MeasuredSupervisor, "same-couch-five-extra-minutes")
		if err != nil {
			late.Error = err.Error()
		}
		manifest.ResourceWindows = append(manifest.ResourceWindows, late)
		if err != nil {
			t.Fatal(err)
		}
	}
	manifest.FinalCohort, err = profileScaleObserveCohort(scenarioCtx, fleet, scenario.Actors, agent, &manifest.InitialCohort)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := fleet.SupervisorIdentity(scenarioCtx); err != nil || current != manifest.MeasuredSupervisor {
		t.Fatalf("measured Couch lifetime changed: %+v: %v", current, err)
	}
	profileScaleStop(t, scenarioCtx, fleet, &manifest)
	if t.Failed() {
		return
	}
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
			t.Fatalf("parse scaled %s with matching retained executable: %v\n%s", name, err, output.data)
		}
	}
	for _, window := range manifest.ResourceWindows {
		t.Logf("%s: actors=%d history_lines/actor=%d cpu=%.3f%% elapsed_ns=%d cpu_delta_ns=%d rss_before=%d rss_after=%d physical_write_delta=%d owner=%+v", window.Name, scenario.Actors, scenario.History, window.CPUPercent, window.ElapsedNS, window.CPUTimeNS, window.Before.Usage.RSSBytes, window.After.Usage.RSSBytes, window.DiskWriteBytes, manifest.MeasuredSupervisor)
	}
}

func profileScaleObserveCohort(ctx context.Context, fleet *testcouch.Fleet, count int, agent string, previous *profileScaleCohort) (profileScaleCohort, error) {
	response, err := fleet.Actors(ctx)
	if err != nil {
		return profileScaleCohort{}, err
	}
	if len(response.Actors) != count {
		return profileScaleCohort{}, fmt.Errorf("exact cohort requires%d actors, observed%d", count, len(response.Actors))
	}
	proof := profileScaleCohort{At: time.Now(), Actors: make([]profileScaleIdentity, count)}
	seen := make(map[string]bool, count)
	for _, candidate := range response.Actors {
		binding := candidate.Binding
		if err := binding.Validate(); err != nil {
			return proof, err
		}
		if seen[binding.Slot] || binding.Agent != agent || binding.Repository != filepath.Join(fleet.Workspaces.Primary, ".git") {
			return proof, fmt.Errorf("duplicate or foreign actor in exact cohort: %s", binding.Slot)
		}
		seen[binding.Slot] = true
	}
	for n := 0; n < count; n++ {
		slot := fmt.Sprintf("fixture:%d", n)
		if !seen[slot] {
			return proof, fmt.Errorf("missing exact cohort member%s", slot)
		}
		ready, err := fleet.Agent(n)
		if err != nil {
			return proof, err
		}
		cwd := fleet.Workspaces.Primary
		if n != 0 {
			cwd = fleet.Workspaces.Workers[n]
		}
		if ready.Agent != agent || ready.RunID != fleet.RunID || ready.CWD != cwd {
			return proof, fmt.Errorf("actor%d readiness differs from owned fixture", n)
		}
		var identity profileScaleIdentity
		for _, candidate := range response.Actors {
			if candidate.Binding.Slot == slot {
				identity = profileScaleIdentity{Slot: slot, WrapperPID: candidate.Binding.PID, WrapperGeneration: candidate.Binding.Start, Ready: ready}
				break
			}
		}
		if previous != nil {
			before := previous.Actors[n]
			if identity.WrapperPID != before.WrapperPID || identity.WrapperGeneration != before.WrapperGeneration || identity.Ready.PID != before.Ready.PID || identity.Ready.Generation != before.Ready.Generation || identity.Ready.NativeID != before.Ready.NativeID {
				return proof, fmt.Errorf("actor%d changed wrapper or fixture incarnation across measured lifetime", n)
			}
		}
		proof.Actors[n] = identity
	}
	return proof, nil
}

func profileScaleSample(ctx context.Context, fleet *testcouch.Fleet, want testcouch.Resource) (profileScaleResourceSample, error) {
	sample := profileScaleResourceSample{ObservationStarted: time.Now()}
	owner, err := fleet.SupervisorIdentity(ctx)
	if err != nil || owner != want {
		return sample, fmt.Errorf("owned supervisor changed before resource read: %+v: %v", owner, err)
	}
	sample.Owner = owner
	pid, err := strconv.Atoi(owner.ID)
	if err != nil || pid <= 0 || owner.Generation == "" {
		return sample, fmt.Errorf("invalid exact supervisor identity: %+v", owner)
	}
	reader := procutil.NativeResourceReader{}
	rows, calls, err := reader.Table(ctx)
	sample.TableCalls = calls
	if err != nil {
		return sample, err
	}
	matches := 0
	for _, row := range rows {
		if row.PID == pid {
			matches++
			if row.Identity != owner.Generation || row.Zombie {
				return sample, fmt.Errorf("supervisor process generation unavailable or changed: %+v", row)
			}
			sample.Process = row
		}
	}
	if matches != 1 {
		return sample, fmt.Errorf("expected one live owned supervisor row, observed%d", matches)
	}
	sample.Usage, sample.UsageCalls, err = reader.Usage(ctx, sample.Process)
	sample.ObservedAt = time.Now()
	if err != nil {
		return sample, err
	}
	if current, err := fleet.SupervisorIdentity(ctx); err != nil || current != want {
		return sample, fmt.Errorf("owned supervisor changed after fenced resource read: %+v: %v", current, err)
	}
	return sample, nil
}

func profileScaleMeasure(ctx context.Context, fleet *testcouch.Fleet, owner testcouch.Resource, name string) (profileScaleResourceWindow, error) {
	window := profileScaleResourceWindow{Name: name}
	var err error
	window.Before, err = profileScaleSample(ctx, fleet, owner)
	if err != nil {
		return window, err
	}
	if err := profileScaleWait(ctx, fleet, 30*time.Second); err != nil {
		return window, err
	}
	window.After, err = profileScaleSample(ctx, fleet, owner)
	if err != nil {
		return window, err
	}
	window.ElapsedNS = window.After.ObservedAt.Sub(window.Before.ObservedAt).Nanoseconds()
	if window.ElapsedNS <= 0 || window.After.Usage.CPUTimeNS < window.Before.Usage.CPUTimeNS || window.After.Usage.DiskWriteBytes < window.Before.Usage.DiskWriteBytes {
		return window, fmt.Errorf("resource window has nonpositive elapsed time or regressing cumulative counters")
	}
	window.CPUTimeNS = window.After.Usage.CPUTimeNS - window.Before.Usage.CPUTimeNS
	window.DiskWriteBytes = window.After.Usage.DiskWriteBytes - window.Before.Usage.DiskWriteBytes
	window.CPUPercent = float64(window.CPUTimeNS) / float64(window.ElapsedNS) * 100
	return window, nil
}

func profileScaleWait(ctx context.Context, fleet *testcouch.Fleet, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-fleet.Done():
		return fmt.Errorf("owned supervisor exited during declared observation interval")
	case <-timer.C:
		return nil
	}
}

func profileScaleStop(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, manifest *profileScaleManifest) {
	t.Helper()
	for i := range manifest.Workloads {
		workload := &manifest.Workloads[i]
		if workload.StopAttempted || workload.Start == nil {
			continue
		}
		workload.StopAttempted = true
		observation, err := fleet.ConfigureWorkload(ctx, workload.Actor, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStop})
		workload.StopAcknowledged = time.Now()
		if err != nil {
			workload.StopError = err.Error()
			t.Errorf("stop actor%d workload: %v", workload.Actor, err)
			continue
		}
		workload.Stop = &observation
		if observation.Status != agentfixture.WorkloadStopped || !observation.Started.Equal(workload.Start.Started) || observation.Finished.IsZero() || observation.Ticks == 0 || observation.Bytes == 0 || observation.ScrollbackLines != workload.Request.ScrollbackLines || observation.IntervalMS != 100 {
			t.Errorf("actor%d workload did not prove bounded running-to-stopped output: %+v", workload.Actor, observation)
		}
	}
}

func profileScaleWriteManifest(t *testing.T, destination string, manifest profileScaleManifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil && len(data) > 1<<20 {
		err = fmt.Errorf("scaled manifest exceeds1MiB")
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(destination, "manifest.json"), data, 0600)
	}
	if err != nil {
		t.Errorf("retain scaled manifest: %v", err)
	}
}
