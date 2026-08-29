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

func TestLedgerStoreAppendOutcomeMatchesRecoveredAuthority(t *testing.T) {
	t.Parallel()
	for _, kind := range []RecordKind{RecordLaunch, RecordBinding} {
		kind := kind
		for _, failure := range []struct {
			name          string
			writeLimit    int
			stage         string
			wantOutcome   AppendOutcome
			wantAuthority bool
		}{
			{name: "write-before-row", writeLimit: 1, stage: "write", wantOutcome: AppendNotAuthoritative},
			{name: "write-after-row", writeLimit: -1, stage: "write", wantOutcome: AppendIndeterminate, wantAuthority: true},
			{name: "file-sync", stage: "sync", wantOutcome: AppendIndeterminate, wantAuthority: true},
			{name: "close", stage: "close", wantOutcome: AppendIndeterminate, wantAuthority: true},
			{name: "directory-sync", stage: "directory-sync", wantOutcome: AppendIndeterminate, wantAuthority: true},
			{name: "unlock", stage: "unlock", wantOutcome: AppendCommitted, wantAuthority: true},
		} {
			failure := failure
			t.Run(string(kind)+"/"+failure.name, func(t *testing.T) {
				t.Parallel()
				path := filepath.Join(t.TempDir(), "ledger.jsonl")
				owner := Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}
				launchOrdinal := uint64(0)
				if kind == RecordBinding {
					launch, err := (LedgerStore{Runtime: OSRuntime{}}).Append(path, launchRecord("scope", "work", 1))
					if err != nil {
						t.Fatal(err)
					}
					launchOrdinal = launch.Ordinal
				}

				runtime := &failurePointRuntime{Runtime: OSRuntime{}, stage: failure.stage, writeLimit: failure.writeLimit, remaining: -1}
				store := LedgerStore{Runtime: runtime}
				var appended Record
				var err error
				if kind == RecordLaunch {
					appended, err = store.Append(path, launchRecord("scope", "work", 2))
				} else {
					appended, err = store.AppendBindingIfCurrent(path, owner, launchOrdinal, "native-a")
				}
				if err == nil {
					t.Fatal("injected append returned nil error")
				}
				if got := AppendOutcomeOf(err); got != failure.wantOutcome {
					t.Fatalf("outcome=%v, want %v (err=%v)", got, failure.wantOutcome, err)
				}
				if failure.wantOutcome != AppendNotAuthoritative && appended.Ordinal == 0 {
					t.Fatalf("append result lost physical ordinal: %#v", appended)
				}

				current, ok := CurrentLaunch(ParseLedger(mustReadFile(t, path)).Records, owner)
				gotAuthority := ok && ((kind == RecordLaunch && current.Launch.PairLogOffset == 2) ||
					(kind == RecordBinding && current.Binding != nil && current.Binding.RootNativeID == "native-a"))
				if gotAuthority != failure.wantAuthority {
					t.Fatalf("recovered authority=%v, want %v; current=%#v", gotAuthority, failure.wantAuthority, current)
				}
			})
		}
	}
}

func TestLedgerStoreEveryIncompleteWriteRemainsNonAuthoritative(t *testing.T) {
	t.Parallel()
	for _, kind := range []RecordKind{RecordLaunch, RecordBinding} {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			owner := Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"}
			record := launchRecord("scope", "work", 2)
			if kind == RecordBinding {
				record = Record{Version: 1, Kind: RecordBinding, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, LaunchOrdinal: 1, RootNativeID: "native-a"}
			}
			encoded, err := EncodeRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			payloadLength := len(encoded) + 1

			for limit := 0; limit < payloadLength; limit++ {
				path := filepath.Join(t.TempDir(), "ledger.jsonl")
				launchOrdinal := uint64(0)
				if kind == RecordBinding {
					launch, appendErr := (LedgerStore{Runtime: OSRuntime{}}).Append(path, launchRecord("scope", "work", 1))
					if appendErr != nil {
						t.Fatal(appendErr)
					}
					launchOrdinal = launch.Ordinal
				}
				runtime := &failurePointRuntime{Runtime: OSRuntime{}, stage: "write", writeLimit: limit, remaining: -1}
				store := LedgerStore{Runtime: runtime}
				if kind == RecordLaunch {
					_, err = store.Append(path, record)
				} else {
					_, err = store.AppendBindingIfCurrent(path, owner, launchOrdinal, "native-a")
				}
				if got := AppendOutcomeOf(err); got != AppendNotAuthoritative {
					t.Fatalf("limit=%d outcome=%v, want %v (err=%v)", limit, got, AppendNotAuthoritative, err)
				}
				current, ok := CurrentLaunch(ParseLedger(mustReadFile(t, path)).Records, owner)
				if kind == RecordLaunch && ok {
					t.Fatalf("limit=%d recovered launch authority: %#v", limit, current)
				}
				if kind == RecordBinding && (!ok || current.Binding != nil) {
					t.Fatalf("limit=%d recovered binding authority: %#v ok=%v", limit, current, ok)
				}
			}
		})
	}
}

func TestLedgerStoreReconcilesIndeterminateRowsWithoutAppendingAgain(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	runtime := &failurePointRuntime{Runtime: OSRuntime{}, stage: "sync", remaining: 1}
	store := LedgerStore{Runtime: runtime}
	launch, err := store.Append(path, launchRecord("scope", "work", 2))
	if AppendOutcomeOf(err) != AppendIndeterminate || launch.Ordinal != 1 {
		t.Fatalf("append=%#v outcome=%v err=%v", launch, AppendOutcomeOf(err), err)
	}
	if err := store.Reconcile(path, launch); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	parsed := ParseLedger(mustReadFile(t, path))
	if len(parsed.Records) != 1 || parsed.Records[0].Ordinal != launch.Ordinal {
		t.Fatalf("parsed=%#v", parsed)
	}

	legacy := []byte(`{"agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair"}`)
	runtime.stage, runtime.remaining = "directory-sync", 1
	ordinal, err := store.AppendLegacy(path, legacy)
	if AppendOutcomeOf(err) != AppendIndeterminate || ordinal != 2 {
		t.Fatalf("legacy ordinal=%d outcome=%v err=%v", ordinal, AppendOutcomeOf(err), err)
	}
	if err := store.ReconcileLegacy(path, ordinal, legacy); err != nil {
		t.Fatalf("reconcile legacy: %v", err)
	}
	parsed = ParseLedger(mustReadFile(t, path))
	if len(parsed.CompatibilityOrdinals) != 1 || parsed.CompatibilityOrdinals[0] != ordinal {
		t.Fatalf("parsed=%#v", parsed)
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

type failurePointRuntime struct {
	Runtime
	stage      string
	writeLimit int
	remaining  int
}

func (r *failurePointRuntime) fail(stage string) bool {
	if r.stage != stage || r.remaining == 0 {
		return false
	}
	if r.remaining > 0 {
		r.remaining--
	}
	return true
}

func (r *failurePointRuntime) Lock(path string) (Unlocker, error) {
	lock, err := r.Runtime.Lock(path)
	if err != nil {
		return nil, err
	}
	return &failurePointUnlocker{Unlocker: lock, runtime: r}, nil
}

func (r *failurePointRuntime) OpenAppend(path string, mode os.FileMode) (AppendFile, error) {
	file, err := r.Runtime.OpenAppend(path, mode)
	if err != nil {
		return nil, err
	}
	return &failurePointFile{AppendFile: file, runtime: r}, nil
}

func (r *failurePointRuntime) SyncDirectory(path string) error {
	if r.fail("directory-sync") {
		return errors.New("injected directory sync failure")
	}
	return r.Runtime.SyncDirectory(path)
}

type failurePointUnlocker struct {
	Unlocker
	runtime *failurePointRuntime
}

func (u *failurePointUnlocker) Close() error {
	err := u.Unlocker.Close()
	if u.runtime.fail("unlock") {
		return errors.Join(err, errors.New("injected unlock failure"))
	}
	return err
}

type failurePointFile struct {
	AppendFile
	runtime *failurePointRuntime
	wrote   bool
}

func (f *failurePointFile) Write(raw []byte) (int, error) {
	if f.runtime.stage != "write" || f.wrote || !f.runtime.fail("write") {
		return f.AppendFile.Write(raw)
	}
	f.wrote = true
	limit := f.runtime.writeLimit
	if limit < 0 || limit > len(raw) {
		limit = len(raw)
	}
	n, writeErr := f.AppendFile.Write(raw[:limit])
	return n, errors.Join(writeErr, errors.New("injected write failure"))
}

func (f *failurePointFile) Sync() error {
	if f.runtime.fail("sync") {
		return errors.New("injected sync failure")
	}
	return f.AppendFile.Sync()
}

func (f *failurePointFile) Close() error {
	err := f.AppendFile.Close()
	if f.runtime.fail("close") {
		return errors.Join(err, errors.New("injected close failure"))
	}
	return err
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
