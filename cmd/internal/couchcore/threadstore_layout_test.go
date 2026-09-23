package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func testLocalThreadStore(t *testing.T) *ThreadStore {
	t.Helper()
	ns := testCouchNamespace(t)
	fleet, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(fleet, "worktree", "pair-slot1")
	slot := SlotIdentity{Repo: "pair", RepoIdentity: filepath.Join(fleet, "pair", ".git"), PrimaryRoot: filepath.Join(fleet, "pair"), EnvironmentRoot: env, WorktreeRoot: filepath.Join(env, "pair"), Number: 1}
	if err := os.MkdirAll(slot.WorktreeRoot, 0700); err != nil {
		t.Fatal(err)
	}
	return newSlotThreadStore(ns, slot)
}

func TestLocalThreadStoreSingleCurrentAndSharedMutation(t *testing.T) {
	s := testLocalThreadStore(t)
	r := validThreadRecord(t)
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	created, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.root, "thread.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.manifestPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("local manifest exists: %v", err)
	}
	other := r
	other.Address.Tag = "couch-1111111111111111"
	if _, err := s.CreateThread(other); err == nil {
		t.Fatal("second current conversation accepted")
	}
	if _, err := s.GetThread(other.Address); err == nil {
		t.Fatal("wrong address read current")
	}
	if _, err := s.updateExistingThread(other.Address, r.Revision, func(r *ThreadRecord) error { r.Name = "wrong"; return nil }); err == nil {
		t.Fatal("wrong address mutated current")
	}
	updated, err := s.updateExistingThread(created.Address, created.Revision, func(r *ThreadRecord) error { r.Name = "local"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot()
	if err != nil || len(snap.Records) != 1 || snap.Records[0].Name != updated.Name {
		t.Fatalf("snapshot=%+v, %v", snap, err)
	}
}

func TestLocalThreadStoreRejectsCorruptMembership(t *testing.T) {
	s := testLocalThreadStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte("{broken")
	if err := os.WriteFile(filepath.Join(s.root, "thread.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Snapshot(); err == nil {
		t.Fatal("corruption treated as empty membership")
	}
	r := validThreadRecord(t)
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	if _, err := s.CreateThread(r); err == nil {
		t.Fatal("corruption overwritten")
	}
	got, _ := os.ReadFile(filepath.Join(s.root, "thread.json"))
	if string(got) != string(raw) {
		t.Fatal("corrupt evidence changed")
	}
}

func TestLocalThreadStoreRejectsForeignPath(t *testing.T) {
	s := testLocalThreadStore(t)
	r := validThreadRecord(t)
	if _, err := s.CreateThread(r); err == nil {
		t.Fatal("foreign path installed as current")
	}
}

func TestLocalThreadStoreArchiveRejectsOtherCurrentTag(t *testing.T) {
	s := testLocalThreadStore(t)
	r := validThreadRecord(t)
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	if _, err := s.CreateThread(r); err != nil {
		t.Fatal(err)
	}
	other := r.Address
	other.Tag = "couch-1111111111111111"
	if err := s.ArchiveThread(other); err == nil {
		t.Fatal("archived another current tag")
	}
	if _, err := s.GetThread(r.Address); err != nil {
		t.Fatalf("current lost: %v", err)
	}
	if err := s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot()
	if err != nil || len(snap.Records) != 0 {
		t.Fatalf("archive membership=%+v %v", snap, err)
	}
	if err := s.RestoreThread(r.Address); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetThread(r.Address); err != nil {
		t.Fatal(err)
	}
}

func TestLocalThreadStoreRefusesSymlinkBackend(t *testing.T) {
	s := testLocalThreadStore(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, s.root); err != nil {
		t.Fatal(err)
	}
	r := validThreadRecord(t)
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	if _, err := s.CreateThread(r); err == nil {
		t.Fatal("symlink backend accepted")
	}
	files, err := os.ReadDir(outside)
	if err != nil || len(files) != 0 {
		t.Fatalf("external writes: %v %v", files, err)
	}
}

func TestSlotManifestVersionPreservedByPrimaryCreate(t *testing.T) {
	s, _ := newTestThreadStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"schema_version":2,"generation":1,"threads":[],"slot_repositories":["/physical/pair"]}`)
	if err := os.WriteFile(s.manifestPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateThread(validThreadRecord(t)); err != nil {
		t.Fatal(err)
	}
	var manifest threadManifest
	got, err := os.ReadFile(s.manifestPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || len(manifest.SlotRepositories) != 1 {
		t.Fatalf("lost format fence: %s", got)
	}
}

func TestLocalSnapshotDoesNotInitializeAbsentState(t *testing.T) {
	s := testLocalThreadStore(t)
	if _, err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read initialized local state: %v", err)
	}
}

func TestLocalSuccessfulStartRecoversRecordAndPreferencesTogether(t *testing.T) {
	s := testLocalThreadStore(t)
	r := validThreadRecord(t)
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "helper", State: IncarnationCreating, RepoIdentity: s.slot.RepoIdentity, Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 43, OwnerIdentity: "owner", LaunchProfile: &profile}}}
	created, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	crash := errors.New("crash after record publication")
	s.hooks.AfterTarget = func(index int) error {
		if index == 0 {
			return crash
		}
		return nil
	}
	_, err = s.AdvanceStart(created.Address, created.Revision, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if !errors.Is(err, crash) {
		t.Fatalf("advance err = %v", err)
	}
	s.hooks.AfterTarget = nil
	got, err := s.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Incarnations) != 1 || got.Incarnations[0].State != IncarnationLive {
		t.Fatalf("start not recovered: %+v", got)
	}
	pref, found, err := s.GetPathLaunchPreference(s.slot.RepoIdentity, r.StartingPath)
	if err != nil || !found || pref.LastAgent != "codex" {
		t.Fatalf("preference not recovered: %+v %v %v", pref, found, err)
	}
	if _, err := os.Stat(filepath.Join(s.root, "preferences.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.manifestPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("synthetic manifest persisted: %v", err)
	}
}
