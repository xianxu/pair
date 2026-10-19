package sessioninventory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

const exactClaudeID = "11111111-1111-4111-8111-111111111111"

func claudeOSFixture(t *testing.T) (OSRuntime, string) {
	t.Helper()
	home := t.TempDir()
	project := filepath.Join(home, ".claude", "projects", "relocated-project")
	if err := os.MkdirAll(project, 0700); err != nil {
		t.Fatal(err)
	}
	writeClaudeExact(t, filepath.Join(project, exactClaudeID+".jsonl"), exactClaudeID, false)
	return NewOSRuntime(home, ""), project
}
func writeClaudeExact(t *testing.T, path, id string, sidechain bool) {
	t.Helper()
	content := fmt.Sprintf("{\"type\":\"user\",\"sessionId\":%q,\"isSidechain\":%t,\"cwd\":\"/relocated\",\"timestamp\":\"2026-08-28T10:00:00Z\",\"message\":{\"role\":\"user\",\"content\":\"hello\"}}\n", id, sidechain)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestClaudeExactOverDescendantCap(t *testing.T) {
	rt, project := claudeOSFixture(t)
	for d := 0; d < 101; d++ {
		dir := filepath.Join(project, "history", fmt.Sprint(d))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for f := 0; f < 100; f++ {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprint(f)), nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, cwd := range []string{"", "/different"} {
		got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: exactClaudeID, CWD: cwd})
		if err != nil || got.Status != NativeSelectorResolved || got.NativeID != exactClaudeID {
			t.Errorf("exact cwd=%q: %+v, %v", cwd, got, err)
		}
	}
	got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, Latest: true, CWD: project})
	if err != nil || got.Status != NativeSelectorIncomplete {
		t.Errorf("latest: %+v, %v", got, err)
	}
}

type mutatingClaudeRuntime struct {
	OSRuntime
	mutate     func(Artifact)
	beforeRead func(FileEntry)
	stats      int
	recursive  int
	opened     []*os.File
}

func (r *mutatingClaudeRuntime) openRecoveryFile(ctx context.Context, root StorageRoot, relative string, directory bool) (*os.File, error) {
	f, err := r.OSRuntime.openRecoveryFile(ctx, root, relative, directory)
	if f != nil {
		r.opened = append(r.opened, f)
	}
	return f, err
}

func (r *mutatingClaudeRuntime) StatRecoveryFile(ctx context.Context, a Artifact) (FileEntry, error) {
	r.stats++
	if r.mutate != nil {
		f := r.mutate
		r.mutate = nil
		f(a)
	}
	return r.OSRuntime.StatRecoveryFile(ctx, a)
}
func (r *mutatingClaudeRuntime) VisitNativeSelectorFiles(ctx context.Context, root StorageRoot, id string, limit int, visit func(FileEntry) error) (int, error) {
	return visitClaudeExactFiles(ctx, r, root, id, limit, visit)
}
func (r *mutatingClaudeRuntime) VisitRecoveryFiles(ctx context.Context, root StorageRoot, limit int, visit func(FileEntry) error) (int, error) {
	r.recursive++
	return r.OSRuntime.VisitRecoveryFiles(ctx, root, limit, visit)
}
func TestClaudeExactParentChanges(t *testing.T) {
	for _, kind := range []string{"project-replaced", "project-removed", "negative-insertion", "root-replaced", "project-metadata", "root-metadata"} {
		t.Run(kind, func(t *testing.T) {
			base, project := claudeOSFixture(t)
			root := filepath.Dir(project)
			leaf := filepath.Join(project, exactClaudeID+".jsonl")
			if kind == "negative-insertion" {
				if err := os.Remove(leaf); err != nil {
					t.Fatal(err)
				}
			}
			rt := &mutatingClaudeRuntime{OSRuntime: base}
			rt.mutate = func(a Artifact) {
				switch kind {
				case "project-replaced":
					if err := os.Rename(project, project+"-old"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(project, 0700); err != nil {
						t.Fatal(err)
					}
				case "project-removed":
					if err := os.RemoveAll(project); err != nil {
						t.Fatal(err)
					}
				case "negative-insertion":
					writeClaudeExact(t, leaf, exactClaudeID, false)
				case "root-replaced":
					if err := os.Rename(root, root+"-old"); err != nil {
						t.Fatal(err)
					}
					if err := os.MkdirAll(project, 0700); err != nil {
						t.Fatal(err)
					}
					writeClaudeExact(t, leaf, exactClaudeID, false)
				case "project-metadata":
					if err := os.Chmod(project, 0750); err != nil {
						t.Fatal(err)
					}
				case "root-metadata":
					if err := os.Chmod(root, 0750); err != nil {
						t.Fatal(err)
					}
				}
			}
			got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: exactClaudeID})
			if err != nil || got.Status != NativeSelectorIncomplete {
				t.Fatalf("got %+v, %v", got, err)
			}
			if rt.stats < 1 || rt.recursive != 0 {
				t.Fatalf("stats=%d recursive=%d", rt.stats, rt.recursive)
			}
		})
	}
}
func TestClaudeExactVisitorGrowthAndLimits(t *testing.T) {
	for _, projects := range []int{3, 6} {
		for _, descendants := range []int{20, 40} {
			t.Run(fmt.Sprintf("p%d-n%d", projects, descendants), func(t *testing.T) {
				base, project := claudeOSFixture(t)
				root := filepath.Dir(project)
				for p := 1; p < projects; p++ {
					if err := os.Mkdir(filepath.Join(root, fmt.Sprint(p)), 0700); err != nil {
						t.Fatal(err)
					}
				}
				for n := 0; n < descendants; n++ {
					if err := os.MkdirAll(filepath.Join(project, "history", fmt.Sprint(n)), 0700); err != nil {
						t.Fatal(err)
					}
				}
				rt := &mutatingClaudeRuntime{OSRuntime: base}
				roots := rt.NativeRoots(AgentClaude)
				visits := 0
				count, err := rt.VisitNativeSelectorFiles(context.Background(), roots[0], exactClaudeID, RecoveryMetadataLimit, func(FileEntry) error { visits++; return nil })
				if err != nil || count != projects || rt.stats != projects || visits != 1 {
					t.Fatalf("count=%d stats=%d visits=%d err=%v", count, rt.stats, visits, err)
				}
				count, err = rt.VisitNativeSelectorFiles(context.Background(), roots[0], exactClaudeID, projects-1, func(FileEntry) error { return nil })
				if !errors.Is(err, ErrReadLimit) || count != projects-1 {
					t.Fatalf("budget count=%d err=%v", count, err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := rt.VisitNativeSelectorFiles(ctx, roots[0], exactClaudeID, 10, func(FileEntry) error { return nil }); !errors.Is(err, context.Canceled) {
					t.Fatalf("cancel: %v", err)
				}
			})
		}
	}
}
func TestClaudeExactRejectsUnsafePathsAndContent(t *testing.T) {
	for _, kind := range []string{"root-symlink", "project-symlink", "target-symlink", "nonregular", "duplicate", "sidechain", "contradiction", "missing", "malformed", "incomplete"} {
		t.Run(kind, func(t *testing.T) {
			rt, project := claudeOSFixture(t)
			root := filepath.Dir(project)
			leaf := filepath.Join(project, exactClaudeID+".jsonl")
			switch kind {
			case "root-symlink", "project-symlink", "target-symlink":
				path := leaf
				if kind == "root-symlink" {
					path = root
				}
				if kind == "project-symlink" {
					path = project
				}
				if err := os.Rename(path, path+"-real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+"-real", path); err != nil {
					t.Fatal(err)
				}
			case "nonregular":
				if err := os.Remove(leaf); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(leaf, 0700); err != nil {
					t.Fatal(err)
				}
			case "duplicate":
				other := filepath.Join(root, "other")
				if err := os.Mkdir(other, 0700); err != nil {
					t.Fatal(err)
				}
				writeClaudeExact(t, filepath.Join(other, exactClaudeID+".jsonl"), exactClaudeID, false)
			case "sidechain":
				writeClaudeExact(t, leaf, exactClaudeID, true)
			case "contradiction":
				writeClaudeExact(t, leaf, "22222222-2222-4222-8222-222222222222", false)
			case "missing":
				if err := os.Remove(leaf); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				if err := os.WriteFile(leaf, []byte("malformed\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "incomplete":
				if err := os.WriteFile(leaf, []byte(`{"type":"user"`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: exactClaudeID})
			if err != nil || got.Status == NativeSelectorResolved {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
}
func TestClaudeExactCanonicalHintAndLatestOrdering(t *testing.T) {
	base, project := claudeOSFixture(t)
	rt := &mutatingClaudeRuntime{OSRuntime: base}
	got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: exactClaudeID})
	if err != nil || got.Status != NativeSelectorResolved || got.NativeID != exactClaudeID || rt.recursive != 0 {
		t.Fatalf("exact %+v, %v recursive=%d", got, err, rt.recursive)
	}
	rt.recursive = 0
	_, _ = ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: "not-canonical"})
	if rt.recursive == 0 {
		t.Fatal("noncanonical ID used exact hint")
	}
	rt.recursive = 0
	_, _ = ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, Latest: true, CWD: project})
	if rt.recursive == 0 {
		t.Fatal("latest skipped global discovery")
	}
}

func TestDirectoryFingerprintFields(t *testing.T) {
	var base unix.Stat_t
	base.Mode = unix.S_IFDIR | 0700
	before, err := fingerprintDirectory(base)
	if err != nil {
		t.Fatal(err)
	}
	changes := []func(*unix.Stat_t){
		func(s *unix.Stat_t) { s.Dev++ }, func(s *unix.Stat_t) { s.Ino++ }, func(s *unix.Stat_t) { s.Mode |= 0040 }, func(s *unix.Stat_t) { s.Size++ },
		func(s *unix.Stat_t) { s.Mtim.Sec++ }, func(s *unix.Stat_t) { s.Mtim.Nsec++ }, func(s *unix.Stat_t) { s.Ctim.Sec++ }, func(s *unix.Stat_t) { s.Ctim.Nsec++ },
	}
	for i, change := range changes {
		s := base
		change(&s)
		got, err := fingerprintDirectory(s)
		if err != nil || got == before {
			t.Fatalf("field %d lost: %+v %v", i, got, err)
		}
	}
	base.Mode = unix.S_IFREG | 0600
	if _, err := fingerprintDirectory(base); err == nil {
		t.Fatal("accepted regular file")
	}
}

func (r *mutatingClaudeRuntime) ReadRecoveryAt(ctx context.Context, e FileEntry, offset, limit int64) ([]byte, bool, error) {
	if r.beforeRead != nil {
		f := r.beforeRead
		r.beforeRead = nil
		f(e)
	}
	return r.OSRuntime.ReadRecoveryAt(ctx, e, offset, limit)
}
func TestClaudeExactChangedContentRefuses(t *testing.T) {
	base, project := claudeOSFixture(t)
	rt := &mutatingClaudeRuntime{OSRuntime: base}
	rt.beforeRead = func(FileEntry) {
		path := filepath.Join(project, exactClaudeID+".jsonl")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.ReplaceAll(raw, []byte("hello"), []byte("other"))
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ResolveNativeSelector(context.Background(), rt, NativeSelectorRequest{Agent: AgentClaude, NativeID: exactClaudeID})
	if err != nil || got.Status != NativeSelectorIncomplete {
		t.Fatalf("changed content: %+v %v", got, err)
	}
	if rt.recursive != 0 {
		t.Fatal("used recursive discovery")
	}
}

func TestClaudeExactVisitorClosesDescriptors(t *testing.T) {
	for _, kind := range []string{"success", "callback-error", "parent-mutation", "probe-cancellation"} {
		t.Run(kind, func(t *testing.T) {
			base, project := claudeOSFixture(t)
			rt := &mutatingClaudeRuntime{OSRuntime: base}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sentinel := errors.New("callback refused")
			var want error
			switch kind {
			case "parent-mutation":
				want = ErrArtifactChanged
				rt.mutate = func(Artifact) {
					if err := os.Chmod(project, 0750); err != nil {
						t.Fatal(err)
					}
				}
			case "probe-cancellation":
				want = context.Canceled
				rt.mutate = func(Artifact) { cancel() }
			case "callback-error":
				want = sentinel
			}
			roots := rt.NativeRoots(AgentClaude)
			_, err := rt.VisitNativeSelectorFiles(ctx, roots[0], exactClaudeID, RecoveryMetadataLimit, func(FileEntry) error {
				if kind == "callback-error" {
					return sentinel
				}
				return nil
			})
			if !errors.Is(err, want) {
				t.Fatalf("err=%v want=%v", err, want)
			}
			if len(rt.opened) != 2 {
				t.Fatalf("opened %d descriptors, want root and project", len(rt.opened))
			}
			for _, file := range rt.opened {
				if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) && !errors.Is(err, os.ErrInvalid) {
					t.Errorf("descriptor %s remains open: %v", file.Name(), err)
				}
			}
		})
	}
}
