package diagnosticlog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyExpiredDiagnosticPreservedUntilBackup(t *testing.T) {
	path, now, opts := fixture(t)
	os.WriteFile(path, []byte("old debug"), 0600)
	old := now.Add(-RetentionPeriod - time.Hour)
	os.Chtimes(path, old, old)
	rows, err := PreviewLegacy(path, opts)
	if err != nil || len(rows) != 1 || rows[0].Eligible || rows[0].Reason != BackupFenceReason || !rows[0].LastWrite.Equal(old) {
		t.Fatal(rows, err)
	}
	removed, err := CollectLegacy(path, opts)
	if err != nil || len(removed) != 0 {
		t.Fatal(removed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "old debug" {
		t.Fatal(string(body), err)
	}
	if _, err = os.Stat(lockPath(path)); !os.IsNotExist(err) {
		t.Fatal("retained legacy acquired lock", err)
	}
}

func TestLegacyUnknownOrYoungDoesNotInitialize(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "young", true: "unknown"}[unknown], func(t *testing.T) {
			path, now, opts := fixture(t)
			os.WriteFile(path, []byte("keep"), 0600)
			old := now.Add(-time.Hour)
			if unknown {
				old = now.Add(-(RetentionPeriod + 24*time.Hour))
				opts.Proof = func(context.Context, string, []Registration) error { return errors.New("legacy writer alive") }
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
