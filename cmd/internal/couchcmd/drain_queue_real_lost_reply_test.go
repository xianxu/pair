package couchcmd

import (
	"context"
	"os"
	"testing"
)

func TestPair722RealQueueLostReplyConformance(t *testing.T) {
	fixture := newRealDrainQueueFixture(t)
	probe, binary := fixture.root, fixture.binary
	t.Setenv("PAIR722_REAL_BINARY", binary)
	t.Setenv("PAIR722_DROP_REPLY", "")
	script := `#!/bin/sh
if [ "$PAIR722_DROP_REPLY" = "$2" ]; then
  "$PAIR722_REAL_BINARY" "$@" >/dev/null
  outcome=$?
  if [ "$outcome" -ne 0 ]; then exit "$outcome"; fi
  printf '%s\n' 'private fixture: effect applied, response lost' >&2
  exit 29
fi
exec "$PAIR722_REAL_BINARY" "$@"
`
	if err := os.WriteFile(probe+"/drop-reply", []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q := sdlcDrainQueue{workspace: fixture.cwd, executable: probe + "/drop-reply"}
	ctx := context.Background()
	const owner = "pair722-lost-reply"
	const other = "pair722-preserved-owner"
	t.Cleanup(func() { _ = os.Unsetenv("PAIR722_DROP_REPLY"); _ = q.Resume(ctx, owner); _ = q.Resume(ctx, other) })
	initial, err := q.Status(ctx)
	if err != nil || !initial.Complete || initial.State != "running" || len(initial.PauseOwners) != 0 || len(initial.Active) != 0 {
		t.Fatalf("fixture not clean: %+v %v", initial, err)
	}
	t.Setenv("PAIR722_DROP_REPLY", "pause")
	if err := q.Pause(ctx, owner, "literal lost response", false); err == nil {
		t.Fatal("lost pause reply reported success")
	} else {
		t.Logf("pause uncertain: %v", err)
	}
	s, err := q.Status(ctx)
	if err != nil || !s.Complete || s.State != "paused" || len(s.PauseOwners) != 1 || s.PauseOwners[0].Owner != owner {
		t.Fatalf("lost pause did not reconcile: %+v %v", s, err)
	}
	t.Logf("pause reconciled owner=%s complete=%v quiescent=%v", s.PauseOwners[0].Owner, s.Complete, s.Quiescent())
	t.Setenv("PAIR722_DROP_REPLY", "")
	if err := q.Pause(ctx, other, "preserve", false); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR722_DROP_REPLY", "resume")
	if err := q.Resume(ctx, owner); err == nil {
		t.Fatal("lost resume reply reported success")
	} else {
		t.Logf("resume uncertain: %v", err)
	}
	s, err = q.Status(ctx)
	if err != nil || !s.Complete || s.State != "paused" || len(s.PauseOwners) != 1 || s.PauseOwners[0].Owner != other {
		t.Fatalf("lost resume lost foreign owner: %+v %v", s, err)
	}
	t.Logf("resume reconciled remaining owner=%s complete=%v", s.PauseOwners[0].Owner, s.Complete)
	t.Setenv("PAIR722_DROP_REPLY", "")
	if err := q.Resume(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if err := q.Resume(ctx, other); err != nil {
		t.Fatal(err)
	}
	s, err = q.Status(ctx)
	if err != nil || !s.Complete || s.State != "running" || len(s.PauseOwners) != 0 || len(s.Active) != 0 {
		t.Fatalf("fixture not restored: %+v %v", s, err)
	}
	t.Log("final running/complete/empty; no replayed pause or automatic restart")
}
