package couchsingleton

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(t *testing.T) Manager {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return Manager{AuthorityDir: filepath.Join(p, "singleton"), Defaults: Roots{filepath.Join(p, "pair", "couch"), filepath.Join(p, "pair"), filepath.Join(p, "identity")}}
}
func TestPreviewReadOnlyAndAcquireImmutable(t *testing.T) {
	m := fixture(t)
	r, e := m.Preview(Request{})
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	if _, e = os.Stat(m.Defaults.PairDataDir); !os.IsNotExist(e) {
		t.Fatal("preview created data")
	}
	if _, e = m.Read(Request{}); e == nil {
		t.Fatal("read adopted")
	}
	s, l, e := m.Acquire(Request{})
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = m.Acquire(Request{}); e == nil {
		t.Fatal("second owner")
	}
	l.Close()
	got, e := m.Read(Request{})
	if e != nil || got.Roots != s.Roots {
		t.Fatal(got, e)
	}
	if _, e = m.Read(Request{Roots: Roots{StoreDir: m.Defaults.StoreDir + "other"}}); e == nil {
		t.Fatal("override accepted")
	}
}
func TestAdoptStalePreviewAndPublicationFailure(t *testing.T) {
	m := fixture(t)
	r, e := m.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(m.Defaults.StoreDir, 0700)
	os.WriteFile(filepath.Join(m.Defaults.StoreDir, "arbitrary"), []byte("changed"), 0600)
	if _, e = m.Adopt(Request{}, r.Digest); e == nil {
		t.Fatal("stale accepted")
	}
	r, e = m.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	m.Publish = func(p string, b []byte) error { return errors.New("write failed") }
	if _, e = m.Adopt(Request{}, r.Digest); e == nil {
		t.Fatal("failure hidden")
	}
	if _, e = m.Read(Request{}); e == nil {
		t.Fatal("published failed write")
	}
	m.Publish = func(p string, b []byte) error {
		if e := durablefile.WriteAtomicStaged(p, b, p+".publication"); e != nil {
			return e
		}
		return errors.New("lost ack")
	}
	if _, e = m.Adopt(Request{}, r.Digest); e == nil {
		t.Fatal("lost ack hidden")
	}
	if _, e = m.Adopt(Request{}, r.Digest); e != nil {
		t.Fatal("retry", e)
	}
}
func TestIsolationRejectsPoisonedSelectionBeforeLease(t *testing.T) {
	m := fixture(t)
	outside := fixture(t)
	s, l, e := outside.Acquire(Request{})
	if e != nil {
		t.Fatal(e)
	}
	l.Close()
	m.IsolationRoot = filepath.Dir(m.AuthorityDir)
	os.MkdirAll(m.AuthorityDir, 0700)
	b, _ := json.Marshal(s)
	os.WriteFile(m.selectionPath(), b, 0600)
	if _, _, e = m.Acquire(Request{}); e == nil {
		t.Fatal("escaped selection")
	}
	if _, e = m.Preview(Request{Stores: []string{outside.Defaults.StoreDir}}); e == nil {
		t.Fatal("escaped request")
	}
}
func TestAdoptionRevalidationAndContenderOrder(t *testing.T) {
	m := fixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	m.AfterInspect = func() error { close(entered); <-release; return nil }
	done := make(chan error, 1)
	go func() {
		_, l, e := m.Acquire(Request{})
		if l != nil {
			l.Close()
		}
		done <- e
	}()
	<-entered
	second := m
	second.AfterInspect = func() error { t.Error("contender reached inspection"); return nil }
	if _, l, e := second.Acquire(Request{}); e == nil {
		l.Close()
		t.Fatal("second admitted owner")
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	m = fixture(t)
	m.BeforePublish = func() error { return os.WriteFile(filepath.Join(m.Defaults.StoreDir, "changed"), []byte("new"), 0600) }
	if _, _, e := m.Acquire(Request{}); e == nil {
		t.Fatal("changed source published")
	}
	if _, e := m.Read(Request{}); e == nil {
		t.Fatal("changed selection exists")
	}
}
func TestFinalRevalidationHonorsAllocationTransaction(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.Defaults.IdentityDir, 0700)
	f, e := os.OpenFile(filepath.Join(m.Defaults.IdentityDir, "couch-identities.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)
	if _, l, e := m.Acquire(Request{}); e == nil {
		l.Close()
		t.Fatal("published during allocation transaction")
	}
	if _, e = m.Read(Request{}); e == nil {
		t.Fatal("published selection")
	}
}
func TestAdoptsSoleLegacyIdentityWithoutChangingBytes(t *testing.T) {
	m := fixture(t)
	legacy := filepath.Join(filepath.Dir(m.AuthorityDir), "legacy")
	id := couchidentity.IdentityStore{HostDir: m.Defaults.IdentityDir, StoreDir: legacy}
	if _, e := id.Allocate(context.Background(), couchidentity.AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "repo"}); e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(m.Defaults.PairDataDir, 0700)
	if e := (&storagegc.Coordinator{Root: m.Defaults.PairDataDir}).RegisterStore(context.Background(), legacy); e != nil {
		t.Fatal(e)
	}
	hp := filepath.Join(id.HostDir, "couch-identities.json")
	lp := filepath.Join(legacy, "identities.json")
	beforeH, _ := os.ReadFile(hp)
	beforeL, _ := os.ReadFile(lp)
	r, e := m.Preview(Request{})
	if e != nil || len(r.Blockers) > 0 || r.Selection.Roots.StoreDir != legacy {
		t.Fatal(r, e)
	}
	s, e := m.Adopt(Request{}, r.Digest)
	if e != nil || s.Roots.StoreDir != legacy {
		t.Fatal(s, e)
	}
	afterH, _ := os.ReadFile(hp)
	afterL, _ := os.ReadFile(lp)
	if string(beforeH) != string(afterH) || string(beforeL) != string(afterL) {
		t.Fatal("identity changed")
	}
}
func TestPreviewOwnerMetadataDoesNotChangeDigest(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.Defaults.StoreDir, 0700)
	r, e := m.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	ns, e := couchcore.ExistingCouchNamespace(m.Defaults.StoreDir)
	if e != nil {
		t.Fatal(e)
	}
	l, e := couchcore.AcquireSupervisorLease(ns, couchcore.OSProcOps{})
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	held, e := m.Preview(Request{})
	if e != nil || held.Digest != r.Digest || len(held.Blockers) == 0 {
		t.Fatal(held, e)
	}
	q := Request{Exclude: []string{m.Defaults.StoreDir}}
	held, e = m.Preview(q)
	if e != nil || len(held.Blockers) == 0 {
		t.Fatal("excluded live store admitted", held, e)
	}
}
func TestReadRejectsMalformedSymlinkAndMissingSelectedRoots(t *testing.T) {
	for _, mode := range []string{"version", "duplicate", "symlink", "missing"} {
		t.Run(mode, func(t *testing.T) {
			m := fixture(t)
			_, l, e := m.Acquire(Request{})
			if e != nil {
				t.Fatal(e)
			}
			l.Close()
			switch mode {
			case "version":
				os.WriteFile(m.selectionPath(), []byte(`{"version":99}`), 0600)
			case "duplicate":
				os.WriteFile(m.selectionPath(), []byte(`{"version":1,"version":1}`), 0600)
			case "symlink":
				os.Rename(m.selectionPath(), m.selectionPath()+".bak")
				os.Symlink(m.selectionPath()+".bak", m.selectionPath())
			case "missing":
				os.Rename(m.Defaults.StoreDir, m.Defaults.StoreDir+".bak")
			}
			if _, e = m.Read(Request{}); e == nil {
				t.Fatal("invalid selected state accepted")
			}
			if _, l, e = m.Acquire(Request{}); e == nil {
				l.Close()
				t.Fatal("invalid selected state acquired")
			}
		})
	}
}
func TestDigestBindsResolvedDefaults(t *testing.T) {
	a := fixture(t)
	b := fixture(t)
	ra, e := a.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	rb, e := b.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	if ra.Digest == rb.Digest {
		t.Fatal("receipt does not bind destination roots")
	}
}
func TestSelectedRootDisappearingAfterReadIsNotRecreated(t *testing.T) {
	m := fixture(t)
	_, l, e := m.Acquire(Request{})
	if e != nil {
		t.Fatal(e)
	}
	l.Close()
	m.AfterInspect = func() error { return os.Rename(m.Defaults.StoreDir, m.Defaults.StoreDir+"-gone") }
	if _, l, e = m.Acquire(Request{}); e == nil {
		l.Close()
		t.Fatal("missing selected root recreated")
	}
	if _, e = os.Stat(m.Defaults.StoreDir); !os.IsNotExist(e) {
		t.Fatal("root recreated", e)
	}
}
func TestMissingRegisteredStoreNeedsExplicitDisposition(t *testing.T) {
	m := fixture(t)
	legacy := filepath.Join(filepath.Dir(m.AuthorityDir), "retired")
	id := couchidentity.IdentityStore{HostDir: m.Defaults.IdentityDir, StoreDir: legacy}
	_, e := id.Allocate(context.Background(), couchidentity.AllocationRequest{Terminal: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(legacy, legacy+".preserved"); e != nil {
		t.Fatal(e)
	}
	r, e := m.Preview(Request{})
	if e != nil || len(r.Blockers) == 0 {
		t.Fatal(r, e)
	}
	q := Request{Exclude: []string{legacy}}
	r, e = m.Preview(q)
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	s, e := m.Adopt(q, r.Digest)
	if e != nil || len(s.Excluded) != 1 {
		t.Fatal(s, e)
	}
	if _, e = os.Stat(legacy + ".preserved/identities.json"); e != nil {
		t.Fatal("retired data changed", e)
	}
}
func TestPreviewFilesystemSnapshotIsUnchanged(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.Defaults.StoreDir, 0700)
	os.WriteFile(filepath.Join(m.Defaults.StoreDir, "record"), []byte("legacy data"), 0600)
	snapshot := func() map[string]string {
		out := map[string]string{}
		e := filepath.WalkDir(filepath.Dir(m.AuthorityDir), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				out[p] = "directory"
				return nil
			}
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			out[p] = string(b)
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	before := snapshot()
	if _, e := m.Preview(Request{}); e != nil {
		t.Fatal(e)
	}
	after := snapshot()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview mutated files", before, after)
	}
}
func TestInspectionBudgetIncludesDirectoriesAndSpansStores(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "empty"), 0700)
	budget := &inspectionBudget{ctx: context.Background(), entries: 65535}
	if _, e := hashTree(root, map[string]string{}, budget); e == nil {
		t.Fatal("empty directories escaped limit")
	}
	a, b := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(a, "one"), []byte("a"), 0600)
	os.WriteFile(filepath.Join(b, "two"), []byte("b"), 0600)
	budget = &inspectionBudget{ctx: context.Background(), bytes: (64 << 20) - 1}
	if _, e := hashTree(a, map[string]string{}, budget); e != nil {
		t.Fatal(e)
	}
	if _, e := hashTree(b, map[string]string{}, budget); e == nil {
		t.Fatal("per-store budget reset")
	}
}

type forbiddenProc struct{ couchcore.ProcOps }

func (forbiddenProc) Exists(int) couchcore.Liveness { panic("read probed a process") }
func (forbiddenProc) Identity(int) (string, error)  { panic("read probed identity") }
func TestSelectedReadDoesNotScanLegacySourcesOrProcesses(t *testing.T) {
	m := fixture(t)
	_, l, e := m.Acquire(Request{})
	if e != nil {
		t.Fatal(e)
	}
	l.Close()
	os.WriteFile(filepath.Join(m.Defaults.IdentityDir, "couch-identities.json"), []byte("bad legacy registry"), 0600)
	m.Proc = forbiddenProc{}
	if _, e = m.Read(Request{}); e != nil {
		t.Fatal(e)
	}
}
func TestAcquireRejectsSymlinkedSupervisorLock(t *testing.T) {
	for _, host := range []bool{true, false} {
		m := fixture(t)
		root := m.Defaults.StoreDir
		if host {
			root = m.AuthorityDir
		}
		os.MkdirAll(root, 0700)
		target := filepath.Join(filepath.Dir(m.AuthorityDir), "external-lock")
		os.WriteFile(target, nil, 0600)
		os.Symlink(target, filepath.Join(root, "supervisor.lock"))
		if _, l, e := m.Acquire(Request{}); e == nil {
			l.Close()
			t.Fatal("symlink supervisor lock acquired")
		}
	}
}
func TestIdentityOnlyLegacyStoreRequiresSupportingRoots(t *testing.T) {
	m := fixture(t)
	legacy := filepath.Join(filepath.Dir(m.AuthorityDir), "custom-store")
	id := couchidentity.IdentityStore{HostDir: m.Defaults.IdentityDir, StoreDir: legacy}
	if _, e := id.Allocate(context.Background(), couchidentity.AllocationRequest{Conversation: true}); e != nil {
		t.Fatal(e)
	}
	r, e := m.Preview(Request{})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Blockers) == 0 {
		t.Fatal("guessed supporting Pair root")
	}
	q := Request{Roots: Roots{StoreDir: legacy, PairDataDir: m.Defaults.PairDataDir, IdentityDir: m.Defaults.IdentityDir}}
	r, e = m.Preview(q)
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	if _, e = m.Adopt(q, r.Digest); e != nil {
		t.Fatal(e)
	}
}
