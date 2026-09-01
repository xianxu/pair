package sessionwatch

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func testLifecycleRecord() LifecycleRecord {
	return LifecycleRecord{
		Version: 1, Agent: "codex", LaunchOrdinal: 7, ArtifactGeneration: "dev:1",
		Source: "transcript", Outcome: "started", TurnID: "turn-1",
		TranscriptPath: "/rollout.jsonl", TranscriptOffset: 42,
		EventTimestamp: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestAppendLifecycleRecordCommitsOneNewlineTerminatedRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lifecycle.jsonl")
	runtime := sessionledger.OSRuntime{}
	if err := AppendLifecycleRecord(runtime, path, testLifecycleRecord()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(raw, []byte{'\n'}) != 1 || raw[len(raw)-1] != '\n' {
		t.Fatalf("journal = %q, want one committed line", raw)
	}
}

func TestAppendLifecycleRecordLoopsShortWrites(t *testing.T) {
	runtime := newLifecycleMemoryRuntime()
	runtime.maxWrite = 3
	if err := AppendLifecycleRecord(runtime, "/journal", testLifecycleRecord()); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(runtime.data, []byte{'\n'}) || runtime.writes < 2 {
		t.Fatalf("data=%q writes=%d", runtime.data, runtime.writes)
	}
}

func TestAppendLifecycleRecordReconcilesIndeterminateStableIdentity(t *testing.T) {
	runtime := newLifecycleMemoryRuntime()
	runtime.syncErrOnce = errors.New("sync uncertain")
	if err := AppendLifecycleRecord(runtime, "/journal", testLifecycleRecord()); err != nil {
		t.Fatalf("stable record did not reconcile: %v", err)
	}
	if bytes.Count(runtime.data, []byte{'\n'}) != 1 {
		t.Fatalf("reconciliation duplicated record: %q", runtime.data)
	}
}

func TestAppendLifecycleRecordReconcilesIndeterminateDirectorySync(t *testing.T) {
	runtime := newLifecycleMemoryRuntime()
	runtime.dirSyncErrOnce = errors.New("directory sync uncertain")
	if err := AppendLifecycleRecord(runtime, "/journal", testLifecycleRecord()); err != nil {
		t.Fatalf("stable record did not reconcile directory sync: %v", err)
	}
	if bytes.Count(runtime.data, []byte{'\n'}) != 1 {
		t.Fatalf("reconciliation duplicated record: %q", runtime.data)
	}
}

func TestAppendLifecycleRecordRetrySeparatesUncommittedPartialRow(t *testing.T) {
	runtime := newLifecycleMemoryRuntime()
	runtime.writeErrOnce = errors.New("interrupted write")
	runtime.errorWriteBytes = 5
	if err := AppendLifecycleRecord(runtime, "/journal", testLifecycleRecord()); err == nil {
		t.Fatal("partial append unexpectedly succeeded")
	}
	if err := AppendLifecycleRecord(runtime, "/journal", testLifecycleRecord()); err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSuffix(runtime.data, []byte{'\n'}), []byte{'\n'})
	if len(lines) != 2 || !bytes.Contains(lines[1], []byte(`"transcript_record_offset":42`)) {
		t.Fatalf("retry did not establish a committed record boundary: %q", runtime.data)
	}
}

type lifecycleMemoryRuntime struct {
	data            []byte
	maxWrite        int
	writes          int
	syncErrOnce     error
	dirSyncErrOnce  error
	writeErrOnce    error
	errorWriteBytes int
}

func newLifecycleMemoryRuntime() *lifecycleMemoryRuntime             { return &lifecycleMemoryRuntime{} }
func (r *lifecycleMemoryRuntime) MkdirAll(string, os.FileMode) error { return nil }
func (r *lifecycleMemoryRuntime) Lock(string) (sessionledger.Unlocker, error) {
	return lifecycleNopCloser{}, nil
}
func (r *lifecycleMemoryRuntime) ReadFile(string) ([]byte, error) {
	if r.data == nil {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), r.data...), nil
}
func (r *lifecycleMemoryRuntime) OpenAppend(string, os.FileMode) (sessionledger.AppendFile, error) {
	return &lifecycleMemoryFile{runtime: r}, nil
}
func (r *lifecycleMemoryRuntime) SyncDirectory(string) error {
	err := r.dirSyncErrOnce
	r.dirSyncErrOnce = nil
	return err
}

type lifecycleNopCloser struct{}

func (lifecycleNopCloser) Close() error { return nil }

type lifecycleMemoryFile struct{ runtime *lifecycleMemoryRuntime }

func (f *lifecycleMemoryFile) Write(p []byte) (int, error) {
	if f.runtime.writeErrOnce != nil {
		n := f.runtime.errorWriteBytes
		if n > len(p) {
			n = len(p)
		}
		f.runtime.data = append(f.runtime.data, p[:n]...)
		f.runtime.writes++
		err := f.runtime.writeErrOnce
		f.runtime.writeErrOnce = nil
		return n, err
	}
	n := len(p)
	if f.runtime.maxWrite > 0 && n > f.runtime.maxWrite {
		n = f.runtime.maxWrite
	}
	f.runtime.data = append(f.runtime.data, p[:n]...)
	f.runtime.writes++
	return n, nil
}
func (f *lifecycleMemoryFile) Sync() error {
	err := f.runtime.syncErrOnce
	f.runtime.syncErrOnce = nil
	return err
}
func (*lifecycleMemoryFile) Close() error { return nil }

var _ io.Writer = (*lifecycleMemoryFile)(nil)
