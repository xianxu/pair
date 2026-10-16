package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestLegacyLockSlots(t *testing.T) {
	got := LegacyLockSlots([]LegacyLockObservation{
		{Session: "pair-z", Label: "repo:2", Live: true},
		{Session: "pair-a", Label: "repo:1", Live: true},
		{Session: "pair-a", Label: "repo:1", Live: true},
		{Session: "pair-new", Live: true, NativeLockVersion: 1},
		{Session: "pair-future", Live: true, NativeLockVersion: 2},
		{Session: "pair-dead"},
		{Session: "pair-stale", NativeLockVersion: 0},
	})
	if !reflect.DeepEqual(got, []string{"repo:1", "repo:2"}) {
		t.Fatal(got)
	}
}

type capabilityFixture struct {
	sessions []Session
	files    map[string][]byte
	dirs     map[string][]os.DirEntry
	readErr  map[string]error
	liveErr  error
}

func (f *capabilityFixture) io() nativeLockCapabilityIO {
	return nativeLockCapabilityIO{
		live: func(context.Context) ([]Session, error) { return f.sessions, f.liveErr },
		read: func(_ context.Context, p string, _ int64) ([]byte, error) {
			if err := f.readErr[p]; err != nil {
				return nil, err
			}
			b, ok := f.files[p]
			if !ok {
				return nil, os.ErrNotExist
			}
			return b, nil
		},
		readDir: func(p string) ([]os.DirEntry, error) { return f.dirs[p], nil },
	}
}
func TestCheckNativeLockCapabilityHostInventory(t *testing.T) {
	const root = "/data"
	scope, _ := ResolveRepoScope("/repo")
	scopeDir, _ := artifactpath.ResolveScopeDir(root, scope.Key)
	// Real directory entries are just names/type; all reads stay in the fake store.
	temp := t.TempDir()
	if err := os.Mkdir(filepath.Join(temp, scope.Key), 0700); err != nil {
		t.Fatal(err)
	}
	dirs, err := os.ReadDir(temp)
	if err != nil {
		t.Fatal(err)
	}
	entry := SessionNameEntry{SessionName: "📁repo-slot2", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: "repo", Tag: "slot2"}
	line, _ := BuildSessionNameIndexLine(entry)
	paths, _ := artifactpath.ResolveScoped(scopeDir, entry.Tag)
	selected, _ := artifactpath.ResolveSelectedScope(scopeDir)
	record := readiness.ReadyRecord{Tag: entry.Tag, Agent: "codex", Session: entry.SessionName, Nonce: "current", PID: 1}
	old, _ := readiness.Encode(record)
	f := capabilityFixture{sessions: []Session{{Name: entry.SessionName, State: SessionLive}, {Name: "pair-dead", State: SessionExited}}, files: map[string][]byte{selected.SessionBindings(): []byte(line), paths.AgentReady("codex"): []byte(old)}, dirs: map[string][]os.DirEntry{filepath.Join(root, "repos"): dirs}}
	check := func(ex NativeLockExemption) error {
		return checkNativeLockCapability(context.Background(), root, ex, f.io())
	}
	if err := check(NativeLockExemption{}); err == nil || !strings.Contains(err.Error(), "relaunch older slots first: repo/slot2") {
		t.Fatalf("legacy error = %v", err)
	}
	if err := check(NativeLockExemption{Session: entry.SessionName, Nonce: "current"}); err != nil {
		t.Fatal(err)
	}
	if err := check(NativeLockExemption{Session: entry.SessionName, Nonce: "stale"}); err == nil {
		t.Fatal("stale exemption admitted")
	}
	if err := check(NativeLockExemption{Session: "other", Nonce: "current"}); err == nil {
		t.Fatal("wrong session admitted")
	}
	record.NativeLockVersion = 1
	capable, _ := readiness.Encode(record)
	f.files[paths.AgentReady("codex")] = []byte(capable)
	if err := check(NativeLockExemption{}); err != nil {
		t.Fatal(err)
	}
	record.Session = "pair-stale"
	stale, _ := readiness.Encode(record)
	f.files[paths.AgentReady("codex")] = []byte(stale)
	if err := check(NativeLockExemption{}); err == nil {
		t.Fatal("stale readiness admitted")
	}
	delete(f.files, paths.AgentReady("codex"))
	if err := check(NativeLockExemption{}); err == nil {
		t.Fatal("missing readiness admitted")
	}
	f.sessions = append(f.sessions, Session{Name: "📁unknown", State: SessionLive})
	if err := check(NativeLockExemption{}); err == nil || !strings.Contains(err.Error(), "📁unknown") {
		t.Fatal(err)
	}
}
func TestCheckNativeLockCapabilityErrors(t *testing.T) {
	boom := errors.New("inventory unavailable")
	f := capabilityFixture{liveErr: boom}
	if err := checkNativeLockCapability(context.Background(), "/data", NativeLockExemption{}, f.io()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	f.liveErr = nil
	f.sessions = []Session{{Name: "pair-work", State: SessionLive}}
	legacy, _ := artifactpath.ResolveLegacyRoot("/data")
	f.readErr = map[string]error{legacy.SessionBindings(): boom}
	if err := checkNativeLockCapability(context.Background(), "/data", NativeLockExemption{}, f.io()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	f.readErr = nil
	f.files = map[string][]byte{legacy.SessionBindings(): []byte("broken")}
	if err := checkNativeLockCapability(context.Background(), "/data", NativeLockExemption{}, f.io()); err == nil {
		t.Fatal("malformed index admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := checkNativeLockCapability(ctx, "/data", NativeLockExemption{}, f.io()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestCheckNativeLockCapabilityEveryAgentAcrossScopes(t *testing.T) {
	const root = "/data"
	temp := t.TempDir()
	for _, name := range []string{"first", "second"} {
		if err := os.Mkdir(filepath.Join(temp, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	dirs, err := os.ReadDir(temp)
	if err != nil {
		t.Fatal(err)
	}
	f := capabilityFixture{files: map[string][]byte{}, dirs: map[string][]os.DirEntry{filepath.Join(root, "repos"): dirs}}
	var readyPaths []string
	for i, agent := range AgentInventory() {
		key := []string{"first", "second"}[i%2]
		dir, _ := artifactpath.ResolveScopeDir(root, key)
		tag := "tag-" + agent
		name := "📁repo-" + agent
		entry := SessionNameEntry{SessionName: name, ScopeKey: key, RepoRoot: "/repo", RepoName: "repo", Tag: tag}
		line, _ := BuildSessionNameIndexLine(entry)
		scope, _ := artifactpath.ResolveSelectedScope(dir)
		f.files[scope.SessionBindings()] = append(f.files[scope.SessionBindings()], []byte(line+"\n")...)
		paths, _ := artifactpath.ResolveScoped(dir, tag)
		path := paths.AgentReady(agent)
		raw, err := readiness.Encode(readiness.ReadyRecord{Tag: tag, Agent: agent, Session: name, Nonce: "n", PID: 1, NativeLockVersion: 1})
		if err != nil {
			t.Fatal(err)
		}
		f.files[path] = []byte(raw)
		readyPaths = append(readyPaths, path)
		f.sessions = append(f.sessions, Session{Name: name, State: SessionLive})
	}
	check := func() error {
		return checkNativeLockCapability(context.Background(), root, NativeLockExemption{}, f.io())
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
	for _, path := range readyPaths {
		raw := f.files[path]
		record, err := readiness.Decode(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		record.NativeLockVersion = 0
		old, _ := readiness.Encode(record)
		f.files[path] = []byte(old)
		if err := check(); err == nil || !strings.Contains(err.Error(), "repo/"+record.Tag) {
			t.Fatalf("old %s: %v", record.Agent, err)
		}
		f.files[path] = raw
	}
}

func TestCheckNativeLockCapabilityReadinessReadErrors(t *testing.T) {
	const root = "/data"
	entry := SessionNameEntry{SessionName: "pair-work", ScopeKey: "scope", RepoRoot: "/repo", RepoName: "repo", Tag: "work"}
	legacy, _ := artifactpath.ResolveLegacyRoot(root)
	paths, _ := artifactpath.ResolveScoped(root, entry.Tag)
	line, _ := BuildSessionNameIndexLine(entry)
	boom := errors.New("read interrupted")
	f := capabilityFixture{sessions: []Session{{Name: entry.SessionName, State: SessionLive}}, files: map[string][]byte{legacy.SessionBindings(): []byte(line)}, readErr: map[string]error{paths.AgentReady("codex"): boom}}
	if err := checkNativeLockCapability(context.Background(), root, NativeLockExemption{}, f.io()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	f.readErr = nil
	f.files[paths.AgentReady("codex")] = []byte("broken")
	if err := checkNativeLockCapability(context.Background(), root, NativeLockExemption{}, f.io()); err == nil {
		t.Fatal("corrupt readiness admitted")
	}
}
