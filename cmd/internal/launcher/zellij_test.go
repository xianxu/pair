package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestZellijSnapshotQueryFailuresDiscardAllSessions(t *testing.T) {
	for _, stage := range []string{"list-sessions --no-formatting", "--session pair-b action list-clients"} {
		t.Run(stage, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "zellij")
			script := `#!/bin/sh
if [ "$*" = '` + stage + `' ]; then
  ` + zellijtest.ShellCase(t, "launcher-snapshot-failure-"+map[string]string{"list-sessions --short": "short", "list-sessions --no-formatting": "full", "--session pair-b action list-clients": "clients"}[stage]) + `
fi
case "$*" in
  'list-sessions --short') ` + zellijtest.ShellCase(t, "launcher-snapshot-short") + ` ;;
  'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "launcher-snapshot-full") + ` ;;
  *) ` + zellijtest.ShellCase(t, "launcher-snapshot-empty-clients") + ` ;;
esac
`
			if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := (ZellijSource{Path: path}).Snapshot()
			if err == nil || !strings.Contains(err.Error(), stage) {
				t.Errorf("error = %v, want failed query %q", err, stage)
			}
			if got != nil {
				t.Errorf("failed snapshot returned partial authority: %+v", got)
			}
		})
	}
}

func TestZellijSnapshotCancellation(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			want := error(context.Canceled)
			if deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			defer cancel()
			got, err := (ZellijSource{Path: "/bin/sh"}).SnapshotContext(ctx)
			if !errors.Is(err, want) || got != nil {
				t.Fatalf("snapshot = %+v, %v; want nil, %v", got, err, want)
			}
		})
	}
}

func TestZellijSnapshotCancellationDuringQuery(t *testing.T) {
	for _, stage := range []string{"list-sessions --no-formatting", "--session pair-b action list-clients"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "zellij")
			ready := filepath.Join(dir, "ready")
			script := `#!/bin/sh
if [ "$*" = '` + stage + `' ]; then
  touch '` + ready + `'
  exec sleep 30
fi
case "$*" in
  'list-sessions --short') ` + zellijtest.ShellCase(t, "launcher-snapshot-short") + ` ;;
  'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "launcher-snapshot-full") + ` ;;
  *) ` + zellijtest.ShellCase(t, "launcher-snapshot-empty-clients") + ` ;;
esac
`
			if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})
			var got []Session
			var err error
			go func() {
				got, err = (ZellijSource{Path: path}).SnapshotContext(ctx)
				close(done)
			}()
			limit := testwait.After(t)
			tick := time.NewTicker(time.Millisecond)
			defer tick.Stop()
			for {
				if _, statErr := os.Stat(ready); statErr == nil {
					break
				}
				select {
				case <-done:
					t.Fatalf("query exited before cancellation barrier: %v", err)
				case <-tick.C:
				case <-limit:
					cancel()
					<-done
					t.Fatal("query did not reach cancellation barrier")
				}
			}
			cancel()
			<-done
			if got != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("snapshot = %+v, %v; want nil, context canceled", got, err)
			}
		})
	}
}

func TestZellijSnapshotEmptyInventoryDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		empty bool
	}{
		{"empty", true}, {"unknown error", false}, {"wrong exit status", false}, {"partial stdout", false}, {"extra diagnostic", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "zellij")
			caseName := "launcher-empty-diagnostic-" + strings.ReplaceAll(tc.name, " ", "-")
			script := "#!/bin/sh\n" + zellijtest.ShellCase(t, caseName)
			if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := (ZellijSource{Path: path}).Snapshot()
			if got != nil || (err == nil) != tc.empty {
				t.Fatalf("snapshot = %+v, %v; want empty success = %v", got, err, tc.empty)
			}
		})
	}
}

func TestZellijSourceClassifiesSessions(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "zellij.log")
	zellij := filepath.Join(dir, "zellij")
	script := `#!/usr/bin/env bash
printf '%s\n' "$*" >> "` + log + `"
case "$*" in
  "list-sessions --short") ` + zellijtest.ShellCase(t, "launcher-classification-short") + ` ;;
  "list-sessions --no-formatting") ` + zellijtest.ShellCase(t, "launcher-classification-full") + ` ;;
  "--session pair-live action list-clients") ` + zellijtest.ShellCase(t, "launcher-classification-attached") + ` ;;
  "--session pair-detached action list-clients") ` + zellijtest.ShellCase(t, "launcher-snapshot-empty-clients") + ` ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ZellijSource{Path: zellij}.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	want := []Session{
		{Name: "pair-detached", State: SessionDetached},
		{Name: "pair-gone", State: SessionExited},
		{Name: "pair-live", State: SessionAttached},
	}
	if len(got) != len(want) {
		t.Fatalf("Snapshot returned %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Snapshot[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

// The counted invariant, at the seam: asking about named sessions costs one
// list-clients per NAMED live session, however many sessions the host has
// (pair#228). Exited and absent names cost none.
func TestSnapshotSessionsAsksOnlyTheNamedSessions(t *testing.T) {
	sessions := map[string]string{"pair-a": "detached", "pair-b": "detached", "pair-c": "attached", "pair-d": "exited"}
	for i := 0; i < 20; i++ {
		sessions[fmt.Sprintf("pair-x%02d", i)] = "detached"
	}
	path, log := pairlifecycletest.StubZellij(t, sessions)
	got, err := ZellijSource{Path: path}.SnapshotSessionsContext(context.Background(), []string{"pair-a", "pair-c", "pair-d", "pair-absent"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{Name: "pair-a", State: SessionDetached}, {Name: "pair-c", State: SessionAttached}, {Name: "pair-d", State: SessionExited}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if n := pairlifecycletest.CountCalls(t, log, "list-clients"); n != 2 {
		t.Fatalf("list-clients calls = %d, want 2 (a and c) regardless of the 20 other sessions", n)
	}
}

// Liveness asks nobody for clients, and says so in the state it reports: a
// live session is SessionLive, never a guess at attached or detached.
func TestLivenessAsksNoSessionForClients(t *testing.T) {
	path, log := pairlifecycletest.StubZellij(t, map[string]string{"pair-a": "detached", "pair-b": "attached", "pair-c": "exited"})
	got, err := ZellijSource{Path: path}.LivenessContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{Name: "pair-a", State: SessionLive}, {Name: "pair-b", State: SessionLive}, {Name: "pair-c", State: SessionExited}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if n := pairlifecycletest.CountCalls(t, log, "list-clients"); n != 0 {
		t.Fatalf("list-clients calls = %d, want 0", n)
	}
}
