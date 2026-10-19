package couchcore

import (
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/lifetimelease"
)

// lifetimeLeaseEnvForTest is the lease entry a launch of a thread starting at
// root carries, and it checks that the launch prepared the file.
func lifetimeLeaseEnvForTest(t *testing.T, namespace, root string) string {
	t.Helper()
	address, err := NewSlotAddress(SlotIdentity{}, root)
	if err != nil {
		t.Fatal(err)
	}
	path, err := lifetimelease.Path(namespace, address.Key())
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("launch did not prepare the lifetime lease %s: %v", path, err)
	}
	return lifetimelease.Env + "=" + path
}

// One slot, one lease: two threads launched from one slot share it, and
// another slot's lease is different (pair#695 P1).
func TestLifetimeLeaseIsPerSlot(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	a, err := env.Couch.lifetimeLeaseEnv(ThreadRecord{StartingPath: "/w/pair-slot1/pair", Address: ThreadAddress{Tag: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := env.Couch.lifetimeLeaseEnv(ThreadRecord{StartingPath: "/w/pair-slot1/pair", Address: ThreadAddress{Tag: "b"}})
	c, _ := env.Couch.lifetimeLeaseEnv(ThreadRecord{StartingPath: "/w/pair-slot2/pair", Address: ThreadAddress{Tag: "a"}})
	if a != b || a == c || !strings.HasPrefix(a, lifetimelease.Env+"="+env.Dir) {
		t.Fatalf("leases a=%q b=%q c=%q", a, b, c)
	}
	if _, err := env.Couch.lifetimeLeaseEnv(ThreadRecord{StartingPath: "relative"}); err == nil {
		t.Fatal("a thread without a clean absolute starting path got a lease")
	}
}
