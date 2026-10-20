package couchcmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPair722RealQueueLegacyWaiterConformance(t *testing.T) {
	private := newRealDrainQueueFixture(t)
	base, fixture := private.root, private.cwd
	t.Setenv("WF_SDLC_QUEUE_ROOT", base+"/queue-legacy")
	t.Setenv("WF_TEST_LOCK_FILE", base+"/capacity-legacy")
	q := sdlcDrainQueue{workspace: fixture, executable: private.binary}
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	status := func(label string) QueueSnapshot {
		t.Helper()
		s, err := q.Status(ctx)
		must(err)
		raw, _ := json.Marshal(s)
		t.Logf("%s %s", label, raw)
		return s
	}
	const owner = "pair722-legacy-owner"
	t.Cleanup(func() { _ = q.Resume(ctx, owner) })
	s := status("initial")
	if !s.Complete || s.State != "running" || len(s.Active) != 0 || len(s.PauseOwners) != 0 {
		t.Fatal("not initially clean")
	}
	must(q.Pause(ctx, owner, "private legacy check", false))
	hold := func(path string, raw []byte) func() {
		t.Helper()
		must(os.MkdirAll(filepath.Dir(path), 0700))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		must(err)
		must(syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
		_, err = f.Write(raw)
		must(err)
		must(f.Sync())
		closed := false
		release := func() {
			if !closed {
				closed = true
				must(f.Close())
			}
		}
		t.Cleanup(release)
		return release
	}
	token := base + "/capacity-legacy.queue"
	ticket := base + "/capacity-legacy.tickets/000000000001"
	raw := []byte(`{"repo":"old","worktree":"/fixture/old","pid":123,"started":"fixture"}`)
	for _, path := range []string{token, ticket} {
		release := hold(path, raw)
		s = status("unbound-held-" + filepath.Base(path))
		if s.Complete || s.State != "paused" || len(s.Active) != 0 || s.Detail == "" || !strings.Contains(s.Detail, path) || s.Quiescent() {
			t.Fatal("unbound pending admission became complete or active")
		}
		current, err := os.ReadFile(path)
		must(err)
		if !bytes.Equal(current, raw) {
			t.Fatal("status mutated held pending admission")
		}
		release()
		s = status("unbound-released")
		if !s.Complete || len(s.Active) != 0 || !s.Quiescent() {
			t.Fatal("released pending admission remained unknown")
		}
	}
	hc, cancel := context.WithTimeout(context.Background(), time.Minute)
	cmd := exec.CommandContext(hc, base+"/lease-helper")
	cmd.Dir = fixture
	cmd.WaitDelay = time.Second
	in, err := cmd.StdinPipe()
	must(err)
	out, err := cmd.StdoutPipe()
	must(err)
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	must(cmd.Start())
	t.Cleanup(func() {
		_ = in.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("helper: %v %s", err, diagnostics.String())
		}
		cancel()
	})
	scanner := bufio.NewScanner(out)
	encoder := json.NewEncoder(in)
	call := func(action string) map[string]any {
		t.Helper()
		must(encoder.Encode(map[string]any{"Action": action, "ID": "pair722-verified-waiter", "Kinds": []string{"suite"}}))
		if !scanner.Scan() {
			t.Fatalf("helper lost: %v %s", scanner.Err(), diagnostics.String())
		}
		var result map[string]any
		must(json.Unmarshal(scanner.Bytes(), &result))
		return result
	}
	if result := call("enroll"); result["error"] != nil {
		t.Fatal(result)
	}
	data, err := os.ReadFile(base + "/queue-legacy/operations/pair722-verified-waiter.json")
	must(err)
	var op map[string]any
	must(json.Unmarshal(data, &op))
	binding, err := json.Marshal(map[string]any{"queue_operation_id": op["id"], "queue_pid": op["pid"], "queue_pid_started": op["pid_started"], "queue_root": op["root"]})
	must(err)
	releaseToken := hold(token, binding)
	releaseTicket := hold(ticket, binding)
	s = status("verified-waiting-bindings")
	if !s.Complete || len(s.Active) != 0 || s.Detail != "" || !s.Quiescent() {
		t.Fatal("verified waiting binding damaged completeness")
	}
	if result := call("admit"); result["paused"] != true || result["admitted"] != false {
		t.Fatalf("waiter escaped pause: %+v", result)
	}
	releaseTicket()
	releaseToken()
	must(q.Resume(ctx, owner))
	if result := call("admit"); result["error"] != nil || result["admitted"] != true {
		t.Fatalf("resume failed: %+v", result)
	}
	s = status("verified-waiter-admitted")
	if !s.Complete || len(s.Active) != 1 || s.Active[0].State != "running" {
		t.Fatal("admitted waiter not visible")
	}
	if result := call("retire"); result["error"] != nil || result["interrupted"] != false {
		t.Fatal(result)
	}
	s = status("final")
	if !s.Complete || s.State != "running" || len(s.Active) != 0 || len(s.PauseOwners) != 0 {
		t.Fatal("not finally clean")
	}
}
