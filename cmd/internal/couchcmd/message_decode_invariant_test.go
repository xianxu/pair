package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

// messageDecodeSnapshot counts actual index body IO independently of decoder
// input. Paths identify which owner's mutation caused work. The copy returned
// by snapshot remains stable as later observations advance the fixture.
type messageDecodeSnapshot struct {
	ReadBytes, DecodeBytes int64
	ByPath                 map[string]int64
}

func (s messageDecodeSnapshot) delta(before messageDecodeSnapshot) messageDecodeSnapshot {
	out := messageDecodeSnapshot{ReadBytes: s.ReadBytes - before.ReadBytes, DecodeBytes: s.DecodeBytes - before.DecodeBytes, ByPath: map[string]int64{}}
	for path, n := range s.ByPath {
		if d := n - before.ByPath[path]; d != 0 {
			out.ByPath[path] = d
		}
	}
	return out
}

type messageDecodeCounter struct{ current messageDecodeSnapshot }

func (c *messageDecodeCounter) snapshot() messageDecodeSnapshot {
	out := c.current
	out.ByPath = map[string]int64{}
	for p, n := range c.current.ByPath {
		out.ByPath[p] = n
	}
	return out
}
func (c *messageDecodeCounter) io(t *testing.T) launcher.SessionNameIndexCacheIO {
	t.Helper()
	c.current.ByPath = map[string]int64{}
	return launcher.SessionNameIndexCacheIO{ReadFile: func(path string) ([]byte, error) {
		// This is the complete production index-reader seam. Message authority
		// deliberately has no inventory runtime: ledgers are not an input here.
		if filepath.Base(path) != "session-names.jsonl" {
			t.Errorf("message index reader touched unrelated artifact: %s", path)
		}
		raw, err := os.ReadFile(path)
		c.current.ReadBytes += int64(len(raw))
		c.current.ByPath[path] += int64(len(raw))
		return raw, err
	}, Decode: func(raw string) (launcher.SessionNameIndex, error) {
		c.current.DecodeBytes += int64(len(raw))
		return launcher.DecodeSessionNameIndex(raw)
	}}
}

// TestMessageAuthorityUnchangedFilesDecodeZeroBytes is the message half of
// #204's counted release invariant. The 20 owner ledgers have their real 30 MiB
// logical sizes but are sparse, metadata-only fixtures: the production message
// path must not consult their bodies, and no ledger parser is wired here.
// Ready/PID files remain intentionally fresh small-file reads. Counters cover
// session indexes (the reported hot loop), not those security checks.
func TestMessageAuthorityUnchangedFilesDecodeZeroBytes(t *testing.T) {
	const slots = 20
	const ledgerBytes = 30 * 1024 * 1024
	dir := t.TempDir()
	ctx := context.Background()
	world := newMessageWorld()
	counter := &messageDecodeCounter{}
	checker := couchcore.NewScopedThreadArtifactCollisionChecker(dir)
	checker.SessionIndexes = launcher.NewSessionNameIndexCacheWithIO(counter.io(t))
	reader := couchcore.OSReadyReader{DataDir: dir, Proc: couchcore.OSProcOps{}}
	authority := world.authority()
	pidChecks, threadChecks := 0, 0
	originalThread := authority.thread
	authority.thread = func(ctx context.Context, b couchmessage.Binding) (string, error) {
		threadChecks++
		return originalThread(ctx, b)
	}
	authority.wrapperPID = func(b couchmessage.Binding) (int, error) {
		pidChecks++
		p, e := artifactpath.Resolve(artifactpath.Address{DataDir: dir, RepoScope: b.Scope, Tag: b.Tag})
		if e != nil {
			return 0, e
		}
		return readMessageWrapperPID(p.PairWrapPID())
	}
	authority.recorded = func(ctx context.Context, b couchmessage.Binding) error {
		address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
		ready, ok, e := reader.RecordedSession(ctx, address, b.Agent, b.Nonce)
		if e != nil {
			return e
		}
		if !ok || ready != b.Session {
			return errors.New("wrapper launch registration unavailable")
		}
		name, e := checker.PairSessionName(ctx, address)
		if e != nil {
			return e
		}
		if name != b.Session {
			return errors.New("wrapper conversation changed")
		}
		return nil
	}
	bindings := make([]couchmessage.Binding, slots)
	indexPaths := make([]string, slots)
	var legacy strings.Builder
	writeIndex := func(b couchmessage.Binding, name string) string {
		t.Helper()
		line, e := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: name, ScopeKey: b.Scope, Tag: b.Tag, RepoRoot: "/tmp/project", RepoName: "project"})
		if e != nil {
			t.Fatal(e)
		}
		return line + "\n"
	}
	for i := range slots {
		old := world.add(i + 1)
		b := old
		b.Scope = fmt.Sprintf("%016x", i+1)
		b.Session = fmt.Sprintf("pair-slot-%d", i+1)
		world.mu.Lock()
		s := world.slots[old.Thread()]
		delete(world.slots, old.Thread())
		s.binding = b
		world.slots[b.Thread()] = s
		world.mu.Unlock()
		bindings[i] = b
		p, e := artifactpath.Resolve(artifactpath.Address{DataDir: dir, RepoScope: b.Scope, Tag: b.Tag})
		if e != nil {
			t.Fatal(e)
		}
		if e := os.MkdirAll(p.ScopeDir(), 0700); e != nil {
			t.Fatal(e)
		}
		indexPaths[i] = p.SessionBindings()
		if e := os.WriteFile(indexPaths[i], []byte(writeIndex(b, b.Session)), 0600); e != nil {
			t.Fatal(e)
		}
		legacy.WriteString(writeIndex(b, "pair-obsolete"))
		raw, e := readiness.Encode(readiness.ReadyRecord{Tag: b.Tag, Agent: b.Agent, Session: b.Session, Nonce: b.Nonce, PID: b.PID})
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p.AgentReady(b.Agent), []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p.PairWrapPID(), []byte(fmt.Sprint(b.PID)), 0600); e != nil {
			t.Fatal(e)
		}
		// Sparse files supply the exact operator scale with bounded resident memory.
		f, e := os.OpenFile(p.Ledger(), os.O_CREATE|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := f.Truncate(ledgerBytes); e != nil {
			f.Close()
			t.Fatal(e)
		}
		if e := f.Close(); e != nil {
			t.Fatal(e)
		}
		info, e := os.Stat(p.Ledger())
		if e != nil || info.Size() != ledgerBytes {
			t.Fatalf("ledger scale %v %v", info, e)
		}
	}
	if e := os.WriteFile(filepath.Join(dir, "session-names.jsonl"), []byte(legacy.String()), 0600); e != nil {
		t.Fatal(e)
	}
	pass := func() {
		t.Helper()
		for _, b := range bindings {
			root, e := authority.current(ctx, b)
			if e != nil || root == "" {
				t.Fatalf("slot %s: %q %v", b.Slot, root, e)
			}
		}
	}
	pass()
	cold := counter.snapshot()
	if cold.ReadBytes == 0 || cold.DecodeBytes == 0 || len(cold.ByPath) != slots+1 {
		t.Fatalf("disconnected cold counters: %+v", cold)
	}
	threads, pids := threadChecks, pidChecks
	for range 3 {
		pass()
	}
	warm := counter.snapshot().delta(cold)
	if warm.ReadBytes != 0 || warm.DecodeBytes != 0 {
		t.Fatalf("unchanged authority repeated work: %+v", warm)
	}
	if threadChecks-threads != 3*slots || pidChecks-pids != 3*slots {
		t.Fatal("cache skipped current pane/PID authority checks")
	}
	// A same-size rewrite that retains the same binding still requires exactly
	// this owner's strict revalidation, followed by another zero-work pass.
	b := bindings[7]
	original, e := os.ReadFile(indexPaths[7])
	if e != nil {
		t.Fatal(e)
	}
	rewritten := strings.Replace(string(original), "project", "PROJECT", 1)
	before := counter.snapshot()
	if e := os.WriteFile(indexPaths[7], []byte(rewritten), 0600); e != nil {
		t.Fatal(e)
	}
	pass()
	mutation := counter.snapshot().delta(before)
	if mutation.ReadBytes == 0 || mutation.DecodeBytes == 0 || len(mutation.ByPath) != 1 || mutation.ByPath[indexPaths[7]] == 0 {
		t.Fatalf("mutation leaked or disconnected work: %+v", mutation)
	}
	before = counter.snapshot()
	pass()
	if d := counter.snapshot().delta(before); d.ReadBytes != 0 || d.DecodeBytes != 0 {
		t.Fatalf("mutation never warmed: %+v", d)
	}
	// Replacing an index while the old wrapper is alive must revoke its use-time
	// authority. Metadata caching cannot preserve the old session identity.
	temp := indexPaths[7] + ".new"
	if e := os.WriteFile(temp, []byte(writeIndex(b, "pair-replaced")), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(temp, indexPaths[7]); e != nil {
		t.Fatal(e)
	}
	if _, e := authority.current(ctx, b); e == nil {
		t.Fatal("replaced session index retained stale wrapper authority")
	}
	before = counter.snapshot()
	if _, e := authority.current(ctx, b); e == nil {
		t.Fatal("warm stale wrapper authority")
	}
	if d := counter.snapshot().delta(before); d.ReadBytes != 0 || d.DecodeBytes != 0 {
		t.Fatalf("stable refusal repeated decoding: %+v", d)
	}
	// The index cache must not hide a fresh PID-file change either.
	p, e := artifactpath.Resolve(artifactpath.Address{DataDir: dir, RepoScope: bindings[0].Scope, Tag: bindings[0].Tag})
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p.PairWrapPID(), []byte("1"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := authority.current(ctx, bindings[0]); e == nil {
		t.Fatal("cached index bypassed current PID guard")
	}
	t.Logf("20 slots x %d-byte metadata-only ledgers; cold index read=%d decoded=%d; warm index read=0 decoded=0; PID and pane checks remain fresh", ledgerBytes, cold.ReadBytes, cold.DecodeBytes)
}
