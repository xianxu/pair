package wrapcmd

import (
	"bufio"
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
	_ = group.Wait()
	if err := syscall.Kill(-pgid, 0); err != syscall.ESRCH {
		t.Fatalf("owned group survived: %v", err)
	}
	_ = group.Wait() // cleanup is joined and idempotent
}
