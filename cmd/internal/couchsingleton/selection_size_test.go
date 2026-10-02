package couchsingleton

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Build actual encoded boundary cases from ordinary short missing paths. Each
// component stays below filesystem limits; escaping is measured by json.Marshal.
func selectionAtSize(t *testing.T, roots Roots, size int, escaped bool) Selection {
	t.Helper()
	s := Selection{Version: 1, Roots: roots}
	for i := 0; i < 300; i++ {
		p := fmt.Sprintf("%s-retired/%04d", roots.StoreDir, i)
		if escaped {
			p += "<&>"
		}
		s.Excluded = append(s.Excluded, p)
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	left := size - len(raw)
	if left < 0 {
		t.Fatalf("fixture already exceeds requested size: %d > %d", len(raw), size)
	}
	for i := range s.Excluded {
		n := min(left, 500)
		for remaining := n; remaining > 0; {
			part := min(remaining, 180)
			s.Excluded[i] += strings.Repeat("x", part)
			remaining -= part
			if remaining == 1 {
				s.Excluded[i] += "x"
				remaining = 0
			}
			if remaining > 0 {
				s.Excluded[i] += "/"
				remaining--
			}
		}
		left -= n
	}
	raw, err = json.Marshal(s)
	if err != nil || left != 0 || len(raw) != size {
		t.Fatalf("fixture size %d, remaining %d: %v", len(raw), left, err)
	}
	return s
}

func TestSelectionSizeContractRoundTripAtReadBoundary(t *testing.T) {
	for _, escaped := range []bool{false, true} {
		t.Run(fmt.Sprintf("escaped=%v", escaped), func(t *testing.T) {
			m := fixture(t)
			want := selectionAtSize(t, m.Defaults, 65536, escaped)
			q := Request{Roots: want.Roots, Exclude: want.Excluded}
			report, err := m.Preview(q)
			if err != nil || report.Status != "READY" {
				t.Fatalf("boundary preview: %+v %v", report, err)
			}
			selected, err := m.Adopt(q, report.Digest)
			if err != nil || !reflect.DeepEqual(selected, want) {
				t.Fatalf("boundary adoption: %+v %v", selected, err)
			}
			read, err := m.Read(q)
			if err != nil || !reflect.DeepEqual(read, want) {
				t.Fatalf("successful writer did not round trip: %v", err)
			}
			raw, err := os.ReadFile(m.selectionPath())
			if err != nil || len(raw) != 65536 {
				t.Fatalf("published bytes: %d %v", len(raw), err)
			}
		})
	}
}

func TestSelectionSizeContractRejectsBeforePublication(t *testing.T) {
	for _, size := range []int{65537, 100000} {
		for _, escaped := range []bool{false, true} {
			t.Run(fmt.Sprintf("size=%d/escaped=%v", size, escaped), func(t *testing.T) {
				m := fixture(t)
				s := selectionAtSize(t, m.Defaults, size, escaped)
				q := Request{Roots: s.Roots, Exclude: s.Excluded}
				report, err := m.Preview(q)
				if err == nil && (report.Status == "READY" || len(report.Blockers) == 0) {
					t.Error("oversized selection advertised as applicable")
				}
				published := false
				m.Publish = func(string, []byte) error { published = true; return nil }
				if _, err := m.Adopt(q, report.Digest); err == nil || !strings.Contains(err.Error(), "selection") {
					t.Errorf("oversized explicit adoption: %v", err)
				}
				_, lease, err := m.Acquire(q)
				if lease != nil {
					lease.Close()
				}
				if err == nil || !strings.Contains(err.Error(), "selection") {
					t.Errorf("oversized automatic adoption: %v", err)
				}
				if published {
					t.Error("oversized selection reached publication")
				}
				if _, err := os.Stat(m.selectionPath()); !os.IsNotExist(err) {
					t.Errorf("oversized selection exists: %v", err)
				}
				for _, root := range []string{s.Roots.StoreDir, s.Roots.PairDataDir, s.Roots.IdentityDir} {
					if _, err := os.Stat(root); !os.IsNotExist(err) {
						t.Errorf("oversized selection initialized source %s: %v", root, err)
					}
				}
			})
		}
	}
}
