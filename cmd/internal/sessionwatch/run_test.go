package sessionwatch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestRunEstablishesOnlyAfterCompletedCorroboratedRound(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(nativeRoot)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "please inspect the durable watcher boundary now"
	native.PutFile(sessioninventory.FileEntry{Artifact: artifact}, codexRound(sid, text))
	native.SetProcess("1234", "native-identity", nil, []string{filepath.Join(nativeRoot.Path, filepath.FromSlash(relative))})

	paths := mustScopedPaths(t, dataDir, "work")
	launch := mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = launch
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: time.Second, Poll: time.Millisecond}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	if got := string(runtime.writes[paths.Config("codex")]); !strings.Contains(got, sid) {
		t.Fatalf("config=%s", got)
	}
	if !runtime.hasLog(adapt.Fired) {
		t.Fatalf("logs=%#v", runtime.logs)
	}
}

func TestRunDoesNotUseChronologyToResolveRepeatedRound(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
	native.AddRoot(root)
	text := "please inspect the durable watcher boundary now"
	for index, sid := range []string{"019eff64-6ceb-7e72-9d41-a735a97029ac", "123e4567-e89b-12d3-a456-426614174000"} {
		relative := "2026/08/28/rollout-" + string(rune('a'+index)) + "-" + sid + ".jsonl"
		native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}}, codexRound(sid, text))
	}
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})
	runtime.files[paths.Log()] = []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	runtime.onSleep = func() { runtime.identities["1234"] = "changed" }
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second, Timeout: time.Second, Poll: time.Millisecond}, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 0 || len(runtime.writes) != 0 {
		t.Fatalf("records=%#v writes=%#v", runtime.store.records, runtime.writes)
	}
}

func TestRunRejectsSupersededLaunch(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	paths := mustScopedPaths(t, dataDir, "work")
	first := strings.TrimSpace(string(mustLaunchRecord(t, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex"})))
	second := first
	runtime := newWatcherRuntime(sessioninventorytest.NewFakeRuntime())
	runtime.files[paths.Ledger()] = []byte(first + "\n" + second + "\n")
	runtime.now = time.Unix(10, 0)
	if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Nanosecond, Timeout: time.Nanosecond, Poll: time.Nanosecond}, runtime); !errors.Is(err, sessionledger.ErrStaleLaunch) {
		t.Fatalf("err=%v", err)
	}
}

func TestPIDFileFreshUsesExactNativeBoundAndLegacySecondTolerance(t *testing.T) {
	bound := time.Unix(100, 500)
	if !pidFileFresh(bound, bound, time.Time{}) || pidFileFresh(time.Unix(100, 499), bound, time.Time{}) || !pidFileFresh(time.Unix(100, 0), time.Time{}, time.Unix(100, 900)) {
		t.Fatal("pid file freshness boundary changed")
	}
}

type watcherRuntime struct {
	now        time.Time
	files      map[string][]byte
	modTimes   map[string]time.Time
	identities map[string]string
	writes     map[string][]byte
	logs       []adapt.Outcome
	native     sessioninventory.Runtime
	store      *fakeLifecycleStore
	onSleep    func()
}

func newWatcherRuntime(native sessioninventory.Runtime) *watcherRuntime {
	return &watcherRuntime{now: time.Unix(100, 0), files: map[string][]byte{}, modTimes: map[string]time.Time{}, identities: map[string]string{}, writes: map[string][]byte{}, native: native, store: &fakeLifecycleStore{}}
}

func (r *watcherRuntime) Now() time.Time { return r.now }
func (r *watcherRuntime) Sleep(duration time.Duration) {
	r.now = r.now.Add(duration)
	if r.onSleep != nil {
		r.onSleep()
	}
}
func (r *watcherRuntime) ReadFile(path string) ([]byte, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), raw...), nil
}
func (r *watcherRuntime) ModTime(path string) (time.Time, error) {
	value, ok := r.modTimes[path]
	if !ok {
		return time.Time{}, os.ErrNotExist
	}
	return value, nil
}
func (r *watcherRuntime) ProcessIdentity(pid string) string { return r.identities[pid] }
func (r *watcherRuntime) AtomicWrite(path string, raw []byte) error {
	r.writes[path] = append([]byte(nil), raw...)
	return nil
}
func (r *watcherRuntime) Log(outcome adapt.Outcome, _ string)                { r.logs = append(r.logs, outcome) }
func (r *watcherRuntime) NativeRuntime(_, _ string) sessioninventory.Runtime { return r.native }
func (r *watcherRuntime) LedgerAppender() LedgerAppender                     { return r.store }
func (r *watcherRuntime) hasLog(want adapt.Outcome) bool {
	for _, outcome := range r.logs {
		if outcome == want {
			return true
		}
	}
	return false
}

func codexRound(sid, text string) []byte {
	return []byte(`{"timestamp":"2026-08-28T01:00:00Z","type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + text + `"}]}}` + "\n" +
		`{"type":"response_item","payload":{"type":"function_call"}}` + "\n")
}

func mustLaunchRecord(t *testing.T, record sessionledger.Record) []byte {
	t.Helper()
	raw, err := sessionledger.EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

type scopedTestPaths struct{ dataDir, tag string }

func mustScopedPaths(t *testing.T, dataDir, tag string) scopedTestPaths {
	t.Helper()
	return scopedTestPaths{dataDir: dataDir, tag: tag}
}
func (p scopedTestPaths) Ledger() string   { return filepath.Join(p.dataDir, "ledger-"+p.tag+".jsonl") }
func (p scopedTestPaths) Log() string      { return filepath.Join(p.dataDir, "log-"+p.tag+".md") }
func (p scopedTestPaths) AgentPID() string { return filepath.Join(p.dataDir, "agent-pid-"+p.tag) }
func (p scopedTestPaths) Config(agent string) string {
	return filepath.Join(p.dataDir, "config-"+p.tag+"-"+agent+".json")
}
