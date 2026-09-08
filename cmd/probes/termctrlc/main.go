package main

// Does Ctrl-C stop a flooding child in `pair term`, and how long does the
// backlog take to drain?
//
//	make test-term-ctrlc            # or: go run ./cmd/probes/termctrlc ./bin/pair
//
// It lives under cmd/probes/ rather than probes/ because it takes an argument
// (the pair binary) that `make test-smoke`'s wholesale loop cannot supply --
// the documented exception in atlas/index.md -- and because a 270 MB flood is
// not something to run on every smoke pass.
//
// WHY IT EXISTS. The operator reported "I can't ctrl-c to stop `yes aaa`" after
// pair#199 M3 put a reserved row in the pane. That report is equally consistent
// with input being LOST and with input WORKING while a buffered backlog keeps
// rendering, and the two have opposite fixes.
//
// Result on 2026-09-08, against #199 M3's binary:
//
//	flood produced 5.5 MB in 3s
//	QUIET after Ctrl-C: 0.04s (drained 0.1 MB of backlog)
//	RESULT: Ctrl-C STOPPED the child.
//
// So `pair term` is not where the input goes missing. The reader here is
// deliberately THROTTLED to ~2 MB/s: an unthrottled one drains at 274 MB/s, no
// backlog ever forms, and Ctrl-C looks instant no matter what the code does --
// a probe that cannot reproduce the operator's condition cannot exonerate it.

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
)

type counter struct {
	mu sync.Mutex
	n  int
	at time.Time
}

func (c *counter) Write(p []byte) (int, error) {
	c.mu.Lock()
	c.n += len(p)
	c.at = time.Now()
	c.mu.Unlock()
	return len(p), nil
}
func (c *counter) snap() (int, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n, c.at
}

// main is two lines on purpose: os.Exit SKIPS DEFERS, and this probe's cleanup
// (which tears down the child and its pty) is registered as one. Every path RETURNS a
// code so the defers unwind. TestNoProbeExitsPastItsOwnCleanup enforces the
// shape across every probe (pair#199 BR-69).
func main() { os.Exit(run()) }

func run() int {
	bin := os.Args[1]
	cmd := exec.Command(bin, "term")
	env := []string{}
	for _, kv := range os.Environ() {
		if len(kv) >= 6 && kv[:6] == "ZELLIJ" {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "TERM=xterm-256color", "SHELL=/bin/sh")

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Println("PROBE-ERROR start:", err)
		return 1
	}
	defer func() { _ = cmd.Process.Kill(); f.Close() }()

	// A THROTTLED reader, because that is the condition the operator is in and
	// an unthrottled one cannot reproduce it: a real terminal renders at some
	// finite rate, so a flood builds a backlog in front of it. Reading as fast
	// as the pty can produce (274 MB/s in the first version of this probe)
	// means no backlog ever forms and Ctrl-C always looks instant.
	const bytesPerSecond = 2 << 20 // ~2 MB/s, a generous terminal
	c := &counter{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				c.Write(buf[:n])
				time.Sleep(time.Duration(n) * time.Second / bytesPerSecond)
			}
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)
	before, _ := c.snap()
	if before == 0 {
		fmt.Println("PROBE-INCONCLUSIVE: pair term produced no output; nothing was measured")
		return 2
	}

	_, _ = f.Write([]byte("yes aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"))
	time.Sleep(3 * time.Second)
	flooded, _ := c.snap()
	if flooded-before < 100000 {
		fmt.Printf("PROBE-INCONCLUSIVE: only %d bytes flowed; the flood did not start\n", flooded-before)
		return 2
	}
	fmt.Printf("flood produced %.1f MB in 3s\n", float64(flooded-before)/1e6)

	sent := time.Now()
	_, _ = f.Write([]byte{0x03})

	// Quiet = 400ms with no new bytes.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		_, last := c.snap()
		if time.Since(last) > 400*time.Millisecond {
			total, _ := c.snap()
			fmt.Printf("QUIET after Ctrl-C: %.2fs (drained %.1f MB of backlog)\n",
				last.Sub(sent).Seconds(), float64(total-flooded)/1e6)
			fmt.Println("RESULT: Ctrl-C STOPPED the child.")
			return 0
		}
	}
	total, _ := c.snap()
	fmt.Printf("RESULT: STILL FLOODING 30s after Ctrl-C (%.1f MB since)\n",
		float64(total-flooded)/1e6)
	return 1
}
