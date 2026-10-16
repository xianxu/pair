package sessioninventory

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func codexSnapshotFixture(t *testing.T) (OSRuntime, *nativeSelectorRuntime, Artifact, string) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	state := filepath.Join(home, ".codex")
	if err := os.Mkdir(state, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(state, "state_5.sqlite")
	if out, err := exec.Command("sqlite3", "-batch", "-init", os.DevNull, path, "CREATE TABLE threads(id TEXT);INSERT INTO threads VALUES('candidate')").CombinedOutput(); err != nil {
		t.Fatalf("private SQLite: %v %s", err, out)
	}
	codexTestFreezeWAL(t, path)
	runtime, err := nativeSelectorCodexRuntime(NewOSRuntime(home, filepath.Join(home, "pair")), state)
	if err != nil {
		t.Fatal(err)
	}
	r := runtime.(OSRuntime)
	bounded, err := newNativeSelectorRuntime(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := bounded.Close(); err != nil {
			t.Error(err)
		}
	})
	return r, bounded, Artifact{StorageRoot: "codex-state", RelativePath: "state_5.sqlite", Kind: ArtifactMetadata}, path
}
func TestNativeSelectorSQLiteSnapshotRejectsCorruptWALHeader(t *testing.T) {
	rt, source, a, path := codexSnapshotFixture(t)
	raw, err := os.ReadFile(path + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	raw[24] ^= 1
	if err = os.WriteFile(path+"-wal", raw, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
	if snapshot != nil {
		defer snapshot.Close()
	}
	if err == nil {
		t.Fatal("corrupt WAL header acquired snapshot authority")
	}
}

// This source models metadata with coarse generation/mtime resolution. Actual
// byte reads remain bounded and descriptor-checked against fresh native stats.
// It lets the test isolate the full-content and WAL-generation fences.
type codexCoarseSnapshotSource struct {
	NativeSelectorSQLiteSnapshotSource
	runtime   OSRuntime
	receipts  map[string]FileEntry
	afterRead func(FileEntry)
}

func (s *codexCoarseSnapshotSource) StatRecoveryFile(ctx context.Context, a Artifact) (FileEntry, error) {
	fresh, err := s.NativeSelectorSQLiteSnapshotSource.StatRecoveryFile(ctx, a)
	if err != nil {
		return fresh, err
	}
	key := targetArtifactKey(a)
	if old, ok := s.receipts[key]; ok {
		fresh.ModTime = old.ModTime
		fresh.MutationToken = old.MutationToken
		return fresh, nil
	}
	s.receipts[key] = fresh
	return fresh, nil
}
func (s *codexCoarseSnapshotSource) ReadRecoveryAt(ctx context.Context, e FileEntry, offset, count int64) ([]byte, bool, error) {
	fresh, err := s.runtime.StatRecoveryFile(ctx, e.Artifact)
	if err != nil {
		return nil, false, err
	}
	if fresh.StableFileID != e.StableFileID || fresh.GenerationToken != e.GenerationToken || fresh.Size != e.Size {
		return nil, false, ErrArtifactChanged
	}
	raw, eof, err := s.NativeSelectorSQLiteSnapshotSource.ReadRecoveryAt(ctx, fresh, offset, count)
	if err == nil && s.afterRead != nil {
		s.afterRead(e)
	}
	return raw, eof, err
}
func TestNativeSelectorSQLiteSnapshotRejectsRecycledWALAfterVerification(t *testing.T) {
	rt, budget, a, path := codexSnapshotFixture(t)
	_, _, _, replacement := codexSnapshotFixture(t)
	wal, err := os.ReadFile(replacement + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	old, err := os.Stat(path + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	dbReads := 0
	changed := false
	source := &codexCoarseSnapshotSource{NativeSelectorSQLiteSnapshotSource: budget, runtime: rt, receipts: map[string]FileEntry{}}
	source.afterRead = func(e FileEntry) {
		if e.Artifact.RelativePath != a.RelativePath {
			return
		}
		dbReads++
		if dbReads != 2 {
			return
		}
		changed = true
		if int64(len(wal)) != old.Size() {
			t.Fatal("replacement WAL changed size")
		}
		if err := os.WriteFile(path+"-wal", wal, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path+"-wal", old.ModTime(), old.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
	if snapshot != nil {
		defer snapshot.Close()
	}
	if !changed || err == nil {
		t.Fatalf("recycled WAL after second DB read accepted: changed=%v err=%v", changed, err)
	}
}
func TestNativeSelectorSQLiteSnapshotRejectsChangedDBWithCoarseMetadata(t *testing.T) {
	rt, budget, a, path := codexSnapshotFixture(t)
	old, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	source := &codexCoarseSnapshotSource{NativeSelectorSQLiteSnapshotSource: budget, runtime: rt, receipts: map[string]FileEntry{}}
	source.afterRead = func(e FileEntry) {
		if changed || e.Artifact.RelativePath != a.RelativePath {
			return
		}
		changed = true
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw[60] ^= 1
		if err = os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.Chtimes(path, old.ModTime(), old.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
	if snapshot != nil {
		defer snapshot.Close()
	}
	if !changed || err == nil {
		t.Fatalf("changed DB content accepted: changed=%v err=%v", changed, err)
	}
}
func TestNativeSelectorSQLiteSnapshotCopyBudgetAndClosedOwnership(t *testing.T) {
	rt, source, a, path := codexSnapshotFixture(t)
	db, _ := os.Stat(path)
	wal, _ := os.Stat(path + "-wal")
	expected := 2 * (db.Size() + wal.Size())
	snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, expected-1)
	if snapshot != nil {
		defer snapshot.Close()
	}
	if !errors.Is(err, ErrReadLimit) || source.bytes != 0 {
		t.Fatalf("oversize copied bytes=%d err=%v", source.bytes, err)
	}
	snapshot, err = rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
	if err != nil {
		t.Fatal(err)
	}
	if source.bytes < expected || source.bytes > expected+4096 {
		t.Fatalf("source copy and verification bytes=%d expected >=%d", source.bytes, expected)
	}
	if source.failure != nil {
		t.Fatalf("expected absent journal poisoned attempt: %v", source.failure)
	}
	private := snapshot.(*nativeSQLiteSnapshot)
	directory := private.directory
	if err = snapshot.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("snapshot directory survived: %v", err)
	}
	if _, err = snapshot.Query(t.Context(), "SELECT id FROM threads", 1024); err == nil {
		t.Fatal("closed snapshot queried")
	}
}

func TestNativeSelectorSQLiteSnapshotCopyMutationNeverAcquiresAuthority(t *testing.T) {
	for _, mutation := range []string{"replace-db", "replace-wal", "truncate-db", "truncate-wal", "checkpoint", "journal-create"} {
		t.Run(mutation, func(t *testing.T) {
			rt, budget, a, path := codexSnapshotFixture(t)
			scratch := t.TempDir()
			t.Setenv("TMPDIR", scratch)
			changed := false
			source := &codexCoarseSnapshotSource{NativeSelectorSQLiteSnapshotSource: budget, runtime: rt, receipts: map[string]FileEntry{}}
			source.afterRead = func(e FileEntry) {
				if changed || e.Artifact.RelativePath != a.RelativePath+"-wal" {
					return
				}
				changed = true
				switch mutation {
				case "replace-db", "replace-wal":
					p := path
					if mutation == "replace-wal" {
						p += "-wal"
					}
					raw, err := os.ReadFile(p)
					if err != nil {
						t.Fatal(err)
					}
					replacement := p + ".replacement"
					if err = os.WriteFile(replacement, raw, 0600); err != nil {
						t.Fatal(err)
					}
					if err = os.Rename(replacement, p); err != nil {
						t.Fatal(err)
					}
				case "truncate-db":
					if err := os.Truncate(path, 0); err != nil {
						t.Fatal(err)
					}
				case "truncate-wal":
					if err := os.Truncate(path+"-wal", 0); err != nil {
						t.Fatal(err)
					}
				case "journal-create":
					if err := os.WriteFile(path+"-journal", []byte("pending"), 0600); err != nil {
						t.Fatal(err)
					}
				case "checkpoint":
					python, err := exec.LookPath("python3")
					if err != nil {
						t.Fatal(err)
					}
					if out, err := exec.Command(python, "-c", "import sqlite3,sys;c=sqlite3.connect(sys.argv[1]);c.execute('PRAGMA wal_checkpoint(TRUNCATE)');c.close()", path).CombinedOutput(); err != nil {
						t.Fatalf("checkpoint fixture: %s %v", out, err)
					}
				}
			}
			snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
			if snapshot != nil {
				defer snapshot.Close()
			}
			if !changed || err == nil {
				t.Fatalf("changed source acquired authority: changed=%v err=%v", changed, err)
			}
			files, err := os.ReadDir(scratch)
			if err != nil || len(files) != 0 {
				t.Fatalf("partial snapshot residue=%v err=%v", files, err)
			}
		})
	}
}

type codexRecordingSnapshotSource struct {
	NativeSelectorSQLiteSnapshotSource
	paths []string
}

func (s *codexRecordingSnapshotSource) StatRecoveryFile(ctx context.Context, a Artifact) (FileEntry, error) {
	s.paths = append(s.paths, a.RelativePath)
	return s.NativeSelectorSQLiteSnapshotSource.StatRecoveryFile(ctx, a)
}
func (s *codexRecordingSnapshotSource) ReadRecoveryAt(ctx context.Context, e FileEntry, offset, count int64) ([]byte, bool, error) {
	s.paths = append(s.paths, e.Artifact.RelativePath)
	return s.NativeSelectorSQLiteSnapshotSource.ReadRecoveryAt(ctx, e, offset, count)
}
func TestNativeSelectorSQLiteSnapshotNeverObservesSHM(t *testing.T) {
	rt, budget, a, path := codexSnapshotFixture(t)
	if err := os.WriteFile(path+"-shm", []byte("must not inspect"), 0600); err != nil {
		t.Fatal(err)
	}
	source := &codexRecordingSnapshotSource{NativeSelectorSQLiteSnapshotSource: budget}
	snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, source, RecoveryTotalReadLimit)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if _, err = snapshot.Query(t.Context(), "SELECT id FROM threads", 1024); err != nil {
		t.Fatal(err)
	}
	for _, path := range source.paths {
		if path != a.RelativePath && path != a.RelativePath+"-wal" && path != a.RelativePath+"-journal" {
			t.Fatalf("observed unauthorized source component %q", path)
		}
	}
	for _, root := range budget.NativeRoots(AgentCodex) {
		if root.Name == "codex-state" {
			t.Fatal("SQLite metadata root leaked into transcript inventory listing")
		}
	}
}

func TestNativeSelectorSQLiteSnapshotRejectsNativeScratchParentWithoutMutation(t *testing.T) {
	for _, which := range []string{"state", "codex-home", "sessions", "alias"} {
		t.Run(which, func(t *testing.T) {
			rt, budget, a, path := codexSnapshotFixture(t)
			state := filepath.Dir(path)
			sessions := filepath.Join(state, "sessions")
			if err := os.MkdirAll(sessions, 0700); err != nil {
				t.Fatal(err)
			}
			parent := state
			if which == "codex-home" {
				codexHome := filepath.Join(filepath.Dir(state), "separate-codex-home")
				if err := os.Mkdir(codexHome, 0700); err != nil {
					t.Fatal(err)
				}
				scoped, err := rt.WithCodexScope(CodexSelectionScope{CodexHome: codexHome, SQLiteHome: state})
				if err != nil {
					t.Fatal(err)
				}
				rt = scoped
				parent = codexHome
			}
			if which == "sessions" {
				parent = sessions
			}
			if which == "alias" {
				alias := filepath.Join(t.TempDir(), "native-alias")
				if err := os.Symlink(state, alias); err != nil {
					t.Fatal(err)
				}
				parent = alias
			}
			t.Setenv("TMPDIR", parent)
			watched, err := filepath.EvalSymlinks(parent)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(watched)
			if err != nil {
				t.Fatal(err)
			}
			beforeNames, err := os.ReadDir(watched)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := rt.NewNativeSelectorSQLiteSnapshot(t.Context(), a, budget, RecoveryTotalReadLimit)
			if snapshot != nil {
				_ = snapshot.Close()
			}
			if err == nil {
				t.Error("native scratch parent acquired authority")
			}
			after, statErr := os.Stat(watched)
			if statErr != nil {
				t.Fatal(statErr)
			}
			afterNames, readErr := os.ReadDir(watched)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !before.ModTime().Equal(after.ModTime()) || len(beforeNames) != len(afterNames) {
				t.Fatal("scratch observation mutated native directory")
			}
		})
	}
}
