// probes/mousemodesmoke drives a real `pair term` under a pty, puts nvim in
// tab 1 and a shell in tab 2, and logs every mouse DECSET/DECRST pair term
// writes to the OUTER terminal on each tab switch. It exists because #240's
// defect was invisible to content-marker probes (termsmoke) and to unit tests
// on fakes: zellij's view of the pane kept nvim's ?1002h after a switch to
// the shell, so the shell tab could not select. The property is "what modes
// the outer terminal was told", which only the bytes on the pty can show.
//
// Expected on a fixed build: the switch to the shell writes a DECRST of
// nvim's modes (a build without #240 writes nothing there), and the switch
// back writes the DECSET twice -- once as the reconcile prefix, once as
// nvim's own startup bytes in the replay; a build without #240 writes it
// once, and only for as long as the ring still holds it.
//
// Committed rather than run from a scratch dir: a probe whose output gets
// quoted in an issue Log has to be re-runnable against a later commit.
//
//	go run ./probes/mousemodesmoke            # against ./bin/pair
//	go run ./probes/mousemodesmoke /path/pair
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// main is two lines on purpose: os.Exit SKIPS DEFERS, and this probe's cleanup
// is registered as one. TestNoProbeExitsPastItsOwnCleanup enforces the shape.
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
	cmd := exec.Command(bin, "term")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "SHELL=/bin/sh")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer func() { _ = ptmx.Close(); _ = cmd.Process.Kill() }()
	var mu sync.Mutex
	var out strings.Builder
	go func() {
		buf := make([]byte, 65536)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				mu.Lock()
				out.Write(buf[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	take := func() string { mu.Lock(); defer mu.Unlock(); s := out.String(); out.Reset(); return s }
	modes := func(s string) string {
		f := regexp.MustCompile(`\x1b\[\?[0-9;]*(?:100[0236])[0-9;]*[hl]`).FindAllString(s, -1)
		if len(f) == 0 {
			return "(no mouse DECSET/DECRST)"
		}
		for i := range f {
			f[i] = strings.TrimPrefix(f[i], "\x1b[")
		}
		return strings.Join(f, " ")
	}
	send := func(s string) { _, _ = io.WriteString(ptmx, s) }
	time.Sleep(700 * time.Millisecond)
	take()
	send("nvim --clean\r")
	time.Sleep(1200 * time.Millisecond)
	fmt.Println("tab1 nvim start        :", modes(take()))
	send("\x1bt") // Alt+t: new shell tab
	time.Sleep(900 * time.Millisecond)
	fmt.Println("Alt+t -> tab2 (shell)  :", modes(take()))
	send("\x1b[1;3D") // Alt+Left back to nvim
	time.Sleep(900 * time.Millisecond)
	fmt.Println("Alt+Left -> tab1 (nvim):", modes(take()))
	send("\x1b[1;3C") // Alt+Right to shell again
	time.Sleep(900 * time.Millisecond)
	fmt.Println("Alt+Right -> tab2 shell:", modes(take()))
	send("\x1b[1;3D")
	time.Sleep(600 * time.Millisecond)
	send("\x1b:q!\r") // quit nvim in tab1
	time.Sleep(900 * time.Millisecond)
	fmt.Println("nvim :q! in tab1       :", modes(take()))
	return 0
}
