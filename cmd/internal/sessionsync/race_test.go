package sessionsync

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestHelperSessionSyncWriter is one writer process for the race below; it
// is a no-op unless the parent sets SESSIONSYNC_RACE_FILE.
func TestHelperSessionSyncWriter(t *testing.T) {
	file := os.Getenv("SESSIONSYNC_RACE_FILE")
	if file == "" {
		t.Skip("helper process")
	}
	for _, err := os.Stat(os.Getenv("SESSIONSYNC_RACE_GO")); err != nil; _, err = os.Stat(os.Getenv("SESSIONSYNC_RACE_GO")) {
		time.Sleep(time.Millisecond)
	}
	s := &Sync{Ledger: &Ledger{Dir: os.Getenv("SESSIONSYNC_RACE_STORE")}, Now: time.Now, Diff: UnifiedDiff, Canonical: Canonical}
	_, err := s.Write(WriteRequest{File: file, Section: "Trains", Body: os.Getenv("SESSIONSYNC_RACE_BODY"), IfHash: os.Getenv("SESSIONSYNC_RACE_HASH")})
	var r *Refusal
	switch {
	case err == nil:
		os.Exit(0)
	case errors.As(err, &r):
		os.Exit(r.Exit)
	default:
		os.Exit(1)
	}
}

// Two processes write the same section with the hash they both read:
// exactly one wins, the other is told it is stale.
func TestTwoProcessWritersExactlyOneWins(t *testing.T) {
	r := newRig(t)
	h := r.hash("Trains")
	start := filepath.Join(t.TempDir(), "go")
	var cmds []*exec.Cmd
	for _, body := range []string{"writer one", "writer two"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHelperSessionSyncWriter$")
		cmd.Env = append(os.Environ(), "SESSIONSYNC_RACE_FILE="+r.p.File, "SESSIONSYNC_RACE_GO="+start,
			"SESSIONSYNC_RACE_STORE="+r.sync.Ledger.Dir, "SESSIONSYNC_RACE_HASH="+h, "SESSIONSYNC_RACE_BODY="+body)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, cmd)
	}
	r.put(start, "")
	var codes []int
	for _, cmd := range cmds {
		err := cmd.Wait()
		var exit *exec.ExitError
		switch {
		case err == nil:
			codes = append(codes, 0)
		case errors.As(err, &exit):
			codes = append(codes, exit.ExitCode())
		default:
			t.Fatal(err)
		}
	}
	sort.Ints(codes)
	if codes[0] != 0 || codes[1] != ExitStale {
		t.Fatalf("exit codes %v, want one winner and one stale", codes)
	}
	got := r.read(r.p.File)
	if strings.Contains(got, "writer one") == strings.Contains(got, "writer two") {
		t.Fatalf("exactly one body must land: %q", got)
	}
}
