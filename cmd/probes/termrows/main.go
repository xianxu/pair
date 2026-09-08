package main

// What height does a `pair term` child actually believe it has?
//
//	make test-term-rows        # or: go run ./cmd/probes/termrows ./bin/pair
//
// Written because the operator's shell drew its right-hand prompt ON the
// strip's row, and the obvious explanation was a child sized to the whole pane.
// Reasoning said the sizing was right; this asked the child, and the child
// agreed -- both the first tab and one opened with Alt+t believe 23 rows in a
// 24-row pane. That RULED OUT the sizing and sent the search to the cursor
// save/restore instead, which is shared state the child also uses.
//
// Result 2026-09-08 against #199 M3:
//
//	pane is 24 rows; a correct child believes 23
//	  TAB1 believes 23 rows — OK
//	  TAB2 believes 23 rows — OK

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

type buf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (x *buf) Write(p []byte) (int, error) {
	x.mu.Lock()
	x.b.Write(p)
	x.mu.Unlock()
	return len(p), nil
}
func (x *buf) String() string { x.mu.Lock(); defer x.mu.Unlock(); return x.b.String() }

// main is two lines on purpose: os.Exit SKIPS DEFERS, and this probe's cleanup
// (which tears down the child and its pty) is registered as one. Every path RETURNS a
// code so the defers unwind. TestNoProbeExitsPastItsOwnCleanup enforces the
// shape across every probe (pair#199 BR-69).
func main() { os.Exit(run()) }

func run() int {
	cmd := exec.Command(os.Args[1], "term")
	env := []string{}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "ZELLIJ") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "TERM=xterm-256color", "SHELL=/bin/sh", "PS1=$ ")

	const paneRows, paneCols = 24, 80
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: paneRows, Cols: paneCols})
	if err != nil {
		fmt.Println("PROBE-ERROR:", err)
		return 1
	}
	defer func() { _ = cmd.Process.Kill(); f.Close() }()

	out := &buf{}
	go func() {
		b := make([]byte, 1<<16)
		for {
			n, e := f.Read(b)
			if n > 0 {
				out.Write(b[:n])
			}
			if e != nil {
				return
			}
		}
	}()

	ask := func(label string) {
		time.Sleep(1200 * time.Millisecond)
		_, _ = f.Write([]byte("echo " + label + "=$(tput lines)\n"))
		time.Sleep(1200 * time.Millisecond)
	}

	ask("TAB1")
	_, _ = f.Write([]byte{0x1b, 't'}) // Alt+t: new tab
	ask("TAB2")

	got := out.String()
	re := regexp.MustCompile(`TAB([12])=(\d+)`)
	seen := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(got, -1) {
		seen["TAB"+m[1]] = m[2]
	}
	fmt.Printf("pane is %d rows; a correct child believes %d\n", paneRows, paneRows-1)
	for _, k := range []string{"TAB1", "TAB2"} {
		v, ok := seen[k]
		if !ok {
			fmt.Printf("  %s: NO ANSWER (the child never reported)\n", k)
			continue
		}
		verdict := "OK"
		if v != fmt.Sprint(paneRows-1) {
			verdict = "WRONG — it can scroll onto the reserved row"
		}
		fmt.Printf("  %s believes %s rows — %s\n", k, v, verdict)
	}
	if len(seen) == 0 {
		fmt.Println("PROBE-INCONCLUSIVE: no child answered; nothing was measured")
		return 2
	}
	return 0
}
