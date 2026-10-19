package diagnosticlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type leaseProcessWorld struct {
	Processes map[int]string
	Unknown   map[int]bool
}

func (w *leaseProcessWorld) Inspect(p storagegc.ProcessIdentity) storagegc.Liveness {
	if w.Unknown[p.PID] {
		return storagegc.ProcessUnknown
	}
	if w.Processes[p.PID] == p.Birth {
		return storagegc.ProcessAlive
	}
	return storagegc.ProcessDead
}

func supervisorFixture(t *testing.T) (*storagegc.Coordinator, *fakeInspection, storagegc.ProcessIdentity) {
	t.Helper()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	identity := storagegc.ProcessIdentity{PID: 123, Birth: "darwin:1234.5"}
	c.Probe = &leaseProcessWorld{Processes: map[int]string{123: identity.Birth}}
	if err := c.RegisterRuntime(context.Background(), identity, storagegc.CouchRuntimeRole, storagegc.RuntimeRegistrationOptions{DiagnosticProtocol: storagegc.DiagnosticProtocolVersion}); err != nil {
		t.Fatal(err)
	}
	return c, &fakeInspection{runtimes: []Runtime{{Process: Registration{PID: identity.PID, Birth: identity.Birth}}}}, identity
}

func TestMatchesRuntimeRegistrationExactCapability(t *testing.T) {
	identity := storagegc.ProcessIdentity{PID: 123, Birth: "darwin:1234.5"}
	entry := storagegc.ProcessRegistration{ID: "lease", Process: identity, Role: storagegc.CouchRuntimeRole, DiagnosticProtocol: storagegc.DiagnosticProtocolVersion}
	process := Registration{PID: identity.PID, Birth: identity.Birth}
	if !matchesRuntimeRegistration(entry, process) {
		t.Fatal("valid capability refused")
	}
	for _, mutate := range []func(*storagegc.ProcessRegistration){
		func(e *storagegc.ProcessRegistration) { e.DiagnosticProtocol = 0 },
		func(e *storagegc.ProcessRegistration) { e.DiagnosticProtocol = 2 },
		func(e *storagegc.ProcessRegistration) { e.Role = "wrapper" },
		func(e *storagegc.ProcessRegistration) { e.Process.PID++ },
		func(e *storagegc.ProcessRegistration) { e.Process.Birth = "darwin:1234.6" },
		func(e *storagegc.ProcessRegistration) { e.Process.Birth = "" },
	} {
		e := entry
		mutate(&e)
		if matchesRuntimeRegistration(e, process) {
			t.Fatalf("bad lease qualified: %+v", e)
		}
	}
}

func TestRuntimeRegistryInspectionRefreshesAndFencesIdentity(t *testing.T) {
	c, native, identity := supervisorFixture(t)
	inspector := WithRuntimeRegistry(c, native)
	proof := InspectionProof(inspector)
	assert := func(want bool) {
		t.Helper()
		err := proof(context.Background(), "/owned/trace", nil)
		if (err == nil) != want {
			t.Fatalf("accepted=%t err=%v", want, err)
		}
	}
	assert(true)
	if native.runtimes[0].Protocol {
		t.Fatal("lease recognition mutated native observation")
	}
	native.holders = []Registration{{PID: identity.PID, Birth: identity.Birth}}
	assert(false)
	native.holders = nil
	registry := filepath.Join(c.Root, ".retention", "runtimes.json")
	saved, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(registry); err != nil {
		t.Fatal(err)
	}
	assert(false)
	if err := os.WriteFile(registry, saved, 0600); err != nil {
		t.Fatal(err)
	}
	assert(true)
	c.Probe.(*leaseProcessWorld).Processes[identity.PID] = "darwin:1234.6"
	assert(false)
	c.Probe.(*leaseProcessWorld).Processes[identity.PID] = identity.Birth
	c.Probe.(*leaseProcessWorld).Unknown = map[int]bool{identity.PID: true}
	assert(false)
	other, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyWriters(context.Background(), "/owned/trace", nil, WithRuntimeRegistry(other, native)); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("wrong root accepted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := proof(ctx, "/owned/trace", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRuntimeRegistryUnsafeMetadataFailsClosedWithoutBlockingReaders(t *testing.T) {
	for _, kind := range []string{"malformed", "symlink", "directory", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			c, native, _ := supervisorFixture(t)
			registry := filepath.Join(c.Root, ".retention", "runtimes.json")
			if err := os.Remove(registry); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "malformed":
				if err := os.WriteFile(registry, []byte(`{"version":1,"processes":null}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), registry); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(registry, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.WriteFile(registry, make([]byte, (1<<20)+1), 0600); err != nil {
					t.Fatal(err)
				}
			}
			inspector := WithRuntimeRegistry(c, native)
			ctx := SweepContext(context.Background(), []string{"/owned/trace"})
			if err := InspectionProof(inspector)(ctx, "/owned/trace", nil); !errors.Is(err, ErrInspectionUnavailable) {
				t.Fatalf("unsafe metadata accepted: %v", err)
			}
			// Closed-generation proof needs only the complete holder inventory.
			if err := InspectionReaderProof(inspector)(ctx, "/owned/trace"); err != nil {
				t.Fatalf("unheld immutable generation blocked: %v", err)
			}
		})
	}
}

type countedLeaseProbe struct {
	calls  int
	state  storagegc.Liveness
	cancel context.CancelFunc
}

func (p *countedLeaseProbe) Inspect(storagegc.ProcessIdentity) storagegc.Liveness {
	p.calls++
	if p.cancel != nil {
		p.cancel()
	}
	return p.state
}

type countedRuntimeWorld struct {
	*processWorld
	snapshots int
}

func (w *countedRuntimeWorld) Snapshot(ctx context.Context) ([]procutil.Observation, error) {
	w.snapshots++
	return w.processWorld.Snapshot(ctx)
}

func TestRuntimeRegistrySweepCountIsConstant(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			c, _, identity := supervisorFixture(t)
			world := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: identity.PID, Name: "couch", Identity: "observed"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/couch"}}}, generation: "observed", strict: identity.Birth, env: map[string]string{}}
			commands := 0
			countedWorld := &countedRuntimeWorld{processWorld: world}
			native := OSInspection{processes: countedWorld, openFileCommand: func(context.Context, ...string) ([]byte, []byte, error) { commands++; return nil, nil, nil }}
			reads := 0
			probe := &countedLeaseProbe{state: storagegc.ProcessAlive}
			inspector := &runtimeInspection{native: native, read: func() ([]storagegc.ProcessRegistration, error) { reads++; return c.ReadRuntimes() }, probe: probe}
			paths := make([]string, n)
			for i := range paths {
				paths[i] = filepath.Join(c.Root, fmt.Sprintf("trace-%d", i))
				if err := os.WriteFile(paths[i], nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			proof := InspectionProof(inspector)
			ctx := SweepContext(context.Background(), paths)
			for _, path := range paths {
				for range 2 {
					if err := proof(ctx, path, nil); err != nil {
						t.Fatal(err)
					}
				}
			}
			if reads != 1 || commands != 1 || world.envCalls != 1 || countedWorld.snapshots != 1 || probe.calls != 1 {
				t.Fatalf("N=%d reads=%d file snapshots=%d process snapshots=%d probes=%d", n, reads, commands, world.envCalls, probe.calls)
			}
			if err := os.Remove(filepath.Join(c.Root, ".retention", "runtimes.json")); err != nil {
				t.Fatal(err)
			}
			if err := proof(SweepContext(context.Background(), paths), paths[0], nil); !errors.Is(err, ErrUnknownWriters) {
				t.Fatalf("new sweep retained stale lease: %v", err)
			}
			if reads != 2 || commands != 2 || world.envCalls != 2 || countedWorld.snapshots != 2 || probe.calls != 1 {
				t.Fatal("sweep evidence escaped its lifetime")
			}
		})
	}
}

func TestRuntimeRegistryRechecksAfterReadAndCancellation(t *testing.T) {
	c, native, _ := supervisorFixture(t)
	for _, state := range []storagegc.Liveness{storagegc.ProcessDead, storagegc.ProcessUnknown, storagegc.ProcessAlive} {
		ctx, cancel := context.WithCancel(context.Background())
		probe := &countedLeaseProbe{state: storagegc.ProcessAlive}
		if state == storagegc.ProcessAlive {
			probe.cancel = cancel
		}
		inspector := &runtimeInspection{native: native, probe: probe, read: func() ([]storagegc.ProcessRegistration, error) {
			entries, err := c.ReadRuntimes()
			probe.state = state // recycle or lose identity after the metadata observation
			return entries, err
		}}
		err := InspectionProof(inspector)(ctx, "/trace", nil)
		if err == nil {
			t.Fatalf("identity/cancellation fence bypassed for %s", state)
		}
		cancel()
	}
}

func TestRuntimeRegistryCapabilityFuturePreservedWithoutAuthority(t *testing.T) {
	c, native, _ := supervisorFixture(t)
	registry := filepath.Join(c.Root, ".retention", "runtimes.json")
	raw, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["processes"].([]any)[0].(map[string]any)["diagnostic_protocol"] = 2
	raw, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := InspectionProof(WithRuntimeRegistry(c, native))(context.Background(), "/trace", nil); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("future capability accepted: %v", err)
	}
}

func TestRuntimeRegistryOrdinaryWriterMaintenance(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	identity, err := storagegc.CurrentProcessIdentity(child.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterRuntime(context.Background(), identity, storagegc.CouchRuntimeRole, storagegc.RuntimeRegistrationOptions{DiagnosticProtocol: storagegc.DiagnosticProtocolVersion}); err != nil {
		t.Fatal(err)
	}
	native := &fakeInspection{runtimes: []Runtime{{Process: Registration{PID: identity.PID, Birth: identity.Birth}}}}
	path, clock, defaults := fixture(t)
	opts := environmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return c.Root
		}
		return ""
	}, native)
	opts.Now = defaults.Now
	opts.SynchronousMaintenance = true
	writer, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(8 * 24 * time.Hour)
	registry := filepath.Join(c.Root, ".retention", "runtimes.json")
	saved, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(registry); err != nil {
		t.Fatal(err)
	}
	if err := Maintain(path, opts); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("unregistered supervisor accepted: %v", err)
	}
	if err := os.WriteFile(registry, saved, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Maintain(path, opts); err != nil {
		t.Fatalf("registered supervisor blocked maintenance: %v", err)
	}
}
