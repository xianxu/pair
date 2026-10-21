package testcouch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCandidatePathsRejectInvalidPlacement(t *testing.T) {
	for _, dir := range []string{"", ".", "relative/bin", "/", "/owned/../foreign"} {
		if _, err := candidatePaths(dir); err == nil {
			t.Errorf("accepted candidate directory %q", dir)
		}
	}
	root := t.TempDir()
	paths, err := candidatePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pair", "couch", "pair-launch-helper", "fixture-agent"} {
		if paths[name] != filepath.Join(root, name) {
			t.Fatalf("%s destination = %q", name, paths[name])
		}
	}
	if len(paths) != 4 {
		t.Fatalf("unexpected candidates: %v", paths)
	}
}

func TestValidateToolFactsRejectNonExecutableOrUnscopedTools(t *testing.T) {
	for _, tc := range []struct {
		path string
		mode os.FileMode
		want bool
	}{
		{"/tools/git", 0755, true}, {"relative/git", 0755, false}, {"", 0755, false},
		{"/tools/git", 0644, false}, {"/tools/git", os.ModeDir | 0755, false},
		{"/tools/git", os.ModeSymlink | 0755, false}, {"/tools/git", os.ModeNamedPipe | 0755, false},
	} {
		if err := validateToolFacts(tc.path, tc.mode); (err == nil) != tc.want {
			t.Errorf("validate(%q,%v) = %v", tc.path, tc.mode, err)
		}
	}
}

func TestDiscoverToolRequiresPhysicalExecutable(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "tool")
	if err := os.WriteFile(file, []byte("#!/bin/sh\nexit 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTool(file); err == nil {
		t.Fatal("accepted non-executable tool")
	}
	if err := os.Chmod(file, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(file, alias); err != nil {
		t.Fatal(err)
	}
	got, err := resolveTool(alias)
	physical, physicalErr := filepath.EvalSymlinks(file)
	if err != nil || physicalErr != nil || got != physical {
		t.Fatalf("physical tool = %q, %v; want %q (%v)", got, err, physical, physicalErr)
	}
	if _, err := resolveTool(filepath.Join(root, "missing")); err == nil {
		t.Fatal("accepted missing tool")
	}
}

func TestBuildCommandBoundsDiagnosticsAndHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	output, err := buildCommand(ctx, t.TempDir(), "/bin/sh", "-c", "printf '%070000d' 0; exit 7")
	if err == nil || len(output) > buildOutputLimit+100 || !strings.Contains(output, "truncated") {
		t.Fatalf("bounded failure: length=%d err=%v", len(output), err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := buildCommand(cancelled, t.TempDir(), "/bin/sh", "-c", "exit 0"); err == nil {
		t.Fatal("ignored build cancellation")
	}
}
