package artifactpath

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestTerminalLeasePathsUseCanonicalSocket(t *testing.T) {
	cache, _ := ResolvePairCache("/home/test")
	socket := "/tmp/sockets/session"
	lock, err := cache.TerminalLease(socket)
	if err != nil {
		t.Fatal(err)
	}
	key := sha256.Sum256([]byte(socket))
	if want := fmt.Sprintf("/home/test/.cache/pair/session-locks-v2/%02x.lock", key[0]); lock != want {
		t.Fatalf("lock = %s, want %s", lock, want)
	}
	for _, bad := range []string{"relative", "/tmp/../session", "/", "/tmp/s\x00"} {
		if _, err := cache.TerminalLease(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if _, err := (PairCachePaths{}).TerminalLease(socket); err == nil {
		t.Fatal("accepted unresolved cache")
	}
}

func TestTerminalLeasePathsHaveFixedBound(t *testing.T) {
	cache, _ := ResolvePairCache("/home/test")
	locks := map[string]bool{}
	for i := 0; i < 10000; i++ {
		socket := fmt.Sprintf("/tmp/sockets/session-%d", i)
		lock, err := cache.TerminalLease(socket)
		if err != nil {
			t.Fatal(err)
		}
		again, err := cache.TerminalLease(socket)
		if err != nil || lock != again {
			t.Fatalf("unstable key: %s %s %v", lock, again, err)
		}
		locks[lock] = true

	}
	if len(locks) > 256 {
		t.Fatalf("%d unique lock paths exceeds fixed bound 256", len(locks))
	}
}
