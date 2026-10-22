package testcouch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

func TestActorCLITimeoutUsesRemainingCallerDeadline(t *testing.T) {
	now := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(90*time.Second))
	defer cancel()
	got, err := actorCLITimeout(ctx, now, 120*time.Second)
	if err != nil || got != 90*time.Second {
		t.Fatalf("caller budget=%v error=%v", got, err)
	}
	got, err = actorCLITimeout(context.Background(), now, 120*time.Second)
	if err != nil || got != 120*time.Second {
		t.Fatalf("fleet fallback=%v error=%v", got, err)
	}
	got, err = actorCLITimeout(context.Background(), now, 0)
	if err != nil || got != 15*time.Second {
		t.Fatalf("direct default=%v error=%v", got, err)
	}
	got, err = actorCLITimeout(context.Background(), now, 48*time.Hour)
	if err != nil || got != 24*time.Hour {
		t.Fatalf("bounded maximum=%v error=%v", got, err)
	}
	if _, err := actorCLITimeout(ctx, now.Add(91*time.Second), time.Second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired caller budget accepted: %v", err)
	}
	cancel()
	if _, err := actorCLITimeout(ctx, now, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller budget accepted: %v", err)
	}
}

func TestFleetStartupCommandSelectsExplicitPublicHubOnlyWhenRequested(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		f := &Fleet{Roots: RootsPlan{Mutable: map[string]string{"bin": "/private/fixture/bin"}}, Workspaces: Workspaces{Primary: "/private/fixture/repo"}, Environment: []string{"PRIVATE_RUN=fixture"}, Options: Options{ExplicitHub: explicit}}
		cmd := f.consoleCommand()
		want := []string{"/private/fixture/bin/couch"}
		if explicit {
			want = append(want, "/private/fixture/repo")
		}
		if !reflect.DeepEqual(cmd.Args, want) || cmd.Dir != f.Workspaces.Primary || !reflect.DeepEqual(cmd.Env, f.Environment) {
			t.Fatalf("explicit=%v args%v dir%s env%v", explicit, cmd.Args, cmd.Dir, cmd.Env)
		}
	}
}
