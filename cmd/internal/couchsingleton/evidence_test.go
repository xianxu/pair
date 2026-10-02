package couchsingleton

import (
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func slotEvidenceFixture(t *testing.T) (Manager, string) {
	t.Helper()
	m := fixture(t)
	fleet := filepath.Dir(m.AuthorityDir)
	primary := filepath.Join(fleet, "repo")
	local := filepath.Join(fleet, "worktree", "repo-slot1", ".couch")
	for _, p := range []string{primary, filepath.Join(filepath.Dir(local), "repo"), local, filepath.Join(m.Defaults.StoreDir, "threadstore")} {
		if e := os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	manifest := map[string]any{"schema_version": 2, "generation": 1, "threads": []any{}, "slot_repositories": []string{primary}, "slot_repository_identities": map[string]string{primary: filepath.Join(primary, ".git")}}
	raw, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(m.Defaults.StoreDir, "threadstore", "manifest.json"), raw, 0600)
	os.WriteFile(filepath.Join(m.Defaults.StoreDir, "threadstore", "store.lock"), nil, 0600)
	os.WriteFile(filepath.Join(local, "store.lock"), nil, 0600)
	return m, local
}
func TestAdoptionRejectsUnreadableSlotEvidence(t *testing.T) {
	for _, kind := range []string{"corrupt", "journal", "symlink", "missing-checkout"} {
		t.Run(kind, func(t *testing.T) {
			m, local := slotEvidenceFixture(t)
			switch kind {
			case "corrupt":
				os.WriteFile(filepath.Join(local, "thread.json"), []byte("{"), 0600)
			case "journal":
				os.WriteFile(filepath.Join(local, "journal.json"), []byte("{}"), 0600)
			case "symlink":
				os.Symlink("/missing", filepath.Join(local, "thread.json"))
			case "missing-checkout":
				os.Remove(filepath.Join(filepath.Dir(local), "repo"))
			}
			r, e := m.Preview(Request{})
			if e == nil && len(r.Blockers) == 0 {
				t.Fatal("unresolved slot admitted", r)
			}
		})
	}
}
func TestSlotSourceChangesInvalidateAdoption(t *testing.T) {
	for _, point := range []string{"after-preview", "before-publish"} {
		t.Run(point, func(t *testing.T) {
			m, local := slotEvidenceFixture(t)
			path := filepath.Join(local, "launch-preferences.json")
			os.WriteFile(path, []byte("before"), 0600)
			r, e := m.Preview(Request{})
			if e != nil || len(r.Blockers) > 0 {
				t.Fatal(r, e)
			}
			change := func() error { return os.WriteFile(path, []byte("after"), 0600) }
			if point == "after-preview" {
				change()
			} else {
				m.BeforePublish = change
			}
			if _, e = m.Adopt(Request{}, r.Digest); e == nil {
				t.Fatal("changed slot source published")
			}
			if _, e = os.Stat(m.selectionPath()); !os.IsNotExist(e) {
				t.Fatal("selection was published", e)
			}
		})
	}
}
func TestSlotTransactionContentionRefusesBeforePublication(t *testing.T) {
	m, local := slotEvidenceFixture(t)
	r, e := m.Preview(Request{})
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	var f *os.File
	m.BeforePublish = func() error {
		var e error
		f, e = os.OpenFile(filepath.Join(local, "store.lock"), os.O_RDWR, 0600)
		if e != nil {
			return e
		}
		return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	started := time.Now()
	if _, e = m.Adopt(Request{}, r.Digest); e == nil {
		t.Fatal("published through slot transaction")
	}
	if time.Since(started) > time.Second {
		t.Fatal("waited for slot transaction")
	}
}
func migrationRecord() couchcore.ThreadRecord {
	return couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}, StartingPath: "/repo", WorkingPath: "/repo", CreatedAt: time.Now(), Revision: 1}
}
func TestExcludedIncarnationRequiresConfirmedAbsence(t *testing.T) {
	for _, state := range []string{"live", "unknown", "recycled", "dead", "identity-error"} {
		t.Run(state, func(t *testing.T) {
			m := fixture(t)
			other := filepath.Join(filepath.Dir(m.AuthorityDir), "other")
			ns, e := couchcore.ResolveCouchNamespace(other, "")
			if e != nil {
				t.Fatal(e)
			}
			record := migrationRecord()
			record.Incarnations = []couchcore.ThreadIncarnation{{PID: 12345, Identity: "original", State: couchcore.IncarnationLive, StartedAt: time.Now()}}
			if _, e = couchcore.NewThreadStore(ns).CreateThread(record); e != nil {
				t.Fatal(e)
			}
			proc := couchcore.NewFakeProcOps()
			proc.Set(os.Getpid(), "manager")
			switch state {
			case "live":
				proc.Set(12345, "original")
			case "unknown":
				proc.SetUnknown(12345)
			case "recycled":
				proc.Set(12345, "different")
			case "identity-error":
				proc.Set(12345, "original")
				proc.IdentityErr[12345] = true
			}
			m.Proc = proc
			q := Request{Exclude: []string{other}}
			r, e := m.Preview(q)
			if e != nil {
				t.Fatal(e)
			}
			wantBlock := state == "live" || state == "unknown" || state == "identity-error"
			if (len(r.Blockers) > 0) != wantBlock {
				t.Fatal(state, r)
			}
			if wantBlock && !strings.Contains(strings.Join(r.Blockers, " "), "incarnation") {
				t.Fatal("missing process diagnostic", r)
			}
		})
	}
}
func TestIncarnationLivenessIsRecheckedBeforePublication(t *testing.T) {
	m := fixture(t)
	other := filepath.Join(filepath.Dir(m.AuthorityDir), "other")
	ns, _ := couchcore.ResolveCouchNamespace(other, "")
	record := migrationRecord()
	record.Incarnations = []couchcore.ThreadIncarnation{{PID: 12345, Identity: "same", State: couchcore.IncarnationUnknown}}
	if _, e := couchcore.NewThreadStore(ns).CreateThread(record); e != nil {
		t.Fatal(e)
	}
	proc := couchcore.NewFakeProcOps()
	proc.Set(os.Getpid(), "manager")
	m.Proc = proc
	q := Request{Exclude: []string{other}}
	r, e := m.Preview(q)
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	m.BeforePublish = func() error { proc.SetUnknown(12345); return nil }
	if _, e = m.Adopt(q, r.Digest); e == nil {
		t.Fatal("unknown incarnation published")
	}
	if _, e = os.Stat(m.selectionPath()); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
func TestSlotTransactionRemainsHeldDuringPublication(t *testing.T) {
	m, local := slotEvidenceFixture(t)
	m.Publish = func(path string, raw []byte) error {
		f, e := os.OpenFile(filepath.Join(local, "store.lock"), os.O_RDWR, 0600)
		if e != nil {
			return e
		}
		defer f.Close()
		if e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); e == nil {
			unix.Flock(int(f.Fd()), unix.LOCK_UN)
			return errors.New("slot transaction released before publication")
		}
		return durablefile.WriteAtomicStaged(path, raw, path+".publication")
	}
	if _, l, e := m.Acquire(Request{}); e != nil {
		t.Fatal(e)
	} else {
		l.Close()
	}
}
func TestSlotTopologyChangeInvalidatesAdoption(t *testing.T) {
	for _, change := range []string{"new-slot", "removed-checkout", "replaced-root"} {
		t.Run(change, func(t *testing.T) {
			m, local := slotEvidenceFixture(t)
			r, e := m.Preview(Request{})
			if e != nil || len(r.Blockers) > 0 {
				t.Fatal(r, e)
			}
			m.BeforePublish = func() error {
				switch change {
				case "new-slot":
					return os.MkdirAll(filepath.Join(filepath.Dir(filepath.Dir(local)), "repo-slot2", "repo"), 0700)
				case "removed-checkout":
					return os.Rename(filepath.Join(filepath.Dir(local), "repo"), filepath.Join(filepath.Dir(local), "repo-old"))
				default:
					return os.Rename(local, local+"-old")
				}
			}
			if _, e = m.Adopt(Request{}, r.Digest); e == nil {
				t.Fatal("changed slot topology published")
			}
		})
	}
}
func TestExcludedSlotIncarnationIsObserved(t *testing.T) {
	m, local := slotEvidenceFixture(t)
	fleet := filepath.Dir(m.AuthorityDir)
	primary := filepath.Join(fleet, "repo")
	checkout := filepath.Join(filepath.Dir(local), "repo")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", primary}, args...)...)
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	git("init")
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "fixture")
	git("worktree", "add", "--detach", checkout, "HEAD")
	scope, e := launcher.ResolveRepoScope(checkout)
	if e != nil {
		t.Fatal(e)
	}
	record := migrationRecord()
	record.Address.RepoScope = scope.Key
	record.StartingPath = checkout
	record.WorkingPath = checkout
	record.Incarnations = []couchcore.ThreadIncarnation{{PID: 12345, Identity: "slot-wrapper", State: couchcore.IncarnationLive}}
	raw, _ := json.Marshal(record)
	os.WriteFile(filepath.Join(local, "thread.json"), raw, 0600)
	proc := couchcore.NewFakeProcOps()
	proc.Set(12345, "slot-wrapper")
	m.Proc = proc
	other := m.Defaults.StoreDir
	m.Defaults.StoreDir = filepath.Join(m.Defaults.PairDataDir, "fresh")
	q := Request{Exclude: []string{other}}
	r, e := m.Preview(q)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Blockers) == 0 || !strings.Contains(strings.Join(r.Blockers, " "), "incarnation pid 12345") {
		t.Fatal("slot process ignored", r)
	}
	proc.Kill(12345)
	r, e = m.Preview(q)
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal("dead slot owner did not release conflict", r, e)
	}
}
func TestSlotPreviewPreservesEntireFixture(t *testing.T) {
	m, local := slotEvidenceFixture(t)
	os.WriteFile(filepath.Join(local, "preferences.json"), []byte("legacy bytes"), 0600)
	capture := func() map[string]string {
		out := map[string]string{}
		e := filepath.WalkDir(filepath.Dir(m.AuthorityDir), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				out[p] = "directory"
				return nil
			}
			raw, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			out[p] = string(raw)
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	before := capture()
	r, e := m.Preview(Request{})
	if e != nil || len(r.Blockers) > 0 {
		t.Fatal(r, e)
	}
	if !reflect.DeepEqual(before, capture()) {
		t.Fatal("slot preview modified fixture")
	}
}
func TestIsolatedAdoptionRejectsExternalEnrolledSlotSources(t *testing.T) {
	m, _ := slotEvidenceFixture(t)
	m.IsolationRoot = filepath.Dir(m.AuthorityDir)
	outside, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	primary := filepath.Join(outside, "repo")
	os.MkdirAll(primary, 0700)
	manifest := map[string]any{"schema_version": 2, "generation": 1, "threads": []any{}, "slot_repositories": []string{primary}, "slot_repository_identities": map[string]string{primary: filepath.Join(primary, ".git")}}
	raw, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(m.Defaults.StoreDir, "threadstore", "manifest.json"), raw, 0600)
	r, e := m.Preview(Request{})
	if e == nil && !strings.Contains(strings.Join(r.Blockers, " "), "escapes isolated root") {
		t.Fatal("external slot inventory probed without containment", r)
	}
}
