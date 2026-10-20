package couchcmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This opt-in check uses the production argv adapter and strict decoder.
func TestPair722RealQueueConformance(t *testing.T) {
	fixture := newRealDrainQueueFixture(t)
	base, probe, cwd, executable := fixture.root, fixture.root, fixture.cwd, fixture.binary
	workerTree := fixture.workerTree
	q := sdlcDrainQueue{workspace: cwd, executable: executable}
	ctx := context.Background()
	status := func(label string) QueueSnapshot {
		t.Helper()
		s, err := q.Status(ctx)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		b, _ := json.Marshal(s)
		t.Logf("%s %s", label, b)
		return s
	}
	empty := func(label string) {
		t.Helper()
		s := status(label)
		if !s.Complete || s.State != "running" || len(s.Active) != 0 || len(s.PauseOwners) != 0 {
			t.Fatalf("not clean: %+v", s)
		}
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	const first = "couch-drain:00000000-0000-4000-8000-000000000722"
	const other = "pair722-other-owner"
	t.Cleanup(func() { _ = q.Resume(ctx, first); _ = q.Resume(ctx, other) })
	empty("initial")
	reason := "Couch drain 00000000-0000-4000-8000-000000000722: literal $(false); quoted reason"
	must(q.Pause(ctx, first, reason, false))
	must(q.Pause(ctx, first, "changed replay reason", false))
	must(q.Pause(ctx, other, "other owner", true))
	s := status("two-owners")
	if !s.Complete || s.State != "paused" || len(s.PauseOwners) != 2 || len(s.Active) != 0 {
		t.Fatal("two-owner mismatch")
	}
	found := false
	for _, p := range s.PauseOwners {
		if p.Owner == first {
			found = p.Reason == reason && !p.CancelRunning
		}
	}
	if !found {
		t.Fatal("literal reason or replay identity changed")
	}
	must(q.Resume(ctx, first))
	must(q.Resume(ctx, first))
	must(q.Resume(ctx, "absent-owner"))
	s = status("only-other-owner")
	if len(s.PauseOwners) != 1 || s.PauseOwners[0].Owner != other || s.State != "paused" {
		t.Fatal("resume released foreign owner")
	}
	must(q.Resume(ctx, other))
	empty("owners-released")
	if workerTree != "" {
		t.Run("worker-refuses", func(t *testing.T) {
			t.Setenv("COUCH_TREE", workerTree)
			if err := q.Pause(ctx, "forbidden-worker", "private refusal", false); err == nil {
				t.Fatal("worker mutation accepted")
			}
		})
		empty("worker-refusal-no-effect")
	}

	start := func() func(string, string, []string) map[string]any {
		t.Helper()
		hc, cancel := context.WithTimeout(context.Background(), time.Minute)
		cmd := exec.CommandContext(hc, probe+"/lease-helper")
		cmd.Dir = cwd
		cmd.WaitDelay = time.Second
		in, err := cmd.StdinPipe()
		must(err)
		out, err := cmd.StdoutPipe()
		must(err)
		var diag bytes.Buffer
		cmd.Stderr = &diag
		must(cmd.Start())
		scanner := bufio.NewScanner(out)
		encoder := json.NewEncoder(in)
		t.Cleanup(func() {
			_ = in.Close()
			if err := cmd.Wait(); err != nil {
				t.Errorf("helper: %v %s", err, diag.String())
			}
			cancel()
		})
		return func(action, id string, kinds []string) map[string]any {
			t.Helper()
			must(encoder.Encode(map[string]any{"Action": action, "ID": id, "Kinds": kinds}))
			if !scanner.Scan() {
				t.Fatalf("helper lost: %v %s", scanner.Err(), diag.String())
			}
			var r map[string]any
			must(json.Unmarshal(scanner.Bytes(), &r))
			return r
		}
	}
	send := start()
	good := func(action, id string, kinds []string) map[string]any {
		t.Helper()
		r := send(action, id, kinds)
		if e, ok := r["error"]; ok {
			t.Fatalf("helper %s %s: %v", action, id, e)
		}
		return r
	}
	good("enroll", "pair722-suite", []string{"suite"})
	good("enroll", "pair722-merge", []string{"merge", "landing"})
	good("enroll", "pair722-waiter", []string{"suite"})
	empty("waiting-not-active")
	for _, id := range []string{"pair722-suite", "pair722-merge"} {
		if good("admit", id, nil)["admitted"] != true {
			t.Fatal("not admitted")
		}
	}
	s = status("all-three-active")
	if !s.Complete || len(s.Active) != 3 {
		t.Fatal("missing active kinds")
	}
	kinds := map[string]bool{}
	for _, a := range s.Active {
		if a.Workspace != "fixture:0" || a.Head != "pair722-private-head" || a.State != "running" {
			t.Fatalf("activity mismatch: %+v", a)
		}
		kinds[a.Kind] = true
	}
	if !kinds["suite"] || !kinds["merge"] || !kinds["landing"] {
		t.Fatal("kind omitted")
	}
	must(q.Pause(ctx, first, reason, false))
	r := send("admit", "pair722-waiter", nil)
	if r["paused"] != true || r["admitted"] != false {
		t.Fatal("pause admitted waiting work")
	}
	s = status("graceful-preserves-active")
	for _, a := range s.Active {
		if a.State != "running" {
			t.Fatal("graceful interrupted active work")
		}
	}
	must(q.Pause(ctx, other, "fast private drain", true))
	s = status("fast-cancel-requested")
	for _, a := range s.Active {
		if a.Kind == "suite" && a.State != "interrupting" || a.Kind != "suite" && a.State != "running" {
			t.Fatalf("wrong fast state: %+v", a)
		}
	}
	if good("cancel", "pair722-suite", nil)["cancel_requested"] != true || good("cancel", "pair722-merge", nil)["cancel_requested"] != false {
		t.Fatal("wrong cancellation scope")
	}
	must(q.Resume(ctx, other))
	must(q.Resume(ctx, first))
	s = status("resume-does-not-restart")
	if len(s.Active) != 3 || s.State != "running" {
		t.Fatal("resume lost active work")
	}
	for _, a := range s.Active {
		if a.Kind == "suite" && a.State != "interrupting" {
			t.Fatal("resume restarted suite")
		}
	}
	if good("retire", "pair722-suite", nil)["interrupted"] != true {
		t.Fatal("cancellation recorded success")
	}
	if good("retire", "pair722-merge", nil)["interrupted"] != false || good("retire", "pair722-waiter", nil)["interrupted"] != false {
		t.Fatal("noncancelled retirement interrupted")
	}
	b, err := os.ReadFile(base + "/queue/interrupted/pair722-suite.json")
	must(err)
	var retained map[string]any
	must(json.Unmarshal(b, &retained))
	if retained["outcome"] != "interrupted" {
		t.Fatal("missing interrupted evidence")
	}
	t.Logf("retained-interruption %s", b)
	empty("retired-final")
	t.Run("unknown-private-residue", func(t *testing.T) {
		t.Setenv("WF_SDLC_QUEUE_ROOT", base+"/queue-unknown-pair5")
		orphan := start()
		r := orphan("enroll", "pair722-orphan", []string{"suite", "merge", "landing"})
		if r["error"] != nil {
			t.Fatal(r)
		}
		r = orphan("admit", "pair722-orphan", nil)
		if r["admitted"] != true {
			t.Fatal(r)
		}
		r = orphan("orphan", "pair722-orphan", nil)
		if r["error"] != nil {
			t.Fatal(r)
		}
		s := status("unknown-lifetime")
		if s.Complete || s.Detail == "" || len(s.Active) != 3 || s.Quiescent() {
			t.Fatal("unknown became quiescent")
		}
		for _, a := range s.Active {
			if a.State != "unknown" {
				t.Fatal("lost lease remained known")
			}
		}
		must(q.Pause(ctx, first, reason, false))
		s = status("unknown-with-owner")
		if s.Complete || s.State != "paused" || s.Quiescent() {
			t.Fatal("pause cleared unknown")
		}
		must(q.Resume(ctx, first))
		s = status("unknown-after-resume")
		if s.Complete || s.Quiescent() || len(s.Active) != 3 {
			t.Fatal("resume erased unknown")
		}
		must(os.MkdirAll(filepath.Join(base, "queue-unknown-pair5", "operations"), 0700))
		must(os.WriteFile(filepath.Join(base, "queue-unknown-pair5", "operations", "corrupt.json"), []byte("{broken"), 0600))
		s = status("corrupt-lease")
		if s.Complete || s.Detail == "" || s.Quiescent() {
			t.Fatal("corrupt observation proved absence")
		}
	})
	empty("primary-fixture-final")
}
