package diagnosticlog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyExpiredDiagnosticKeepsOriginalAge(t *testing.T) {
	path, now, opts := fixture(t)
	os.WriteFile(path, []byte("old debug"), 0600)
	old := now.Add(-8 * 24 * time.Hour)
	os.Chtimes(path, old, old)
	rows, e := PreviewLegacy(path, opts)
	if e != nil || len(rows) != 1 || !rows[0].Eligible {
		t.Fatalf("preview %+v %v", rows, e)
	}
	if _, e = os.Stat(lockPath(path)); !os.IsNotExist(e) {
		t.Fatal("legacy preview initialized lock")
	}
	rows, e = CollectLegacy(path, opts)
	if e != nil || len(rows) != 1 {
		t.Fatalf("collect %+v %v", rows, e)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("expired legacy file retained")
	}
	if _, e = os.Stat(lockPath(path)); e != nil {
		t.Fatal("stable lock removed")
	}
}
func TestLegacyUnknownOrYoungDoesNotInitialize(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "young", true: "unknown"}[unknown], func(t *testing.T) {
			path, now, opts := fixture(t)
			os.WriteFile(path, []byte("keep"), 0600)
			old := now.Add(-time.Hour)
			if unknown {
				old = now.Add(-8 * 24 * time.Hour)
				opts.Proof = func(string, []Registration) error { return errors.New("legacy writer alive") }
			}
			os.Chtimes(path, old, old)
			rows, e := PreviewLegacy(path, opts)
			if e != nil || len(rows) != 1 || rows[0].Eligible {
				t.Fatalf("preview %+v %v", rows, e)
			}
			CollectLegacy(path, opts)
			names, _ := os.ReadDir(filepath.Dir(path))
			if len(names) != 1 {
				t.Fatalf("retained legacy acquired metadata %v", names)
			}
		})
	}
}
