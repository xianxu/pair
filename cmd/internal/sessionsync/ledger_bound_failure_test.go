package sessionsync

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLedgerRetentionFailureStopsGrowthAcrossOwners(t *testing.T) {
	for _, attempts := range []int{8, 16} {
		for _, size := range []int{trimBytes - 1, trimBytes + 1} {
			t.Run(fmt.Sprintf("bytes=%d/attempts=%d", size, attempts), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "ops%3A0.jsonl")
				bad := `{"ID":"bad","Origin":"remote","RemoteSource":{"IP":null}}` + "\n"
				raw := []byte(bad + strings.Repeat(" ", size-len(bad)-1) + "\n")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				for i := range attempts {
					owner := &Ledger{Dir: dir}
					if err := owner.Append("ops:0", Entry{ID: fmt.Sprint(i), At: t0, Head: "delivery"}); err == nil {
						t.Fatal("retention failure hidden")
					}
					got, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, raw) {
						t.Fatalf("attempt %d grew or altered damaged evidence: bytes %d -> %d", i, len(raw), len(got))
					}
				}
			})
		}
	}
}

func TestLedgerReadFailureStopsGrowthAcrossOwners(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops%3A0.jsonl")
	// A line over the existing scanner limit is a read failure, even without
	// recognizable provenance. It must not allow another row on each restart.
	raw := []byte(strings.Repeat("x", 1024*1024+1) + "\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		if err := (&Ledger{Dir: dir}).Append("ops:0", Entry{ID: fmt.Sprint(i), At: t0}); err == nil {
			t.Fatal("read failure hidden")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, raw) {
			t.Fatalf("read failure allowed append %d", i)
		}
	}
}

func TestLedgerRetentionPublicationFailureStopsGrowthAcrossOwners(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires non-root Unix directory permission enforcement")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ops%3A0.jsonl")
	old := `{"ID":"old","At":"2020-01-01T00:00:00Z","Head":"` + strings.Repeat("x", 200) + `"}` + "\n"
	raw := []byte(strings.Repeat(old, 1200))
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	for i := range 16 {
		if err := (&Ledger{Dir: dir}).Append("ops:0", Entry{ID: fmt.Sprint(i), At: t0}); err == nil {
			t.Fatal("atomic trim publication failure hidden")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, raw) {
			t.Fatalf("failed atomic trim allowed append %d", i)
		}
	}
}

func TestLedgerWriteFailurePreservesExistingEvidence(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires non-root Unix file permission enforcement")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ops%3A0.jsonl")
	raw := []byte(`{"ID":"old","At":"2026-10-10T12:00:00Z"}` + "\n")
	if err := os.WriteFile(path, raw, 0400); err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		if err := (&Ledger{Dir: dir}).Append("ops:0", Entry{ID: fmt.Sprint(i), At: t0}); err == nil {
			t.Fatal("write-open failure hidden")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, raw) {
			t.Fatalf("failed write changed evidence %d", i)
		}
	}
}

func TestLedgerGapExpiryFailureDoesNotDisableRetention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops%3A0.jsonl")
	// This real IO failure occurs after append. Retention preflight must still
	// run before the next append, rather than depending on expireGap succeeding.
	if err := os.Mkdir(filepath.Join(dir, "ops%3A0.gap"), 0700); err != nil {
		t.Fatal(err)
	}
	old := `{"ID":"old","At":"2020-01-01T00:00:00Z","Head":"` + strings.Repeat("x", 200) + `"}` + "\n"
	if err := os.WriteFile(path, []byte(strings.Repeat(old, 1200)), 0600); err != nil {
		t.Fatal(err)
	}
	for i := range 16 {
		if err := (&Ledger{Dir: dir}).Append("ops:0", Entry{ID: fmt.Sprint(i), At: t0}); err == nil {
			t.Fatal("gap read IO failure hidden")
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > trimBytes {
			t.Fatalf("gap failure bypassed retention at append %d: %d", i, info.Size())
		}
	}
}
