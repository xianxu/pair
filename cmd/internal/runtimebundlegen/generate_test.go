package runtimebundlegen

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGeneratePreservesExistingOutputOnFailure(t *testing.T) {
	repo := t.TempDir()
	out := filepath.Join(t.TempDir(), "runtime")
	writeMinimalRuntimeRepo(t, repo)

	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: fakeTerminfoCompiler{}}); err != nil {
		t.Fatalf("initial Generate error = %v", err)
	}
	before, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(initial manifest) error = %v", err)
	}

	if err := os.Remove(filepath.Join(repo, "bin", "pair-help")); err != nil {
		t.Fatalf("Remove(pair-help) error = %v", err)
	}
	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: fakeTerminfoCompiler{}}); err == nil {
		t.Fatal("Generate error = nil, want missing asset error")
	}

	after, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(preserved manifest) error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("failed generation changed existing output manifest")
	}
}

func TestGenerateConcurrentSameOutputSucceeds(t *testing.T) {
	repo := t.TempDir()
	out := filepath.Join(t.TempDir(), "runtime")
	writeMinimalRuntimeRepo(t, repo)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: fakeTerminfoCompiler{}})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Generate error = %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "manifest.json")); err != nil {
		t.Fatalf("Stat(manifest) error = %v", err)
	}
}

// This fake models the compiler's filesystem result, including partial output
// on failure. Generate still performs its real validation and atomic staging.
type fakeTerminfoCompiler struct {
	directory string
	empty     bool
	fail      error
}

func (f fakeTerminfoCompiler) Compile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(data), "pair-vt-256color|") {
		return errors.New("invalid source")
	}
	if !f.empty {
		directory := f.directory
		if directory == "" {
			directory = "p"
		}
		path := filepath.Join(destination, directory, "pair-vt-256color")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte("compiled-profile"), 0644); err != nil {
			return err
		}
	}
	return f.fail
}

func TestGenerateCompilerResultsAndFailures(t *testing.T) {
	// Unit coverage must not depend on an installed compiler.
	t.Setenv("PATH", t.TempDir())
	for _, directory := range []string{"p", "70"} {
		t.Run(directory, func(t *testing.T) {
			repo, out := t.TempDir(), filepath.Join(t.TempDir(), "runtime")
			writeMinimalRuntimeRepo(t, repo)
			if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: fakeTerminfoCompiler{directory: directory}}); err != nil {
				t.Fatal(err)
			}
			profile := filepath.Join(out, "files", "terminfo", "70", "pair-vt-256color")
			if data, err := os.ReadFile(profile); err != nil || string(data) != "compiled-profile" {
				t.Fatalf("profile=%q err=%v", data, err)
			}
			before, err := os.ReadFile(filepath.Join(out, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			for _, compiler := range []fakeTerminfoCompiler{{empty: true}, {fail: errors.New("compiler failed after output")}} {
				if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: compiler}); err == nil {
					t.Fatal("accepted missing or failed compilation")
				}
				after, err := os.ReadFile(filepath.Join(out, "manifest.json"))
				if err != nil || string(after) != string(before) {
					t.Fatalf("failed compilation replaced output: %v", err)
				}
				entries, err := os.ReadDir(filepath.Dir(out))
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 1 || entries[0].Name() != "runtime" {
					t.Fatalf("temporary compiler/stage residue: %v", entries)
				}
			}
		})
	}
}

func TestTicCompilerConformance(t *testing.T) {
	for _, command := range []string{"tic", "infocmp"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("native compiler conformance requires %s", command)
		}
	}
	repo, out := t.TempDir(), filepath.Join(t.TempDir(), "runtime")
	writeMinimalRuntimeRepo(t, repo)
	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out, Compiler: TicCompiler{}}); err != nil {
		t.Fatal(err)
	}
	data, err := exec.Command("infocmp", "-A", filepath.Join(out, "files", "terminfo"), "pair-vt-256color").CombinedOutput()
	if err != nil || !strings.Contains(string(data), "cols#80") || !strings.Contains(string(data), "lines#24") || !strings.Contains(string(data), "clear=\\E[H\\E[2J") {
		t.Fatalf("compiled capabilities=%s err=%v", data, err)
	}
}

func writeMinimalRuntimeRepo(t *testing.T, repo string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "terminfo", "pair-vt-256color.ti"), "pair-vt-256color|Pair test terminal,\n cols#80, lines#24, clear=\\E[H\\E[2J,\n", 0o644)
	for _, logical := range explicitAssetPaths {
		writeFile(t, filepath.Join(repo, filepath.FromSlash(logical)), "#!/bin/sh\n", 0o755)
	}
	writeFile(t, filepath.Join(repo, "bin", "lib", "shared.sh"), "shared\n", 0o644)
	writeFile(t, filepath.Join(repo, "nvim", "init.lua"), "-- init\n", 0o644)
	writeFile(t, filepath.Join(repo, "zellij", "config.kdl"), "keybinds {}\n", 0o644)
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
