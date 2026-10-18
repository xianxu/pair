package diagnosticlog

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestWriterProofRejectsLegacyAndUnknownHolders(t *testing.T) {
	me := Registration{PID: 123, Birth: "birth", Root: "/root"}
	for _, tt := range []struct {
		name     string
		holders  []Registration
		runtimes []Runtime
		err      error
		want     bool
	}{
		{"registered", []Registration{me}, []Runtime{{Process: me}}, nil, true},
		{"unknown holder", []Registration{{PID: 9, Birth: "old"}}, nil, nil, false},
		{"legacy transient", nil, []Runtime{{Process: Registration{PID: 8, Birth: "old", Root: "/root"}}}, nil, false},
		{"other root legacy", nil, []Runtime{{Process: Registration{PID: 8, Birth: "old", Root: "/other"}}}, nil, true},
		{"upgraded transient", nil, []Runtime{{Process: Registration{PID: 8, Birth: "new", Root: "/root"}, Protocol: true}}, nil, true},
		{"inspection denied", nil, nil, errors.New("denied"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			probe := &fakeInspection{tt.holders, tt.runtimes, tt.err}
			e := VerifyWriters(context.Background(), "/trace", []Registration{me}, probe)
			if (e == nil) != tt.want {
				t.Fatalf("proof=%v", e)
			}
		})
	}
}

type fakeInspection struct {
	holders  []Registration
	runtimes []Runtime
	err      error
}

func (p *fakeInspection) OpenFiles(context.Context, string) ([]Registration, error) {
	return p.holders, p.err
}
func (p *fakeInspection) Runtimes(context.Context) ([]Runtime, error) { return p.runtimes, p.err }

func TestOpenFileInspectionConformance(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(testwait.HostState())
		}
	})
	if _, e := exec.LookPath("lsof"); e != nil {
		requireLsofObservation(t, errors.New("lsof unavailable or incomplete"))
	}
	path := filepath.Join(t.TempDir(), "trace")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	holders, e := (OSInspection{}).OpenFiles(context.Background(), path)
	requireLsofObservation(t, e)
	assertLsofHolder(t, holders, os.Getpid(), procutil.StrictIdentity(strconv.Itoa(os.Getpid())))
	f.Close()
	holders, e = (OSInspection{}).OpenFiles(context.Background(), path)
	requireLsofObservation(t, e)
	for _, p := range holders {
		if p.PID == os.Getpid() {
			t.Fatalf("closed fd remains in inspection; %s", testwait.HostState())
		}
	}
}

func TestUnregisteredChildOpenFileBlocksLegacyExpiry(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(testwait.HostState())
		}
	})
	if _, e := exec.LookPath("lsof"); e != nil {
		requireLsofObservation(t, errors.New("lsof unavailable or incomplete"))
	}
	path := filepath.Join(t.TempDir(), "legacy-trace")
	if e := os.WriteFile(path, []byte("old"), 0600); e != nil {
		t.Fatal(e)
	}
	child := exec.Command("sh", "-c", `exec 3<"$1"; printf 'ready\n'; read done`, "trace-reader", path)
	input, e := child.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	output, e := child.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = child.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	ready := make(chan error, 1)
	go func() { _, e := bufio.NewReader(output).ReadString('\n'); ready <- e }()
	select {
	case e := <-ready:
		if e != nil {
			t.Fatal(e)
		}
	case <-testwait.After(t):
		t.Fatalf("reader not ready; %s", testwait.HostState())
	}
	self := Registration{PID: os.Getpid(), Birth: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	holders, e := (OSInspection{}).OpenFiles(context.Background(), path)
	requireLsofObservation(t, e)
	assertLsofHolder(t, holders, child.Process.Pid, procutil.StrictIdentity(strconv.Itoa(child.Process.Pid)))
	// Only the test-owned file needs native observation. Runtime proof semantics
	// are covered by the stateful fake, without scanning unrelated processes.
	if e = VerifyWriters(context.Background(), path, []Registration{self}, &fakeInspection{holders: holders}); !errors.Is(e, ErrUnknownWriters) {
		t.Fatalf("legacy child holder proof %v; %s", e, testwait.HostState())
	}
	input.Write([]byte("done\n"))
	input.Close()
	if e = child.Wait(); e != nil {
		t.Fatal(e)
	}
	holders, e = (OSInspection{}).OpenFiles(context.Background(), path)
	requireLsofObservation(t, e)
	for _, p := range holders {
		if p.PID == child.Process.Pid {
			t.Fatalf("exited child still reported live; %s", testwait.HostState())
		}
	}
}

// An unanswered native probe is not proof that the implementation is wrong.
// A complete answer that violates the contract is always a failure.
func requireLsofObservation(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	unavailable := err.Error() == "lsof unavailable or incomplete" || err.Error() == "lsof reported incomplete inspection"
	if unavailable && os.Getenv("PAIR_REQUIRE_LSOF_CONFORMANCE") != "1" {
		t.Skipf("native lsof conformance unavailable/incomplete: %v; run PAIR_REQUIRE_LSOF_CONFORMANCE=1 to require it; %s", err, testwait.HostState())
	}
	t.Fatalf("native lsof conformance failed: %v; %s", err, testwait.HostState())
}

func assertLsofHolder(t *testing.T, holders []Registration, pid int, birth string) {
	t.Helper()
	for _, holder := range holders {
		if holder.PID == pid && birth != "" && holder.Birth == birth {
			return
		}
	}
	t.Fatalf("open holder pid=%d identity=%q absent or wrong: %+v; %s", pid, birth, holders, testwait.HostState())
}

func TestLsofConformanceVerdicts(t *testing.T) {
	if fixture := os.Getenv("PAIR_TEST_LSOF_VERDICT"); fixture != "" {
		switch fixture {
		case "unavailable":
			requireLsofObservation(t, errors.New("lsof unavailable or incomplete"))
		case "incomplete":
			requireLsofObservation(t, errors.New("lsof reported incomplete inspection"))
		case "malformed":
			requireLsofObservation(t, errors.New("unexpected lsof output"))
		case "unknown":
			requireLsofObservation(t, ErrUnknownWriters)
		case "wrong-holder":
			assertLsofHolder(t, []Registration{{PID: 2, Birth: "expected"}}, 1, "expected")
		case "wrong-identity":
			assertLsofHolder(t, []Registration{{PID: 1, Birth: "wrong"}}, 1, "expected")
		case "correct":
			requireLsofObservation(t, nil)
			assertLsofHolder(t, []Registration{{PID: 1, Birth: "expected"}}, 1, "expected")
		default:
			t.Fatal("unknown fixture", fixture)
		}
		return
	}
	for _, required := range []string{"0", "1"} {
		for _, fixture := range []string{"unavailable", "incomplete", "malformed", "unknown", "wrong-holder", "wrong-identity", "correct"} {
			t.Run(fixture+"/required="+required, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), testwait.Budget(t))
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLsofConformanceVerdicts$", "-test.v")
				cmd.Env = append(os.Environ(), "PAIR_TEST_LSOF_VERDICT="+fixture, "PAIR_REQUIRE_LSOF_CONFORMANCE="+required)
				out, err := cmd.CombinedOutput()
				skip := (fixture == "unavailable" || fixture == "incomplete") && required == "0"
				wantPass := skip || fixture == "correct"
				if (err == nil) != wantPass || strings.Contains(string(out), "--- SKIP:") != skip {
					t.Fatalf("verdict fixture=%s required=%s err=%v output=%s; %s", fixture, required, err, out, testwait.HostState())
				}
			})
		}
	}
}

type cancelingInspection struct {
	cancel       context.CancelFunc
	runtimeCalls int
}

func (p *cancelingInspection) OpenFiles(context.Context, string) ([]Registration, error) {
	p.cancel()
	return nil, nil
}
func (p *cancelingInspection) Runtimes(context.Context) ([]Runtime, error) {
	p.runtimeCalls++
	return nil, nil
}
func TestCanceledProofDoesNotStartNextInspection(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("keep")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(RetentionPeriod + 24*time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &cancelingInspection{cancel: cancel}
	opts.Context = ctx
	opts.Proof = func(ctx context.Context, path string, regs []Registration) error {
		return VerifyWriters(ctx, path, regs, probe)
	}
	_, err = Collect(path, opts, 100)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if probe.runtimeCalls != 0 {
		t.Fatalf("started %d runtime inspections after cancel", probe.runtimeCalls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cancellation deleted payload: %v", err)
	}
}

// cuedInspectionDeadline expires only after the real command proves readiness.
type cuedInspectionDeadline struct{ context.Context }

func (c cuedInspectionDeadline) Err() error {
	if c.Context.Err() != nil {
		return context.DeadlineExceeded
	}
	return nil
}

func TestOSInspectionCommandHonorsMaintenanceDeadline(t *testing.T) {
	for _, name := range []string{"lsof"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "trace")
			ready := filepath.Join(dir, "ready")
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nprintf ready > \"$PAIR_TEST_INSPECTION_READY\"\nexec /bin/sleep 10\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			t.Setenv("PAIR_TEST_INSPECTION_READY", ready)
			parent, expire := context.WithCancel(t.Context())
			defer expire()
			ctx := cuedInspectionDeadline{parent}
			done := make(chan error, 1)
			go func() { _, err := (OSInspection{}).OpenFiles(ctx, path); done <- err }()
			testwait.Until(t, "inspection command readiness", func() bool { _, err := os.Stat(ready); return err == nil })
			expire()
			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("deadline error=%v", err)
				}
			case <-testwait.After(t):
				t.Fatal("inspection command did not finish after maintenance deadline")
			}
		})
	}
}
func TestCanceledLegacyPreviewStopsAfterProof(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.log")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := Options{Context: ctx, Proof: func(context.Context, string, []Registration) error { cancel(); return nil }, Now: func() time.Time { t.Fatal("preview continued after canceled inspection"); return time.Time{} }}
	if _, _, _, err := PreviewLegacyPage(path, options, "", 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("preview cancellation=%v", err)
	}
}
