package runtimebundlegen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePreservesExistingOutputOnFailure(t *testing.T) {
	repo := t.TempDir()
	out := filepath.Join(t.TempDir(), "runtime")
	writeMinimalRuntimeRepo(t, repo)

	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err != nil {
		t.Fatalf("initial Generate error = %v", err)
	}
	before, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(initial manifest) error = %v", err)
	}

	if err := os.Remove(filepath.Join(repo, "bin", "pair-shell")); err != nil {
		t.Fatalf("Remove(pair-shell) error = %v", err)
	}
	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err == nil {
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

func writeMinimalRuntimeRepo(t *testing.T, repo string) {
	t.Helper()
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
