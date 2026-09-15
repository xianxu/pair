package diagnosticlog

import (
	"bufio"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
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
	if _, e := exec.LookPath("lsof"); e != nil {
		t.Skip("lsof unavailable")
	}
	path := filepath.Join(t.TempDir(), "trace")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	holders, e := (OSInspection{}).OpenFiles(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, p := range holders {
		if p.PID == os.Getpid() {
			found = p.Birth == procutil.StrictIdentity(strconv.Itoa(os.Getpid()))
		}
	}
	if !found {
		t.Fatalf("current open holder absent %+v", holders)
	}
	f.Close()
	holders, e = (OSInspection{}).OpenFiles(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range holders {
		if p.PID == os.Getpid() {
			t.Fatal("closed fd remains in inspection")
		}
	}
}

func TestUnregisteredChildOpenFileBlocksLegacyExpiry(t *testing.T) {
	if _, e := exec.LookPath("lsof"); e != nil {
		t.Skip("lsof unavailable")
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
	case <-time.After(5 * time.Second):
		t.Fatal("reader not ready")
	}
	self := Registration{PID: os.Getpid(), Birth: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	if e = VerifyWriters(context.Background(), path, []Registration{self}, OSInspection{}); !errors.Is(e, ErrUnknownWriters) {
		t.Fatalf("legacy child holder proof %v", e)
	}
	input.Write([]byte("done\n"))
	input.Close()
	if e = child.Wait(); e != nil {
		t.Fatal(e)
	}
	holders, e := (OSInspection{}).OpenFiles(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range holders {
		if p.PID == child.Process.Pid {
			t.Fatal("exited child still reported live")
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
	*now = now.Add(8 * 24 * time.Hour)
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
func TestOSInspectionCommandHonorsMaintenanceDeadline(t *testing.T) {
	for _, name := range []string{"lsof", "ps"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "trace")
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			start := time.Now()
			var err error
			if name == "lsof" {
				_, err = (OSInspection{}).OpenFiles(ctx, path)
			} else {
				_, err = (OSInspection{}).Runtimes(ctx)
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline error=%v", err)
			}
			if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
				t.Fatalf("inspection ignored worker deadline: %s", elapsed)
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
