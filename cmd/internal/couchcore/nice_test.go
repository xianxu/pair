package couchcore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// #502 replaced #498's launch nice: a blocked start runs the target, and
// everything it forks, at Couch's own niceness; Couch's renicer lowers what
// a slot's agent spawns later. The grandchild reads its own, so this pins
// the inheritance through the helper's exec.
func TestBlockedStartRunsTheTargetAtCouchsOwnNiceness(t *testing.T) {
	t.Parallel()

	own := ownNice(t)
	marker := filepath.Join(t.TempDir(), "nice")
	r := ExecRunner{LaunchHelper: os.Args[0]}
	h, err := r.StartBlocked(context.Background(), t.TempDir(), []string{"sh", "-c", "sh -c 'ps -o nice= -p $$' > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_TARGET_MARKER=" + marker,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("Wait = %d", code)
	}
	raw, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(raw)); got != strconv.Itoa(own) {
		t.Fatalf("grandchild nice = %q, want Couch's own %d", got, own)
	}
}

func ownNice(t *testing.T) int {
	t.Helper()
	out, err := exec.Command("ps", "-o", "nice=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Skipf("ps: %v", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("ps nice %q: %v", out, err)
	}
	return n
}
