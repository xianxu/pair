// probes/escsmoke drives a real `pair term` under a pty with a real nvim child
// and asks nvim ITSELF — over its RPC socket, not by parsing the screen — what
// mode it is in after one ESC. It exists because #234's fix is a deadline in
// the terminal's stdin loop, and the property the operator reported ("two
// presses of ESC to leave insert mode") is one that pump-level unit tests can
// only approximate: they prove the mux forwards the byte, not that the editor
// behind a real pty acts on it.
//
// Committed rather than run from a scratch dir: a probe whose output gets
// quoted in an issue Log has to be re-runnable against a later commit.
//
//	go run ./probes/escsmoke            # against ./bin/pair
//	go run ./probes/escsmoke /path/pair
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/creack/pty"
)

// main is two lines on purpose: os.Exit SKIPS DEFERS, and this probe's cleanup
// (which quits nvim, kills the spawned `pair term` and closes its pty) is
// registered as one. Every path RETURNS a code so the defers unwind.
// TestNoProbeExitsPastItsOwnCleanup enforces the shape across every probe.
func main() { os.Exit(run()) }

func run() int {
	bin := "./bin/pair"
	if len(os.Args) > 1 {
		bin = os.Args[1]
	}
	if _, err := exec.LookPath("nvim"); err != nil {
		fmt.Println("SKIP  nvim not on PATH")
		return 0
	}

	sock := filepath.Join(os.TempDir(), fmt.Sprintf("escsmoke-%d.sock", os.Getpid()))
	defer os.Remove(sock)

	cmd := exec.Command(bin, "term")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "SHELL=/bin/sh")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pty.StartWithSize: %v\n", err)
		return 1
	}
	cleanup := func() {
		// Ask nvim to leave through its own socket so it does not outlive
		// the pty it was started in; then take the terminal down.
		_ = exec.Command("nvim", "--server", sock, "--remote-send", "<Esc>:qa!<CR>").Run()
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}
	defer cleanup()

	// Drain the pty: an undrained master blocks the child on its next write.
	go func() { _, _ = io.Copy(io.Discard, ptmx) }()

	send := func(s string) { _, _ = io.WriteString(ptmx, s) }
	ask := func(expr string) string {
		out, err := exec.Command("nvim", "--server", sock, "--remote-expr", expr).Output()
		if err != nil {
			return "<rpc error: " + err.Error() + ">"
		}
		return strings.TrimSpace(string(out))
	}

	time.Sleep(700 * time.Millisecond)
	send("nvim --clean --listen " + sock + "\r")
	if !waitFor(func() bool { _, err := os.Stat(sock); return err == nil }, 5*time.Second) {
		fmt.Println("FAIL  nvim never opened its socket")
		return 1
	}
	time.Sleep(300 * time.Millisecond)

	failures := 0
	step := func(name string, want map[string]string, do func()) {
		do()
		time.Sleep(200 * time.Millisecond)
		var bad []string
		for expr, wantValue := range want {
			if got := ask(expr); got != wantValue {
				bad = append(bad, fmt.Sprintf("%s = %q, want %q", expr, got, wantValue))
			}
		}
		if len(bad) == 0 {
			fmt.Printf("PASS  %s\n", name)
			return
		}
		failures++
		fmt.Printf("FAIL  %s\n      %s\n", name, strings.Join(bad, "; "))
	}

	step("i enters insert mode", map[string]string{"mode()": "i"}, func() { send("i") })
	// The operator's report: ONE press, then a pause with nothing typed after
	// it. Before #234 this ESC sat in pair term's held buffer until the next
	// keystroke, so mode() would still read "i" here.
	step("one ESC, nothing after it, leaves insert mode", map[string]string{"mode()": "n"}, func() {
		send("\x1b")
	})
	step("two lines of text, cursor on line 2", map[string]string{"mode()": "n", "line('.')": "2"}, func() {
		send("ione\rtwo")
		send("\x1b")
	})
	step("gg then i: insert mode on line 1", map[string]string{"mode()": "i", "line('.')": "1"}, func() {
		send("gg")
		time.Sleep(100 * time.Millisecond)
		send("i")
	})
	// ESC, then j after a human inter-key gap. Before #234 the held ESC and
	// the j were joined into Alt+j, which pair term consumed: nvim saw neither,
	// stayed in insert mode, and the cursor never moved.
	step("ESC then j after the deadline: normal mode, cursor moved down", map[string]string{"mode()": "n", "line('.')": "2"}, func() {
		send("\x1b")
		time.Sleep(120 * time.Millisecond)
		send("j")
	})
	// The other side of the deadline: a chord typed AS a chord arrives in one
	// write and is still pair term's, never nvim's. Had it been forwarded as
	// ESC+j, nvim in normal mode on line 1 would move down to line 2 — so
	// start on line 1 and check the cursor stays there.
	step("Alt+j as one write is still consumed by pair term", map[string]string{"mode()": "n", "line('.')": "1"}, func() {
		send("gg")
		time.Sleep(100 * time.Millisecond)
		send("\x1bj")
	})

	fmt.Println()
	if failures > 0 {
		fmt.Printf("%d step(s) failed\n", failures)
		return 1
	}
	fmt.Println("all steps passed")
	return 0
}

func waitFor(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
