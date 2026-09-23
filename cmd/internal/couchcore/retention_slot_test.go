package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func slotRetentionFixture(t *testing.T) (*ThreadStore, *ThreadStore, *storagegc.Coordinator, ThreadRecord) {
	t.Helper()
	global, c := retentionStore(t)
	local := testLocalThreadStore(t)
	local.namespace = global.namespace
	local.coordinator = c
	if err := os.MkdirAll(local.slot.PrimaryRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := global.withLock(func() error {
		raw, err := json.Marshal(threadManifest{SchemaVersion: 2, Threads: []ThreadAddress{}, SlotRepositories: []string{local.slot.PrimaryRoot}})
		if err != nil {
			return err
		}
		return os.WriteFile(global.manifestPath(), raw, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	r := actionableTestThread("couch-0000000000000001", c.Now())
	r.StartingPath = local.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	created, err := local.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	return global, local, c, created
}

func TestSlotRetentionSnapshotFindsLocalOnlyCurrent(t *testing.T) {
	global, local, c, r := slotRetentionFixture(t)
	if err := c.WithReadLock(context.Background(), func(l *storagegc.Locked) error {
		got, err := ReadStoreRetention(global.namespace, c, l)
		if err != nil {
			return err
		}
		if len(got.Visible) != 1 || got.Visible[0] != r.Address {
			t.Fatalf("local reference lost: %+v", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(local.manifestPath()); !os.IsNotExist(err) {
		t.Fatal("preview initialized local manifest")
	}
}

func TestSlotRetentionSnapshotBlocksUncertainMembership(t *testing.T) {
	for _, kind := range []string{"missing-current", "bad-current", "missing-root", "missing-global-manifest", "missing-local-store"} {
		t.Run(kind, func(t *testing.T) {
			global, local, c, _ := slotRetentionFixture(t)
			switch kind {
			case "missing-current":
				os.Remove(filepath.Join(local.root, "thread.json"))
			case "bad-current":
				os.WriteFile(filepath.Join(local.root, "thread.json"), []byte("broken"), 0600)
			case "missing-root":
				os.RemoveAll(local.slot.PrimaryRoot)
			case "missing-global-manifest":
				os.Remove(global.manifestPath())
			case "missing-local-store":
				os.RemoveAll(local.root)
			}
			if err := c.WithReadLock(context.Background(), func(l *storagegc.Locked) error {
				_, err := ReadStoreRetention(global.namespace, c, l)
				if err == nil {
					t.Fatal("uncertain membership accepted")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if kind == "missing-local-store" {
				if _, err := os.Stat(local.root); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("preview initialized missing .couch")
				}
			}
		})
	}
}

func slotRetentionArchive(t *testing.T) (*ThreadStore, *ThreadStore, *storagegc.Coordinator, ArchiveDetachRequest) {
	t.Helper()
	global, local, c, r := slotRetentionFixture(t)
	if err := local.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	grace, err := readTestArchiveGrace(local, r.Address)
	if err != nil {
		t.Fatal(err)
	}
	next := r
	next.Address.Tag = "couch-0000000000000002"
	next.Revision = 1
	if _, err := local.CreateThread(next); err != nil {
		t.Fatal(err)
	}
	return global, local, c, ArchiveDetachRequest{OperationID: "local-operation", Address: r.Address, RecordHash: grace.RecordHash, ArchivedAt: grace.ArchivedAt, SlotEnvironment: local.slot.EnvironmentRoot}
}

func TestSlotRetentionEntrypointsRouteIndependentOfSnapshot(t *testing.T) {
	for _, op := range []string{"onboard", "recover", "detach", "forget"} {
		t.Run(op, func(t *testing.T) {
			global, local, c, request := slotRetentionArchive(t)
			switch op {
			case "onboard":
				if err := os.Remove(local.archiveGracePath(request.Address)); err != nil {
					t.Fatal(err)
				}
			case "recover":
				local.hooks.AfterJournal = func() error { return errors.New("injected crash") }
				_, err := local.updateExistingThread(ThreadAddress{RepoScope: request.Address.RepoScope, Tag: "couch-0000000000000002"}, 1, func(r *ThreadRecord) error { r.Name = "recovered"; return nil })
				if err == nil {
					t.Fatal("fault not reached")
				}
			case "forget":
				if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return local.DetachArchive(l, request) }); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
				switch op {
				case "onboard":
					return OnboardStoreArchiveGrace(context.Background(), global.namespace, c, l, []ThreadAddress{request.Address})
				case "recover":
					return RecoverStoreRetention(context.Background(), global.namespace, c, l)
				case "detach":
					return DetachStoreArchive(global.namespace, c, l, request)
				default:
					return ForgetStoreArchiveReceipt(global.namespace, c, l, request)
				}
			}); err != nil {
				t.Fatal(err)
			}
			switch op {
			case "onboard":
				if _, err := readTestArchiveGrace(local, request.Address); err != nil {
					t.Fatal(err)
				}
			case "recover":
				if _, err := os.Stat(local.journalPath()); !os.IsNotExist(err) {
					t.Fatal("local journal not recovered")
				}
			case "detach":
				if _, err := os.Stat(local.archivePath(request.Address)); !os.IsNotExist(err) {
					t.Fatal("local archive survived detach")
				}
			case "forget":
				if _, err := os.Stat(local.archiveReceiptPath(request.OperationID)); !os.IsNotExist(err) {
					t.Fatal("local receipt survived forget")
				}
			}
		})
	}
}

func TestSlotArchiveLocatorAndReceiptCannotReachReplacement(t *testing.T) {
	global, local, c, request := slotRetentionArchive(t)
	wrong := request
	wrong.SlotEnvironment = filepath.Join(filepath.Dir(request.SlotEnvironment), "pair-slot99")
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return DetachStoreArchive(global.namespace, c, l, wrong) }); err == nil {
		t.Fatal("unenrolled locator accepted")
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return DetachStoreArchive(global.namespace, c, l, request) }); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicBytes(local.archivePath(request.Address), []byte("new owner")); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
		if err := DetachStoreArchive(global.namespace, c, l, request); err != nil {
			return err
		}
		return ForgetStoreArchiveReceipt(global.namespace, c, l, request)
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(local.archivePath(request.Address))
	if err != nil || string(raw) != "new owner" {
		t.Fatal("stale receipt touched replacement")
	}
}

func TestSlotRetentionEveryMutationBlocksMissingCurrent(t *testing.T) {
	for _, op := range []string{"onboard", "recover", "detach", "forget"} {
		t.Run(op, func(t *testing.T) {
			global, local, c, request := slotRetentionArchive(t)
			if err := os.Remove(filepath.Join(local.root, "thread.json")); err != nil {
				t.Fatal(err)
			}
			err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
				switch op {
				case "onboard":
					return OnboardStoreArchiveGrace(context.Background(), global.namespace, c, l, []ThreadAddress{request.Address})
				case "recover":
					return RecoverStoreRetention(context.Background(), global.namespace, c, l)
				case "detach":
					return DetachStoreArchive(global.namespace, c, l, request)
				default:
					return ForgetStoreArchiveReceipt(global.namespace, c, l, request)
				}
			})
			if err == nil {
				t.Fatal("mutation accepted missing current")
			}
			if _, err := os.Stat(local.archivePath(request.Address)); err != nil {
				t.Fatal("failed operation removed archive")
			}
		})
	}
}

func TestSlotRestoreRoutesArchivedConversation(t *testing.T) {
	global, local, _, request := slotRetentionArchive(t)
	if err := global.RestoreThread(request.Address); err == nil {
		t.Fatal("restore replaced current conversation")
	}
	if err := os.Remove(filepath.Join(local.root, "thread.json")); err != nil {
		t.Fatal(err)
	}
	if err := global.RestoreThread(request.Address); err != nil {
		t.Fatal(err)
	}
	record, err := local.GetThread(request.Address)
	if err != nil || record.Address != request.Address {
		t.Fatalf("local restore %+v %v", record, err)
	}
	if _, err := os.Stat(global.recordPath(request.Address)); !os.IsNotExist(err) {
		t.Fatal("restore created global owner")
	}
}
