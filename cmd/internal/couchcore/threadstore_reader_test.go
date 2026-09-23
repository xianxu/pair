package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalCurrentReadersRejectSymlink(t *testing.T) {
	for _, operation := range []string{"get", "metadata", "snapshot", "locked", "start", "park"} {
		t.Run(operation, func(t *testing.T) {
			s := testLocalThreadStore(t)
			r := validThreadRecord(t)
			r.StartingPath = s.slot.WorktreeRoot
			r.WorkingPath = r.StartingPath
			if operation == "park" {
				profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
				r.Reservation = false
				r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "park-helper", State: IncarnationLive, LaunchProfile: &profile}}
			}
			if operation == "start" {
				profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
				r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "helper", State: IncarnationCreating, RepoIdentity: s.slot.RepoIdentity, Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 43, OwnerIdentity: "owner", LaunchProfile: &profile}}}
			}
			r, err := s.CreateThread(r)
			if err != nil {
				t.Fatal(err)
			}
			path := s.recordPath(r.Address)
			raw, _ := os.ReadFile(path)
			outside := filepath.Join(t.TempDir(), "outside.json")
			if err := os.WriteFile(outside, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "get":
				_, err = s.GetThread(r.Address)
			case "metadata":
				_, err = s.ApplyThreadMetadata(r.Address, r.Revision, ThreadMetadataPatch{})
			case "snapshot":
				_, err = s.Snapshot()
			case "locked":
				_, err = s.readThreadLocked(r.Address)
			case "start":
				_, err = s.AdvanceStart(r.Address, r.Revision, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
			case "park":
				_, err = s.BeginPark(r.Address, r.Revision, ParkIdentity{Nonce: "park-0123456789abcdef", Address: r.Address, PID: 42, ProcessIdentity: "park-helper"})
			}
			if err == nil {
				t.Fatal("accepted symlink current record")
			}
			after, _ := os.ReadFile(outside)
			if !bytes.Equal(after, raw) {
				t.Fatal("outside target changed")
			}
			info, err := os.Lstat(path)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				t.Fatal("mutated symlink current record")
			}
		})
	}
}

func TestLocalJournalRejectsSymlinkAuthorityAndTargets(t *testing.T) {
	for _, kind := range []string{"journal", "current", "parent"} {
		t.Run(kind, func(t *testing.T) {
			s := testLocalThreadStore(t)
			if err := os.MkdirAll(s.root, 0700); err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			foreign := filepath.Join(outside, "thread.json")
			before := []byte("foreign")
			os.WriteFile(foreign, before, 0600)
			changed := []byte("changed")
			entry := storeJournalEntry{Path: "thread.json", Expected: &before, After: &changed}
			switch kind {
			case "journal":
				entry.Expected = nil
			case "current":
				os.Symlink(foreign, filepath.Join(s.root, "thread.json"))
			case "parent":
				os.Symlink(outside, filepath.Join(s.root, "archive"))
				entry.Path = "archive/thread.json"
			}
			journal, err := assignStoreJournalNonce(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{entry}})
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(journal)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "journal" {
				foreign = filepath.Join(outside, "journal.json")
				before = raw
				os.WriteFile(foreign, raw, 0600)
				os.Symlink(foreign, s.journalPath())
			} else {
				os.WriteFile(s.journalPath(), raw, 0600)
			}
			if err := s.RecoverStoreJournal(); err == nil {
				t.Fatal("accepted symlink journal authority or target")
			}
			after, _ := os.ReadFile(foreign)
			if !bytes.Equal(before, after) {
				t.Fatal("foreign target changed")
			}
		})
	}
}

func TestLocalReaderRejectsNonregularAndOversizedMetadata(t *testing.T) {
	for _, kind := range []string{"directory", "fifo", "oversized-journal", "environment-sibling"} {
		t.Run(kind, func(t *testing.T) {
			s := testLocalThreadStore(t)
			if err := os.MkdirAll(s.root, 0700); err != nil {
				t.Fatal(err)
			}
			path := s.recordPath(ThreadAddress{})
			switch kind {
			case "environment-sibling":
				path = filepath.Join(s.slot.EnvironmentRoot, "sibling.json")
				if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := unix.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "oversized-journal":
				path = s.journalPath()
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.Truncate(localJournalLimit + 1); err != nil {
					t.Fatal(err)
				}
				f.Close()
			}
			if _, err := s.readPayload(path); err == nil {
				t.Fatal("accepted unsafe metadata")
			}
		})
	}
}

func TestLocalJournalRecoversImagesLargerThanPayloadReadLimit(t *testing.T) {
	s := testLocalThreadStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	before := bytes.Repeat([]byte("b"), 3<<20)
	after := bytes.Repeat([]byte("a"), 3<<20)
	path := filepath.Join(s.root, "continuation.md")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("interrupted after journal publication")
	s.hooks.AfterJournal = func() error { return interrupted }
	err := s.withLock(func() error {
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{{Path: "continuation.md", Expected: &before, After: &after}}})
	})
	if !errors.Is(err, interrupted) {
		t.Fatalf("commit: %v", err)
	}
	info, err := os.Stat(s.journalPath())
	if err != nil || info.Size() <= localPayloadLimit {
		t.Fatalf("journal: %v %v", info, err)
	}
	s.hooks.AfterJournal = nil
	if err := s.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	got, err := s.readPayload(path)
	if err != nil || !bytes.Equal(got, after) {
		t.Fatalf("recovered image: %v", err)
	}
}

func TestLocalJournalRefusesOversizedImagesBeforePublication(t *testing.T) {
	s := testLocalThreadStore(t)
	oversized := make([]byte, localPayloadLimit+1)
	err := s.withLock(func() error {
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{{Path: "thread.json", After: &oversized}}})
	})
	if err == nil {
		t.Fatal("oversized image accepted")
	}
	if _, err := os.Lstat(s.journalPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unrecoverable journal published: %v", err)
	}
}
