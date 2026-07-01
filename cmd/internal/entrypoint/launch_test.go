package entrypoint

import (
	"reflect"
	"testing"
)

func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"claude", "--", "--resume"})
	if req.Path != "/repo/bin/pair-shell" {
		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
	}
	want := []string{"pair", "claude", "--", "--resume"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}

func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"resume", "demo"})
	if req.Path != "/repo/bin/pair-shell" {
		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
	}
	want := []string{"pair", "resume", "demo"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}
