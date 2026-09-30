package couchidentity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) IdentityStore {
	t.Helper()
	return IdentityStore{HostDir: filepath.Join(t.TempDir(), "host"), StoreDir: filepath.Join(t.TempDir(), "store")}
}
func alloc(t *testing.T, s IdentityStore) AllocationResult {
	t.Helper()
	r, e := s.Allocate(context.Background(), AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "repo"})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestStoreRestoredLocalAndMoved(t *testing.T) {
	s := fixture(t)
	a := alloc(t, s)
	saved, _ := os.ReadFile(filepath.Join(s.StoreDir, "identities.json"))
	b := alloc(t, s)
	if a.C != b.C || b.N != 2 || b.M != 2 {
		t.Fatal(a, b)
	}
	os.WriteFile(filepath.Join(s.StoreDir, "identities.json"), saved, 0600)
	c := alloc(t, s)
	if c.N != 3 || c.M != 3 {
		t.Fatal(c)
	}
	moved := filepath.Join(t.TempDir(), "moved")
	if e := os.Rename(s.StoreDir, moved); e != nil {
		t.Fatal(e)
	}
	s.StoreDir = moved
	d := alloc(t, s)
	if d.C != 2 || d.N != 1 || d.M != 1 {
		t.Fatal(d)
	}
}
func TestStoreRefusesLostAuthority(t *testing.T) {
	for _, which := range []string{"host", "local"} {
		t.Run(which, func(t *testing.T) {
			s := fixture(t)
			alloc(t, s)
			p := filepath.Join(s.StoreDir, "identities.json")
			if which == "host" {
				p = filepath.Join(s.HostDir, "couch-identities.json")
			}
			os.Remove(p)
			_, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true})
			if e == nil || !strings.Contains(e.Error(), p) {
				t.Fatalf("missing exact authority diagnostic: %v", e)
			}
		})
	}
}
func TestStoreRefusesHostRollbackReenrollment(t *testing.T) {
	s := fixture(t)
	alloc(t, s)
	p := filepath.Join(s.HostDir, "couch-identities.json")
	snapshot, _ := os.ReadFile(p)
	other := s
	other.StoreDir = filepath.Join(t.TempDir(), "second")
	alloc(t, other)
	os.WriteFile(p, snapshot, 0600)
	moved := filepath.Join(t.TempDir(), "moved")
	os.Rename(other.StoreDir, moved)
	other.StoreDir = moved
	if _, e := other.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
		t.Fatal("re-enrolled with regressed host authority")
	}
}
func TestStorePublicationFailureBurnsNumber(t *testing.T) {
	s := fixture(t)
	alloc(t, s)
	s.beforePublish = func(path string) error {
		if filepath.Base(path) == "identities.json" {
			return errors.New("injected")
		}
		return nil
	}
	if _, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
		t.Fatal("wanted failure")
	}
	s.beforePublish = nil
	if r := alloc(t, s); r.M != 3 {
		t.Fatal(r)
	}
}
func TestStoreStrictFiles(t *testing.T) {
	for _, raw := range []string{`{"schema_version":1,"schema_version":1}`, strings.Repeat(" ", 4097), `null`} {
		t.Run(raw[:min(8, len(raw))], func(t *testing.T) {
			s := fixture(t)
			alloc(t, s)
			os.WriteFile(filepath.Join(s.StoreDir, "identities.json"), []byte(raw), 0600)
			if _, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
				t.Fatal("accepted malformed local")
			}
		})
	}
}
func TestStoreContextLock(t *testing.T) {
	s := fixture(t)
	alloc(t, s)
	lock, e := acquireLock(context.Background(), filepath.Join(s.HostDir, "couch-identities.lock"))
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, e = s.Allocate(ctx, AllocationRequest{Terminal: true})
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
}
func TestStoreSubprocessContention(t *testing.T) {
	if os.Getenv("PAIR_IDENTITY_CHILD") == "1" {
		s := IdentityStore{HostDir: os.Getenv("PAIR_IDENTITY_HOST"), StoreDir: os.Getenv("PAIR_IDENTITY_STORE")}
		r, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true})
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := json.Marshal(r)
		if e = os.WriteFile(os.Getenv("PAIR_IDENTITY_RESULT"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		return
	}
	s := fixture(t)
	dir := t.TempDir()
	cmds := make([]*exec.Cmd, 12)
	for i := range cmds {
		p := filepath.Join(dir, string(rune('a'+i)))
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestStoreSubprocessContention$")
		cmds[i].Env = append(os.Environ(), "PAIR_IDENTITY_CHILD=1", "PAIR_IDENTITY_HOST="+s.HostDir, "PAIR_IDENTITY_STORE="+s.StoreDir, "PAIR_IDENTITY_RESULT="+p)
		if e := cmds[i].Start(); e != nil {
			t.Fatal(e)
		}
	}
	seen := map[uint64]bool{}
	for i, c := range cmds {
		if e := c.Wait(); e != nil {
			t.Fatal(e)
		}
		raw, e := os.ReadFile(filepath.Join(dir, string(rune('a'+i))))
		if e != nil {
			t.Fatal(e)
		}
		var r AllocationResult
		json.Unmarshal(raw, &r)
		if seen[r.M] || r.C != 1 || r.M == 0 {
			t.Fatal(r)
		}
		seen[r.M] = true
	}
}

func TestEnrollmentPublicationRetry(t *testing.T) {
	for _, moved := range []bool{false, true} {
		t.Run(fmt.Sprint(moved), func(t *testing.T) {
			s := fixture(t)
			if moved {
				alloc(t, s)
				p := filepath.Join(t.TempDir(), "moved")
				if e := os.Rename(s.StoreDir, p); e != nil {
					t.Fatal(e)
				}
				s.StoreDir = p
			}
			s.beforePublish = func(p string) error {
				if filepath.Base(p) == "identities.json" {
					return errors.New("injected enrollment failure")
				}
				return nil
			}
			if _, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
				t.Fatal("expected publication failure")
			}
			s.beforePublish = nil
			r := alloc(t, s)
			if r.N != 1 || r.M != 1 {
				t.Fatal(r)
			}
		})
	}
}
func TestStoreAuthorityValidation(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "oversized-host", "bad-host", "local-ahead"} {
		t.Run(kind, func(t *testing.T) {
			s := fixture(t)
			alloc(t, s)
			lp := filepath.Join(s.StoreDir, "identities.json")
			hp := filepath.Join(s.HostDir, "couch-identities.json")
			switch kind {
			case "symlink":
				raw, _ := os.ReadFile(lp)
				p := filepath.Join(t.TempDir(), "copy")
				os.WriteFile(p, raw, 0600)
				os.Remove(lp)
				os.Symlink(p, lp)
			case "directory":
				os.Remove(lp)
				os.Mkdir(lp, 0700)
			case "oversized-host":
				os.WriteFile(hp, []byte(strings.Repeat(" ", (4<<20)+1)), 0600)
			case "bad-host":
				os.WriteFile(hp, []byte(`{"schema_version":1,"next_c":1,"stores":[]}`), 0600)
			case "local-ahead":
				raw, _ := os.ReadFile(lp)
				var state AllocationState
				json.Unmarshal(raw, &state)
				state.LastM++
				raw, _ = json.Marshal(state)
				os.WriteFile(lp, raw, 0600)
			}
			if _, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
				t.Fatal("accepted invalid authority")
			}
		})
	}
}
func TestStoreCapacityAndOverflow(t *testing.T) {
	for _, capacity := range []bool{true, false} {
		t.Run(fmt.Sprint(capacity), func(t *testing.T) {
			s := fixture(t)
			os.MkdirAll(s.HostDir, 0700)
			h := hostRegistry{SchemaVersion: 1, NextC: math.MaxUint64}
			if capacity {
				h.NextC = maxStores + 1
				for i := 1; i <= maxStores; i++ {
					h.Stores = append(h.Stores, StoreRegistration{C: uint64(i), StorePath: fmt.Sprintf("/retired/%d", i)})
				}
			}
			raw, _ := json.Marshal(h)
			p := filepath.Join(s.HostDir, "couch-identities.json")
			os.WriteFile(p, raw, 0600)
			if _, e := s.Allocate(context.Background(), AllocationRequest{Terminal: true}); e == nil {
				t.Fatal("allocated beyond capacity")
			}
			after, _ := os.ReadFile(p)
			if string(after) != string(raw) {
				t.Fatal("changed authority on refusal")
			}
		})
	}
}

func TestHostAndStoreMayShareDirectory(t *testing.T) {
	root := t.TempDir()
	s := IdentityStore{HostDir: root, StoreDir: root}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := s.Allocate(ctx, AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "repo"})
	if err != nil || result.C != 1 || result.N != 1 || result.M != 1 {
		t.Fatal(result, err)
	}
}

func TestDescriptiveRepositoryNamesNeverGateIdentityAllocation(t *testing.T) {
	s := fixture(t)
	for i, name := range []string{"项目", "!!!", strings.Repeat("a", 255)} {
		r, err := s.Allocate(context.Background(), AllocationRequest{Conversation: true, RepositoryToken: name})
		if err != nil {
			t.Fatalf("name=%q: %v", name, err)
		}
		if r.C != 1 || r.N != uint64(i+1) || len(r.PairTag) > 106 {
			t.Fatalf("allocation=%+v", r)
		}
		if i < 2 && !strings.HasPrefix(r.PairTag, "1-repo-") {
			t.Fatal(r)
		}
	}
}
