package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func taggedPreviousPeekFixture(t *testing.T) (*Couch, ThreadRecord, ThreadAddress, threadrecord.RecoveryCapture, OSRecoveryEvidenceReader) {
	t.Helper()
	reader, source := recoveryEvidenceFixture(t, false)
	observed, err := reader.Read(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := reader.Preserve(context.Background(), source, observed)
	if err != nil {
		t.Fatal(err)
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
	return &Couch{Threads: store}, record, source.Address, archived.Capture, reader
}

func TestPeekTaggedFreshPreviousAllReadsExactRetainedOldTag(t *testing.T) {
	c, record, old, capture, reader := taggedPreviousPeekFixture(t)
	calls := 0
	c.SlotRetainedCapture = func(ctx context.Context, address ThreadAddress, agent string, selected threadrecord.RecoveryCapture, limit int) (TerminalRecording, error) {
		calls++
		if address != record.Address || address == old || agent != "codex" || selected != capture || limit != 0 {
			t.Fatalf("wrong retained descriptor: %+v %s %+v %d", address, agent, selected, limit)
		}
		return reader.ReadRetainedTerminal(ctx, address, agent, selected, limit)
	}
	c.SlotCapture = func(ThreadAddress, string, string, int) (TerminalRecording, error) {
		t.Fatal("tagged previous used current owner renderer")
		return TerminalRecording{}, nil
	}
	result, err := c.PeekThreadSelection(context.Background(), "pair:2", record.Address, PeekSelection{Previous: true, All: true})
	if err != nil || calls != 1 || !result.Complete || !result.Previous || result.Source != "previous-recording" || result.Raw != capture.Raw || !strings.Contains(strings.Join(result.Lines, "\n"), "previous objective") {
		t.Fatalf("peek=%+v calls=%d err=%v", result, calls, err)
	}
}

func TestPeekTaggedPreviousRejectsUnownedOrMissingCaptureWithoutCurrentFallback(t *testing.T) {
	for _, mode := range []string{"outside-root", "other-scope", "wrong-token", "wrong-events", "missing", "changed", "no-capture"} {
		t.Run(mode, func(t *testing.T) {
			c, record, _, capture, reader := taggedPreviousPeekFixture(t)
			changed := capture
			switch mode {
			case "outside-root":
				changed.Raw = filepath.Join(t.TempDir(), filepath.Base(capture.Raw))
			case "other-scope":
				other, _ := artifactpath.Resolve(artifactpath.Address{DataDir: reader.DataDir, RepoScope: "0123456789abcdef", Tag: "source"})
				p, _ := other.ParkedScrollbackArtifacts(capture.Token)
				changed.Raw, changed.Events = p.Raw, p.Events
			case "wrong-token":
				changed.Token = "20261015T020000"
			case "wrong-events":
				changed.Events = capture.Raw
			case "missing":
				if err := os.Remove(capture.Raw); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(capture.Raw, []byte("replaced capture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := c.Threads.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error {
				r.RecoveryIntent.Orientation.Capture = &changed
				if mode == "no-capture" {
					r.RecoveryIntent.Orientation.Capture = nil
				}
				return nil
			})
			// A malformed structurally persisted member is rejected even before peek.
			if mode == "wrong-events" && err != nil {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c.SlotRetainedCapture = reader.ReadRetainedTerminal
			c.SlotCapture = func(ThreadAddress, string, string, int) (TerminalRecording, error) {
				t.Fatal("unproved previous capture reached reader")
				return TerminalRecording{}, nil
			}
			c.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) {
				t.Fatal("previous capture fell back to current")
				return nil, nil
			}
			result, err := c.PeekThreadSelection(context.Background(), "pair:2", record.Address, PeekSelection{Previous: true, All: true})
			if err != nil || result.Complete || len(result.Lines) != 0 || len(result.Unavailable) == 0 {
				t.Fatalf("unavailable=%+v err=%v", result, err)
			}
		})
	}
}

func TestRetainedPeekLeasesRemainIndependentWithinOneProcess(t *testing.T) {
	_, _, old, capture, reader := taggedPreviousPeekFixture(t)
	paths, err := reader.recoveryPaths(ThreadRecord{Address: old})
	if err != nil {
		t.Fatal(err)
	}
	first, err := retainedPeekLease(context.Background(), paths, capture, "codex")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := retainedPeekLease(context.Background(), paths, capture, "codex")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	state, err := second.Coordinator.ReadOwner(second.Owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, process := range state.Processes {
		if process.ID == second.ID {
			return
		}
	}
	t.Fatal("finishing one peek removed the other reader's capture protection")
}
