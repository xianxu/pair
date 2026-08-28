package sessionledger

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLedgerStoreAppendConsumesMalformedTailOrdinal(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	if err := os.WriteFile(path, []byte("partial-tail"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := LedgerStore{Runtime: OSRuntime{}}
	appended, err := store.Append(path, launchRecord("scope", "work", 12))
	if err != nil {
		t.Fatal(err)
	}
	if appended.Ordinal != 2 {
		t.Fatalf("ordinal=%d, want 2", appended.Ordinal)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := ParseLedger(raw)
	if len(parsed.MalformedOrdinals) != 1 || parsed.MalformedOrdinals[0] != 1 || len(parsed.Records) != 1 || parsed.Records[0].Ordinal != 2 {
		t.Fatalf("parsed=%#v raw=%q", parsed, raw)
	}
}

func TestLedgerStoreEncodesBeforeTakingLock(t *testing.T) {
	t.Parallel()
	runtime := &countingRuntime{Runtime: OSRuntime{}}
	store := LedgerStore{Runtime: runtime}
	if _, err := store.Append(filepath.Join(t.TempDir(), "ledger.jsonl"), Record{}); err == nil {
		t.Fatal("invalid record appended")
	}
	if runtime.locks != 0 {
		t.Fatalf("locks=%d, want 0", runtime.locks)
	}
}

func TestLedgerStoreFailureRetryUsesNextPhysicalOrdinal(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"short-write", "fsync"} {
		failure := failure
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "ledger.jsonl")
			runtime := &failingAppendRuntime{Runtime: OSRuntime{}, failure: failure}
			if _, err := (LedgerStore{Runtime: runtime}).Append(path, launchRecord("scope", "work", 1)); err == nil {
				t.Fatal("injected append returned nil error")
			}
			appended, err := (LedgerStore{Runtime: OSRuntime{}}).Append(path, launchRecord("scope", "work", 2))
			if err != nil {
				t.Fatal(err)
			}
			if appended.Ordinal != 2 {
				t.Fatalf("retry ordinal=%d, want 2", appended.Ordinal)
			}
		})
	}
}

func TestLedgerStoreAppendBindingIfCurrent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	store := LedgerStore{Runtime: OSRuntime{}}
	first, err := store.Append(path, launchRecord("scope", "work", 1))
	if err != nil {
		t.Fatal(err)
	}
	owner := Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}
	bound, err := store.AppendBindingIfCurrent(path, owner, first.Ordinal, "native-a")
	if err != nil || bound.Kind != RecordBinding || bound.LaunchOrdinal != first.Ordinal {
		t.Fatalf("bound=%#v err=%v", bound, err)
	}
	second, err := store.Append(path, launchRecord("scope", "work", 2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendBindingIfCurrent(path, owner, first.Ordinal, "stale"); !errors.Is(err, ErrStaleLaunch) {
		t.Fatalf("stale append err=%v", err)
	}
	current, ok := CurrentLaunch(ParseLedger(mustReadFile(t, path)).Records, owner)
	if !ok || current.Launch.Ordinal != second.Ordinal || current.Binding != nil {
		t.Fatalf("current=%#v ok=%v", current, ok)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type countingRuntime struct {
	Runtime
	locks int
}

func (r *countingRuntime) Lock(path string) (Unlocker, error) {
	r.locks++
	return r.Runtime.Lock(path)
}

type failingAppendRuntime struct {
	Runtime
	failure string
}

func (r *failingAppendRuntime) OpenAppend(path string, mode os.FileMode) (AppendFile, error) {
	file, err := r.Runtime.OpenAppend(path, mode)
	if err != nil {
		return nil, err
	}
	return &failingAppendFile{AppendFile: file, failure: r.failure}, nil
}

type failingAppendFile struct {
	AppendFile
	failure string
	failed  bool
}

func (f *failingAppendFile) Write(raw []byte) (int, error) {
	if f.failure == "short-write" && !f.failed {
		f.failed = true
		n, _ := f.AppendFile.Write(raw[:max(1, len(raw)/2)])
		return n, io.ErrShortWrite
	}
	return f.AppendFile.Write(raw)
}

func (f *failingAppendFile) Sync() error {
	if f.failure == "fsync" && !f.failed {
		f.failed = true
		return errors.New("injected fsync failure")
	}
	return f.AppendFile.Sync()
}

func launchRecord(scope, tag string, offset uint64) Record {
	return Record{Version: 1, Kind: RecordLaunch, ScopeKey: scope, Tag: tag, Agent: "claude", PairLogOffset: offset}
}
