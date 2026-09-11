package launcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestZellijSourceClassifiesSessions(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "zellij.log")
	zellij := filepath.Join(dir, "zellij")
	script := `#!/usr/bin/env bash
printf '%s\n' "$*" >> "` + log + `"
case "$*" in
  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
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
