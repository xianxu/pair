package couchcore

import (
	"errors"
	"os"
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
