package wrapcmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOwnedGroupStopsDescendantsBeforeReap(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "sleep 60 & echo $!; wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	child, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	pgid, err := syscall.Getpgid(child)
	if err != nil || pgid != cmd.Process.Pid {
		t.Fatalf("child group=%d err=%v", pgid, err)
	}
	group := newOwnedGroup(cmd, time.Millisecond)
	group.StartStop()
	group.StartStop()
	waited := make(chan struct{})
	go func() { _ = group.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not stop the owned group: the leader is still waiting on its descendant")
	}
	// The real end state: the descendant and then the whole group are gone.
	// Right after the leader's reap the killed descendant may still be an
	// unreaped zombie under launchd (kill(pid, 0) succeeds on it), and darwin
	// answers kill(-pgid, 0) with EPERM while the last member is mid-exit
	// (measured: that member already reads ESRCH and ps lists nothing). Both
	// settle to ESRCH promptly; only one still present after the bound survived.
	settled := func(probe func() error, transient error) error {
		deadline := time.Now().Add(2 * time.Second)
		for {
			err := probe()
			if err == syscall.ESRCH {
				return nil
			}
			if (err != nil && err != transient) || time.Now().After(deadline) {
				return fmt.Errorf("still present: %v", err)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	if err := settled(func() error { return syscall.Kill(child, 0) }, nil); err != nil {
		t.Fatalf("descendant %d survived: %v", child, err)
	}
	if err := settled(func() error { return syscall.Kill(-pgid, 0) }, syscall.EPERM); err != nil {
		t.Fatalf("owned group survived: %v", err)
	}
	_ = group.Wait() // cleanup is joined and idempotent
}

// pair#595 close review BR-3: a group that exits on SIGTERM is reaped at once,
// not after the whole grace.
func TestOwnedGroupFinishesEarlyWhenTheGroupExits(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "sleep 60 & wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	time.Sleep(50 * time.Millisecond)
	group := newOwnedGroup(cmd, 5*time.Second)
	start := time.Now()
	_ = group.Wait()
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cleanup spent %s of a 5s grace on a group that exited on SIGTERM", elapsed)
	}
}

// The Linux probe's parser: a zombie member, including our unreaped leader,
// does not keep the group live; a command name with spaces and parens parses.
func TestProcStatLiveInGroup(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"4242 (codex) S 1 4242 4242 0 -1", true},
		{"4242 (sh) Z 1 4242 4242 0 -1", false},
		{"4243 (sleep) S 4242 4242 4242 0 -1", true},
		{"4244 (other) S 1 9999 9999 0 -1", false},
		{"4245 (we (ird) name) R 1 4242 4242 0 -1", true},
		{"garbage", false},
	} {
		if got := procStatLiveInGroup([]byte(tc.line), 4242); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.line, got, tc.want)
		}
	}
}
