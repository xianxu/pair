package couchcore

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestDiagnosticRotationPreservesWitnessedPreviousCapture(t *testing.T) {
	ctx := context.Background()
	reader, source := recoveryEvidenceFixture(t, true)
	observed, err := reader.Read(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve invokes the real launcher ParkScrollbackContext publisher and
	// certifies the copied raw/events pair against its producer witness.
	archived, err := reader.Preserve(ctx, source, observed)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Generation != observed.Generation || archived.Capture.Token == "" {
		t.Fatalf("park lost witnessed producer: %+v", archived)
	}
	paths, err := reader.recoveryPaths(source)
	if err != nil {
		t.Fatal(err)
	}
	live, err := paths.ScrollbackArtifacts("codex")
	if err != nil {
		t.Fatal(err)
	}
	parked, err := paths.ParkedScrollbackArtifacts(archived.Capture.Token)
	if err != nil {
		t.Fatal(err)
	}
	type memberSnapshot struct {
		identity artifactpath.CaptureFileIdentity
		bytes    string
	}
	members := map[string]memberSnapshot{}
	for _, path := range []string{live.Raw, live.Events, parked.Raw, parked.Events, parked.Metadata} {
		id, err := storagegc.CaptureIdentity(path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		members[path] = memberSnapshot{id, string(body)}
	}

	record := source
	record.Address.Tag = "fresh-target"
	record.Incarnations = nil
	record.RecoveryIntent = taggedRecoveryIntent(record, threadrecord.RecoveryFresh)
	record.RecoveryIntent.Agent = "codex"
	record.RecoveryIntent.Orientation.Capture = &archived.Capture
	store, _ := newTestThreadStore(t)
	record, err = store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: store}
	reads := 0
	c.SlotRetainedCapture = func(ctx context.Context, address ThreadAddress, agent string, selected threadrecord.RecoveryCapture, limit int) (TerminalRecording, error) {
		reads++
		if selected != archived.Capture || address != record.Address || agent != "codex" || limit != 0 {
			t.Fatalf("previous selection changed: %+v %s %+v %d", address, agent, selected, limit)
		}
		return reader.ReadRetainedTerminal(ctx, address, agent, selected, limit)
	}
	c.SlotCapture = func(ThreadAddress, string, string, int) (TerminalRecording, error) {
		t.Fatal("previous capture fell back to current capture")
		return TerminalRecording{}, nil
	}
	c.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) {
		t.Fatal("previous capture fell back to current terminal")
		return nil, nil
	}
	peek := func() PeekResult {
		t.Helper()
		result, err := c.PeekThreadSelection(ctx, "pair:2", record.Address, PeekSelection{Previous: true, All: true})
		if err != nil || !result.Complete || !result.Previous || result.Source != "previous-recording" || result.Raw != archived.Capture.Raw || result.Events != archived.Capture.Events || !strings.Contains(strings.Join(result.Lines, "\n"), "previous objective") {
			t.Fatalf("previous peek=%+v err=%v", result, err)
		}
		return result
	}
	before := peek()
	lease, err := retainedPeekLease(ctx, paths, archived.Capture, "codex")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()

	// This separate artifact family shares the same owner/root. Repeated size
	// rotation must preserve all diagnostic history and both capture pairs.
	logPath := paths.WrapEvents()
	logNow := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	opts := diagnosticlog.Options{
		Root: reader.DataDir, MaxBytes: 4, MaxGenerations: 2,
		Now:                    func() time.Time { return logNow },
		Proof:                  func(context.Context, string, []diagnosticlog.Registration) error { return nil },
		SynchronousMaintenance: true,
	}
	writer, err := diagnosticlog.Open(logPath, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	for n := 0; n < 8; n++ {
		logNow = logNow.Add(time.Second)
		if written, err := writer.Write([]byte(fmt.Sprintf("%04d", n))); err != nil || written != 4 {
			t.Fatalf("diagnostic write %d: %d %v", n, written, err)
		}
	}
	segments, _, complete, err := diagnosticlog.PreviewLegacyPage(logPath, opts, "", 10)
	if err != nil || !complete || len(segments) != 8 {
		t.Fatalf("preserved diagnostic generations=%+v complete=%t err=%v", segments, complete, err)
	}
	retained := map[string]bool{}
	for _, segment := range segments {
		body, err := os.ReadFile(segment.Path)
		if err != nil || len(body) != 4 || retained[string(body)] {
			t.Fatalf("diagnostic retention kept wrong bytes: %q %v", body, err)
		}
		retained[string(body)] = true
	}
	for n := 0; n < 8; n++ {
		if !retained[fmt.Sprintf("%04d", n)] {
			t.Fatalf("lost diagnostic record %04d", n)
		}
	}
	for path, original := range members {
		id, err := storagegc.CaptureIdentity(path)
		if err != nil || id != original.identity {
			t.Fatalf("diagnostic rotation changed capture identity %s: %+v %v", path, id, err)
		}
		body, err := os.ReadFile(path)
		if err != nil || string(body) != original.bytes {
			t.Fatalf("diagnostic rotation changed capture bytes %s: %q %v", path, body, err)
		}
	}
	after := peek()
	if reads != 2 || strings.Join(after.Lines, "\n") != strings.Join(before.Lines, "\n") {
		t.Fatalf("retained replay changed: before=%+v after=%+v reads=%d", before, after, reads)
	}
	state, err := lease.Coordinator.ReadOwner(lease.Owner)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, process := range state.Processes {
		if process.ID == lease.ID && process.Target == parked.Raw {
			found = true
		}
	}
	if !found {
		t.Fatal("retained reader/diagnostic rotation removed independent peek lease")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	state, err = lease.Coordinator.ReadOwner(lease.Owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, process := range state.Processes {
		if process.ID == lease.ID {
			t.Fatal("completed peek retained its lease")
		}
	}
}
