// probes/zellijpark answers one question with an observation instead of an
// assertion: does a zellij SESSION survive its CLIENT being killed?
//
// #146's Decision 7 rests on it. couch's child is `pair` -> a zellij client, so
// if the session outlives the client then the work outlives the console, and
// couch needs no daemon -- only deterministic re-entry. If it does not, the
// daemon question reopens. `workshop/projects/couch.md` separately asserts
// "a direct SIGTERM is a kill, not TUI Park", which is the same fact stated the other
// way round and has never been measured.
//
// It creates a throwaway session, kills the client, looks, and deletes the
// session. It touches nothing else.
//
//	go run ./probes/zellijpark
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

const sessionName = "couch-park-probe"

func main() {
	cleanup()

	fmt.Printf("== starting a throwaway zellij session %q\n", sessionName)
	cmd := exec.Command("zellij", "--session", sessionName)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pty.StartWithSize: %v\n", err)
		os.Exit(1)
	}
	go drain(ptmx)

	if !waitForSession(true, 10*time.Second) {
		fmt.Fprintln(os.Stderr, "FAIL: the session never appeared")
		_ = cmd.Process.Kill()
		cleanup()
		os.Exit(1)
	}
	fmt.Println("   session is up")

	for _, step := range []struct {
		name string
		sig  syscall.Signal
	}{
		{"SIGTERM (direct process stop)", syscall.SIGTERM},
		{"SIGKILL (what a crashed console leaves behind)", syscall.SIGKILL},
	} {
		fmt.Printf("== killing the CLIENT with %s\n", step.name)
		if err := cmd.Process.Signal(step.sig); err != nil {
			fmt.Printf("   (client already gone: %v)\n", err)
		}
		time.Sleep(2 * time.Second)

		alive, state := sessionState()
		fmt.Printf("   session present=%v state=%q\n", alive, state)
		if alive && !strings.Contains(state, "EXITED") {
			fmt.Printf("   => PARK: the session outlived the client\n")
		} else if alive {
			fmt.Printf("   => EXITED-but-resurrectable: the panes are gone, the name remains\n")
		} else {
			fmt.Printf("   => KILL: the session went with the client\n")
		}
		if step.sig == syscall.SIGTERM && !alive {
			break // nothing left to SIGKILL
		}
	}

	fmt.Println("== cleaning up")
	_ = cmd.Process.Kill()
	cleanup()
}

func drain(f *os.File) {
	buf := make([]byte, 4096)
	for {
		if _, err := f.Read(buf); err != nil {
			return
		}
	}
}

func sessionState() (bool, string) {
	out, _ := exec.Command("zellij", "list-sessions", "--no-formatting").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, sessionName+" ") || line == sessionName {
			return true, strings.TrimSpace(line)
		}
	}
	return false, ""
}

func waitForSession(want bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if got, _ := sessionState(); got == want {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func cleanup() {
	_ = exec.Command("zellij", "delete-session", sessionName, "--force").Run()
}
