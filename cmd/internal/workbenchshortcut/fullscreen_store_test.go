package workbenchshortcut

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

func TestFullscreenStoreRoundTripAndStableLock(t *testing.T) {
	s := FullscreenReturnStore{DataDir: t.TempDir(), Tag: "thread"}
	var _ FullscreenStore = s
	p, _ := artifactpath.ResolveScoped(s.DataDir, s.Tag)
	if got, err := s.Read(); err != nil || got != "" {
		t.Fatalf("missing = %q, %v", got, err)
	}
	unlock, ok, err := s.TryLock()
	if err != nil || !ok {
		t.Fatalf("lock = %v, %v", ok, err)
	}
	defer unlock()
	before, err := os.Stat(p.FullscreenLock())
	if err != nil {
		t.Fatal(err)
	}
	if release, acquired, err := s.TryLock(); err != nil || acquired || release != nil {
		t.Fatalf("busy = %v, %v", acquired, err)
	}
	for _, id := range []string{"12", "terminal_42"} {
		if err := s.Write(id); err != nil {
			t.Fatal(err)
		}
		if got, err := s.Read(); err != nil || got != id {
			t.Fatalf("read = %q, %v", got, err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := s.Clear(); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.Read(); err != nil || got != "" {
		t.Fatalf("cleared = %q, %v", got, err)
	}
	unlock()
	unlock()
	again, ok, err := s.TryLock()
	if err != nil || !ok {
		t.Fatalf("reacquire = %v, %v", ok, err)
	}
	defer again()
	after, err := os.Stat(p.FullscreenLock())
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("lock inode replaced", err)
	}
}

func TestFullscreenStoreRejectsMalformedAndOversizedRecords(t *testing.T) {
	s := FullscreenReturnStore{DataDir: t.TempDir(), Tag: "thread"}
	p, _ := artifactpath.ResolveScoped(s.DataDir, s.Tag)
	if err := s.Write("12"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "  ", "12\n13", "12\x0013", strings.Repeat("1", 257)} {
		if err := s.Write(value); err == nil {
			t.Fatalf("accepted write %q", value)
		}
		if got, _ := s.Read(); got != "12" {
			t.Fatalf("bad write replaced record: %q", got)
		}
	}
	for _, value := range []string{"", "\n", "12\n13\n", "12\x00", strings.Repeat("1", 258)} {
		if err := os.WriteFile(p.FullscreenReturn(), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Read(); err == nil {
			t.Fatalf("accepted malformed disk record %q", value)
		}
	}
}

func TestFullscreenStoreInvalidNamespaceHasNoEffects(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", root)
	t.Setenv("HOME", root)
	for _, s := range []FullscreenReturnStore{{DataDir: filepath.Join(root, "absent"), Tag: "../bad"}, {DataDir: "relative", Tag: "thread"}, {DataDir: "", Tag: "thread"}} {
		if _, err := s.Read(); err == nil {
			t.Fatal("invalid read accepted")
		}
		if err := s.Write("12"); err == nil {
			t.Fatal("invalid write accepted")
		}
		if err := s.Clear(); err == nil {
			t.Fatal("invalid clear accepted")
		}
		if _, ok, err := s.TryLock(); ok || err == nil {
			t.Fatal("invalid lock accepted")
		}
		s.LogFailure(errors.New("failure"))
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("side effects: %v, %v", entries, err)
	}
}

func TestFullscreenStoreDiagnosticsManagedAndRetained(t *testing.T) {
	root := t.TempDir()
	s := FullscreenReturnStore{DataDir: root, Tag: "thread"}
	p, _ := artifactpath.ResolveScoped(root, s.Tag)
	s.LogFailure(nil)
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("nil error created artifacts")
	}
	s.LogFailure(errors.New("focus return=12 target=42: failed"))
	data, err := os.ReadFile(p.FullscreenDiagnostics())
	if err != nil || !strings.Contains(string(data), "focus return=12 target=42: failed") {
		t.Fatalf("diagnostic = %q, %v", data, err)
	}
	opts := diagnosticlog.Options{Now: func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }, Proof: func(context.Context, string, []diagnosticlog.Registration) error { return nil }}
	rows, err := diagnosticlog.Preview(p.FullscreenDiagnostics(), opts, 100)
	if err != nil || len(rows) != 1 || !rows[0].Eligible {
		t.Fatalf("retention = %+v, %v", rows, err)
	}
	if _, err := diagnosticlog.Collect(p.FullscreenDiagnostics(), opts, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.FullscreenDiagnostics()); !os.IsNotExist(err) {
		t.Fatalf("expired diagnostic retained: %v", err)
	}
	// Logging into an unusable directory is deliberately silent and best effort.
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	FullscreenReturnStore{DataDir: blocked, Tag: "thread"}.LogFailure(errors.New("ignored"))
}

func TestFullscreenStoreSeparatesScopesAndTags(t *testing.T) {
	root := t.TempDir()
	var stores []FullscreenReturnStore
	for _, scope := range []string{"repo", "other"} {
		for _, tag := range []string{"thread", "neighbor"} {
			p, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: scope, Tag: tag})
			if err != nil {
				t.Fatal(err)
			}
			stores = append(stores, FullscreenReturnStore{DataDir: p.ScopeDir(), Tag: tag})
		}
	}
	for i, s := range stores {
		unlock, ok, err := s.TryLock()
		if err != nil || !ok {
			t.Fatalf("independent lock %d: %v, %v", i, ok, err)
		}
		defer unlock()
		if err := s.Write(strings.Repeat("1", i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := stores[0].Clear(); err != nil {
		t.Fatal(err)
	}
	for i, s := range stores[1:] {
		if got, err := s.Read(); err != nil || got != strings.Repeat("1", i+2) {
			t.Fatalf("neighbor changed: %q, %v", got, err)
		}
	}
}

func TestFullscreenStoreBoundsPayloadAndRejectsNonregularFiles(t *testing.T) {
	s := FullscreenReturnStore{DataDir: t.TempDir(), Tag: "thread"}
	p, _ := artifactpath.ResolveScoped(s.DataDir, s.Tag)
	id := strings.Repeat("1", 256)
	if err := s.Write(id); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Read(); err != nil || got != id {
		t.Fatalf("maximum ID = %q, %v", got, err)
	}
	if err := os.WriteFile(p.FullscreenReturn(), []byte(strings.Repeat("1", 1<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("oversized read: %v", err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p.FullscreenReturn(), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil || !strings.Contains(err.Error(), "not regular") {
		t.Fatalf("directory read: %v", err)
	}
	if err := os.Symlink(p.FullscreenReturn(), p.FullscreenLock()); err != nil {
		t.Fatal(err)
	}
	if unlock, ok, err := s.TryLock(); unlock != nil || ok || err == nil {
		t.Fatal("symlink lock accepted")
	}
	s.LogFailure(errors.New(strings.Repeat("x", 1<<20)))
	data, err := os.ReadFile(p.FullscreenDiagnostics())
	if err != nil {
		t.Fatal(err)
	}
	var entry struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if len(entry.Error) != 4096 {
		t.Fatalf("diagnostic detail length = %d", len(entry.Error))
	}
}
