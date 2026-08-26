package couchcore

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestAllocateThreadTagClaimsCryptographicOpaqueAddress(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record, err := store.AllocateThreadTag(
		"0123456789abcdef", ns.Dir(), time.Date(2026, 8, 26, 13, 0, 0, 0, time.UTC),
		bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}),
	)
	if err != nil {
		t.Fatalf("AllocateThreadTag: %v", err)
	}
	want := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	if record.Address != want {
		t.Fatalf("address = %+v, want %+v", record.Address, want)
	}
	if _, err := NewThreadStore(ns).GetThread(want); err != nil {
		t.Fatalf("opaque address was returned before its durable claim: %v", err)
	}
}

func TestAllocateThreadTagRetriesCollisionWithoutReplacingExistingThread(t *testing.T) {
	store, ns := newTestThreadStore(t)
	existing := validThreadRecord(t)
	existing.Address = ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000000"}
	existing.StartingPath, existing.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(existing)
	if err != nil {
		t.Fatalf("CreateThread(existing): %v", err)
	}

	entropy := append(make([]byte, 8), []byte{1, 1, 1, 1, 1, 1, 1, 1}...)
	allocated, err := store.AllocateThreadTag(existing.Address.RepoScope, ns.Dir(), existing.CreatedAt.Add(time.Second), bytes.NewReader(entropy))
	if err != nil {
		t.Fatalf("AllocateThreadTag: %v", err)
	}
	if allocated.Address.Tag != "couch-0101010101010101" {
		t.Fatalf("allocated tag = %q", allocated.Address.Tag)
	}
	unchanged, err := store.GetThread(existing.Address)
	if err != nil {
		t.Fatalf("GetThread(existing): %v", err)
	}
	if unchanged.CreatedAt != created.CreatedAt || unchanged.Revision != created.Revision {
		t.Fatalf("collision replaced existing thread: before=%+v after=%+v", created, unchanged)
	}
}

func TestAllocateThreadTagFailsAfterEightCollisionsAndOnEntropyError(t *testing.T) {
	store, ns := newTestThreadStore(t)
	existing := validThreadRecord(t)
	existing.Address = ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000000"}
	existing.StartingPath, existing.WorkingPath = ns.Dir(), ns.Dir()
	if _, err := store.CreateThread(existing); err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if _, err := store.AllocateThreadTag(existing.Address.RepoScope, ns.Dir(), existing.CreatedAt, bytes.NewReader(make([]byte, 8*8))); err == nil {
		t.Fatal("eight collisions must fail, not reuse an existing tag")
	}

	broken := errorReader{err: errors.New("entropy unavailable")}
	if _, err := store.AllocateThreadTag("fedcba9876543210", ns.Dir(), existing.CreatedAt, broken); !errors.Is(err, broken.err) {
		t.Fatalf("entropy failure err = %v", err)
	}
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
