package zellijquery

import (
	"context"
	"errors"
	"testing"
)

func TestBoundedCommandClassificationExcludesInteractiveAndCreators(t *testing.T) {
	for _, args := range [][]string{{"action", "rename-pane", "agent"}, {"--session", "demo", "action", "list-clients"}, {"kill-session", "demo"}, {"delete-session", "demo"}, {"list-sessions", "--no-formatting"}} {
		if !IsBoundedNativeCommand(args) {
			t.Fatalf("bounded command rejected: %q", args)
		}
	}
	for _, args := range [][]string{nil, {"attach", "demo"}, {"--new-session-with-layout", "layout", "--session", "demo"}, {"watch"}, {"subscribe"}, {"--server", "/native/server"}} {
		if IsBoundedNativeCommand(args) {
			t.Fatalf("interactive/creator command admitted: %q", args)
		}
	}
}

func TestBoundedCommandWaitDoesNotRunAfterCancellation(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	err = WithProbe(ctx, env, func(context.Context) error { called = true; return nil })
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("cancelled admission: called=%v err=%v", called, err)
	}
}
