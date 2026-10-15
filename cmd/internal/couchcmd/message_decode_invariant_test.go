package couchcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/workbenchperftest"
)

// countedMessageIndexIO observes actual reader and decoder inputs separately.
// Decoder inputs retain their source identity even with concurrent admissions.
func countedMessageIndexIO(t testing.TB, counts *workbenchperftest.Counter) launcher.SessionNameIndexCacheIO {
	t.Helper()
	var mu sync.Mutex
	sources := map[string]workbenchperftest.Key{}
	return launcher.SessionNameIndexCacheIO{
		ReadFile: func(path string) ([]byte, error) {
			if filepath.Base(path) != "session-names.jsonl" {
				t.Errorf("message index reader touched unrelated artifact: %s", path)
			}
			raw, err := os.ReadFile(path)
			owner := filepath.Base(filepath.Dir(path))
			key := workbenchperftest.Key{Family: "index", Owner: owner, Path: path}
			counts.RecordRead(key.Family, key.Owner, key.Path, int64(len(raw)))
			mu.Lock()
			sources[string(raw)] = key
			mu.Unlock()
			return raw, err
		},
		Decode: func(raw string) (launcher.SessionNameIndex, error) {
			mu.Lock()
			key, ok := sources[raw]
			mu.Unlock()
			if !ok {
				t.Errorf("decoder input has no observed source")
			}
			counts.RecordDecode(key.Family, key.Owner, key.Path, int64(len(raw)))
			return launcher.DecodeSessionNameIndex(raw)
		},
	}
}

type messageEvidenceFixture struct {
	workbenchperftest.Fixture
	checker  couchcore.ScopedThreadArtifactCollisionChecker
	reader   couchcore.OSReadyReader
	indexes  []string
	bindings []couchmessage.Binding
}

func newMessageEvidenceFixture(t testing.TB, slots, ledgerBytes int) *messageEvidenceFixture {
	t.Helper()
	f := &messageEvidenceFixture{Fixture: workbenchperftest.NewFixture(t, slots, ledgerBytes)}
	f.checker = couchcore.NewScopedThreadArtifactCollisionChecker(f.DataDir)
	f.checker.SessionIndexes = launcher.NewSessionNameIndexCacheWithIO(countedMessageIndexIO(t, f.Counts))
	f.checker.InventoryWork = f.Counts.Observer("")
	f.reader = couchcore.OSReadyReader{DataDir: f.DataDir, Proc: couchcore.OSProcOps{}}
	return f
}

func (f *messageEvidenceFixture) add(t testing.TB, world *messageWorld, i int) couchmessage.Binding {
	t.Helper()
	owner := f.Owners[i]
	old := world.add(i + 1)
	b := old
	b.Scope = owner.Scope
	b.Tag = owner.Tag
	b.Agent = owner.Agent
	b.Session = fmt.Sprintf("pair-slot-%d", i+1)
	world.mu.Lock()
	s := world.slots[old.Thread()]
	delete(world.slots, old.Thread())
	s.binding = b
	world.slots[b.Thread()] = s
	world.mu.Unlock()
	paths, e := artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: b.Scope, Tag: b.Tag})
	if e != nil {
		t.Fatal(e)
	}
	index := paths.SessionBindings()
	if e = os.WriteFile(index, []byte(messageIndexLine(t, b, b.Session)), 0600); e != nil {
		t.Fatal(e)
	}
	raw, e := readiness.Encode(readiness.ReadyRecord{Tag: b.Tag, Agent: b.Agent, Session: b.Session, Nonce: b.Nonce, PID: b.PID})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(paths.AgentReady(b.Agent), []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(paths.PairWrapPID(), []byte(fmt.Sprint(b.PID)), 0600); e != nil {
		t.Fatal(e)
	}
	legacy, e := os.OpenFile(filepath.Join(f.DataDir, "session-names.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = legacy.WriteString(messageIndexLine(t, b, "pair-obsolete")); e != nil {
		legacy.Close()
		t.Fatal(e)
	}
	if e = legacy.Close(); e != nil {
		t.Fatal(e)
	}
	f.indexes = append(f.indexes, index)
	f.bindings = append(f.bindings, b)
	return b
}

func messageIndexLine(t testing.TB, b couchmessage.Binding, name string) string {
	t.Helper()
	line, e := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: name, ScopeKey: b.Scope, Tag: b.Tag, RepoRoot: "/tmp/project", RepoName: "project"})
	if e != nil {
		t.Fatal(e)
	}
	return line + "\n"
}

func (f *messageEvidenceFixture) authority(a messageAuthority) messageAuthority {
	a.wrapperPID = func(b couchmessage.Binding) (int, error) {
		p, e := artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: b.Scope, Tag: b.Tag})
		if e != nil {
			return 0, e
		}
		return readMessageWrapperPID(p.PairWrapPID())
	}
	a.recorded = recordedMessageAuthority(f.reader, f.checker.PairSessionName)
	return a
}

// The large consumer test owns the release-scale fixture; this focused case
// retains #559's invalidation and fresh-security-evidence regressions.
func TestMessageAuthorityUnchangedFilesDecodeZeroBytes(t *testing.T) {
	f := newMessageEvidenceFixture(t, 2, 4096)
	world := newMessageWorld()
	for i := range f.Owners {
		f.add(t, world, i)
	}
	a := f.authority(world.authority())
	pass := func() {
		t.Helper()
		for _, b := range f.bindings {
			root, e := a.current(context.Background(), b)
			if e != nil || root == "" {
				t.Fatalf("slot %s: %q %v", b.Slot, root, e)
			}
		}
	}
	pass()
	cold := f.Counts.Snapshot()
	if cold.ReadFamily("index") == 0 || cold.DecodeFamily("index") == 0 {
		t.Fatalf("disconnected cold counters: %+v", cold)
	}
	for range 3 {
		before := f.Counts.Snapshot()
		pass()
		f.Counts.Snapshot().Delta(before).RequireZero(t)
	}
	original, e := os.ReadFile(f.indexes[0])
	if e != nil {
		t.Fatal(e)
	}
	before := f.Counts.Snapshot()
	if e = os.WriteFile(f.indexes[0], []byte(strings.Replace(string(original), "project", "PROJECT", 1)), 0600); e != nil {
		t.Fatal(e)
	}
	pass()
	requireMessageIndexMutation(t, f.Counts.Snapshot().Delta(before), f.indexes[0])
	before = f.Counts.Snapshot()
	pass()
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	replacement := f.indexes[0] + ".new"
	if e = os.WriteFile(replacement, []byte(messageIndexLine(t, f.bindings[0], "pair-replaced")), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(replacement, f.indexes[0]); e != nil {
		t.Fatal(e)
	}
	if _, e = a.current(context.Background(), f.bindings[0]); e == nil {
		t.Fatal("replaced session index retained stale wrapper authority")
	}
	before = f.Counts.Snapshot()
	if _, e = a.current(context.Background(), f.bindings[0]); e == nil {
		t.Fatal("warm stale wrapper authority")
	}
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	p, e := artifactpath.Resolve(artifactpath.Address{DataDir: f.DataDir, RepoScope: f.bindings[1].Scope, Tag: f.bindings[1].Tag})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(p.PairWrapPID(), []byte("1"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.current(context.Background(), f.bindings[1]); e == nil {
		t.Fatal("cached index bypassed current PID guard")
	}
}

func requireMessageIndexMutation(t testing.TB, d workbenchperftest.Snapshot, path string) {
	t.Helper()
	if d.ReadFamily("index") == 0 || d.DecodeFamily("index") == 0 {
		t.Fatalf("index mutation counters disconnected: %+v", d)
	}
	for k, n := range d.Read {
		if n != 0 && (k.Family != "index" || k.Path != path) {
			t.Errorf("mutation read unrelated evidence %+v: %d", k, n)
		}
	}
	for k, n := range d.Decoded {
		if n != 0 && (k.Family != "index" || k.Path != path) {
			t.Errorf("mutation decoded unrelated evidence %+v: %d", k, n)
		}
	}
}
