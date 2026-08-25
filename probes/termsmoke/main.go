// probes/termsmoke drives a real `pair term` under a pty and reports what the
// operator would see. It exists because #146 M1 migrated termcmd's multiplexer
// onto ptychild + hostty, and the property that matters -- landing on a tab
// repaints its content -- is a visual one that unit tests only approximate.
//
// Committed rather than run from a scratch dir: a probe whose output gets
// quoted in an issue Log has to be re-runnable against a later commit.
//
//	go run ./probes/termsmoke            # against ./bin/pair
//	go run ./probes/termsmoke /path/pair
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

const (
	altT     = "\x1bt"
	altLeft  = "\x1b[1;3D"
	altRight = "\x1b[1;3C"
)

func main() {
	bin := "./bin/pair"
	if len(os.Args) > 1 {
		bin = os.Args[1]
	}

	cmd := exec.Command(bin, "term")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "SHELL=/bin/sh")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pty.StartWithSize: %v\n", err)
		os.Exit(1)
	}
	// Not a defer: the failure path below calls os.Exit, which skips defers and
	// would leave the spawned pair process alive.
	cleanup := func() { _ = ptmx.Close(); _ = cmd.Process.Kill() }
	defer cleanup()

	var mu sync.Mutex
	var seen strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				mu.Lock()
				seen.Write(buf[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	output := func() string { mu.Lock(); defer mu.Unlock(); return seen.String() }
	reset := func() { mu.Lock(); seen.Reset(); mu.Unlock() }
	send := func(s string) { _, _ = io.WriteString(ptmx, s); time.Sleep(400 * time.Millisecond) }

	failures := 0
	step := func(name, want string, do func()) {
		reset()
		do()
		got := output()
		if strings.Contains(got, want) {
			fmt.Printf("PASS  %-44s (found %q)\n", name, want)
			return
		}
		failures++
		fmt.Printf("FAIL  %-44s (wanted %q)\n      saw: %q\n", name, want, trim(got))
	}

	time.Sleep(700 * time.Millisecond)
	fmt.Printf("startup output: %q\n\n", trim(output()))

	step("tab 1 runs a command", "MARKER-ONE", func() { send("echo MARKER-ONE\r") })
	step("Alt+t opens a second tab", "MARKER-TWO", func() { send(altT); send("echo MARKER-TWO\r") })
	step("Alt+Left repaints tab 1 from its ring", "MARKER-ONE", func() { send(altLeft) })
	step("Alt+Right repaints tab 2 from its ring", "MARKER-TWO", func() { send(altRight) })
	step("resize reaches the child", "40 100", func() {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 100})
		send("stty size\r")
	})
	step("still usable afterwards", "STILL-ALIVE", func() { send("echo STILL-ALIVE\r") })

	// The alt-screen case, which is the one #146's plan flags as risky: a full
	// screen app's ring holds partial redraws, not a screen. Landing on it must
	// still repaint something coherent rather than a blank pane.
	if _, err := exec.LookPath("nvim"); err != nil {
		fmt.Println("SKIP  nvim alt-screen round trip                  (nvim not on PATH)")
	} else {
		step("nvim enters the alt screen", "\x1b[?1049h", func() {
			send("nvim -u NONE -c 'norm iALT-SCREEN-MARKER'\r")
			time.Sleep(1200 * time.Millisecond)
		})
		step("switching away and back repaints nvim", "ALT-SCREEN-MARKER", func() {
			send(altT)
			time.Sleep(300 * time.Millisecond)
			send(altLeft)
			time.Sleep(600 * time.Millisecond)
		})
		// Leave nvim so the harness does not exit with it holding the tty.
		send("\x1b:q!\r")
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println()
	if failures > 0 {
		fmt.Printf("%d step(s) failed\n", failures)
		cleanup()
		os.Exit(1)
	}
	fmt.Println("all steps passed")
}

func trim(s string) string {
	if len(s) > 400 {
		return s[:200] + " … " + s[len(s)-200:]
	}
	return s
}
