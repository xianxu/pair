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
		ValidRoot:       existingRoots("/env/root", "/repo", "/default/root"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/env/root" {
		t.Fatalf("Root = %q, want /env/root", root.Root)
	}
	if root.Source != "PAIR_HOME" {
		t.Fatalf("Source = %q, want PAIR_HOME", root.Source)
	}
}

func TestResolveAssetRootUsesExecutableSiblingRoot(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/repo/bin/pair",
		DefaultPairHome: "/default/root",
		ValidRoot:       existingRoots("/repo", "/default/root"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/repo" {
		t.Fatalf("Root = %q, want /repo", root.Root)
	}
}

func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/repo",
		ValidRoot:       existingRoots("/repo"),
	})
	if err != nil {
		t.Fatalf("ResolveAssetRoot error = %v", err)
	}
	if root.Root != "/repo" {
		t.Fatalf("Root = %q, want /repo", root.Root)
	}
}

func TestResolveAssetRootFallsBackToEmbeddedRootAfterAdjacentRoots(t *testing.T) {
	root, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/default/root",
		EmbeddedRoot:    "/data/pair/runtime/abc/pair-home",
		ValidRoot:       existingRoots("/data/pair/runtime/abc/pair-home"),
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

func TestResolveAssetRootReportsMissingRootAndPairHome(t *testing.T) {
	_, err := ResolveAssetRoot(AssetRootInput{
		Executable:      "/home/me/.local/bin/pair",
		DefaultPairHome: "/repo",
		ValidRoot:       existingRoots(),
	})
	if err == nil {
		t.Fatal("ResolveAssetRoot error = nil, want missing-root error")
	}
	for _, want := range []string{"main.kdl", "PAIR_HOME", "/home/me/.local", "/repo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidRootMarker(t *testing.T) {
	if got := ValidRootMarker("/repo"); got != "/repo/zellij/layouts/main.kdl" {
		t.Fatalf("ValidRootMarker = %q", got)
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
