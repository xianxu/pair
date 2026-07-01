package entrypoint

import (
	"strings"
	"testing"
)

func TestResolveAssetRootPrefersPairHome(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		PairHome:        "/env/root",
		Executable:      "/repo/bin/pair",
		DefaultPairHome: "/default/root",
		PairShellExists: existingRoots("/env/root", "/repo", "/default/root"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/env/root" {
		t.Fatalf("Root = %q, want /env/root", root.Root)
	}
	if root.ShellPath != "/env/root/bin/pair-shell" {
		t.Fatalf("ShellPath = %q, want /env/root/bin/pair-shell", root.ShellPath)
	}
}

func TestResolveAssetRootUsesExecutableSiblingRoot(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/repo/bin/pair",
		DefaultPairHome: "/default/root",
		PairShellExists: existingRoots("/repo", "/default/root"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/repo" {
		t.Fatalf("Root = %q, want /repo", root.Root)
	}
	if root.ShellPath != "/repo/bin/pair-shell" {
		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
	}
}

func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/repo",
		PairShellExists: existingRoots("/repo"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/repo" {
		t.Fatalf("Root = %q, want /repo", root.Root)
	}
	if root.ShellPath != "/repo/bin/pair-shell" {
		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
	}
}

func TestResolveAssetRootFallsBackToEmbeddedRootAfterAdjacentRoots(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/default/root",
		EmbeddedRoot:    "/data/pair/runtime/abc/pair-home",
		PairShellExists: existingRoots("/data/pair/runtime/abc/pair-home"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/data/pair/runtime/abc/pair-home" {
		t.Fatalf("Root = %q, want embedded root", root.Root)
	}
	if root.Source != "embedded runtime" {
		t.Fatalf("Source = %q, want embedded runtime", root.Source)
	}
}

func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
	_, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/repo",
		PairShellExists: existingRoots(),
	})
	if err == nil {
		t.Fatal("ResolveAssetRoot error = nil, want missing-root error")
	}
	for _, want := range []string{"pair-shell", "PAIR_HOME", "/home/me/.local", "/repo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func existingRoots(roots ...string) func(string) bool {
	set := make(map[string]bool, len(roots))
	for _, root := range roots {
		set[root] = true
	}
	return func(root string) bool {
		return set[root]
	}
}
