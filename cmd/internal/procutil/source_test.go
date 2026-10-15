package procutil

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

type sourceTestReader struct{ *snapshotWorld }

func (r sourceTestReader) Table() ([]Process, error)            { return r.table() }
func (r sourceTestReader) Command(pid int) (CommandInfo, error) { return r.command(pid) }
func (r sourceTestReader) Identity(pid int) (string, error)     { return r.identity(pid) }

func TestSourceSharesReaderAndPreservesSnapshotEvidence(t *testing.T) {
	w := &snapshotWorld{rows: []Process{{PID: 1, Name: "zellij", Identity: "old"}, {PID: 2, Name: "other", Identity: "stable"}}, commands: map[int]CommandInfo{1: {Args: []string{"zellij"}}, 2: {Args: []string{"other"}}}, identities: map[int]string{1: "replacement", 2: "stable"}}
	source := NewSource(sourceTestReader{w})
	copy := source
	rows, err := source.Table()
	if err != nil || len(rows) != 2 {
		t.Fatalf("table=%v %v", rows, err)
	}
	named, err := copy.SnapshotNamed(t.Context(), "zellij")
	if err != nil || len(named) != 0 {
		t.Fatalf("reused PID survived named snapshot: %v %v", named, err)
	}
	full, err := source.Snapshot(t.Context())
	if err != nil || len(full) != 1 || full[0].PID != 2 || w.tableCalls != 3 || !reflect.DeepEqual(w.calls, []int{1, 1, 2}) {
		t.Fatalf("full=%v err=%v tables=%d commands=%v", full, err, w.tableCalls, w.calls)
	}
}

func TestSourceCancellationAndErrorsNeverFallBackToHost(t *testing.T) {
	w := &snapshotWorld{err: os.ErrPermission}
	source := NewSource(sourceTestReader{w})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := source.Snapshot(ctx); !errors.Is(err, context.Canceled) || w.tableCalls != 0 {
		t.Fatalf("cancellation=%v tables=%d", err, w.tableCalls)
	}
	if _, err := source.SnapshotNamed(t.Context(), "zellij"); !errors.Is(err, os.ErrPermission) || w.tableCalls != 1 {
		t.Fatalf("failure=%v tables=%d", err, w.tableCalls)
	}
}
