package sessionsync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestRegisterIsDurableAndIdempotentWithoutEditingDocument(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "note.md")
	r.put(path, "## Work\n\noperator text\n")
	first, err := r.sync.Register(path, "pair:5")
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.sync.Register(path, "pair:5")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || second.Changed || first.Registration.ID != second.Registration.ID {
		t.Fatalf("registration identity changed: %+v %+v", first, second)
	}
	reopened := New(filepath.Dir(r.sync.Ledger.Dir))
	// The rig's ledger root is explicit rather than New's standard layout.
	reopened.Ledger.Dir = r.sync.Ledger.Dir
	got, err := reopened.Lookup(path)
	if err != nil || got == nil || got.ID != first.Registration.ID || got.Owner != "pair:5" {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	if r.read(path) != "## Work\n\noperator text\n" {
		t.Fatal("registration changed markdown")
	}
	_, err = r.sync.Register(path, "ops:0")
	wantRefusal(t, err, 1, "owner-conflict")
}

func TestWaitingWriterRechecksRegistrationAfterMetadataMutation(t *testing.T) {
	for _, operation := range []string{"rename", "forget"} {
		t.Run(operation, func(t *testing.T) {
			r := newRig(t)
			old := filepath.Join(filepath.Dir(r.p.File), "plain.md")
			r.put(old, "## Work\n\noriginal\n")
			if _, err := r.sync.Register(old, "ops:0"); err != nil {
				t.Fatal(err)
			}
			read, err := r.sync.Read(old, "Work")
			if err != nil {
				t.Fatal(err)
			}
			newPath := filepath.Join(filepath.Dir(old), "new.md")
			r.put(newPath, r.read(old)) // saveas leaves OLD present
			root := r.sync.registryStore().Dir
			publishing, release, waiting := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			var workers sync.WaitGroup
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }); workers.Wait() })
			await := func(ch <-chan struct{}, name string) {
				t.Helper()
				select {
				case <-ch:
				case <-testwait.After(t):
					t.Fatalf("waiting for %s", name)
				}
			}
			mutator := *r.sync
			mutator.Registry = &registryStore{Dir: root, Publish: func(path string, raw []byte) error {
				if filepath.Base(path) == "registry.json" {
					close(publishing)
					<-release
				}
				return writeAtomic(path, raw)
			}}
			var once sync.Once
			writer := *r.sync
			writer.Registry = &registryStore{Dir: root, TryLock: func(fd int) error {
				err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
				if errors.Is(err, syscall.EWOULDBLOCK) {
					once.Do(func() { close(waiting) })
				}
				return err
			}}
			mutated := make(chan error, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				var err error
				if operation == "rename" {
					_, err = mutator.Rename(old, newPath)
				} else {
					_, err = mutator.Forget(old)
				}
				mutated <- err
			}()
			await(publishing, "metadata publisher")
			wrote := make(chan error, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				_, err := writer.Write(WriteRequest{File: old, Section: "Work", Body: "must not land", IfHash: read.Hash})
				wrote <- err
			}()
			await(waiting, "writer excluded by real flock")
			releaseOnce.Do(func() { close(release) })
			if err := <-mutated; err != nil {
				t.Fatal(err)
			}
			if err := <-wrote; err == nil {
				t.Fatal("waiting writer reused removed registration")
			}
			if r.read(old) != "## Work\n\noriginal\n" {
				t.Fatal("waiting write changed OLD")
			}
		})
	}
}

func TestForgetExportFailureKeepsCentralAuthorityUntilRetry(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	h := r.hash("Trains")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "latest\n", IfHash: h}); err != nil {
		t.Fatal(err)
	}
	root := r.sync.registryStore().Dir
	failExport := true
	r.sync.Registry = &registryStore{Dir: root, Publish: func(path string, raw []byte) error {
		if filepath.Base(path) == filepath.Base(r.p.State) && failExport {
			return errors.New("injected legacy state export failure")
		}
		return writeAtomic(path, raw)
	}}
	if _, err := r.sync.Forget(r.p.File); err == nil {
		t.Fatal("forget bypassed failed metadata publication")
	}
	if registration, err := r.sync.Lookup(r.p.File); err != nil || registration == nil {
		t.Fatalf("lost central authority: %+v %v", registration, err)
	}
	if _, diff, err := r.sync.DiffText(r.p.File); err != nil || diff != "" {
		t.Fatalf("read stale handoff: %q %v", diff, err)
	}
	failExport = false
	if _, err := r.sync.Forget(r.p.File); err != nil {
		t.Fatal(err)
	}
	if _, diff, err := r.sync.DiffText(r.p.File); err != nil || diff != "" {
		t.Fatalf("retry baseline: %q %v", diff, err)
	}
}

func TestRegisteredDocumentUsesSyncProtocolAndRetainsRenamedSubmit(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "note.md")
	r.put(path, "## Work\n\noperator text\n")
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if err := r.sync.Ledger.Append("ops:0", Entry{ID: "old-submit", From: "ops:0", At: r.now, Head: SubmitPrefix + path}); err != nil {
		t.Fatal(err)
	}
	read, err := r.sync.Read(path, "Work")
	if err != nil {
		t.Fatal(err)
	}
	if _, diff, err := r.sync.DiffText(path); err != nil || !strings.Contains(diff, "operator text") {
		t.Fatalf("diff %q %v", diff, err)
	}
	newPath := filepath.Join(filepath.Dir(path), "moved.md")
	if err := os.Rename(path, newPath); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(time.Minute)
	move, err := r.sync.Rename(path, newPath)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := r.sync.Rename(path, newPath)
	if err != nil || replay.Changed || move.Registration.ID != replay.Registration.ID {
		t.Fatalf("replay %+v %v", replay, err)
	}
	st, err := r.sync.Status(newPath)
	if err != nil || len(st.Owed) != 1 || st.Owed[0].ID != "old-submit" {
		t.Fatalf("owed %+v %v", st, err)
	}
	if _, err := r.sync.Write(WriteRequest{File: newPath, Section: "Work", Body: "answer\n", IfHash: read.Hash, Fold: []string{"old-submit"}}); err != nil {
		t.Fatal(err)
	}
	st, err = r.sync.Status(newPath)
	if err != nil || len(st.Owed) != 0 {
		t.Fatalf("folded %+v %v", st, err)
	}
	if strings.Contains(r.read(newPath), "---") {
		t.Fatal("write added frontmatter")
	}
	if _, err := r.sync.Forget(newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Read(newPath, ""); err == nil {
		t.Fatal("forgotten plain document still admitted")
	}
}

func TestForgetLegacyHandsBackCurrentSnapshotAndHistory(t *testing.T) {
	r := newRig(t)
	r.deliver("before-registration", SubmitPrefix+r.p.File)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	st, err := r.sync.Status(r.p.File)
	if err != nil || len(st.Owed) != 1 {
		t.Fatalf("migration lost horizon %+v %v", st, err)
	}
	h := r.hash("Trains")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "central answer\n", IfHash: h, Fold: []string{"before-registration"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Forget(r.p.File); err != nil {
		t.Fatal(err)
	}
	st, err = r.sync.Status(r.p.File)
	if err != nil || len(st.Owed) != 0 || st.LastWrite == nil {
		t.Fatalf("handoff %+v %v", st, err)
	}
	if _, diff, err := r.sync.DiffText(r.p.File); err != nil || diff != "" {
		t.Fatalf("stale baseline %q %v", diff, err)
	}
}

func TestRenamedLegacyForgetAndReregisterKeepOldSubmitHistory(t *testing.T) {
	r := newRig(t)
	r.deliver("old-submit", SubmitPrefix+r.p.File)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(filepath.Dir(r.p.File), "moved-session.md")
	if err := os.Rename(r.p.File, newPath); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(time.Minute)
	if _, err := r.sync.Rename(r.p.File, newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Forget(newPath); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		st, err := r.sync.Status(newPath)
		if err != nil || len(st.Owed) != 1 || st.Owed[0].ID != "old-submit" {
			t.Fatalf("lost handoff history %+v %v", st, err)
		}
	}
	check()
	if _, err := r.sync.Register(newPath, "ops:0"); err != nil {
		t.Fatal(err)
	}
	check()
}

func TestRegisteredWholeFileWritePreservesEffectiveOwnerWithoutType(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "plain.md")
	r.put(path, "plain\n")
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	read, err := r.sync.Read(path, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.sync.Write(WriteRequest{File: path, Body: "---\nowner: business-team\n---\nanswer\n", IfHash: read.Hash})
	if err != nil {
		t.Fatal(err)
	}
	if result.Owner != "ops:0" {
		t.Fatalf("business metadata rerouted write: %+v", result)
	}
	_, err = r.sync.Write(WriteRequest{File: path, Body: "---\ncouch-owner: pair:1\n---\nreroute\n", IfHash: result.Hash})
	if err == nil {
		t.Fatal("whole-file write changed its effective owner")
	}
}

func TestOperatorLockAgeAdvisesButNeverExpires(t *testing.T) {
	for _, age := range []time.Duration{30 * time.Minute, 30*time.Minute + time.Second} {
		t.Run(age.String(), func(t *testing.T) {
			r := newRig(t)
			h := r.hash("Trains")
			r.put(r.p.Lock, "holder: operator\ntime: "+r.now.Add(-age).Format(time.RFC3339)+"\n")
			_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "answer", IfHash: h})
			wantRefusal(t, err, 3, "operator-lock")
			if strings.Contains(err.Error(), "ping the operator") != (age > 30*time.Minute) {
				t.Fatalf("age %s: %v", age, err)
			}
			if r.read(r.p.File) != statusDoc {
				t.Fatal("aged lock was expired")
			}
		})
	}
}

func TestUnknownOperatorLockTimeStillRefuses(t *testing.T) {
	for _, stamp := range []string{"", "nonsense", "2027-01-01T00:00:00Z"} {
		t.Run(stamp, func(t *testing.T) {
			r := newRig(t)
			h := r.hash("Trains")
			r.put(r.p.Lock, "holder: operator\ntime: "+stamp+"\n")
			_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "answer", IfHash: h})
			wantRefusal(t, err, 3, "operator-lock")
			if strings.Contains(err.Error(), "ping the operator") {
				t.Fatalf("fabricated age: %v", err)
			}
		})
	}
}

func TestEffectiveOwnerDoesNotRouteBusinessMetadata(t *testing.T) {
	for _, c := range []struct{ content, want string }{
		{"plain", "pair:5"},
		{"---\nowner: business-team\n---\nplain", "pair:5"},
		{"---\ncouch-owner: ops:0\nowner: business-team\n---\nplain", "ops:0"},
		{"---\ntype: session-sync\nowner: ops:0\n---\nplain", "ops:0"},
	} {
		if got := EffectiveOwner(c.content, "pair:5"); got != c.want {
			t.Fatalf("owner %q want %q", got, c.want)
		}
	}
}

func TestLookupAbsenceDoesNotRegisterAndForgetAllowsMissingDocument(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "note.md")
	r.put(path, "note\n")
	if got, err := r.sync.Lookup(path); err != nil || got != nil {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Forget(path); err != nil {
		t.Fatal(err)
	}
	if entries, err := r.sync.List(); err != nil || len(entries) != 0 {
		t.Fatalf("list: %+v %v", entries, err)
	}
}

func TestRegisterConfirmsAnUncertainPublicationWithoutResettingHorizon(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "note.md")
	r.put(path, "plain\n")
	root := r.sync.registryStore().Dir
	uncertain := true
	r.sync.Registry = &registryStore{Dir: root, Publish: func(p string, raw []byte) error {
		if err := writeAtomic(p, raw); err != nil {
			return err
		}
		if filepath.Base(p) == "registry.json" && uncertain {
			uncertain = false
			return errors.New("lost directory-sync acknowledgment")
		}
		return nil
	}}
	_, err := r.sync.Register(path, "ops:0")
	wantRefusal(t, err, 1, "registry-uncertain")
	observed, err := r.sync.Lookup(path)
	if err != nil || observed == nil {
		t.Fatalf("published record %+v %v", observed, err)
	}
	originalTime := r.now
	r.now = r.now.Add(time.Hour)
	replay, err := r.sync.Register(path, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	if replay.Changed || replay.Registration.ID != observed.ID || !replay.Registration.RegisteredAt.Equal(originalTime) {
		t.Fatalf("retry reset record: %+v", replay)
	}
}

func TestRegistrySymlinkAliasAndPostRenameMissingLeaf(t *testing.T) {
	r := newRig(t)
	dir := filepath.Dir(r.p.File)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(alias, "note.md")
	r.put(old, "## Work\n\nplain\n")
	first, err := r.sync.Register(old, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	realPath, err := filepath.EvalSymlinks(filepath.Join(dir, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.sync.Register(realPath, "ops:0")
	if err != nil || second.Changed || second.Registration.ID != first.Registration.ID {
		t.Fatalf("canonical retry %+v %v", second, err)
	}
	leafAlias := filepath.Join(t.TempDir(), "leaf-alias.md")
	if err := os.Symlink(realPath, leafAlias); err != nil {
		t.Fatal(err)
	}
	additional, err := r.sync.Register(leafAlias, "ops:0")
	if err != nil || !additional.Changed || additional.Registration.ID != first.Registration.ID {
		t.Fatalf("additional alias %+v %v", additional, err)
	}
	retry, err := r.sync.Register(leafAlias, "ops:0")
	if err != nil || retry.Changed || retry.Registration.ID != first.Registration.ID {
		t.Fatalf("alias retry %+v %v", retry, err)
	}
	if _, err := r.sync.Read(old, ""); err != nil {
		t.Fatal(err)
	}
	if _, diff, err := r.sync.DiffText(realPath); err != nil || diff != "" {
		t.Fatalf("split baseline %q %v", diff, err)
	}
	newPath := filepath.Join(alias, "moved.md")
	if err := os.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(time.Second)
	move, err := r.sync.Rename(old, newPath)
	if err != nil || move.Registration.ID != first.Registration.ID {
		t.Fatalf("missing old alias %+v %v", move, err)
	}
}

func TestLegacyLeafSymlinkImportsAndHandsBackItsOwnBaseline(t *testing.T) {
	r := newRig(t)
	link := filepath.Join(filepath.Dir(r.p.File), "linked.md")
	if err := os.Symlink(r.p.File, link); err != nil {
		t.Fatal(err)
	}
	r.put(PathsOf(link).Shadow, "an older linked baseline\n")
	_, before, err := r.sync.DiffText(link)
	if err != nil || before == "" {
		t.Fatalf("control diff %q %v", before, err)
	}
	if _, err := r.sync.Register(link, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if _, after, err := r.sync.DiffText(link); err != nil || after != before {
		t.Fatalf("import lost alias baseline: %q => %q (%v)", before, after, err)
	}
	if _, err := r.sync.Read(link, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Forget(link); err != nil {
		t.Fatal(err)
	}
	if _, diff, err := r.sync.DiffText(link); err != nil || diff != "" {
		t.Fatalf("handoff revived alias baseline: %q %v", diff, err)
	}
}

func TestRegisteredLeafAliasLockAndAliasRenameKeepOperatorAuthority(t *testing.T) {
	r := newRig(t)
	link := filepath.Join(filepath.Dir(r.p.File), "note-link.md")
	if err := os.Symlink(r.p.File, link); err != nil {
		t.Fatal(err)
	}
	registered, err := r.sync.Register(link, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	h := r.hash("Trains")
	checkLock := func(alias string) {
		t.Helper()
		r.put(alias+".lock", "holder: operator\n")
		_, err := r.sync.Write(WriteRequest{File: registered.Registration.File, Section: "Trains", Body: "blocked", IfHash: h})
		wantRefusal(t, err, 3, "operator-lock")
		if err := os.Remove(alias + ".lock"); err != nil {
			t.Fatal(err)
		}
	}
	checkLock(link)
	renamed := filepath.Join(filepath.Dir(link), "moved-link.md")
	if err := os.Rename(link, renamed); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(time.Second)
	move, err := r.sync.Rename(link, renamed)
	if err != nil || move.Registration.ID != registered.Registration.ID {
		t.Fatalf("alias rename %+v %v", move, err)
	}
	replay, err := r.sync.Rename(link, renamed)
	if err != nil || replay.Changed {
		t.Fatalf("alias replay %+v %v", replay, err)
	}
	checkLock(renamed)
}
