package couchcore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// #498: the exec'd target, and everything it forks, runs at the runner's
// niceness. The grandchild reads its own, so this pins inheritance through the
// helper's exec, not just a call made on the helper.
func TestBlockedStartRunsTheTargetsDescendantsAtTheRunnersNiceness(t *testing.T) {
	t.Parallel()

	nice := min(ownNice(t)+3, 20)
	marker := filepath.Join(t.TempDir(), "nice")
	r := ExecRunner{LaunchHelper: os.Args[0], Nice: nice}
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
	if got := strings.TrimSpace(string(raw)); got != strconv.Itoa(nice) {
		t.Fatalf("grandchild nice = %q, want %d", got, nice)
	}
}

// A niceness the kernel refuses (raising priority needs root) cancels the
// launch before the target ever runs, and the error names the setting.
func TestBlockedStartRefusedNicenessCancelsBeforeTheTargetRuns(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root may raise priority")
	}

	marker := filepath.Join(t.TempDir(), "target-ran")
	r := ExecRunner{LaunchHelper: os.Args[0], Nice: ownNice(t) - 1}
	h, err := r.StartBlocked(context.Background(), t.TempDir(), []string{"sh", "-c", "printf exec > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_TARGET_MARKER=" + marker,
	}, 2*time.Second)
	if err == nil || h != nil {
		t.Fatalf("StartBlocked = %v, %v; want a refusal and no handle", h, err)
	}
	if !strings.Contains(err.Error(), "COUCH_SLOT_NICE") || !(strings.Contains(err.Error(), syscall.EACCES.Error()) || strings.Contains(err.Error(), syscall.EPERM.Error())) {
		t.Fatalf("error = %v, want the kernel's refusal naming COUCH_SLOT_NICE", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("refused launch ran its target: %v", err)
	}
}

func ownNice(t *testing.T) int {
	t.Helper()
	out, err := exec.Command("ps", "-o", "nice=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("ps nice %q: %v", out, err)
	}
	return n
}
