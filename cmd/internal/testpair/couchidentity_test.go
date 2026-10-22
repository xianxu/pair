package testpair

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
)

func TestCouchIdentitySeedBounds(t *testing.T) {
	for _, tc := range []struct {
		pid  int64
		seq  uint64
		want uint64
	}{
		{1, 1, 1<<63 | 1<<32 | 1},
		{math.MaxInt32, math.MaxUint32 - 1, math.MaxUint64 - 1},
	} {
		got, err := couchIdentitySeed(tc.pid, tc.seq)
		if err != nil || got != tc.want {
			t.Fatalf("seed(%d, %d) = %d, %v; want %d", tc.pid, tc.seq, got, err, tc.want)
		}
	}
	for _, tc := range []struct {
		pid int64
		seq uint64
	}{
		{0, 1}, {-1, 1}, {math.MaxInt32 + 1, 1}, {1, 0}, {1, math.MaxUint32 + 1},
		{math.MaxInt32, math.MaxUint32},
	} {
		if got, err := couchIdentitySeed(tc.pid, tc.seq); err == nil || got != 0 {
			t.Errorf("invalid seed(%d, %d) = %d, %v", tc.pid, tc.seq, got, err)
		}
	}
}

func TestCouchIdentitySequenceExhaustion(t *testing.T) {
	s := couchIdentitySequence{last: math.MaxUint32 - 1}
	if got, err := s.next(); got != math.MaxUint32 || err != nil {
		t.Fatalf("last sequence = %d, %v", got, err)
	}
	for range 2 {
		if got, err := s.next(); got != 0 || err == nil {
			t.Fatalf("exhausted sequence = %d, %v", got, err)
		}
	}
}

func TestSeedCouchIdentityExclusive(t *testing.T) {
	host := filepath.Join(t.TempDir(), "host")
	c, err := SeedCouchIdentity(host)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(host, "couch-identities.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		SchemaVersion int               `json:"schema_version"`
		NextC         uint64            `json:"next_c"`
		Stores        []json.RawMessage `json:"stores"`
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	if registry.SchemaVersion != 1 || registry.NextC != c || registry.Stores == nil || len(registry.Stores) != 0 {
		t.Fatalf("invalid registry: %s", raw)
	}
	for path, want := range map[string]os.FileMode{host: 0700, path: 0600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s permissions = %o; want %o", path, info.Mode().Perm(), want)
		}
	}
	if got, err := SeedCouchIdentity(host); err == nil || got != 0 {
		t.Fatalf("reseed = %d, %v", got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(raw) {
		t.Fatal("reseed overwrote registry")
	}
}

func TestSeedCouchIdentityRefusesExistingAuthority(t *testing.T) {
	for _, residue := range []string{"", "broken registry"} {
		host := t.TempDir()
		path := filepath.Join(host, "couch-identities.json")
		if err := os.WriteFile(path, []byte(residue), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := SeedCouchIdentity(host); err == nil {
			t.Fatal("accepted existing authority")
		}
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != residue {
			t.Fatalf("changed residue: %q, %v", raw, err)
		}
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := SeedCouchIdentity(filepath.Join(blocked, "host")); err == nil {
		t.Fatal("ignored mkdir error")
	}
	if _, err := SeedCouchIdentity(""); err == nil {
		t.Fatal("accepted empty host")
	}
}

func TestSeedCouchIdentityIndependentAuthorities(t *testing.T) {
	seen := map[uint64]bool{}
	for range 2 {
		root := t.TempDir()
		host := filepath.Join(root, "host")
		c, err := SeedCouchIdentity(host)
		if err != nil {
			t.Fatal(err)
		}
		if seen[c] {
			t.Fatalf("duplicate C: %d", c)
		}
		seen[c] = true
		store := couchidentity.IdentityStore{HostDir: host, StoreDir: filepath.Join(root, "store")}
		for n := uint64(1); n <= 2; n++ {
			got, err := store.Allocate(context.Background(), couchidentity.AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "fixture"})
			if err != nil {
				t.Fatal(err)
			}
			if got.C != c || got.N != n || got.M != n {
				t.Fatalf("allocation = %+v; want C=%d N=M=%d", got, c, n)
			}
		}
	}
}

func TestSeedCouchIdentityConcurrent(t *testing.T) {
	for _, count := range []int{8, 16} {
		root := t.TempDir()
		type result struct {
			c   uint64
			err error
		}
		results := make(chan result, count)
		var wg sync.WaitGroup
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c, err := SeedCouchIdentity(filepath.Join(root, string(rune('a'+i))))
				results <- result{c, err}
			}()
		}
		wg.Wait()
		close(results)
		seen := map[uint64]bool{}
		for r := range results {
			if r.err != nil {
				t.Fatal(r.err)
			}
			if r.c>>63 != 1 || uint64(os.Getpid()) != (r.c>>32)&math.MaxInt32 || r.c&math.MaxUint32 == 0 || seen[r.c] {
				t.Fatalf("invalid or duplicate C: %d", r.c)
			}
			seen[r.c] = true
		}
	}
}
