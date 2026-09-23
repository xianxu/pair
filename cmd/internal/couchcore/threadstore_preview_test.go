package couchcore

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestThreadStorePreviewDoesNotInitialize(t *testing.T) {
	s, _ := newTestThreadStore(t)
	snapshot, err := s.PreviewSnapshot()
	if err != nil || len(snapshot.Records) != 0 {
		t.Fatalf("preview=%+v %v", snapshot, err)
	}
	_, found, err := s.PreviewPathLaunchPreference("repo", "/repo")
	if err != nil || found {
		t.Fatalf("preference=%v %v", found, err)
	}
	if _, err := os.Lstat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created root: %v", err)
	}
}
func TestThreadStorePreviewDoesNotRecoverJournal(t *testing.T) {
	s, _ := newTestThreadStore(t)
	fail := errors.New("interrupted")
	s.hooks.AfterJournal = func() error { return fail }
	if _, err := s.CreateThread(validThreadRecord(t)); !errors.Is(err, fail) {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewSnapshot(); err == nil {
		t.Fatal("preview accepted pending journal")
	}
	after, err := os.ReadFile(s.journalPath())
	if err != nil || string(before) != string(after) {
		t.Fatalf("preview recovered journal: %v", err)
	}
}

func TestSlotPreferenceRefusesSymlinkedPayload(t *testing.T) {
	s := testLocalThreadStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	external := s.root + "-external-preferences"
	if err := os.WriteFile(external, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, s.pathLaunchPreferencePath(s.slot.RepoIdentity, s.slot.WorktreeRoot)); err != nil {
		t.Fatal(err)
	}
	_, _, err := s.GetPathLaunchPreference(s.slot.RepoIdentity, s.slot.WorktreeRoot)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("unsafe preference read: %v", err)
	}
}
