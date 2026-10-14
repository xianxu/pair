package wrapcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestAutomaticInputReporterWritesEvidenceAndOneRPC(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "wrap.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	p := &proxy{agentBasename: "codex", wrapEventsFD: file}
	b := peerTestMessage(time.Now()).To
	b.Repository = "/repo/.git"
	b.Scope = "scope"
	b.Tag = "tag"
	b.Session = "session"
	b.Nonce = "nonce"
	b.Agent = "codex"
	b.Version = "test"
	b.PID = os.Getpid()
	b.Start = "fixture"
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan couchmessage.Request, 2)
	server, err := couchmessage.StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var r couchmessage.Request
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		requests <- r
		return json.Marshal(couchmessage.Response{Code: "uncertain", Error: "write may have landed"})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	queue := make(chan automaticInputEvidence, 1)
	done := make(chan struct{})
	go func() { defer close(done); runAutomaticInputReporter(ctx, p, b, dir, queue) }()
	report := couchmessage.AutomaticInputStall{Sequence: 1, Cause: couchmessage.AutomaticInputStallNoRender, WaitMillis: 30000, Fence: 7, Epoch: 7, Released: true}
	queue <- captureAutomaticInputEvidence(report, peerSnapshot(t, codexStatusPaint()))
	select {
	case r := <-requests:
		if r.ID != couchmessage.AutomaticInputStallID(b, 1) || couchmessage.ValidateRequest(r) != nil {
			t.Fatalf("bad request %+v", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no RPC")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		raw, _ := os.ReadFile(file.Name())
		if strings.Contains(string(raw), "write may have landed") {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("uncertain outcome not recorded")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("reporter not joined")
	}
	if len(requests) != 0 {
		t.Fatal("uncertain RPC was retried")
	}
	raw, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "automatic-input-stall") || !strings.Contains(string(raw), "Ask Codex") || !strings.Contains(string(raw), "write may have landed") {
		t.Fatalf("missing diagnostic: %s", raw)
	}
}
func TestAutomaticInputEvidenceBounded(t *testing.T) {
	s := peerSnapshot(t, "\x1b[10;1H\x1b[1m›\x1b[22m "+strings.Repeat("x", 110)+"\x1b[?25h\x1b[10;3H")
	for i := range s.Cells {
		s.Cells[i].Content = strings.Repeat("x", 256) + "\x1b"
	}
	e := captureAutomaticInputEvidence(couchmessage.AutomaticInputStall{}, s)
	n := 0
	for _, line := range e.Screen {
		n += len(line)
		if strings.ContainsRune(line, '\x1b') {
			t.Fatal("terminal control leaked")
		}
	}
	if n > 16384 || len(e.Screen) > 40 || !e.Truncated {
		t.Fatalf("unbounded evidence %d bytes, %d rows", n, len(e.Screen))
	}
}

func TestAutomaticInputReporterDoesNotBlockInputAndJoins(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "wrap.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	p := &proxy{agentBasename: "codex", wrapEventsFD: file}
	b := peerTestMessage(time.Now()).To
	b.Repository = "/repo/.git"
	b.Scope = "scope"
	b.Tag = "tag"
	b.Session = "session"
	b.Nonce = "nonce"
	b.Agent = "codex"
	b.Version = "test"
	b.PID = os.Getpid()
	b.Start = "fixture"
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		close(entered)
		select {
		case <-ctx.Done():
		case <-release:
		}
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	p.startAutomaticInputReporter(ctx, b, dir, &workers)
	report := couchmessage.AutomaticInputStall{Sequence: 1, Cause: couchmessage.AutomaticInputStallNoRender, WaitMillis: 30000, Fence: 7, Epoch: 7, Released: true}
	snapshot := peerSnapshot(t, codexStatusPaint())
	p.automaticReport(report, snapshot)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("no pending RPC")
	}
	handed := make(chan struct{})
	go func() {
		report.Sequence = 2
		p.automaticReport(report, snapshot)
		report.Sequence = 3
		p.automaticReport(report, snapshot)
		close(handed)
	}()
	select {
	case <-handed:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("report IO blocked input owner")
	}
	cancel()
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("report worker outlived cancellation")
	}
	raw, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "report_cancelled") || !strings.Contains(string(raw), "report queue full") {
		t.Fatalf("missing teardown/overflow evidence: %s", raw)
	}
}

func TestAutomaticInputReporterRPCDeadline(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "wrap.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	p := &proxy{agentBasename: "codex", wrapEventsFD: file}
	b := peerTestMessage(time.Now()).To
	b.Repository = "/repo/.git"
	b.Scope = "scope"
	b.Tag = "tag"
	b.Session = "session"
	b.Nonce = "nonce"
	b.Agent = "codex"
	b.Version = "test"
	b.PID = os.Getpid()
	b.Start = "fixture"
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		entered <- struct{}{}
		select {
		case <-ctx.Done():
		case <-release:
		}
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var workers sync.WaitGroup
	p.startAutomaticInputReporter(ctx, b, dir, &workers)
	p.automaticReport(couchmessage.AutomaticInputStall{Sequence: 1, Cause: couchmessage.AutomaticInputStallNoRender, WaitMillis: 30000}, peerSnapshot(t, codexStatusPaint()))
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no RPC")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		raw, _ := os.ReadFile(file.Name())
		if strings.Contains(string(raw), "automatic-input-report-failed") {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("RPC exceeded its bounded attempt")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(entered) != 0 {
		t.Fatal("timed-out RPC retried")
	}
	cancel()
	workers.Wait()
}
