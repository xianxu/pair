package testcouch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFleetOptionsRefuseBeforeAllocation(t *testing.T) {
	for _, workers := range []int{-1, 0, 9, 1000000} {
		if err := validateOptions(Options{Workers: workers}); err == nil {
			t.Fatalf("accepted workers=%d", workers)
		}
	}
	if err := validateOptions(Options{Workers: 3, AgentKind: "claude"}); err != nil {
		t.Fatal(err)
	}
	if err := validateOptions(Options{Workers: 3, AgentKind: "installed"}); err == nil {
		t.Fatal("allowed installed-agent fallback")
	}
}

func TestCanceledStartDoesNotLaunchOrPublish(t *testing.T) {
	root := t.TempDir()
	f := &Fleet{Roots: RootsPlan{Root: root, Mutable: map[string]string{"bin": root}}, Workspaces: Workspaces{Primary: root}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.Start(ctx); err == nil {
		t.Fatal("canceled Start admitted")
	}
	if _, err := os.Stat(filepath.Join(root, "ownership.json")); !os.IsNotExist(err) {
		t.Fatalf("canceled Start published ownership: %v", err)
	}
	if f.console != nil || f.active != 0 {
		t.Fatal("canceled Start retained activity")
	}
}
