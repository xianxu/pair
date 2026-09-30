package couchidentity

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestAdvanceIndependentCounters(t *testing.T) {
	s := AllocationState{SchemaVersion: 1, C: 7, StorePath: "/store"}
	n, r, e := AdvanceAllocation(s, AllocationRequest{Conversation: true, RepositoryToken: "My Repo"})
	if e != nil || n.LastN != 1 || n.LastM != 0 || r.PairTag != "7-my-repo-1" {
		t.Fatalf("%+v %+v %v", n, r, e)
	}
	n, r, e = AdvanceAllocation(n, AllocationRequest{Terminal: true})
	if e != nil || n.LastN != 1 || n.LastM != 1 || r.SessionName != "📁7-1" {
		t.Fatalf("%+v %+v %v", n, r, e)
	}
}
func TestAdvanceRefusesOverflowWithoutMutation(t *testing.T) {
	s := AllocationState{SchemaVersion: 1, C: 1, StorePath: "/store", LastN: math.MaxUint64}
	n, _, e := AdvanceAllocation(s, AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "repo"})
	if e == nil || n != s {
		t.Fatalf("%+v %v", n, e)
	}
}
func TestFormatsAndBinding(t *testing.T) {
	if _, e := FormatPairTag(0, "repo", 1); e == nil {
		t.Fatal("zero C")
	}
	if _, e := FormatSessionName(1, 0); e == nil {
		t.Fatal("zero M")
	}
	b := SessionBinding{C: 2, M: 3, Name: "📁2-3", ScopeKey: "scope", Tag: "2-repo-1", StartNonce: "nonce"}
	if e := b.Validate(); e != nil {
		t.Fatal(e)
	}
	b.Name = "📁2-4"
	if b.Validate() == nil {
		t.Fatal("mismatch")
	}
}
func FuzzAdvanceAllocation(f *testing.F) {
	f.Add(uint64(1), uint64(0), uint64(0), true, false)
	f.Fuzz(func(t *testing.T, c, n, m uint64, conversation, terminal bool) {
		s := AllocationState{SchemaVersion: 1, C: c, StorePath: "/store", LastN: n, LastM: m}
		next, _, err := AdvanceAllocation(s, AllocationRequest{Conversation: conversation, Terminal: terminal, RepositoryToken: "repo"})
		if err != nil {
			if next != s {
				t.Fatal("mutated on refusal")
			}
			return
		}
		if conversation && next.LastN != n+1 || !conversation && next.LastN != n || terminal && next.LastM != m+1 || !terminal && next.LastM != m {
			t.Fatal("independence")
		}
	})
}
func TestParseNames(t *testing.T) {
	c, token, n, e := ParsePairTag("12-my-repo-34")
	if e != nil || c != 12 || token != "my-repo" || n != 34 {
		t.Fatal(c, token, n, e)
	}
	c, m, e := ParseSessionName("📁12-34")
	if e != nil || c != 12 || m != 34 {
		t.Fatal(c, m, e)
	}
	for _, bad := range []string{"📁01-2", "📁1-0", "📁1-2-extra", "📁-1-2"} {
		if _, _, e := ParseSessionName(bad); e == nil {
			t.Fatal(bad)
		}
	}
	for _, bad := range []string{"01-repo-1", "1-Repo-1", "1--1", "1-repo-0", "1-repo--1"} {
		if _, _, _, e := ParsePairTag(bad); e == nil {
			t.Fatal(bad)
		}
	}
}

func TestBindingRejectsMalformedPersistedNamesAndNonces(t *testing.T) {
	valid := SessionBinding{Name: "📁repo-work", ScopeKey: "scope", Tag: "work", StartNonce: "start-123", Legacy: true}
	for _, name := range []string{"foreign", "pair-", "📁", "pair-../work", "📁repo\\work", "📁repo\nwork", "pair-repo work", "pair-repo\u00a0work", "pair-" + strings.Repeat("x", 252)} {
		t.Run("name-"+name, func(t *testing.T) {
			b := valid
			b.Name = name
			raw, _ := json.Marshal(b)
			var persisted SessionBinding
			if err := json.Unmarshal(raw, &persisted); err != nil {
				t.Fatal(err)
			}
			if persisted.Validate() == nil {
				t.Fatal("accepted malformed legacy name")
			}
		})
	}
	for _, nonce := range []string{"../start", "start\\1", "start\n1", "start 1", "start\u00a01", strings.Repeat("x", 257)} {
		t.Run("nonce-"+nonce, func(t *testing.T) {
			for _, legacy := range []bool{true, false} {
				b := valid
				b.StartNonce = nonce
				if !legacy {
					b.Legacy = false
					b.C = 1
					b.M = 1
					b.Name = "📁1-1"
				}
				if b.Validate() == nil {
					t.Fatal("accepted malformed nonce")
				}
			}
		})
	}
	for _, name := range []string{"📁repo-work", "pair-repo-work"} {
		b := valid
		b.Name = name
		if err := b.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
